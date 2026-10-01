"""Thin PostgreSQL access layer.

Uses psycopg3 with short-lived transactions. The pipeline only touches the
``content_staging`` schema (plus read-only canonical taxonomy).
"""

from __future__ import annotations

from collections.abc import Iterator
from contextlib import contextmanager

import psycopg


class Database:
    def __init__(self, database_url: str) -> None:
        self._conn = psycopg.connect(database_url, autocommit=False)

    def close(self) -> None:
        self._conn.close()

    def __enter__(self) -> Database:
        return self

    def __exit__(self, *_: object) -> None:
        self.close()

    @contextmanager
    def transaction(self) -> Iterator[psycopg.Cursor[psycopg.rows.TupleRow]]:
        """Yield a cursor inside a transaction that commits on success."""
        with self._conn.cursor() as cur:
            try:
                yield cur
            except BaseException:
                self._conn.rollback()
                raise
            else:
                self._conn.commit()
