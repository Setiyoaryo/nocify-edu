from __future__ import annotations

import ipaddress
from collections.abc import Callable

import httpx
import pytest

from content_pipeline.fetcher import (
    DisallowedContentTypeError,
    Fetcher,
    FetchError,
    ResponseTooLargeError,
    sha256_hex,
)
from content_pipeline.safety import UnsafeURLError

PUBLIC_IP = ipaddress.ip_address("93.184.216.34")
Handler = Callable[[httpx.Request], httpx.Response]


@pytest.fixture(autouse=True)
def _public_dns(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr("content_pipeline.safety.resolve_host", lambda host, port: {PUBLIC_IP})


def _fetcher(handler: Handler, **kwargs: object) -> Fetcher:
    client = httpx.Client(transport=httpx.MockTransport(handler), follow_redirects=False)
    return Fetcher(allowed_hosts={"example.test"}, user_agent="test-bot", client=client, **kwargs)  # type: ignore[arg-type]


def test_fetch_returns_hash_and_body() -> None:
    def handler(_: httpx.Request) -> httpx.Response:
        return httpx.Response(200, content=b"hello", headers={"content-type": "text/html"})

    with _fetcher(handler) as fetcher:
        result = fetcher.fetch("https://example.test/page")
    assert result.http_status == 200
    assert result.body == b"hello"
    assert result.content_hash == sha256_hex(b"hello")
    assert result.not_modified is False


def test_fetch_sends_conditional_headers_and_handles_304() -> None:
    seen: dict[str, str] = {}

    def handler(request: httpx.Request) -> httpx.Response:
        seen.update(request.headers)
        return httpx.Response(304, headers={"etag": '"abc"'})

    with _fetcher(handler) as fetcher:
        result = fetcher.fetch("https://example.test/page", etag='"abc"', last_modified="Mon, 01 Jan 2024 00:00:00 GMT")
    assert seen.get("if-none-match") == '"abc"'
    assert "if-modified-since" in seen
    assert result.not_modified is True
    assert result.body is None


def test_fetch_rejects_disallowed_content_type() -> None:
    def handler(_: httpx.Request) -> httpx.Response:
        return httpx.Response(200, content=b"\x00\x01", headers={"content-type": "application/octet-stream"})

    with _fetcher(handler) as fetcher, pytest.raises(DisallowedContentTypeError):
        fetcher.fetch("https://example.test/file.bin")


def test_fetch_enforces_size_cap() -> None:
    def handler(_: httpx.Request) -> httpx.Response:
        return httpx.Response(200, content=b"x" * 2048, headers={"content-type": "text/html"})

    with _fetcher(handler, max_response_bytes=1024) as fetcher, pytest.raises(ResponseTooLargeError):
        fetcher.fetch("https://example.test/big")


def test_fetch_raises_on_http_error() -> None:
    def handler(_: httpx.Request) -> httpx.Response:
        return httpx.Response(500, content=b"nope", headers={"content-type": "text/html"})

    with _fetcher(handler) as fetcher, pytest.raises(FetchError):
        fetcher.fetch("https://example.test/boom")


def test_fetch_follows_redirect_to_allowed_host() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        if request.url.path == "/old":
            return httpx.Response(301, headers={"location": "https://example.test/new"})
        return httpx.Response(200, content=b"ok", headers={"content-type": "text/html"})

    with _fetcher(handler) as fetcher:
        result = fetcher.fetch("https://example.test/old")
    assert result.canonical_url == "https://example.test/new"
    assert result.body == b"ok"


def test_fetch_rejects_redirect_to_host_outside_allowlist() -> None:
    def handler(_: httpx.Request) -> httpx.Response:
        return httpx.Response(302, headers={"location": "https://evil.test/x"})

    with _fetcher(handler) as fetcher, pytest.raises(UnsafeURLError):
        fetcher.fetch("https://example.test/old")


def test_fetch_rejects_url_outside_allowlist() -> None:
    def handler(_: httpx.Request) -> httpx.Response:  # pragma: no cover - never reached
        return httpx.Response(200, content=b"", headers={"content-type": "text/html"})

    with _fetcher(handler) as fetcher, pytest.raises(UnsafeURLError):
        fetcher.fetch("https://evil.test/x")
