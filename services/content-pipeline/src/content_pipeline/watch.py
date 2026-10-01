"""Curriculum change watcher (§27, §28, §44).

Polls canonical curriculum source documents, hashes them, and — only when the
hash changed — diffs their CP statements against canonical `curriculum_cp` and
records a frozen impact snapshot in `content_staging`.

The flow (§44)::

    watcher -> source changed -> document diff -> changed CPs
            -> affected skills -> affected questions -> review queue

M7 is *detect + review only*: this module never mutates canonical curriculum.
The reviewer decision lives in the canonical ledger `curriculum_change_reviews`
(Go-owned). The running baseline lives in staging (`curriculum_watch_state`)
because the canonical `document_hash` is the untouched approved anchor.

The store is behind a Protocol so orchestration can be tested without a DB, and
the HTTP client is injectable so tests never touch the network (the `Fetcher`
pattern).
"""

from __future__ import annotations

import logging
from collections.abc import Mapping, Sequence
from dataclasses import dataclass
from datetime import UTC, datetime, timedelta
from pathlib import Path
from typing import Protocol, runtime_checkable

from content_pipeline.config import Settings
from content_pipeline.curriculum_diff import CpChange, diff_cp_statements, extract_cp_statements
from content_pipeline.db import Database
from content_pipeline.fetcher import Fetcher, FetchError, sha256_hex
from content_pipeline.parsers import UnsupportedDocumentError, detect_document_type, parse
from content_pipeline.safety import UnsafeURLError, host_of
from content_pipeline.storage import RawArchive
from content_pipeline.taxonomy import (
    CpRef,
    SourceDocumentRef,
    load_active_version_id,
    load_cp_statements,
    load_source_documents,
)

log = logging.getLogger("content_pipeline.watch")

# Distinct from the Go migration lock (727274).
ADVISORY_LOCK_KEY = 727275

STATUS_PENDING = "pending"
STATUS_UNCHANGED = "unchanged"
STATUS_NOT_MODIFIED = "not_modified"
STATUS_CHANGED = "changed"
STATUS_SKIPPED = "skipped"
STATUS_FAILED = "failed"

_FREQUENCY_DAYS = {"daily": 1, "weekly": 7, "monthly": 30}


# ------------------------------------------------------------------- records


@dataclass(frozen=True)
class WatchTarget:
    document: SourceDocumentRef
    url: str
    baseline_hash: str
    baseline_source: str
    etag: str
    last_modified: str
    check_frequency: str
    last_checked_at: datetime | None


@dataclass(frozen=True)
class QuestionRef:
    question_id: str
    code: str
    skill_id: int
    status: str


@dataclass(frozen=True)
class AffectedQuestion:
    question_id: str
    code: str
    skill_id: int
    status: str
    match: str  # 'skill' | 'provenance'


@dataclass(frozen=True)
class ImpactRecord:
    change: CpChange
    cp_id: int | None
    affected_skills: int
    questions: tuple[AffectedQuestion, ...]


@dataclass(frozen=True)
class WatchSummary:
    document_code: str
    url: str
    status: str
    change_event_id: int | None
    cp_added: int
    cp_changed: int
    cp_removed: int
    affected_questions: int
    message: str


@dataclass(frozen=True)
class WatchReport:
    summaries: list[WatchSummary]
    dry_run: bool
    locked: bool = False

    @property
    def changed(self) -> int:
        return sum(1 for summary in self.summaries if summary.status == STATUS_CHANGED)


