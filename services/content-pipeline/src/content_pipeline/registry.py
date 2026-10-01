"""Source registry CRUD (§4, §5).

The crawler never hardcodes URLs: every fetchable source lives here, with a
governance lifecycle. New sources start ``discovered`` and disabled, so nothing
is crawled until a human moves it to ``active``.
"""

from __future__ import annotations

from dataclasses import dataclass
from datetime import datetime
from typing import Any

from content_pipeline.db import Database

# The initial official allowlist (§45). Inserted disabled so they cannot be
# crawled until verified and activated.
DEFAULT_SOURCES: tuple[dict[str, object], ...] = (
    {
        "code": "kemendikdasmen-kurikulum",
        "name": "Kurikulum Kemendikdasmen",
        "publisher": "Kemendikdasmen",
        "base_url": "https://kurikulum.kemendikdasmen.go.id/",
        "source_type": "official_curriculum",
        "tier": 1,
    },
    {
        "code": "kemendikdasmen-guru",
        "name": "Portal Guru Kemendikdasmen",
        "publisher": "Kemendikdasmen",
        "base_url": "https://guru.kemendikdasmen.go.id/",
        "source_type": "institution",
        "tier": 2,
    },
    {
        "code": "kemendikdasmen",
        "name": "Kemendikdasmen",
        "publisher": "Kemendikdasmen",
        "base_url": "https://www.kemendikdasmen.go.id/",
        "source_type": "official_curriculum",
        "tier": 1,
    },
    {
        "code": "kemendikdasmen-repositori",
        "name": "Repositori Kemendikdasmen",
        "publisher": "Kemendikdasmen",
        "base_url": "https://repositori.kemendikdasmen.go.id/",
        "source_type": "institution",
        "tier": 2,
    },
    {
        "code": "peraturan-bpk",
        "name": "JDIH BPK",
        "publisher": "BPK",
        "base_url": "https://peraturan.bpk.go.id/",
        "source_type": "official_curriculum",
        "tier": 1,
    },
)

_COLUMNS = (
    "id, code, name, publisher, base_url, source_type, tier, license_type, license_url, "
    "terms_url, robots_policy, crawl_frequency, priority, enabled, status, max_pages, "
    "rate_limit_ms, last_crawled_at"
)


@dataclass(frozen=True)
class Source:
    id: int
    code: str
    name: str
    publisher: str
    base_url: str
    source_type: str
    tier: int
    license_type: str
    license_url: str
    terms_url: str
    robots_policy: str
    crawl_frequency: str
    priority: int
    enabled: bool
    status: str
    max_pages: int
    rate_limit_ms: int
    last_crawled_at: datetime | None


def _to_source(row: tuple[Any, ...]) -> Source:
    return Source(
        id=int(row[0]),
        code=str(row[1]),
        name=str(row[2]),
        publisher=str(row[3]),
        base_url=str(row[4]),
        source_type=str(row[5]),
        tier=int(row[6]),
        license_type=str(row[7]),
        license_url=str(row[8]),
        terms_url=str(row[9]),
        robots_policy=str(row[10]),
        crawl_frequency=str(row[11]),
        priority=int(row[12]),
        enabled=bool(row[13]),
        status=str(row[14]),
        max_pages=int(row[15]),
        rate_limit_ms=int(row[16]),
        last_crawled_at=row[17],
    )


def add_source(
    db: Database,
    *,
    code: str,
    name: str,
    base_url: str,
    source_type: str,
    tier: int,
    publisher: str = "",
    license_type: str = "unknown",
    license_url: str = "",
    terms_url: str = "",
    max_pages: int = 50,
    rate_limit_ms: int = 1000,
) -> Source:
    with db.transaction() as cur:
        cur.execute(
            f"""
            INSERT INTO content_staging.source_registry
                (code, name, publisher, base_url, source_type, tier, license_type,
                 license_url, terms_url, max_pages, rate_limit_ms)
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
            ON CONFLICT (code) DO UPDATE SET
                name = EXCLUDED.name,
                publisher = EXCLUDED.publisher,
                base_url = EXCLUDED.base_url,
                source_type = EXCLUDED.source_type,
                tier = EXCLUDED.tier,
                license_type = EXCLUDED.license_type,
                license_url = EXCLUDED.license_url,
                terms_url = EXCLUDED.terms_url,
                max_pages = EXCLUDED.max_pages,
                rate_limit_ms = EXCLUDED.rate_limit_ms,
                updated_at = now()
            RETURNING {_COLUMNS}
            """,
            (
                code, name, publisher, base_url, source_type, tier, license_type,
                license_url, terms_url, max_pages, rate_limit_ms,
            ),
        )
        row = cur.fetchone()
    assert row is not None
    return _to_source(row)


def list_sources(db: Database) -> list[Source]:
    with db.transaction() as cur:
        cur.execute(f"SELECT {_COLUMNS} FROM content_staging.source_registry ORDER BY priority, code")
        return [_to_source(row) for row in cur.fetchall()]


def get_source(db: Database, code: str) -> Source | None:
    with db.transaction() as cur:
        cur.execute(
            f"SELECT {_COLUMNS} FROM content_staging.source_registry WHERE code = %s",
            (code,),
        )
        row = cur.fetchone()
    return _to_source(row) if row is not None else None


def active_sources(db: Database, codes: list[str] | None = None) -> list[Source]:
    """Sources eligible for crawling: lifecycle ``active`` and kill switch on."""
    with db.transaction() as cur:
        if codes:
            cur.execute(
                f"SELECT {_COLUMNS} FROM content_staging.source_registry "
                f"WHERE status = 'active' AND enabled AND code = ANY(%s) ORDER BY priority, code",
                (codes,),
            )
        else:
            cur.execute(
                f"SELECT {_COLUMNS} FROM content_staging.source_registry "
                f"WHERE status = 'active' AND enabled ORDER BY priority, code"
            )
        return [_to_source(row) for row in cur.fetchall()]


def set_status(db: Database, code: str, status: str, *, enabled: bool | None = None) -> Source | None:
    with db.transaction() as cur:
        if enabled is None:
            cur.execute(
                f"UPDATE content_staging.source_registry SET status = %s, updated_at = now() "
                f"WHERE code = %s RETURNING {_COLUMNS}",
                (status, code),
            )
        else:
            cur.execute(
                f"UPDATE content_staging.source_registry SET status = %s, enabled = %s, updated_at = now() "
                f"WHERE code = %s RETURNING {_COLUMNS}",
                (status, enabled, code),
            )
        row = cur.fetchone()
    return _to_source(row) if row is not None else None


def mark_crawled(db: Database, source_id: int) -> None:
    with db.transaction() as cur:
        cur.execute(
            "UPDATE content_staging.source_registry SET last_crawled_at = now(), updated_at = now() WHERE id = %s",
            (source_id,),
        )


def seed_defaults(db: Database) -> int:
    """Insert the official allowlist if absent. Existing rows are untouched."""
    created = 0
    with db.transaction() as cur:
        for source in DEFAULT_SOURCES:
            cur.execute(
                """
                INSERT INTO content_staging.source_registry
                    (code, name, publisher, base_url, source_type, tier)
                VALUES (%s, %s, %s, %s, %s, %s)
                ON CONFLICT (code) DO NOTHING
                """,
                (
                    source["code"], source["name"], source["publisher"],
                    source["base_url"], source["source_type"], source["tier"],
                ),
            )
            created += cur.rowcount
    return created
