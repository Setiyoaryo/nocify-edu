"""robots.txt handling.

The crawler fetches robots.txt through the same SSRF-safe fetcher, then parses
it. When robots.txt is missing or unreadable we treat the site as *allowed* but
the caller records that fact; sources whose policy forbids crawling are paused
rather than bypassed (§43).
"""

from __future__ import annotations

from urllib.robotparser import RobotFileParser


def robots_allows(robots_txt: str | None, user_agent: str, url: str) -> bool:
    """Return True when *url* may be fetched by *user_agent*.

    ``None`` means robots.txt could not be retrieved; the planning document
    treats that as permissive, so we allow it and leave the decision to the
    human source-approval step.
    """
    if robots_txt is None:
        return True
    parser = RobotFileParser()
    parser.parse(robots_txt.splitlines())
    return parser.can_fetch(user_agent, url)
