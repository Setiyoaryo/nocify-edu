"""URL safety: canonicalisation, allowlist matching and SSRF protection.

Everything the crawler fetches must pass through here. The rules are the ones
in the planning document (§42, §43): only public IPs, only allowlisted hosts.
"""

from __future__ import annotations

import ipaddress
import socket
from urllib.parse import parse_qsl, urlencode, urlsplit, urlunsplit

_DEFAULT_PORTS = {"http": 80, "https": 443}


class UnsafeURLError(ValueError):
    """Raised when a URL is malformed or resolves to a non-public address."""


def canonicalize_url(url: str) -> str:
    """Return a stable form of *url* used as the deduplication key.

    Lowercases scheme/host, drops default ports and fragments, and sorts query
    parameters. Path case is preserved because it is case-sensitive on most
    servers.
    """
    parts = urlsplit(url.strip())
    scheme = parts.scheme.lower()
    if scheme not in _DEFAULT_PORTS:
        raise UnsafeURLError(f"unsupported scheme: {parts.scheme!r}")
    host = (parts.hostname or "").lower().rstrip(".")
    if not host:
        raise UnsafeURLError(f"missing host in {url!r}")
    port = parts.port
    netloc = host if port in (None, _DEFAULT_PORTS[scheme]) else f"{host}:{port}"
    path = parts.path or "/"
    query = urlencode(sorted(parse_qsl(parts.query, keep_blank_values=True)))
    return urlunsplit((scheme, netloc, path, query, ""))


def host_matches(host: str, allowed_hosts: set[str]) -> bool:
    """True when *host* is an allowed host or a subdomain of one."""
    host = host.lower().rstrip(".")
    for allowed in allowed_hosts:
        allowed = allowed.lower().rstrip(".")
        if host == allowed or host.endswith("." + allowed):
            return True
    return False


def host_of(url: str) -> str:
    host = (urlsplit(url).hostname or "").lower().rstrip(".")
    if not host:
        raise UnsafeURLError(f"missing host in {url!r}")
    return host


def is_forbidden_ip(ip: ipaddress.IPv4Address | ipaddress.IPv6Address) -> bool:
    """Reject anything that is not a globally routable address.

    This covers loopback, private ranges, link-local (including the cloud
    metadata address 169.254.169.254), multicast, reserved and unspecified.
    """
    return not ip.is_global


def resolve_host(host: str, port: int) -> set[ipaddress.IPv4Address | ipaddress.IPv6Address]:
    infos = socket.getaddrinfo(host, port, proto=socket.IPPROTO_TCP)
    return {ipaddress.ip_address(info[4][0]) for info in infos}


def assert_public_host(host: str, port: int) -> None:
    """Raise UnsafeURLError unless *host* resolves only to public IPs.

    All resolved addresses must be public: a host that answers with one public
    and one private address is still rejected (DNS rebinding defence).
    """
    try:
        addresses = resolve_host(host, port)
    except socket.gaierror as exc:
        raise UnsafeURLError(f"cannot resolve {host!r}: {exc}") from exc
    if not addresses:
        raise UnsafeURLError(f"{host!r} resolved to no addresses")
    for address in addresses:
        if is_forbidden_ip(address):
            raise UnsafeURLError(f"{host!r} resolves to non-public address {address}")


def assert_allowed(url: str, allowed_hosts: set[str]) -> str:
    """Validate *url* against the allowlist and SSRF rules; return its host.

    Raises UnsafeURLError when the host is not allowlisted or does not resolve
    to public addresses.
    """
    parts = urlsplit(url)
    if parts.scheme.lower() not in _DEFAULT_PORTS:
        raise UnsafeURLError(f"unsupported scheme: {parts.scheme!r}")
    host = host_of(url)
    if not host_matches(host, allowed_hosts):
        raise UnsafeURLError(f"host not allowlisted: {host}")
    port = parts.port or _DEFAULT_PORTS[parts.scheme.lower()]
    assert_public_host(host, port)
    return host
