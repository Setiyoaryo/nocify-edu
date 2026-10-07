"""Worker daemon tests (M11).

DB-free by design: the `JobStore` is a fake and stages are monkeypatched, so the
dispatch/progress/cancel/reap/schedule logic is exercised without a database —
matching the `test_watch.py` convention.
"""

from __future__ import annotations

from dataclasses import dataclass, replace
from typing import Any, cast

import pytest

from content_pipeline.config import Settings
from content_pipeline.db import Database
from content_pipeline.worker import (
    DEFAULT_STAGES,
    Job,
    Worker,
    _jsonable,
)


def _settings(**overrides: Any) -> Settings:
    base = Settings(
        database_url="postgres://x",
        user_agent="ua",
        http_timeout_seconds=20.0,
        max_response_bytes=1024,
        max_redirects=5,
        default_rate_limit_ms=1000,
        s3_endpoint="",
        s3_region="us-east-1",
        s3_bucket="",
        s3_use_ssl=True,
        storage_access_key="",
        storage_secret_key="",
        ai_provider="",
        ai_base_url="",
        ai_api_key="",
        ai_embedding_model="",
        ai_chat_model="",
        ai_timeout_seconds=30.0,
        ai_max_output_tokens=700,
        ai_generation_cap=50,
        ai_embedding_batch_size=64,
        ai_prompt_version="scenario-v1",
        worker_poll_seconds=5.0,
        scheduler_seconds=60.0,
    )
    return replace(base, **overrides)


@dataclass
class _Summary:
    source_code: str
    generated: int


class FakeJobStore:
    def __init__(self, jobs: list[Job] | None = None) -> None:
        self._jobs = list(jobs or [])
        self.progress: dict[int, dict[str, Any]] = {}
        self.status: dict[int, str] = {}
        self.finished: list[int] = []
        self.failed: list[tuple[int, str]] = []
        self.requeued: list[int] = []
        self.cancelled: list[int] = []
        self.cancelled_ids: set[int] = set()
        self.enqueued: list[tuple[str, dict[str, Any], str | None]] = []
        self.reaps = 0
        self.locked = True
        self.due: list[str] = []
        self.heartbeats: list[tuple[str, str, int | None]] = []

    def claim(self) -> Job | None:
        return self._jobs.pop(0) if self._jobs else None

    def is_cancelled(self, job_id: int) -> bool:
        return job_id in self.cancelled_ids

    def set_order(self, job_id: int, order: Any) -> None:
        self.progress.setdefault(job_id, {})["order"] = list(order)

    def begin_stage(self, job_id: int, stage: str) -> None:
        self.progress.setdefault(job_id, {})["current"] = stage

    def add_stage(self, job_id: int, stage: str, result: Any) -> None:
        self.progress.setdefault(job_id, {}).setdefault("stages", {})[stage] = result

    def finish(self, job_id: int) -> None:
        self.status[job_id] = "done"
        self.finished.append(job_id)

    def fail(self, job_id: int, error: str) -> None:
        self.status[job_id] = "failed"
        self.failed.append((job_id, error))

    def requeue(self, job_id: int) -> None:
        self.status[job_id] = "queued"
        self.requeued.append(job_id)

    def mark_cancelled(self, job_id: int) -> None:
        self.status[job_id] = "cancelled"
        self.cancelled.append(job_id)

    def reap(self, lease_minutes: int) -> int:
        self.reaps += 1
        return 0

    def try_lock(self) -> bool:
        return self.locked

    def unlock(self) -> None:
        pass

    def due_sources(self) -> list[str]:
        return list(self.due)

    def enqueue(self, kind: str, params: dict[str, Any], *, requested_by: str | None = None) -> int:
        self.enqueued.append((kind, params, requested_by))
        return len(self.enqueued)

    def heartbeat(self, worker_id: str, status: str = "idle", current_job_id: int | None = None) -> None:
        self.heartbeats.append((worker_id, status, current_job_id))


