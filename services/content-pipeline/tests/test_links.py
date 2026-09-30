from __future__ import annotations

from content_pipeline.links import discover_links

BASE = "https://example.test/docs/index.html"
ALLOWED = {"example.test"}

HTML = """
<html><body>
  <a href="/a">A</a>
  <a href="b.html">B</a>
  <a href="https://example.test/a">A duplicate</a>
  <a href="#section">fragment only</a>
  <a href="mailto:hi@example.test">mail</a>
  <a href="https://other.test/x">external</a>
  <a href="https://sub.example.test/c">subdomain</a>
  <a href="javascript:void(0)">js</a>
</body></html>
"""


def test_discover_links_filters_and_deduplicates() -> None:
    links = discover_links(HTML, BASE, ALLOWED)
    assert links == [
        "https://example.test/a",
        "https://example.test/docs/b.html",
        "https://sub.example.test/c",
    ]


def test_discover_links_is_deterministic() -> None:
    assert discover_links(HTML, BASE, ALLOWED) == discover_links(HTML, BASE, ALLOWED)


def test_discover_links_ignores_non_allowlisted_hosts() -> None:
    html = '<a href="https://evil.test/x">evil</a>'
    assert discover_links(html, BASE, ALLOWED) == []
