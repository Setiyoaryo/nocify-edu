from __future__ import annotations

from datetime import UTC, datetime

from content_pipeline.storage import extension_for, local_filename, normalise_endpoint, raw_key


def test_raw_key_layout() -> None:
    key = raw_key("peraturan-bpk", "abc123", "text/html", datetime(2026, 9, 30, tzinfo=UTC))
    assert key == "raw/peraturan-bpk/2026/09/abc123.html"


def test_raw_key_strips_charset_from_content_type() -> None:
    key = raw_key("src", "h", "text/html; charset=utf-8", datetime(2026, 1, 2, tzinfo=UTC))
    assert key.endswith(".html")


def test_extension_for_known_and_unknown() -> None:
    assert extension_for("application/pdf") == ".pdf"
    assert extension_for("application/octet-stream") == ".bin"


def test_local_filename() -> None:
    assert local_filename("raw/s/2026/09/deadbeef.pdf") == "deadbeef.pdf"


def test_normalise_endpoint_adds_scheme() -> None:
    assert normalise_endpoint("s3:8333", use_ssl=False) == "http://s3:8333"
    assert normalise_endpoint("s3:8333", use_ssl=True) == "https://s3:8333"
    assert normalise_endpoint("https://s3.example.com", use_ssl=False) == "https://s3.example.com"
    assert normalise_endpoint("", use_ssl=False) is None
