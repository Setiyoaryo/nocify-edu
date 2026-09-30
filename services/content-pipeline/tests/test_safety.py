from __future__ import annotations

import ipaddress

import pytest

from content_pipeline.safety import (
    UnsafeURLError,
    assert_allowed,
    assert_public_host,
    canonicalize_url,
    host_matches,
    is_forbidden_ip,
)


def test_canonicalize_normalises_host_scheme_port_and_query() -> None:
    assert canonicalize_url("HTTPS://Example.Test:443/a/b?b=2&a=1#frag") == "https://example.test/a/b?a=1&b=2"


def test_canonicalize_keeps_path_case() -> None:
    assert canonicalize_url("https://example.test/CaseSensitive/Path") == "https://example.test/CaseSensitive/Path"


def test_canonicalize_keeps_non_default_port() -> None:
    assert canonicalize_url("http://example.test:8080/x") == "http://example.test:8080/x"


@pytest.mark.parametrize("url", ["ftp://example.test/x", "file:///etc/passwd", "https:///nohost"])
def test_canonicalize_rejects_unsupported_urls(url: str) -> None:
    with pytest.raises(UnsafeURLError):
        canonicalize_url(url)


def test_host_matches_exact_and_subdomain_only() -> None:
    allowed = {"example.test"}
    assert host_matches("example.test", allowed)
    assert host_matches("sub.example.test", allowed)
    assert not host_matches("notexample.test", allowed)
    assert not host_matches("example.test.evil.com", allowed)


@pytest.mark.parametrize(
    "ip",
    ["127.0.0.1", "10.0.0.1", "192.168.1.1", "172.16.0.1", "169.254.169.254", "::1", "0.0.0.0"],
)
def test_forbidden_ips(ip: str) -> None:
    assert is_forbidden_ip(ipaddress.ip_address(ip))


def test_public_ip_is_allowed() -> None:
    assert not is_forbidden_ip(ipaddress.ip_address("8.8.8.8"))


def test_assert_allowed_rejects_host_outside_allowlist() -> None:
    with pytest.raises(UnsafeURLError, match="not allowlisted"):
        assert_allowed("https://evil.test/x", {"example.test"})


def test_assert_allowed_accepts_public_host(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(
        "content_pipeline.safety.resolve_host",
        lambda host, port: {ipaddress.ip_address("93.184.216.34")},
    )
    assert assert_allowed("https://example.test/x", {"example.test"}) == "example.test"


def test_assert_public_host_rejects_private_resolution(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(
        "content_pipeline.safety.resolve_host",
        lambda host, port: {ipaddress.ip_address("10.0.0.5")},
    )
    with pytest.raises(UnsafeURLError, match="non-public"):
        assert_public_host("internal.example.test", 443)


def test_assert_public_host_rejects_mixed_public_and_private(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(
        "content_pipeline.safety.resolve_host",
        lambda host, port: {ipaddress.ip_address("93.184.216.34"), ipaddress.ip_address("10.0.0.5")},
    )
    with pytest.raises(UnsafeURLError):
        assert_public_host("rebind.example.test", 443)