def build_impacts(
    changes: Sequence[CpChange],
    cp_refs: Mapping[str, CpRef],
    skills_by_cp: Mapping[int, Sequence[int]],
    questions_by_skill: Mapping[int, Sequence[QuestionRef]],
    questions_by_cp: Mapping[int, Sequence[QuestionRef]],
) -> list[ImpactRecord]:
    """Assemble per-CP impacts. Pure, so it is tested without a DB.

    A changed CP affects the skills that point at it (`skills.cp_id`) and,
    transitively, every question on those skills. A question may also be linked
    directly through `question_provenance.cp_id`.
    """
    impacts: list[ImpactRecord] = []
    for change in changes:
        ref = cp_refs.get(change.code)
        cp_id = ref.id if ref is not None else None
        skill_ids = tuple(skills_by_cp.get(cp_id, ())) if cp_id is not None else ()
        questions: dict[str, AffectedQuestion] = {}
        for skill_id in skill_ids:
            for question in questions_by_skill.get(skill_id, ()):
                questions[question.question_id] = AffectedQuestion(
                    question_id=question.question_id, code=question.code, skill_id=question.skill_id,
                    status=question.status, match="skill",
                )
        if cp_id is not None:
            for question in questions_by_cp.get(cp_id, ()):
                questions.setdefault(
                    question.question_id,
                    AffectedQuestion(
                        question_id=question.question_id, code=question.code,
                        skill_id=question.skill_id, status=question.status, match="provenance",
                    ),
                )
        impacts.append(
            ImpactRecord(
                change=change,
                cp_id=cp_id,
                affected_skills=len(skill_ids),
                questions=tuple(sorted(questions.values(), key=lambda q: q.question_id)),
            )
        )
    return impacts


# --------------------------------------------------------------------- store


@runtime_checkable
class WatchStore(Protocol):
    """The persistence the watcher needs (fakeable in tests)."""

    def targets(self, codes: list[str] | None, *, require_url: bool = True) -> list[WatchTarget]: ...

    def save_state(
        self, target: WatchTarget, *, status: str, baseline_hash: str, baseline_source: str,
        etag: str, last_modified: str, error: str = "", changed: bool = False,
    ) -> None: ...

    def create_event(
        self, target: WatchTarget, *, observed_hash: str, detection: str, http_status: int | None,
        etag: str, last_modified: str, document_type: str, storage_key: str,
    ) -> int: ...

    def fail_event(self, event_id: int, error: str) -> None: ...

    def finish_event(
        self, event_id: int, *, cp_added: int, cp_changed: int, cp_removed: int
    ) -> None: ...

    def save_impacts(self, event_id: int, impacts: list[ImpactRecord]) -> None: ...

    def cp_statements(self, curriculum_version_id: int) -> dict[str, CpRef]: ...

    def active_version_id(self) -> int | None: ...

    def try_lock(self) -> bool: ...

    def unlock(self) -> None: ...

    def skills_for_cps(self, cp_ids: list[int]) -> dict[int, tuple[int, ...]]: ...

    def questions_for_skills(self, skill_ids: list[int]) -> dict[int, tuple[QuestionRef, ...]]: ...

    def questions_for_cps(self, cp_ids: list[int]) -> dict[int, tuple[QuestionRef, ...]]: ...


@dataclass(frozen=True)
class _StateRow:
    url: str
    baseline_hash: str
    baseline_source: str
    etag: str
    last_modified: str
    check_frequency: str
    last_checked_at: datetime | None


