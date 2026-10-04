"""Watcher tests (§27, §28, §44).

DB-free and network-free by design: the `WatchStore` is a fake, the HTTP client
is an `httpx.MockTransport`, and DNS resolution is stubbed so the SSRF guard
sees a public address (the `Fetcher` pattern).
"""

from __future__ import annotations

import ipaddress
from collections.abc import Callable
from datetime import UTC, datetime, timedelta
from pathlib import Path
from typing import Any, cast

import httpx
import pytest

from content_pipeline.config import Settings
from content_pipeline.curriculum_diff import CHANGED, CpChange
from content_pipeline.db import Database
from content_pipeline.fetcher import Fetcher, sha256_hex
from content_pipeline.taxonomy import CpRef, SourceDocumentRef
from content_pipeline.watch import (
    STATUS_CHANGED,
    STATUS_FAILED,
    STATUS_NOT_MODIFIED,
    STATUS_SKIPPED,
    STATUS_UNCHANGED,
    CurriculumWatcher,
    QuestionRef,
    WatchTarget,
    build_impacts,
)

PUBLIC_IP = ipaddress.ip_address("93.184.216.34")
Handler = Callable[[httpx.Request], httpx.Response]
URL = "https://example.test/kurikulum"

BASELINE_BODY = b"CP-A-MAT-BILANGAN: Menjumlahkan bilangan sampai 10."
CHANGED_BODY = (
    b"CP-A-MAT-BILANGAN: Menjumlahkan bilangan sampai 20.\n"
    b"CP-A-MAT-BARU: Mengenal bangun datar."
)


