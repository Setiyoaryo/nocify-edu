"""Pipeline worker daemon (M11).

The admin UI (Go) writes *commands* into ``content_staging.pipeline_jobs``; this
daemon claims them (one at a time, ``FOR UPDATE SKIP LOCKED``), runs the existing
stage builders, and writes progress back. It also runs the scheduler: sources
whose ``crawl_frequency`` is due are enqueued automatically.

The worker only ever writes ``content_staging`` (Python owns it) and never
canonical — the single control-table exception is that Go may write the queue
itself. Stages are idempotent, so a crashed or interrupted job can be re-queued
safely (see the reaper).
"""

from __future__ import annotations

import json
import logging
import os
import signal
import socket
import time
from collections.abc import Callable, Sequence
from dataclasses import asdict, dataclass, is_dataclass
from typing import Any, Protocol

from content_pipeline.assisted import COMPREHENSION, SCENARIO, AssistedGenerator
from content_pipeline.config import Settings
from content_pipeline.crawler import CrawlRunner
from content_pipeline.db import Database
from content_pipeline.embeddings import OWNER_CHUNK, OWNER_SKILL, EmbeddingBuilder, EmbeddingStore
from content_pipeline.extract import Extractor
from content_pipeline.figures import FigureVocab, load_vocab
from content_pipeline.mapping import Mapper
from content_pipeline.materials import MaterialBuilder
from content_pipeline.provider import Provider, build_provider
from content_pipeline.questions import QuestionBuilder
from content_pipeline.registry import set_governance
from content_pipeline.storage import RawArchive
from content_pipeline.validation import QuestionSchema
from content_pipeline.visuals import DbAssetCatalog, VisualBuilder, VisualPlanner
from content_pipeline.watch import CurriculumWatcher

log = logging.getLogger("content_pipeline.worker")

# Job kinds the worker will execute. The DB CHECK constraint mirrors this.
JOB_KINDS = ("crawl", "run_pipeline", "set_source_status", "watch")

# run_pipeline stage order. ``embed``/``generate`` are opt-in (params.ai) because
# they cost provider calls; the default chain stops at review candidates.
DEFAULT_STAGES = ("crawl", "extract", "map", "materials", "questions", "visuals")
AI_STAGES = ("embed", "generate")

# Distinct from the Go migration lock (727274) and the watch lock (727275).
ADVISORY_LOCK_KEY = 727276
# A running job older than this is presumed dead and reaped.
LEASE_MINUTES = 30


class JobInterrupted(Exception):
    """The worker is shutting down; re-queue the job and stop."""


class JobCancelled(Exception):
    """A reviewer cancelled the job; leave it cancelled."""


@dataclass(frozen=True)
class Job:
    id: int
    kind: str
    params: dict[str, Any]
    attempts: int
    max_attempts: int
    requested_by: str | None


class JobStore(Protocol):
    """Read/write view of the job queue (mirrors ``WatchStore``)."""

    def claim(self) -> Job | None: ...
    def is_cancelled(self, job_id: int) -> bool: ...
    def set_order(self, job_id: int, order: Sequence[str]) -> None: ...
    def begin_stage(self, job_id: int, stage: str) -> None: ...
    def add_stage(self, job_id: int, stage: str, result: Any) -> None: ...
    def finish(self, job_id: int) -> None: ...
    def fail(self, job_id: int, error: str) -> None: ...
    def requeue(self, job_id: int) -> None: ...
    def mark_cancelled(self, job_id: int) -> None: ...
    def reap(self, lease_minutes: int) -> int: ...
    def try_lock(self) -> bool: ...
    def unlock(self) -> None: ...
    def due_sources(self) -> list[str]: ...
    def enqueue(self, kind: str, params: dict[str, Any], *, requested_by: str | None = None) -> int: ...
    def heartbeat(self, worker_id: str, status: str = "idle", current_job_id: int | None = None) -> None: ...


def _jsonable(value: Any) -> Any:
    if is_dataclass(value) and not isinstance(value, type):
        return asdict(value)
    if isinstance(value, (list, tuple)):
        return [_jsonable(item) for item in value]
    return value