class PgWatchStore:
    """PostgreSQL-backed :class:`WatchStore`. Writes only `content_staging`."""

    def __init__(self, db: Database) -> None:
        self._db = db

    def targets(self, codes: list[str] | None, *, require_url: bool = True) -> list[WatchTarget]:
        states = self._states()
        targets: list[WatchTarget] = []
        for document in load_source_documents(self._db):
            if codes and document.code not in codes:
                continue
            state = states.get(document.id)
            url = document.source_url or (state.url if state is not None else "")
            if require_url and not url:
                continue
            targets.append(
                WatchTarget(
                    document=document,
                    url=url,
                    baseline_hash=state.baseline_hash if state is not None else document.document_hash,
                    baseline_source=state.baseline_source if state is not None else "canonical",
                    etag=state.etag if state is not None else "",
                    last_modified=state.last_modified if state is not None else "",
                    check_frequency=state.check_frequency if state is not None else "weekly",
                    last_checked_at=state.last_checked_at if state is not None else None,
                )
            )
        return targets

    def _states(self) -> dict[str, _StateRow]:
        with self._db.transaction() as cur:
            cur.execute(
                "SELECT source_document_id::text, url, baseline_hash, baseline_source, etag, "
                "last_modified, check_frequency, last_checked_at "
                "FROM content_staging.curriculum_watch_state"
            )
            rows = cur.fetchall()
        return {
            str(r[0]): _StateRow(
                url=str(r[1]), baseline_hash=str(r[2]), baseline_source=str(r[3]),
                etag=str(r[4]), last_modified=str(r[5]), check_frequency=str(r[6]),
                last_checked_at=r[7],
            )
            for r in rows
        }

    def save_state(
        self, target: WatchTarget, *, status: str, baseline_hash: str, baseline_source: str,
        etag: str, last_modified: str, error: str = "", changed: bool = False,
    ) -> None:
        document = target.document
        with self._db.transaction() as cur:
            cur.execute(
                """
                INSERT INTO content_staging.curriculum_watch_state
                    (source_document_id, source_document_code, curriculum_version_id, url,
                     baseline_hash, baseline_source, etag, last_modified, check_frequency,
                     last_checked_at, last_changed_at, last_status, last_error, updated_at)
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, now(),
                        CASE WHEN %s THEN now() END, %s, %s, now())
                ON CONFLICT (source_document_id) DO UPDATE SET
                    source_document_code = EXCLUDED.source_document_code,
                    curriculum_version_id = EXCLUDED.curriculum_version_id,
                    url = EXCLUDED.url,
                    baseline_hash = EXCLUDED.baseline_hash,
                    baseline_source = EXCLUDED.baseline_source,
                    etag = EXCLUDED.etag,
                    last_modified = EXCLUDED.last_modified,
                    check_frequency = EXCLUDED.check_frequency,
                    last_checked_at = EXCLUDED.last_checked_at,
                    last_changed_at = CASE WHEN %s THEN now()
                                           ELSE content_staging.curriculum_watch_state.last_changed_at END,
                    last_status = EXCLUDED.last_status,
                    last_error = EXCLUDED.last_error,
                    updated_at = now()
                """,
                (
                    document.id, document.code, document.curriculum_version_id, target.url,
                    baseline_hash, baseline_source, etag, last_modified, target.check_frequency,
                    changed, status, error, changed,
                ),
            )

    def create_event(
        self, target: WatchTarget, *, observed_hash: str, detection: str, http_status: int | None,
        etag: str, last_modified: str, document_type: str, storage_key: str,
    ) -> int:
        document = target.document
        with self._db.transaction() as cur:
            cur.execute(
                """
                INSERT INTO content_staging.curriculum_change_events
                    (source_document_id, source_document_code, curriculum_version_id, url,
                     baseline_hash, observed_hash, http_status, etag, last_modified, detection,
                     document_type, storage_key)
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
                ON CONFLICT (source_document_id, observed_hash) WHERE status IN ('detected', 'failed')
                DO UPDATE SET baseline_hash = EXCLUDED.baseline_hash,
                              http_status = EXCLUDED.http_status,
                              etag = EXCLUDED.etag,
                              last_modified = EXCLUDED.last_modified,
                              updated_at = now()
                RETURNING id
                """,
                (
                    document.id, document.code, document.curriculum_version_id, target.url,
                    target.baseline_hash, observed_hash, http_status, etag, last_modified,
                    detection, document_type, storage_key,
                ),
            )
            row = cur.fetchone()
        if row is None:  # pragma: no cover - RETURNING always yields a row
            raise RuntimeError("curriculum_change_events insert returned no id")
        return int(row[0])

    def fail_event(self, event_id: int, error: str) -> None:
        with self._db.transaction() as cur:
            cur.execute(
                "UPDATE content_staging.curriculum_change_events "
                "SET status = 'failed', error = %s, updated_at = now() WHERE id = %s",
                (error[:500], event_id),
            )

    def finish_event(
        self, event_id: int, *, cp_added: int, cp_changed: int, cp_removed: int
    ) -> None:
        with self._db.transaction() as cur:
            cur.execute(
                "UPDATE content_staging.curriculum_change_events "
                "SET status = 'analyzed', cp_added = %s, cp_changed = %s, cp_removed = %s, "
                "updated_at = now() WHERE id = %s",
                (cp_added, cp_changed, cp_removed, event_id),
            )

    def save_impacts(self, event_id: int, impacts: list[ImpactRecord]) -> None:
        for impact in impacts:
            with self._db.transaction() as cur:
                cur.execute(
                    """
                    INSERT INTO content_staging.curriculum_change_impacts
                        (change_event_id, cp_id, cp_code, change_type, old_statement, new_statement,
                         statement_hash, affected_skills, affected_questions)
                    VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)
                    ON CONFLICT (change_event_id, cp_code) DO UPDATE SET
                        cp_id = EXCLUDED.cp_id,
                        change_type = EXCLUDED.change_type,
                        old_statement = EXCLUDED.old_statement,
                        new_statement = EXCLUDED.new_statement,
                        statement_hash = EXCLUDED.statement_hash,
                        affected_skills = EXCLUDED.affected_skills,
                        affected_questions = EXCLUDED.affected_questions
                    RETURNING id
                    """,
                    (
                        event_id, impact.cp_id, impact.change.code, impact.change.change_type,
                        impact.change.old_statement, impact.change.new_statement,
                        impact.change.statement_hash, impact.affected_skills, len(impact.questions),
                    ),
                )
                row = cur.fetchone()
                if row is None:  # pragma: no cover
                    raise RuntimeError("curriculum_change_impacts insert returned no id")
                impact_id = int(row[0])
                for question in impact.questions:
                    cur.execute(
                        """
                        INSERT INTO content_staging.curriculum_change_questions
                            (impact_id, question_id, question_code, skill_id, match, status)
                        VALUES (%s, %s, %s, %s, %s, %s)
                        ON CONFLICT (impact_id, question_id) DO NOTHING
                        """,
                        (
                            impact_id, question.question_id, question.code, question.skill_id,
                            question.match, question.status,
                        ),
                    )

    def cp_statements(self, curriculum_version_id: int) -> dict[str, CpRef]:
        return load_cp_statements(self._db, curriculum_version_id)

    def active_version_id(self) -> int | None:
        return load_active_version_id(self._db)

    def try_lock(self) -> bool:
        with self._db.transaction() as cur:
            cur.execute("SELECT pg_try_advisory_lock(%s)", (ADVISORY_LOCK_KEY,))
            row = cur.fetchone()
        return bool(row[0]) if row is not None else False

    def unlock(self) -> None:
        with self._db.transaction() as cur:
            cur.execute("SELECT pg_advisory_unlock(%s)", (ADVISORY_LOCK_KEY,))

    def skills_for_cps(self, cp_ids: list[int]) -> dict[int, tuple[int, ...]]:
        if not cp_ids:
            return {}
        with self._db.transaction() as cur:
            cur.execute("SELECT id, cp_id FROM skills WHERE cp_id = ANY(%s)", (cp_ids,))
            rows = cur.fetchall()
        grouped: dict[int, list[int]] = {}
        for row in rows:
            grouped.setdefault(int(row[1]), []).append(int(row[0]))
        return {cp_id: tuple(sorted(ids)) for cp_id, ids in grouped.items()}

    def questions_for_skills(self, skill_ids: list[int]) -> dict[int, tuple[QuestionRef, ...]]:
        if not skill_ids:
            return {}
        with self._db.transaction() as cur:
            cur.execute(
                "SELECT id::text, code, skill_id, status FROM questions WHERE skill_id = ANY(%s)",
                (skill_ids,),
            )
            rows = cur.fetchall()
        grouped: dict[int, list[QuestionRef]] = {}
        for row in rows:
            grouped.setdefault(int(row[2]), []).append(
                QuestionRef(question_id=str(row[0]), code=str(row[1]), skill_id=int(row[2]), status=str(row[3]))
            )
        return {skill_id: tuple(items) for skill_id, items in grouped.items()}

    def questions_for_cps(self, cp_ids: list[int]) -> dict[int, tuple[QuestionRef, ...]]:
        if not cp_ids:
            return {}
        with self._db.transaction() as cur:
            cur.execute(
                "SELECT q.id::text, q.code, q.skill_id, q.status, p.cp_id "
                "FROM questions q JOIN question_provenance p ON p.question_id = q.id "
                "WHERE p.cp_id = ANY(%s)",
                (cp_ids,),
            )
            rows = cur.fetchall()
        grouped: dict[int, list[QuestionRef]] = {}
        for row in rows:
            grouped.setdefault(int(row[4]), []).append(
                QuestionRef(question_id=str(row[0]), code=str(row[1]), skill_id=int(row[2]), status=str(row[3]))
            )
        return {cp_id: tuple(items) for cp_id, items in grouped.items()}