@pytest.fixture(autouse=True)
def _public_dns(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr("content_pipeline.safety.resolve_host", lambda host, port: {PUBLIC_IP})


def _settings() -> Settings:
    return Settings(
        database_url="postgres://x",
        user_agent="ua",
        http_timeout_seconds=20.0,
        max_response_bytes=1024 * 1024,
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


def _document() -> SourceDocumentRef:
    return SourceDocumentRef(
        id="11111111-1111-1111-1111-111111111111",
        code="KEMENDIKDASMEN-KURIKULUM",
        title="Kurikulum",
        source_url=URL,
        document_hash=BASELINE_BODY.hex(),
        curriculum_version_id=1,
    )


def _target(
    *,
    baseline_hash: str = "",
    url: str = URL,
    frequency: str = "weekly",
    last_checked_at: datetime | None = None,
) -> WatchTarget:
    return WatchTarget(
        document=_document(),
        url=url,
        baseline_hash=baseline_hash,
        baseline_source="observed" if baseline_hash else "canonical",
        etag="",
        last_modified="",
        check_frequency=frequency,
        last_checked_at=last_checked_at,
    )


class FakeStore:
    """In-memory `WatchStore`. Records every call so tests can assert on writes."""

    def __init__(
        self,
        targets: list[WatchTarget],
        *,
        cp_refs: dict[str, CpRef] | None = None,
        active_version: int | None = 1,
        lock_available: bool = True,
    ) -> None:
        self._targets = targets
        self._cp_refs = cp_refs or {}
        self._active_version = active_version
        self._lock_available = lock_available
        self.state_saves: list[dict[str, Any]] = []
        self.events: list[dict[str, Any]] = []
        self.failures: list[tuple[int, str]] = []
        self.finished: list[dict[str, Any]] = []
        self.impacts: list[list[Any]] = []
        self.lock_calls = 0
        self.unlock_calls = 0
        self._next_event = 1
        self.skills_by_cp: dict[int, tuple[int, ...]] = {}
        self.questions_by_skill: dict[int, tuple[QuestionRef, ...]] = {}
        self.questions_by_cp: dict[int, tuple[QuestionRef, ...]] = {}

    def targets(self, codes: list[str] | None, *, require_url: bool = True) -> list[WatchTarget]:
        found = [t for t in self._targets if not codes or t.document.code in codes]
        if require_url:
            found = [t for t in found if t.url]
        return found

    def save_state(
        self, target: WatchTarget, *, status: str, baseline_hash: str, baseline_source: str,
        etag: str, last_modified: str, error: str = "", changed: bool = False,
    ) -> None:
        self.state_saves.append({
            "code": target.document.code, "status": status, "baseline_hash": baseline_hash,
            "baseline_source": baseline_source, "error": error, "changed": changed,
        })

    def create_event(
        self, target: WatchTarget, *, observed_hash: str, detection: str, http_status: int | None,
        etag: str, last_modified: str, document_type: str, storage_key: str,
    ) -> int:
        event_id = self._next_event
        self._next_event += 1
        self.events.append({
            "id": event_id, "code": target.document.code, "observed_hash": observed_hash,
            "detection": detection, "http_status": http_status, "document_type": document_type,
        })
        return event_id

    def fail_event(self, event_id: int, error: str) -> None:
        self.failures.append((event_id, error))

    def finish_event(
        self, event_id: int, *, cp_added: int, cp_changed: int, cp_removed: int
    ) -> None:
        self.finished.append({
            "id": event_id, "added": cp_added, "changed": cp_changed, "removed": cp_removed,
        })

    def save_impacts(self, event_id: int, impacts: list[Any]) -> None:
        self.impacts.append(impacts)

    def cp_statements(self, curriculum_version_id: int) -> dict[str, CpRef]:
        return self._cp_refs

    def active_version_id(self) -> int | None:
        return self._active_version

    def try_lock(self) -> bool:
        self.lock_calls += 1
        return self._lock_available

    def unlock(self) -> None:
        self.unlock_calls += 1

    def skills_for_cps(self, cp_ids: list[int]) -> dict[int, tuple[int, ...]]:
        return {cp_id: self.skills_by_cp[cp_id] for cp_id in cp_ids if cp_id in self.skills_by_cp}

    def questions_for_skills(self, skill_ids: list[int]) -> dict[int, tuple[QuestionRef, ...]]:
        return {s: self.questions_by_skill[s] for s in skill_ids if s in self.questions_by_skill}

    def questions_for_cps(self, cp_ids: list[int]) -> dict[int, tuple[QuestionRef, ...]]:
        return {c: self.questions_by_cp[c] for c in cp_ids if c in self.questions_by_cp}


def _fetcher(handler: Handler) -> Fetcher:
    client = httpx.Client(transport=httpx.MockTransport(handler), follow_redirects=False)
    return Fetcher(allowed_hosts={"example.test"}, user_agent="test-bot", client=client)


def _watcher(store: FakeStore, fetcher: Fetcher | None = None) -> CurriculumWatcher:
    return CurriculumWatcher(cast(Database, None), _settings(), store=store, fetcher=fetcher)


def _handler(body: bytes, *, content_type: str = "text/plain", status: int = 200) -> Handler:
    def handler(_: httpx.Request) -> httpx.Response:
        return httpx.Response(status, content=body, headers={"content-type": content_type})

    return handler


# ------------------------------------------------------------------ build_impacts


def _change(code: str, change_type: str = CHANGED) -> CpChange:
    return CpChange(
        code=code, change_type=change_type, old_statement="old", new_statement="new",
        statement_hash="hash",
    )


def test_build_impacts_links_questions_via_skill_and_provenance() -> None:
    impacts = build_impacts(
        changes=[_change("CP-1")],
        cp_refs={"CP-1": CpRef(id=1, code="CP-1", statement="old")},
        skills_by_cp={1: (10, 11)},
        questions_by_skill={10: (QuestionRef("q-a", "Q-A", 10, "published"),)},
        questions_by_cp={1: (QuestionRef("q-b", "Q-B", 99, "draft"),)},
    )
    assert len(impacts) == 1
    assert impacts[0].cp_id == 1
    assert impacts[0].affected_skills == 2
    by_id = {q.question_id: q for q in impacts[0].questions}
    assert by_id["q-a"].match == "skill"
    assert by_id["q-b"].match == "provenance"


def test_build_impacts_prefers_skill_match_when_question_appears_in_both() -> None:
    question = QuestionRef("q-a", "Q-A", 10, "published")
    impacts = build_impacts(
        changes=[_change("CP-1")],
        cp_refs={"CP-1": CpRef(id=1, code="CP-1", statement="old")},
        skills_by_cp={1: (10,)},
        questions_by_skill={10: (question,)},
        questions_by_cp={1: (question,)},
    )
    assert [q.match for q in impacts[0].questions] == ["skill"]


def test_build_impacts_sorts_questions_and_handles_unknown_cp() -> None:
    impacts = build_impacts(
        changes=[_change("CP-UNKNOWN")],
        cp_refs={},
        skills_by_cp={},
        questions_by_skill={},
        questions_by_cp={},
    )
    assert impacts[0].cp_id is None
    assert impacts[0].affected_skills == 0
    assert impacts[0].questions == ()


def test_build_impacts_sorts_questions_by_id() -> None:
    impacts = build_impacts(
        changes=[_change("CP-1")],
        cp_refs={"CP-1": CpRef(id=1, code="CP-1", statement="old")},
        skills_by_cp={1: (10,)},
        questions_by_skill={10: (
            QuestionRef("q-z", "Q-Z", 10, "published"),
            QuestionRef("q-a", "Q-A", 10, "published"),
        )},
        questions_by_cp={},
    )
    assert [q.question_id for q in impacts[0].questions] == ["q-a", "q-z"]


# --------------------------------------------------------------- watcher: no-op


def test_watch_unchanged_records_state_without_event() -> None:
    target = _target(baseline_hash=sha256_hex(BASELINE_BODY))
    store = FakeStore([target], cp_refs={"CP-A-MAT-BILANGAN": CpRef(1, "CP-A-MAT-BILANGAN", "x")})
    report = _watcher(store, _fetcher(_handler(BASELINE_BODY))).run()
    assert [s.status for s in report.summaries] == [STATUS_UNCHANGED]
    assert store.events == []
    assert store.state_saves[-1]["status"] == STATUS_UNCHANGED


def test_watch_bootstrap_records_baseline_only() -> None:
    store = FakeStore([_target(baseline_hash="")])
    report = _watcher(store, _fetcher(_handler(BASELINE_BODY))).run()
    assert [s.status for s in report.summaries] == [STATUS_SKIPPED]
    assert store.events == []
    save = store.state_saves[-1]
    assert save["baseline_hash"] == sha256_hex(BASELINE_BODY)
    assert save["baseline_source"] == "observed"


def test_watch_not_modified_updates_conditional_state() -> None:
    store = FakeStore([_target(baseline_hash=sha256_hex(BASELINE_BODY))])
    report = _watcher(store, _fetcher(_handler(b"", status=304))).run()
    assert [s.status for s in report.summaries] == [STATUS_NOT_MODIFIED]
    assert store.events == []
    assert store.state_saves[-1]["status"] == STATUS_NOT_MODIFIED


def test_watch_fetch_error_records_failed_state() -> None:
    store = FakeStore([_target(baseline_hash=sha256_hex(BASELINE_BODY))])
    report = _watcher(store, _fetcher(_handler(b"boom", status=500))).run()
    assert [s.status for s in report.summaries] == [STATUS_FAILED]
    assert store.events == []
    assert store.state_saves[-1]["status"] == STATUS_FAILED
    assert store.state_saves[-1]["error"]


# ------------------------------------------------------------- watcher: changed


def _changed_store(**kwargs: Any) -> FakeStore:
    target = _target(baseline_hash=sha256_hex(BASELINE_BODY))
    store = FakeStore(
        [target],
        cp_refs={"CP-A-MAT-BILANGAN": CpRef(1, "CP-A-MAT-BILANGAN", "Menjumlahkan bilangan sampai 10.")},
        **kwargs,
    )
    store.skills_by_cp = {1: (10,)}
    store.questions_by_skill = {10: (QuestionRef("q-1", "Q-1", 10, "published"),)}
    return store


def test_watch_changed_creates_event_impacts_and_advances_baseline() -> None:
    store = _changed_store()
    report = _watcher(store, _fetcher(_handler(CHANGED_BODY))).run()

    summary = report.summaries[0]
    assert summary.status == STATUS_CHANGED
    assert (summary.cp_added, summary.cp_changed, summary.cp_removed) == (1, 1, 0)
    assert summary.affected_questions == 1
    assert report.changed == 1

    assert len(store.events) == 1
    assert store.events[0]["detection"] == "content_hash"
    assert store.finished == [{"id": 1, "added": 1, "changed": 1, "removed": 0}]
    impacts = store.impacts[0]
    assert {impact.change.code for impact in impacts} == {"CP-A-MAT-BARU", "CP-A-MAT-BILANGAN"}
    assert store.state_saves[-1]["changed"] is True
    assert store.state_saves[-1]["baseline_hash"] == sha256_hex(CHANGED_BODY)


def test_watch_dry_run_writes_nothing() -> None:
    store = _changed_store()
    report = _watcher(store, _fetcher(_handler(CHANGED_BODY))).run(dry_run=True)
    assert [s.status for s in report.summaries] == [STATUS_CHANGED]
    assert report.dry_run is True
    assert store.events == []
    assert store.state_saves == []
    assert store.impacts == []
    assert store.lock_calls == 0
    assert store.unlock_calls == 0


def test_watch_force_reanalyses_an_unchanged_hash() -> None:
    store = _changed_store()
    store.skills_by_cp = {}
    store.questions_by_skill = {}
    # The body's CP-A-MAT-BILANGAN statement matches the baseline exactly, so a
    # normal run reports "unchanged"; --force re-analyses it anyway.
    body = BASELINE_BODY
    assert _watcher(store, _fetcher(_handler(body))).run().summaries[0].status == STATUS_UNCHANGED

    forced = _watcher(store, _fetcher(_handler(body))).run(force=True)
    assert [s.status for s in forced.summaries] == [STATUS_CHANGED]
    assert len(store.events) == 1


def test_watch_unsupported_document_type_records_failure() -> None:
    store = FakeStore([_target(baseline_hash=sha256_hex(BASELINE_BODY))])
    report = _watcher(store, _fetcher(_handler(b"\x00\x01\x02", content_type="application/octet-stream"))).run()
    assert [s.status for s in report.summaries] == [STATUS_FAILED]
    assert store.events == []
    assert store.state_saves[-1]["status"] == STATUS_FAILED


def test_watch_parse_failure_marks_event_failed(monkeypatch: pytest.MonkeyPatch) -> None:
    def boom(_body: bytes, _content_type: str) -> Any:
        raise ValueError("bad document")

    monkeypatch.setattr("content_pipeline.watch.parse", boom)
    store = _changed_store()
    report = _watcher(store, _fetcher(_handler(CHANGED_BODY))).run()
    assert [s.status for s in report.summaries] == [STATUS_FAILED]
    assert store.failures == [(1, "bad document")]
    assert store.state_saves[-1]["status"] == STATUS_FAILED


# -------------------------------------------------------------- watcher: gating


def test_watch_due_skips_documents_that_are_not_due() -> None:
    fresh = _target(baseline_hash=sha256_hex(BASELINE_BODY), frequency="weekly",
                    last_checked_at=datetime.now(UTC) - timedelta(days=1))
    store = FakeStore([fresh])
    report = _watcher(store, _fetcher(_handler(CHANGED_BODY))).run(due=True)
    assert [s.status for s in report.summaries] == [STATUS_SKIPPED]
    assert report.summaries[0].message == "not due"
    assert store.events == []


def test_watch_due_runs_documents_that_are_due() -> None:
    stale = _target(baseline_hash=sha256_hex(BASELINE_BODY), frequency="daily",
                    last_checked_at=datetime.now(UTC) - timedelta(days=2))
    store = FakeStore([stale], cp_refs={"CP-A-MAT-BILANGAN": CpRef(1, "CP-A-MAT-BILANGAN", "old")})
    report = _watcher(store, _fetcher(_handler(CHANGED_BODY))).run(due=True)
    assert [s.status for s in report.summaries] == [STATUS_CHANGED]


def test_watch_manual_frequency_is_never_due() -> None:
    target = _target(baseline_hash=sha256_hex(BASELINE_BODY), frequency="manual", last_checked_at=None)
    store = FakeStore([target])
    report = _watcher(store, _fetcher(_handler(CHANGED_BODY))).run(due=True)
    assert [s.status for s in report.summaries] == [STATUS_SKIPPED]


def test_watch_lock_held_returns_locked_report() -> None:
    store = _changed_store(lock_available=False)
    report = _watcher(store, _fetcher(_handler(CHANGED_BODY))).run()
    assert report.locked is True
    assert report.summaries == []
    assert store.events == []
    assert store.unlock_calls == 0


def test_watch_releases_lock_after_a_run() -> None:
    store = _changed_store()
    _watcher(store, _fetcher(_handler(CHANGED_BODY))).run()
    assert store.lock_calls == 1
    assert store.unlock_calls == 1


# ------------------------------------------------------------- watcher: offline


def test_watch_from_file_reads_a_local_body(tmp_path: Path) -> None:
    path = tmp_path / "kurikulum.txt"
    path.write_bytes(CHANGED_BODY)
    store = _changed_store()
    report = _watcher(store).run(codes=["KEMENDIKDASMEN-KURIKULUM"], from_file=str(path))
    assert [s.status for s in report.summaries] == [STATUS_CHANGED]
    assert store.state_saves[-1]["baseline_hash"] == sha256_hex(CHANGED_BODY)


def test_watch_no_targets_returns_empty_report() -> None:
    store = FakeStore([])
    report = _watcher(store).run()
    assert report.summaries == []
    assert store.lock_calls == 0
