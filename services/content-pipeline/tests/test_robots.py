from __future__ import annotations

from content_pipeline.robots import robots_allows

ROBOTS = """
User-agent: *
Disallow: /private/
Allow: /
"""


def test_missing_robots_is_permissive() -> None:
    assert robots_allows(None, "bot", "https://example.test/private/x")


def test_disallowed_path_is_blocked() -> None:
    assert not robots_allows(ROBOTS, "bot", "https://example.test/private/x")


def test_allowed_path_passes() -> None:
    assert robots_allows(ROBOTS, "bot", "https://example.test/public/x")


def test_blanket_disallow_blocks_everything() -> None:
    assert not robots_allows("User-agent: *\nDisallow: /\n", "bot", "https://example.test/")