# ------------------------------------------------------------------- watcher


class CurriculumWatcher:
    """Detects curriculum source-document changes and records their impact."""

    def __init__(
        self,
        db: Database,
        settings: Settings,
        *,
        store: WatchStore | None = None,
        archive: RawArchive | None = None,
        fetcher: Fetcher | None = None,
    ) -> None:
        self._db = db
        self._settings = settings
        self._store: WatchStore = store if store is not None else PgWatchStore(db)
        self._archive = archive if archive is not None else RawArchive(settings)
        self._fetcher = fetcher

    def run(
        self,
        *,
        codes: list[str] | None = None,
        limit: int | None = None,
        dry_run: bool = False,
        force: bool = False,
        due: bool = False,
        from_file: str | None = None,
    ) -> WatchReport:
        targets = self._store.targets(codes, require_url=from_file is None)
        if limit:
            targets = targets[:limit]
        if not targets:
            return WatchReport(summaries=[], dry_run=dry_run)

        if not dry_run and not self._store.try_lock():
            log.warning("another watch run holds the advisory lock; skipping")
            return WatchReport(summaries=[], dry_run=dry_run, locked=True)

        fetcher = self._fetcher if from_file is None else None
        owned_fetcher = False
        if fetcher is None and from_file is None:
            fetcher = self._make_fetcher(targets)
            owned_fetcher = True
        try:
            summaries = [
                self._check(target, fetcher=fetcher, dry_run=dry_run, force=force, due=due,
                            from_file=from_file)
                for target in targets
            ]
        finally:
            if owned_fetcher and fetcher is not None:
                fetcher.close()
            if not dry_run:
                self._store.unlock()
        return WatchReport(summaries=summaries, dry_run=dry_run)

    # ----------------------------------------------------------------- internal

    def _make_fetcher(self, targets: list[WatchTarget]) -> Fetcher:
        hosts = {host_of(target.url) for target in targets if target.url}
        return Fetcher(
            allowed_hosts=hosts,
            user_agent=self._settings.user_agent,
            timeout_seconds=self._settings.http_timeout_seconds,
            max_response_bytes=self._settings.max_response_bytes,
            max_redirects=self._settings.max_redirects,
        )

    def _check(
        self, target: WatchTarget, *, fetcher: Fetcher | None, dry_run: bool, force: bool,
        due: bool, from_file: str | None,
    ) -> WatchSummary:
        if due and not self._is_due(target):
            return self._summary(target, STATUS_SKIPPED, "not due")

        if from_file is not None:
            body = Path(from_file).read_bytes()
            content_type = ""
            http_status: int | None = None
            etag = last_modified = ""
            observed_hash = sha256_hex(body)
        else:
            assert fetcher is not None
            try:
                result = fetcher.fetch(target.url, etag=target.etag, last_modified=target.last_modified)
            except (FetchError, UnsafeURLError) as exc:
                if not dry_run:
                    self._store.save_state(
                        target, status=STATUS_FAILED, baseline_hash=target.baseline_hash,
                        baseline_source=target.baseline_source, etag=target.etag,
                        last_modified=target.last_modified, error=str(exc),
                    )
                return self._summary(target, STATUS_FAILED, str(exc))
            if result.not_modified:
                if not dry_run:
                    self._store.save_state(
                        target, status=STATUS_NOT_MODIFIED, baseline_hash=target.baseline_hash,
                        baseline_source=target.baseline_source, etag=result.etag or target.etag,
                        last_modified=result.last_modified or target.last_modified,
                    )
                return self._summary(target, STATUS_NOT_MODIFIED, "not modified")
            body = result.body or b""
            content_type = result.content_type
            http_status = result.http_status
            etag = result.etag or target.etag
            last_modified = result.last_modified or target.last_modified
            observed_hash = result.content_hash

        # Bootstrap: we only learned the baseline, nothing changed yet.
        if not target.baseline_hash:
            if not dry_run:
                self._store.save_state(
                    target, status=STATUS_SKIPPED, baseline_hash=observed_hash,
                    baseline_source="observed", etag=etag, last_modified=last_modified,
                )
            return self._summary(target, STATUS_SKIPPED, "baseline recorded")

        if observed_hash == target.baseline_hash and not force:
            if not dry_run:
                self._store.save_state(
                    target, status=STATUS_UNCHANGED, baseline_hash=target.baseline_hash,
                    baseline_source=target.baseline_source, etag=etag, last_modified=last_modified,
                )
            return self._summary(target, STATUS_UNCHANGED, "unchanged")

        return self._analyze(
            target, body=body, content_type=content_type, observed_hash=observed_hash,
            http_status=http_status, etag=etag, last_modified=last_modified, dry_run=dry_run,
        )

    def _analyze(
        self, target: WatchTarget, *, body: bytes, content_type: str, observed_hash: str,
        http_status: int | None, etag: str, last_modified: str, dry_run: bool,
    ) -> WatchSummary:
        try:
            document_type = detect_document_type(content_type, body)
        except UnsupportedDocumentError as exc:
            document_type = ""
            if not dry_run:
                self._store.save_state(
                    target, status=STATUS_FAILED, baseline_hash=target.baseline_hash,
                    baseline_source=target.baseline_source, etag=etag, last_modified=last_modified,
                    error=str(exc),
                )
            return self._summary(target, STATUS_FAILED, str(exc))

        storage_key = ""
        event_id: int | None = None
        if not dry_run:
            if self._archive.enabled:
                try:
                    storage_key = self._archive.store(
                        target.document.code, body, observed_hash, content_type or "text/plain"
                    )
                except Exception as exc:  # noqa: BLE001 - archiving is best-effort
                    log.warning("archive failed for %s: %s", target.document.code, exc)
            event_id = self._store.create_event(
                target, observed_hash=observed_hash, detection="content_hash",
                http_status=http_status, etag=etag, last_modified=last_modified,
                document_type=document_type, storage_key=storage_key,
            )

        try:
            version_id = target.document.curriculum_version_id or self._store.active_version_id()
            cp_refs = self._store.cp_statements(version_id) if version_id else {}
            parsed = parse(body, content_type)
            text = "\n".join(f"{section.heading}\n{section.text}" for section in parsed.sections)
            current = extract_cp_statements(text)
            previous = {code: ref.statement for code, ref in cp_refs.items()}
            changes = diff_cp_statements(previous, current)
            impacts = self._impact_data(changes, cp_refs)
        except (UnsupportedDocumentError, ValueError) as exc:
            if event_id is not None:
                self._store.fail_event(event_id, str(exc))
                self._store.save_state(
                    target, status=STATUS_FAILED, baseline_hash=target.baseline_hash,
                    baseline_source=target.baseline_source, etag=etag, last_modified=last_modified,
                    error=str(exc),
                )
            return self._summary(target, STATUS_FAILED, str(exc), change_event_id=event_id)

        counts = {
            "added": sum(1 for impact in impacts if impact.change.change_type == "added"),
            "changed": sum(1 for impact in impacts if impact.change.change_type == "changed"),
            "removed": sum(1 for impact in impacts if impact.change.change_type == "removed"),
        }
        if not dry_run and event_id is not None:
            self._store.save_impacts(event_id, impacts)
            self._store.finish_event(
                event_id, cp_added=counts["added"], cp_changed=counts["changed"],
                cp_removed=counts["removed"],
            )
            self._store.save_state(
                target, status=STATUS_CHANGED, baseline_hash=observed_hash,
                baseline_source="observed", etag=etag, last_modified=last_modified, changed=True,
            )
        return WatchSummary(
            document_code=target.document.code,
            url=target.url,
            status=STATUS_CHANGED,
            change_event_id=event_id,
            cp_added=counts["added"],
            cp_changed=counts["changed"],
            cp_removed=counts["removed"],
            affected_questions=sum(len(impact.questions) for impact in impacts),
            message="changed",
        )

    def _impact_data(
        self, changes: Sequence[CpChange], cp_refs: Mapping[str, CpRef]
    ) -> list[ImpactRecord]:
        cp_ids = [cp_refs[change.code].id for change in changes if change.code in cp_refs]
        skills_by_cp = self._store.skills_for_cps(cp_ids)
        skill_ids = sorted({skill for ids in skills_by_cp.values() for skill in ids})
        questions_by_skill = self._store.questions_for_skills(skill_ids)
        questions_by_cp = self._store.questions_for_cps(cp_ids)
        return build_impacts(changes, cp_refs, skills_by_cp, questions_by_skill, questions_by_cp)

    def _is_due(self, target: WatchTarget) -> bool:
        days = _FREQUENCY_DAYS.get(target.check_frequency)
        if days is None:  # 'manual' is only run explicitly
            return False
        if target.last_checked_at is None:
            return True
        return datetime.now(UTC) - target.last_checked_at >= timedelta(days=days)

    def _summary(
        self, target: WatchTarget, status: str, message: str, *, change_event_id: int | None = None
    ) -> WatchSummary:
        return WatchSummary(
            document_code=target.document.code, url=target.url, status=status,
            change_event_id=change_event_id, cp_added=0, cp_changed=0, cp_removed=0,
            affected_questions=0, message=message,
        )


