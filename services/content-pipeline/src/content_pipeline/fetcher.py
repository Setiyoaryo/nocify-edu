"""SSRF-safe HTTP fetcher with conditional requests.

Redirects are followed manually so every hop is validated against the allowlist
and the public-IP rule. Response bodies are streamed and capped. The raw bytes
are returned to the caller, which decides whether to archive them.
"""

from __future__ import annotations

import hashlib
from dataclasses import dataclass

import httpx

from content_pipeline.safety import assert_allowed, canonicalize_url

# Content types we are willing to archive. Anything else is recorded as skipped.
ALLOWED_CONTENT_TYPES = (
    "text/html",
    "text/plain",
    "text/markdown",
    "application/pdf",
    "application/xml",
    "application/json",
    "application/vnd.openxmlformats-officedocument",
    "application/msword",
)

_REDIRECT_STATUSES = {301, 302, 303, 307, 308}


class FetchError(RuntimeError):
    """A fetch failed for a reason the caller should record on the crawl item."""


class ResponseTooLargeError(FetchError):
    pass


class DisallowedContentTypeError(FetchError):
    pass


class TooManyRedirectsError(FetchError):
    pass


@dataclass(frozen=True)
class FetchResult:
    url: str
    canonical_url: str
    http_status: int
    content_type: str
    etag: str
    last_modified: str
    content_hash: str
    content_length: int
    not_modified: bool
    body: bytes | None


def content_type_allowed(content_type: str) -> bool:
    base = content_type.split(";", 1)[0].strip().lower()
    return any(base.startswith(allowed) for allowed in ALLOWED_CONTENT_TYPES)


def sha256_hex(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


class Fetcher:
    """Fetches URLs that belong to an allowlist of hosts.

    ``client`` can be injected for tests (e.g. an ``httpx.MockTransport``).
    """

    def __init__(
        self,
        *,
        allowed_hosts: set[str],
        user_agent: str,
        timeout_seconds: float = 20.0,
        max_response_bytes: int = 10 * 1024 * 1024,
        max_redirects: int = 5,
        client: httpx.Client | None = None,
    ) -> None:
        self._allowed_hosts = allowed_hosts
        self._max_response_bytes = max_response_bytes
        self._max_redirects = max_redirects
        self._owns_client = client is None
        self._client = client or httpx.Client(
            timeout=timeout_seconds,
            headers={"User-Agent": user_agent, "Accept": "*/*"},
        )

    def close(self) -> None:
        if self._owns_client:
            self._client.close()

    def __enter__(self) -> Fetcher:
        return self

    def __exit__(self, *_: object) -> None:
        self.close()

    def fetch(self, url: str, *, etag: str = "", last_modified: str = "") -> FetchResult:
        """Fetch *url*, validating every redirect hop.

        Sends ``If-None-Match``/``If-Modified-Since`` when the caller passes the
        previous ETag/Last-Modified. A 304 comes back as ``not_modified=True``
        with no body.
        """
        headers: dict[str, str] = {}
        if etag:
            headers["If-None-Match"] = etag
        if last_modified:
            headers["If-Modified-Since"] = last_modified

        current = url
        for _ in range(self._max_redirects + 1):
            assert_allowed(current, self._allowed_hosts)
            response = self._request(current, headers)
            if response.status_code in _REDIRECT_STATUSES:
                location = response.headers.get("location")
                response.close()
                if not location:
                    raise FetchError(f"redirect without Location from {current}")
                current = str(httpx.URL(current).join(location))
                headers = {}  # conditional headers only apply to the origin URL
                continue
            return self._read(current, response)

        raise TooManyRedirectsError(f"more than {self._max_redirects} redirects for {url}")

    def _request(self, url: str, headers: dict[str, str]) -> httpx.Response:
        request = self._client.build_request("GET", url, headers=headers)
        return self._client.send(request, stream=True, follow_redirects=False)

    def _read(self, url: str, response: httpx.Response) -> FetchResult:
        content_type = response.headers.get("content-type", "")
        etag = response.headers.get("etag", "")
        last_modified = response.headers.get("last-modified", "")

        if response.status_code == 304:
            response.close()
            return FetchResult(
                url=url,
                canonical_url=canonicalize_url(url),
                http_status=304,
                content_type=content_type,
                etag=etag,
                last_modified=last_modified,
                content_hash="",
                content_length=0,
                not_modified=True,
                body=None,
            )

        try:
            response.raise_for_status()
        except httpx.HTTPStatusError as exc:
            response.close()
            raise FetchError(f"HTTP {response.status_code} for {url}") from exc

        if content_type and not content_type_allowed(content_type):
            response.close()
            raise DisallowedContentTypeError(f"content-type not allowed: {content_type!r}")

        body = self._read_body(response)
        return FetchResult(
            url=url,
            canonical_url=canonicalize_url(url),
            http_status=response.status_code,
            content_type=content_type,
            etag=etag,
            last_modified=last_modified,
            content_hash=sha256_hex(body),
            content_length=len(body),
            not_modified=False,
            body=body,
        )

    def _read_body(self, response: httpx.Response) -> bytes:
        chunks: list[bytes] = []
        size = 0
        try:
            for chunk in response.iter_bytes():
                size += len(chunk)
                if size > self._max_response_bytes:
                    raise ResponseTooLargeError(
                        f"response exceeded {self._max_response_bytes} bytes"
                    )
                chunks.append(chunk)
        finally:
            response.close()
        return b"".join(chunks)