class PgJobStore:
    def __init__(self, db: Database) -> None:
        self._db = db

    def claim(self) -> Job | None:
        with self._db.transaction() as cur:
            cur.execute(
                """
                UPDATE content_staging.pipeline_jobs j
                SET status = 'running', started_at = now(), updated_at = now(),
                    attempts = attempts + 1, error = ''
                WHERE j.id = (
                    SELECT id FROM content_staging.pipeline_jobs
                    WHERE status = 'queued'
                    ORDER BY created_at, id
                    FOR UPDATE SKIP LOCKED
                    LIMIT 1
                )
                RETURNING j.id, j.kind, j.params, j.attempts, j.max_attempts, j.requested_by
                """
            )
            row = cur.fetchone()
        if row is None:
            return None
        return Job(
            id=int(row[0]),
            kind=str(row[1]),
            params=dict(row[2] or {}),
            attempts=int(row[3]),
            max_attempts=int(row[4]),
            requested_by=str(row[5]) if row[5] is not None else None,
        )

    def is_cancelled(self, job_id: int) -> bool:
        with self._db.transaction() as cur:
            cur.execute("SELECT status FROM content_staging.pipeline_jobs WHERE id = %s", (job_id,))
            row = cur.fetchone()
        if row is None:  # deleted mid-run: treat as cancelled, stop cleanly
            return True
        return str(row[0]) == "cancelled"

    def set_order(self, job_id: int, order: Sequence[str]) -> None:
        with self._db.transaction() as cur:
            cur.execute(
                "UPDATE content_staging.pipeline_jobs "
                "SET progress = progress || jsonb_build_object('order', %s::jsonb), updated_at = now() "
                "WHERE id = %s",
                (json.dumps(list(order)), job_id),
            )

    def begin_stage(self, job_id: int, stage: str) -> None:
        with self._db.transaction() as cur:
            cur.execute(
                "UPDATE content_staging.pipeline_jobs "
                "SET progress = jsonb_set(progress, '{current}', to_jsonb(%s::text), true), "
                "    updated_at = now() "
                "WHERE id = %s",
                (stage, job_id),
            )

    def add_stage(self, job_id: int, stage: str, result: Any) -> None:
        with self._db.transaction() as cur:
            cur.execute(
                "UPDATE content_staging.pipeline_jobs "
                "SET progress = jsonb_set("
                "      progress, '{stages}', "
                "      COALESCE(progress->'stages', '{}'::jsonb) || jsonb_build_object(%s::text, %s::jsonb), true), "
                "    updated_at = now() "
                "WHERE id = %s",
                (stage, json.dumps(_jsonable(result)), job_id),
            )

    def finish(self, job_id: int) -> None:
        with self._db.transaction() as cur:
            cur.execute(
                "UPDATE content_staging.pipeline_jobs "
                "SET status = 'done', finished_at = now(), updated_at = now(), error = '' "
                "WHERE id = %s",
                (job_id,),
            )

    def fail(self, job_id: int, error: str) -> None:
        with self._db.transaction() as cur:
            cur.execute(
                "UPDATE content_staging.pipeline_jobs "
                "SET status = 'failed', error = %s, finished_at = now(), updated_at = now() "
                "WHERE id = %s",
                (error[:500], job_id),
            )

    def requeue(self, job_id: int) -> None:
        with self._db.transaction() as cur:
            cur.execute(
                "UPDATE content_staging.pipeline_jobs "
                "SET status = 'queued', started_at = NULL, updated_at = now() "
                "WHERE id = %s",
                (job_id,),
            )

    def mark_cancelled(self, job_id: int) -> None:
        with self._db.transaction() as cur:
            cur.execute(
                "UPDATE content_staging.pipeline_jobs "
                "SET status = 'cancelled', finished_at = now(), updated_at = now() "
                "WHERE id = %s AND status <> 'cancelled'",
                (job_id,),
            )

    def reap(self, lease_minutes: int) -> int:
        with self._db.transaction() as cur:
            cur.execute(
                """
                UPDATE content_staging.pipeline_jobs
                SET status = CASE WHEN attempts < max_attempts THEN 'queued' ELSE 'failed' END,
                    error = CASE WHEN attempts < max_attempts THEN error ELSE 'lease expired' END,
                    started_at = CASE WHEN attempts < max_attempts THEN NULL ELSE started_at END,
                    finished_at = CASE WHEN attempts < max_attempts THEN NULL ELSE now() END,
                    updated_at = now()
                WHERE status = 'running'
                  AND started_at < now() - make_interval(mins => %s)
                """,
                (lease_minutes,),
            )
            return cur.rowcount

    def try_lock(self) -> bool:
        with self._db.transaction() as cur:
            cur.execute("SELECT pg_try_advisory_lock(%s)", (ADVISORY_LOCK_KEY,))
            row = cur.fetchone()
        return bool(row[0]) if row is not None else False

    def unlock(self) -> None:
        with self._db.transaction() as cur:
            cur.execute("SELECT pg_advisory_unlock(%s)", (ADVISORY_LOCK_KEY,))

    def due_sources(self) -> list[str]:
        """Active sources whose crawl cadence is due and not already queued."""
        with self._db.transaction() as cur:
            cur.execute(
                """
                SELECT s.code
                FROM content_staging.source_registry s
                WHERE s.status = 'active' AND s.enabled
                  AND s.crawl_frequency <> 'manual'
                  AND (
                    s.last_crawled_at IS NULL
                    OR s.last_crawled_at < now() - CASE s.crawl_frequency
                        WHEN 'daily'   THEN interval '1 day'
                        WHEN 'weekly'  THEN interval '7 days'
                        WHEN 'monthly' THEN interval '30 days'
                        ELSE interval '7 days' END
                  )
                  AND NOT EXISTS (
                    SELECT 1 FROM content_staging.pipeline_jobs j
                    WHERE j.kind IN ('crawl', 'run_pipeline')
                      AND j.status IN ('queued', 'running')
                      AND (j.params->'sources' IS NULL OR j.params->'sources' @> to_jsonb(s.code))
                  )
                ORDER BY s.priority, s.code
                """
            )
            return [str(row[0]) for row in cur.fetchall()]

    def enqueue(self, kind: str, params: dict[str, Any], *, requested_by: str | None = None) -> int:
        with self._db.transaction() as cur:
            cur.execute(
                "INSERT INTO content_staging.pipeline_jobs (kind, params, requested_by) "
                "VALUES (%s, %s::jsonb, %s) RETURNING id",
                (kind, json.dumps(params), requested_by),
            )
            row = cur.fetchone()
        assert row is not None
        return int(row[0])

    def heartbeat(self, worker_id: str, status: str = "idle", current_job_id: int | None = None) -> None:
        try:
            with self._db.transaction() as cur:
                cur.execute(
                    """
                    INSERT INTO content_staging.worker_heartbeats
                        (worker_id, hostname, pid, status, current_job_id, last_seen_at)
                    VALUES (%s, %s, %s, %s, %s, now())
                    ON CONFLICT (worker_id)
                    DO UPDATE SET status = EXCLUDED.status,
                                  current_job_id = EXCLUDED.current_job_id,
                                  last_seen_at = now()
                    """,
                    (worker_id, socket.gethostname(), os.getpid(), status, current_job_id),
                )
        except Exception as exc:  # noqa: BLE001 - don't crash if table does not exist
            log.debug("heartbeat update skipped: %s", exc)


