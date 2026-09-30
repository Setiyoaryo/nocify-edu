"""Crawl orchestration (§7–§10, §34).

One run per source: fetch the entry page, discover allowlisted same-host links,
then fetch each URL with conditional requests. Every attempt is recorded on a
``crawl_items`` row so the archive can be reprocessed later.
"""

from __future__ import annotations

import logging
import time
from collections.abc import Callable
from dataclasses import dataclass
from urllib.parse import urljoin

from content_pipeline.config import Settings
from content_pipeline.db import Database
from content_pipeline.fetcher import (
    DisallowedContentTypeError,
    Fetcher,
    FetchError,
    FetchResult,
    ResponseTooLargeError,
    TooManyRedirectsError,
)
from content_pipeline.links import discover_links
from content_pipeline.registry import Source, mark_crawled, set_status
from content_pipeline.robots import robots_allows
from content_pipeline.safety import UnsafeURLError, canonicalize_url, host_of
from content_pipeline.storage import RawArchive

log = logging.getLogger("content_pipeline.crawler")

FetcherFactory = Callable[[Source, set[str]], Fetcher]


@dataclass(frozen=True)
class CrawlSummary:
    source_code: str
    run_id: int
    status: str
    pages_discovered: int
    pages_fetched: int
    pages_changed: int
    errors: int
    message: str


def _default_fetcher_factory(settings: Settings) -> FetcherFactory:
    def factory(_source: Source, allowed_hosts: set[str]) -> Fetcher:
        return Fetcher(
            allowed_hosts=allowed_hosts,
            user_agent=settings.user_agent,
            timeout_seconds=settings.http_timeout_seconds,
            max_response_bytes=settings.max_response_bytes,
            max_redirects=settings.max_redirects,
        )

    return factory


def _error_code(exc: Exception) -> str:
    if isinstance(exc, UnsafeURLError):
        return "unsafe_url"
    if isinstance(exc, DisallowedContentTypeError):
        return "content_type"
    if isinstance(exc, ResponseTooLargeError):
        return "too_large"
    if isinstance(exc, TooManyRedirectsError):
        return "too_many_redirects"
    if isinstance(exc, FetchError):
        return "fetch_error"
    return "error"