def _job(kind: str, params: dict[str, Any] | None = None, *, jid: int = 1) -> Job:
    return Job(id=jid, kind=kind, params=params or {}, attempts=1, max_attempts=3, requested_by=None)


def _worker(store: FakeJobStore, **settings_over: Any) -> Worker:
    return Worker(cast(Database, None), _settings(**settings_over), store=store)


# ------------------------------------------------------------------ dispatch


def test_run_pipeline_executes_stages_in_order(monkeypatch: pytest.MonkeyPatch) -> None:
    seen: list[str] = []

    def fake_stage(self: Worker, stage: str, params: dict[str, Any]) -> Any:
        seen.append(stage)
        return {"stage": stage}

    monkeypatch.setattr(Worker, "_run_stage", fake_stage)
    store = FakeJobStore([_job("run_pipeline", {"stages": ["crawl", "extract"]})])
    assert _worker(store).run_once() is True

    assert seen == ["crawl", "extract"]
    assert store.progress[1]["order"] == ["crawl", "extract"]
    assert store.progress[1]["stages"] == {"crawl": {"stage": "crawl"}, "extract": {"stage": "extract"}}
    assert store.finished == [1]


def test_run_pipeline_uses_default_stages(monkeypatch: pytest.MonkeyPatch) -> None:
    seen: list[str] = []

    def fake_stage(self: Worker, stage: str, params: dict[str, Any]) -> Any:
        seen.append(stage)
        return {}

    monkeypatch.setattr(Worker, "_run_stage", fake_stage)
    store = FakeJobStore([_job("run_pipeline", {})])
    _worker(store).run_once()
    assert seen == list(DEFAULT_STAGES)