class JobContext:
    """Per-job handle: reports progress and checks for cancellation/shutdown."""

    def __init__(
        self,
        store: JobStore,
        job: Job,
        stopping: Callable[[], bool],
        *,
        worker_id: str | None = None,
    ) -> None:
        self._store = store
        self._job = job
        self._stopping = stopping
        self._worker_id = worker_id

    @property
    def params(self) -> dict[str, Any]:
        return self._job.params

    def check(self) -> None:
        if self._stopping():
            raise JobInterrupted
        if self._store.is_cancelled(self._job.id):
            raise JobCancelled
        if self._worker_id:
            self._store.heartbeat(self._worker_id, "active", self._job.id)

    def set_order(self, order: Sequence[str]) -> None:
        self._store.set_order(self._job.id, order)

    def begin(self, stage: str) -> None:
        self._store.begin_stage(self._job.id, stage)
        if self._worker_id:
            self._store.heartbeat(self._worker_id, "active", self._job.id)

    def done(self, stage: str, result: Any) -> None:
        self._store.add_stage(self._job.id, stage, result)
        if self._worker_id:
            self._store.heartbeat(self._worker_id, "active", self._job.id)


class Worker:
    def __init__(
        self,
        db: Database,
        settings: Settings,
        *,
        store: JobStore | None = None,
        poll_seconds: float | None = None,
        scheduler_seconds: float | None = None,
    ) -> None:
        self._db = db
        self._settings = settings
        self._store: JobStore = store if store is not None else PgJobStore(db)
        self._poll_seconds = poll_seconds if poll_seconds is not None else settings.worker_poll_seconds
        self._scheduler_seconds = (
            scheduler_seconds if scheduler_seconds is not None else settings.scheduler_seconds
        )
        self._stopping = False
        self._last_schedule = 0.0
        self._worker_id = f"worker-{socket.gethostname()}-{os.getpid()}"

    def stop(self) -> None:
        self._stopping = True

    def _is_stopping(self) -> bool:
        return self._stopping

    # ------------------------------------------------------------- one job
    def run_once(self) -> bool:
        """Claim and execute at most one job. Returns True when a job ran."""
        self._store.reap(LEASE_MINUTES)
        job = self._store.claim()
        if job is None:
            self._store.heartbeat(self._worker_id, "idle", None)
            return False
        self._store.heartbeat(self._worker_id, "active", job.id)
        try:
            self._execute(job)
        finally:
            self._store.heartbeat(self._worker_id, "idle", None)
        return True

    def _execute(self, job: Job) -> None:
        ctx = JobContext(self._store, job, self._is_stopping, worker_id=self._worker_id)
        try:
            self._dispatch(ctx, job)
        except JobInterrupted:
            log.info("job %s interrupted; re-queueing", job.id)
            self._store.requeue(job.id)
            return
        except JobCancelled:
            log.info("job %s cancelled", job.id)
            self._store.mark_cancelled(job.id)
            return
        except Exception as exc:  # noqa: BLE001 - a bad job must not kill the daemon
            log.warning("job %s failed: %s", job.id, exc)
            self._store.fail(job.id, str(exc))
            return
        self._store.finish(job.id)

    def _dispatch(self, ctx: JobContext, job: Job) -> None:
        if job.kind == "crawl":
            self._do_crawl(ctx)
        elif job.kind == "run_pipeline":
            self._do_run_pipeline(ctx)
        elif job.kind == "set_source_status":
            self._do_set_source_status(ctx)
        elif job.kind == "watch":
            self._do_watch(ctx)
        else:
            raise ValueError(f"unknown job kind {job.kind!r}")

    # ------------------------------------------------------------- handlers
    def _do_crawl(self, ctx: JobContext) -> None:
        ctx.set_order(("crawl",))
        ctx.check()
        ctx.begin("crawl")
        ctx.done("crawl", self._stage_crawl(ctx.params))

    def _do_run_pipeline(self, ctx: JobContext) -> None:
        stages = list(ctx.params.get("stages") or DEFAULT_STAGES)
        if ctx.params.get("ai"):
            stages = self._with_ai_stages(stages)
        for stage in stages:
            if stage not in DEFAULT_STAGES and stage not in AI_STAGES:
                raise ValueError(f"unknown pipeline stage {stage!r}")
        ctx.set_order(stages)
        for stage in stages:
            ctx.check()
            ctx.begin(stage)
            ctx.done(stage, self._run_stage(stage, ctx.params))

    def _do_set_source_status(self, ctx: JobContext) -> None:
        ctx.check()
        params = ctx.params
        code = params.get("code")
        if not code:
            raise ValueError("set_source_status requires params.code")
        ctx.set_order(("set_source_status",))
        ctx.begin("set_source_status")
        source = set_governance(
            self._db,
            str(code),
            status=params.get("status"),
            enabled=params.get("enabled"),
            crawl_frequency=params.get("crawl_frequency"),
        )
        if source is None:
            raise ValueError(f"unknown source {code!r}")
        ctx.done(
            "set_source_status",
            {"code": source.code, "status": source.status,
             "enabled": source.enabled, "crawl_frequency": source.crawl_frequency},
        )

    def _do_watch(self, ctx: JobContext) -> None:
        params = ctx.params
        ctx.set_order(("watch",))
        ctx.check()
        ctx.begin("watch")
        report = CurriculumWatcher(self._db, self._settings).run(
            codes=params.get("documents"),
            limit=params.get("limit"),
            dry_run=bool(params.get("dry_run")),
            force=bool(params.get("force")),
            due=bool(params.get("due")),
        )
        ctx.done(
            "watch",
            {"locked": report.locked, "summaries": [asdict(s) for s in report.summaries]},
        )

    # -------------------------------------------------------------- stages
    def _with_ai_stages(self, stages: list[str]) -> list[str]:
        out = list(stages)
        if "embed" not in out and "map" in out:
            out.insert(out.index("map"), "embed")
        if "generate" not in out and "questions" in out:
            out.insert(out.index("questions") + 1, "generate")
        return out

    def _run_stage(self, stage: str, params: dict[str, Any]) -> Any:
        codes = params.get("sources")
        limit = params.get("limit")
        force = bool(params.get("force"))
        if stage == "crawl":
            return self._stage_crawl(params)
        if stage == "extract":
            archive = RawArchive(self._settings)
            if not archive.enabled:
                return {"skipped": "raw archive not configured (set S3_BUCKET)"}
            return _summaries(Extractor(self._db, archive).run(codes=codes, limit=limit, force=force))
        if stage == "embed":
            provider, reason = self._provider()
            if provider is None:
                return {"skipped": reason}
            try:
                report = EmbeddingBuilder(
                    self._db, provider, batch_size=self._settings.ai_embedding_batch_size
                ).run(codes=codes, limit=limit, force=force, kinds=(OWNER_CHUNK, OWNER_SKILL))
            finally:
                provider.close()
            return asdict(report)
        if stage == "map":
            return self._stage_map(params, codes, limit, force)
        if stage == "materials":
            return _summaries(MaterialBuilder(self._db).run(codes=codes, limit=limit, force=force))
        if stage == "questions":
            return self._stage_questions(codes, limit, force)
        if stage == "visuals":
            return self._stage_visuals(codes, limit, force)
        if stage == "generate":
            return self._stage_generate(params, codes, limit, force)
        raise ValueError(f"unknown pipeline stage {stage!r}")

    def _stage_crawl(self, params: dict[str, Any]) -> Any:
        return _summaries(
            CrawlRunner(self._db, self._settings).run(
                codes=params.get("sources"), limit=params.get("limit")
            )
        )

    def _stage_map(
        self, params: dict[str, Any], codes: list[str] | None, limit: int | None, force: bool
    ) -> Any:
        if not params.get("semantic"):
            return _summaries(Mapper(self._db).run(codes=codes, limit=limit, force=force))
        provider, _ = self._provider()
        if provider is None:
            return _summaries(Mapper(self._db).run(codes=codes, limit=limit, force=force))
        try:
            mapper = Mapper(
                self._db, embeddings=EmbeddingStore(self._db), semantic_model=provider.embedding_model
            )
            return _summaries(mapper.run(codes=codes, limit=limit, force=force, semantic=True))
        finally:
            provider.close()

    def _stage_questions(self, codes: list[str] | None, limit: int | None, force: bool) -> Any:
        schema = QuestionSchema.load()
        vocab = self._vocab()
        planner = VisualPlanner(vocab, DbAssetCatalog(self._db)) if vocab is not None else None
        return _summaries(
            QuestionBuilder(self._db, schema, planner=planner, vocab=vocab).run(
                codes=codes, limit=limit, force=force
            )
        )

    def _stage_visuals(self, codes: list[str] | None, limit: int | None, force: bool) -> Any:
        schema = QuestionSchema.load()
        vocab = self._vocab()
        if vocab is None:
            return {"skipped": "figure vocabulary unavailable"}
        return _summaries(
            VisualBuilder(self._db, schema, VisualPlanner(vocab, DbAssetCatalog(self._db))).run(
                codes=codes, limit=limit, force=force, dry_run=False
            )
        )

    def _stage_generate(
        self, params: dict[str, Any], codes: list[str] | None, limit: int | None, force: bool
    ) -> Any:
        provider, reason = self._provider()
        if provider is None:
            return {"skipped": reason}
        schema = QuestionSchema.load()
        templates = tuple(params.get("templates") or (SCENARIO, COMPREHENSION))
        try:
            report = AssistedGenerator(
                self._db,
                provider,
                schema,
                prompt_version=self._settings.ai_prompt_version,
                cap=self._settings.ai_generation_cap,
                max_output_tokens=self._settings.ai_max_output_tokens,
            ).run(
                codes=codes,
                limit=limit,
                templates=templates,
                force=force,
                dry_run=False,
                cap=None,
                allow_unknown_license=bool(params.get("allow_unknown_license")),
            )
        finally:
            provider.close()
        return asdict(report)

    # ------------------------------------------------------------- helpers
    def _provider(self) -> tuple[Provider | None, str]:
        try:
            provider = build_provider(self._settings)
        except ValueError as exc:
            return None, str(exc)
        if provider is None:
            return None, "AI provider not configured"
        return provider, ""

    @staticmethod
    def _vocab() -> FigureVocab | None:
        try:
            return load_vocab()
        except (FileNotFoundError, ValueError) as exc:
            log.warning("visual vocabulary unavailable: %s", exc)
            return None

    # ---------------------------------------------------------- scheduling
    def schedule_due(self) -> int:
        """Enqueue a run for every due source. Returns the number enqueued."""
        if not self._store.try_lock():
            return 0
        try:
            codes = self._store.due_sources()
            if not codes:
                return 0
            self._store.enqueue("run_pipeline", {"sources": codes})
            log.info("scheduled run for due sources: %s", ", ".join(codes))
            return len(codes)
        finally:
            self._store.unlock()

    # --------------------------------------------------------------- serve
    def serve(self) -> None:
        signal.signal(signal.SIGTERM, self._on_signal)
        signal.signal(signal.SIGINT, self._on_signal)
        log.info(
            "worker started (poll=%.1fs scheduler=%.1fs)",
            self._poll_seconds,
            self._scheduler_seconds,
        )
        try:
            while not self._stopping:
                self._store.heartbeat(self._worker_id, "idle", None)
                now = time.monotonic()
                if now - self._last_schedule >= self._scheduler_seconds:
                    try:
                        self.schedule_due()
                    except Exception as exc:  # noqa: BLE001 - the scheduler must not crash
                        log.warning("scheduler error: %s", exc)
                    self._last_schedule = now
                if not self.run_once():
                    self._sleep(self._poll_seconds)
        finally:
            self._store.heartbeat(self._worker_id, "stopped", None)
            log.info("worker stopped")

    def _on_signal(self, signum: int, _frame: object) -> None:
        log.info("signal %s received; finishing current stage and stopping", signum)
        self._stopping = True

    def _sleep(self, seconds: float) -> None:
        # Sleep in small slices so a signal stops us promptly.
        deadline = time.monotonic() + seconds
        while not self._stopping and time.monotonic() < deadline:
            time.sleep(min(0.25, deadline - time.monotonic()))


def _summaries(items: Sequence[Any]) -> list[Any]:
    return [_jsonable(item) for item in items]
