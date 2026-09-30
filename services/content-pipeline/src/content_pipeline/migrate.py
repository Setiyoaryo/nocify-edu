"""Minimal forward-only migration runner for the content_staging schema.

Migrations are plain ``sql/NNNN_*.sql`` files applied in filename order and
recorded in ``content_staging.schema_migrations``. This keeps the pipeline's
schema ownership self-contained without adding a heavier tool; Alembic can
replace it later if the schema starts branching (see roadmap §4.6).
"""

from __future__ import annotations

import hashlib
from dataclasses import dataclass
from pathlib import Path

from content_pipeline.db import Database

SQL_DIR = Path(__file__).resolve().parents[2] / "sql"

_BOOTSTRAP = """
CREATE SCHEMA IF NOT EXISTS content_staging;
CREATE TABLE IF NOT EXISTS content_staging.schema_migrations (
    filename   text PRIMARY KEY,
    checksum   text NOT NULL,
    applied_at timestamptz NOT NULL DEFAULT now()
);
"""


@dataclass(frozen=True)
class MigrationResult:
    applied: list[str]
    skipped: list[str]


def discover_migrations(sql_dir: Path = SQL_DIR) -> list[Path]:
    """Numbered migrations only; files like roles_grants.sql are run manually."""
    return sorted(p for p in sql_dir.glob("[0-9][0-9][0-9][0-9]_*.sql"))


def _checksum(text: str) -> str:
    return hashlib.sha256(text.encode("utf-8")).hexdigest()


def migrate(db: Database, sql_dir: Path = SQL_DIR) -> MigrationResult:
    applied: list[str] = []
    skipped: list[str] = []
    with db.transaction() as cur:
        cur.execute(_BOOTSTRAP)
        for path in discover_migrations(sql_dir):
            sql = path.read_text(encoding="utf-8")
            filename = path.name
            cur.execute(
                "SELECT checksum FROM content_staging.schema_migrations WHERE filename = %s",
                (filename,),
            )
            row = cur.fetchone()
            checksum = _checksum(sql)
            if row is not None:
                if row[0] != checksum:
                    raise RuntimeError(
                        f"{filename} was modified after it was applied "
                        f"(recorded {row[0][:12]}, now {checksum[:12]})"
                    )
                skipped.append(filename)
                continue
            cur.execute(sql)
            cur.execute(
                "INSERT INTO content_staging.schema_migrations (filename, checksum) VALUES (%s, %s)",
                (filename, checksum),
            )
            applied.append(filename)
    return MigrationResult(applied=applied, skipped=skipped)