# --------------------------------------------------------------- helpers (CLI)


@dataclass(frozen=True)
class ChangeEventRow:
    id: int
    document_code: str
    url: str
    baseline_hash: str
    observed_hash: str
    cp_added: int
    cp_changed: int
    cp_removed: int
    status: str
    created_at: datetime


def register_document(
    db: Database, *, code: str, url: str, baseline_hash: str = "", frequency: str = "weekly"
) -> str | None:
    """Register a watched document URL in staging. Returns the code, or None if unknown.

    The URL is a staging override used when canonical `source_url` is empty; this
    never writes canonical (M7 is detect + review only).
    """
    document = next((doc for doc in load_source_documents(db) if doc.code == code), None)
    if document is None:
        return None
    baseline = baseline_hash or document.document_hash
    baseline_source = "observed" if baseline_hash else "canonical"
    with db.transaction() as cur:
        cur.execute(
            """
            INSERT INTO content_staging.curriculum_watch_state
                (source_document_id, source_document_code, curriculum_version_id, url,
                 baseline_hash, baseline_source, check_frequency)
            VALUES (%s, %s, %s, %s, %s, %s, %s)
            ON CONFLICT (source_document_id) DO UPDATE SET
                source_document_code = EXCLUDED.source_document_code,
                curriculum_version_id = EXCLUDED.curriculum_version_id,
                url = EXCLUDED.url,
                baseline_hash = CASE WHEN EXCLUDED.baseline_hash <> ''
                                     THEN EXCLUDED.baseline_hash
                                     ELSE content_staging.curriculum_watch_state.baseline_hash END,
                baseline_source = EXCLUDED.baseline_source,
                check_frequency = EXCLUDED.check_frequency,
                updated_at = now()
            """,
            (
                document.id, document.code, document.curriculum_version_id, url,
                baseline, baseline_source, frequency,
            ),
        )
    return document.code


def list_change_events(
    db: Database, *, codes: list[str] | None = None, status: str | None = None, limit: int = 50
) -> list[ChangeEventRow]:
    conditions: list[str] = []
    params: list[object] = []
    if codes:
        conditions.append("source_document_code = ANY(%s)")
        params.append(codes)
    if status:
        conditions.append("status = %s")
        params.append(status)
    where = f" WHERE {' AND '.join(conditions)}" if conditions else ""
    params.append(limit)
    with db.transaction() as cur:
        cur.execute(
            "SELECT id, source_document_code, url, baseline_hash, observed_hash, cp_added, "
            "cp_changed, cp_removed, status, created_at "
            f"FROM content_staging.curriculum_change_events{where} "
            "ORDER BY created_at DESC, id DESC LIMIT %s",
            params,
        )
        rows = cur.fetchall()
    return [
        ChangeEventRow(
            id=int(r[0]), document_code=str(r[1]), url=str(r[2]), baseline_hash=str(r[3]),
            observed_hash=str(r[4]), cp_added=int(r[5]), cp_changed=int(r[6]), cp_removed=int(r[7]),
            status=str(r[8]), created_at=r[9],
        )
        for r in rows
    ]