def test_unknown_stage_fails(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(Worker, "_run_stage", lambda self, stage, params: {})
    store = FakeJobStore([_job("run_pipeline", {"stages": ["nonsense"]})])
    _worker(store).run_once()
    assert store.status[1] == "failed"
    assert "unknown pipeline stage" in store.failed[0][1]


def test_unknown_kind_fails() -> None:
    store = FakeJobStore([_job("bogus")])
    _worker(store).run_once()
    assert store.status[1] == "failed"
    assert "unknown job kind" in store.failed[0][1]


# --------------------------------------------------------------- cancel/stop


def test_cancel_stops_between_stages(monkeypatch: pytest.MonkeyPatch) -> None:
    store = FakeJobStore([_job("run_pipeline", {"stages": ["crawl", "extract", "map"]})])
    seen: list[str] = []

    def fake_stage(self: Worker, stage: str, params: dict[str, Any]) -> Any:
        seen.append(stage)
        store.cancelled_ids.add(1)  # a reviewer cancels after the first stage
        return {}

    monkeypatch.setattr(Worker, "_run_stage", fake_stage)
    _worker(store).run_once()
    assert seen == ["crawl"]  # stopped before extract
    assert store.status[1] == "cancelled"


def test_stopping_requeues_the_job(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(Worker, "_run_stage", lambda self, stage, params: {})
    store = FakeJobStore([_job("run_pipeline", {"stages": ["crawl"]})])
    worker = _worker(store)
    worker.stop()
    worker.run_once()
    assert store.status[1] == "queued"
    assert store.requeued == [1]


# ------------------------------------------------------------------- stages


def test_extract_stage_skips_without_archive() -> None:
    worker = _worker(FakeJobStore(), s3_bucket="")
    assert worker._run_stage("extract", {}) == {"skipped": "raw archive not configured (set S3_BUCKET)"}


def test_ai_stages_are_inserted() -> None:
    worker = _worker(FakeJobStore())
    assert worker._with_ai_stages(["crawl", "map", "questions"]) == [
        "crawl",
        "embed",
        "map",
        "questions",
        "generate",
    ]


def test_ai_stages_not_duplicated() -> None:
    worker = _worker(FakeJobStore())
    assert worker._with_ai_stages(["crawl", "embed", "map"]) == ["crawl", "embed", "map"]


# ------------------------------------------------------------ set_source_status


def test_set_source_status_records_the_result(monkeypatch: pytest.MonkeyPatch) -> None:
    @dataclass
    class _Source:
        code: str
        status: str
        enabled: bool
        crawl_frequency: str

    monkeypatch.setattr(
        "content_pipeline.worker.set_governance",
        lambda db, code, **kw: _Source("src", "active", True, "daily"),
    )
    store = FakeJobStore([_job("set_source_status", {"code": "src", "enabled": True})])
    _worker(store).run_once()
    assert store.status[1] == "done"
    assert store.progress[1]["stages"]["set_source_status"]["enabled"] is True


def test_set_source_status_requires_code() -> None:
    store = FakeJobStore([_job("set_source_status", {})])
    _worker(store).run_once()
    assert store.status[1] == "failed"
    assert "params.code" in store.failed[0][1]


# ---------------------------------------------------------------- scheduling


def test_schedule_due_enqueues_one_run() -> None:
    store = FakeJobStore()
    store.due = ["a", "b"]
    assert _worker(store).schedule_due() == 2
    assert store.enqueued == [("run_pipeline", {"sources": ["a", "b"]}, None)]


def test_schedule_due_skips_when_locked() -> None:
    store = FakeJobStore()
    store.due = ["a"]
    store.locked = False
    assert _worker(store).schedule_due() == 0
    assert store.enqueued == []


def test_schedule_due_no_due_sources() -> None:
    store = FakeJobStore()
    assert _worker(store).schedule_due() == 0
    assert store.enqueued == []


# -------------------------------------------------------------------- misc


def test_run_once_reaps_then_returns_false_when_idle() -> None:
    store = FakeJobStore()
    assert _worker(store).run_once() is False
    assert store.reaps == 1


def test_jsonable_serialises_dataclasses() -> None:
    assert _jsonable(_Summary("s", 3)) == {"source_code": "s", "generated": 3}
    assert _jsonable([_Summary("s", 1)]) == [{"source_code": "s", "generated": 1}]
    assert _jsonable("x") == "x"


def test_settings_from_env_database_url_precedence(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("CONTENT_PIPELINE_DATABASE_URL", "postgres://cp:cp@db/staging")
    monkeypatch.setenv("DATABASE_URL", "postgres://app:app@db/prod")
    s = Settings.from_env()
    assert s.database_url == "postgres://cp:cp@db/staging"

    monkeypatch.delenv("CONTENT_PIPELINE_DATABASE_URL", raising=False)
    s2 = Settings.from_env()
    assert s2.database_url == "postgres://app:app@db/prod"

    monkeypatch.delenv("DATABASE_URL", raising=False)
    with pytest.raises(ValueError, match="DATABASE_URL is required"):
        Settings.from_env()


def test_worker_records_heartbeats() -> None:
    store = FakeJobStore()
    worker = _worker(store)
    assert not worker.run_once()
    assert len(store.heartbeats) == 1
    assert store.heartbeats[0][1] == "idle"

    # With a job
    job = _job("crawl", {"sources": ["x"]})
    store = FakeJobStore([job])
    worker = _worker(store)
    worker._stage_crawl = lambda _p: [{"source_code": "x"}]  # type: ignore[assignment]
    assert worker.run_once()
    # Expect heartbeat active with job.id recorded during claim, begin, and done, then idle
    active_hbs = [h for h in store.heartbeats if h[1] == "active" and h[2] == 1]
    assert len(active_hbs) >= 3  # claim + begin + done
    assert store.heartbeats[-1][1] == "idle"


def test_worker_serve_records_stopped_heartbeat() -> None:
    store = FakeJobStore()
    worker = _worker(store, worker_poll_seconds=0.01)
    # Stop immediately after one check
    worker._stopping = True
    worker.serve()
    assert len(store.heartbeats) >= 1
    assert store.heartbeats[-1][1] == "stopped"