class CrawlRunner:
    def __init__(
        self,
        db: Database,
        settings: Settings,
        *,
        fetcher_factory: FetcherFactory | None = None,
        archive: RawArchive | None = None,
    ) -> None:
        self._db = db
        self._settings = settings
        self._fetcher_factory = fetcher_factory or _default_fetcher_factory(settings)
        self._archive = archive or RawArchive(settings)

    # ------------------------------------------------------------------ public

    def preview(self, source: Source, *, limit: int | None = None) -> list[str]:
        """Fetch the entry page and list discovered links without writing to the DB."""
        allowed_hosts = {host_of(source.base_url)}
        fetcher = self._fetcher_factory(source, allowed_hosts)
        try:
            result = fetcher.fetch(source.base_url)
            if result.body is None:
                return []
            html = result.body.decode("utf-8", errors="replace")
            links = discover_links(html, result.canonical_url, allowed_hosts)
        finally:
            fetcher.close()
        return links[:limit] if limit else links

    def run(self, *, codes: list[str] | None = None, limit: int | None = None) -> list[CrawlSummary]:
        from content_pipeline.registry import active_sources  # local import avoids a cycle at module load

        summaries: list[CrawlSummary] = []
        for source in active_sources(self._db, codes):
            summaries.append(self.run_source(source, limit=limit))
        return summaries

    def run_source(self, source: Source, *, limit: int | None = None) -> CrawlSummary:
        allowed_hosts = {host_of(source.base_url)}
        fetcher = self._fetcher_factory(source, allowed_hosts)
        try:
            return self._run(source, fetcher, allowed_hosts, limit=limit)
        finally:
            fetcher.close()

    # ----------------------------------------------------------------- internal

    def _run(
        self,
        source: Source,
        fetcher: Fetcher,
        allowed_hosts: set[str],
        *,
        limit: int | None,
    ) -> CrawlSummary:
        if source.robots_policy == "respect" and not self._robots_allows(source, fetcher):
            set_status(self._db, source.code, "paused", enabled=False)
            log.warning("source %s paused: robots.txt disallows crawling", source.code)
            return CrawlSummary(
                source_code=source.code,
                run_id=0,
                status="failed",
                pages_discovered=0,
                pages_fetched=0,
                pages_changed=0,
                errors=0,
                message="robots.txt disallows crawling; source paused",
            )

        run_id = self._open_run(source)
        counts = {"discovered": 0, "fetched": 0, "changed": 0, "errors": 0}
        status = "completed"
        message = ""
        try:
            try:
                entry = fetcher.fetch(source.base_url)
            except Exception as exc:  # noqa: BLE001 - recorded, not swallowed
                self._finish_run(run_id, "failed", counts, message=str(exc))
                mark_crawled(self._db, source.id)
                return CrawlSummary(
                    source_code=source.code,
                    run_id=run_id,
                    status="failed",
                    pages_discovered=0,
                    pages_fetched=0,
                    pages_changed=0,
                    errors=1,
                    message=f"entry page failed: {exc}",
                )

            urls = [entry.canonical_url]
            if entry.body is not None:
                html = entry.body.decode("utf-8", errors="replace")
                cap = min(source.max_pages, limit) if limit else source.max_pages
                for link in discover_links(html, entry.canonical_url, allowed_hosts):
                    if len(urls) >= cap:
                        break
                    if link not in urls:
                        urls.append(link)

            counts["discovered"] = len(urls)
            self._store_and_record(source, run_id, entry, counts)

            for url in urls[1:]:
                if source.rate_limit_ms:
                    time.sleep(source.rate_limit_ms / 1000)
                self._fetch_one(source, fetcher, run_id, url, counts)

            status = "partial" if counts["errors"] else "completed"
        except Exception as exc:  # noqa: BLE001 - a run must never crash the scheduler
            status = "failed"
            message = str(exc)
            log.exception("crawl run failed for %s", source.code)

        self._finish_run(run_id, status, counts, message=message)
        mark_crawled(self._db, source.id)
        return CrawlSummary(
            source_code=source.code,
            run_id=run_id,
            status=status,
            pages_discovered=counts["discovered"],
            pages_fetched=counts["fetched"],
            pages_changed=counts["changed"],
            errors=counts["errors"],
            message=message,
        )

    def _fetch_one(
        self,
        source: Source,
        fetcher: Fetcher,
        run_id: int,
        url: str,
        counts: dict[str, int],
    ) -> None:
        etag, last_modified = self._last_fetch(source.id, url)
        try:
            result = fetcher.fetch(url, etag=etag, last_modified=last_modified)
        except Exception as exc:  # noqa: BLE001 - recorded per item
            counts["errors"] += 1
            self._record(
                run_id, source, url, canonicalize_url(url),
                http_status=None, content_type="", etag="", last_modified="",
                content_hash="", content_length=None, storage_key="",
                status="failed", error_code=_error_code(exc), error_message=str(exc),
            )
            return

        if result.not_modified:
            self._record(
                run_id, source, url, result.canonical_url,
                http_status=304, content_type=result.content_type, etag=result.etag,
                last_modified=result.last_modified, content_hash="", content_length=0,
                storage_key="", status="not_modified",
            )
            return

        self._store_and_record(source, run_id, result, counts)

    def _store_and_record(
        self,
        source: Source,
        run_id: int,
        result: FetchResult,
        counts: dict[str, int],
    ) -> None:
        """Archive the body (when present), then record the crawl item."""
        storage_key = ""
        if result.body is not None:
            try:
                storage_key = self._archive.store(
                    source.code, result.body, result.content_hash, result.content_type
                )
            except Exception as exc:  # noqa: BLE001 - archive failure must not lose crawl metadata
                counts["errors"] += 1
                log.warning("raw archive failed for %s: %s", result.canonical_url, exc)
            counts["changed"] += 1
        counts["fetched"] += 1
        self._record(
            run_id, source, result.url, result.canonical_url,
            http_status=result.http_status, content_type=result.content_type,
            etag=result.etag, last_modified=result.last_modified,
            content_hash=result.content_hash, content_length=result.content_length,
            storage_key=storage_key, status="fetched",
        )

    def _robots_allows(self, source: Source, fetcher: Fetcher) -> bool:
        robots_url = urljoin(source.base_url, "/robots.txt")
        try:
            result = fetcher.fetch(robots_url)
        except Exception:  # noqa: BLE001 - missing/unreadable robots.txt is permissive
            return True
        if result.body is None:
            return True
        return robots_allows(result.body.decode("utf-8", errors="replace"), self._settings.user_agent, source.base_url)

    # ------------------------------------------------------------------- DB ops

    def _open_run(self, source: Source) -> int:
        with self._db.transaction() as cur:
            cur.execute(
                "INSERT INTO content_staging.crawl_runs (source_id, status) VALUES (%s, 'running') RETURNING id",
                (source.id,),
            )
            row = cur.fetchone()
        assert row is not None
        return int(row[0])

    def _finish_run(self, run_id: int, status: str, counts: dict[str, int], *, message: str = "") -> None:
        with self._db.transaction() as cur:
            cur.execute(
                """
                UPDATE content_staging.crawl_runs
                SET finished_at = now(), status = %s, pages_discovered = %s, pages_fetched = %s,
                    pages_changed = %s, errors = %s, message = %s
                WHERE id = %s
                """,
                (
                    status, counts["discovered"], counts["fetched"],
                    counts["changed"], counts["errors"], message, run_id,
                ),
            )

    def _last_fetch(self, source_id: int, canonical_url: str) -> tuple[str, str]:
        with self._db.transaction() as cur:
            cur.execute(
                """
                SELECT etag, last_modified FROM content_staging.crawl_items
                WHERE source_id = %s AND canonical_url = %s AND etag <> ''
                ORDER BY fetched_at DESC NULLS LAST
                LIMIT 1
                """,
                (source_id, canonical_url),
            )
            row = cur.fetchone()
        return (str(row[0]), str(row[1])) if row is not None else ("", "")

    def _record(
        self,
        run_id: int,
        source: Source,
        url: str,
        canonical_url: str,
        *,
        http_status: int | None,
        content_type: str,
        etag: str,
        last_modified: str,
        content_hash: str,
        content_length: int | None,
        storage_key: str,
        status: str,
        error_code: str = "",
        error_message: str = "",
    ) -> None:
        with self._db.transaction() as cur:
            cur.execute(
                """
                INSERT INTO content_staging.crawl_items
                    (crawl_run_id, source_id, url, canonical_url, http_status, content_type,
                     etag, last_modified, content_hash, content_length, storage_key,
                     fetched_at, status, error_code, error_message)
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, now(), %s, %s, %s)
                ON CONFLICT (crawl_run_id, canonical_url) DO NOTHING
                """,
                (
                    run_id, source.id, url, canonical_url, http_status, content_type,
                    etag, last_modified, content_hash, content_length, storage_key,
                    status, error_code, error_message,
                ),
            )
