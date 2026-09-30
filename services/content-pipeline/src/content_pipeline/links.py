"""Link discovery for crawling.

Kept as a pure function so it is easy to test: given HTML and the allowlist,
return the same-host URLs worth fetching, in a deterministic order.
"""

from __future__ import annotations

from urllib.parse import urldefrag, urljoin, urlsplit

from bs4 import BeautifulSoup

from content_pipeline.safety import UnsafeURLError, canonicalize_url, host_matches

_SKIP_SCHEMES = {"mailto", "tel", "javascript", "data", "ftp"}


def discover_links(html: str, base_url: str, allowed_hosts: set[str]) -> list[str]:
    """Return canonical, same-host links found in *html*, deduplicated.

    Only http(s) links whose host is allowlisted are returned. Fragments are
    dropped. Order is deterministic (document order, first occurrence wins).
    """
    soup = BeautifulSoup(html, "html.parser")
    seen: set[str] = set()
    out: list[str] = []
    for anchor in soup.find_all("a", href=True):
        href = str(anchor["href"]).strip()
        if not href or href.startswith("#"):
            continue
        if urlsplit(href).scheme.lower() in _SKIP_SCHEMES:
            continue
        absolute = urldefrag(urljoin(base_url, href))[0]
        parts = urlsplit(absolute)
        if parts.scheme.lower() not in {"http", "https"}:
            continue
        host = (parts.hostname or "").lower().rstrip(".")
        if not host or not host_matches(host, allowed_hosts):
            continue
        try:
            canonical = canonicalize_url(absolute)
        except UnsafeURLError:
            continue
        if canonical in seen:
            continue
        seen.add(canonical)
        out.append(canonical)
    return out
