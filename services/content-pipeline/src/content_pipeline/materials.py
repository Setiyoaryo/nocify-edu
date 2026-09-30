"""Material candidates (§21, §30, §40).

Turns a mapped chunk into a reviewable teaching snippet with a deterministic
title, summary and learning objective — no AI here (§23: deterministic templates
first). Each candidate carries its child-readability metadata (§30), a content
fingerprint for duplicate detection (§29) and the result of the automatic quality
checks (§40). A human remains the final authority (§24); this module only
prepares and flags.

Idempotent: one material per chunk, re-running replaces it in place.
"""

from __future__ import annotations

import hashlib
import json
import logging
import re
from dataclasses import dataclass

from content_pipeline.db import Database
from content_pipeline.mapping import REVIEW_CONFIDENCE
from content_pipeline.readability import Readability, analyse, difficulty_hint, sentences
from content_pipeline.taxonomy import Taxonomy, load_taxonomy

log = logging.getLogger("content_pipeline.materials")

MIN_WORDS = 15
TITLE_MAX = 80
SUMMARY_MAX_CHARS = 220
SUMMARY_MAX_SENTENCES = 2

_WHITESPACE = re.compile(r"\s+")


@dataclass(frozen=True)
class QaIssue:
    code: str
    message: str
    blocking: bool

    def as_dict(self) -> dict[str, object]:
        return {"code": self.code, "message": self.message, "blocking": self.blocking}


@dataclass(frozen=True)
class MaterialDraft:
    title: str
    summary: str
    learning_objective: str
    difficulty_hint: int
    readability: Readability
    content_hash: str


def material_hash(text: str) -> str:
    """Text fingerprint for duplicate detection (§29 exact-duplicate step)."""
    return hashlib.sha256(_WHITESPACE.sub(" ", text).strip().lower().encode("utf-8")).hexdigest()


def _lower_first(text: str) -> str:
    return text[:1].lower() + text[1:] if text else text


def build_title(heading: str, text: str, skill_name: str) -> str:
    title = _WHITESPACE.sub(" ", heading).strip()
    if not title:
        first = sentences(text)
        title = first[0] if first else ""
    if not title:
        title = skill_name or "Materi"
    if len(title) > TITLE_MAX:
        title = title[:TITLE_MAX].rstrip() + "…"
    return title


def build_summary(text: str) -> str:
    picked: list[str] = []
    for sentence in sentences(text)[:SUMMARY_MAX_SENTENCES]:
        candidate = " ".join([*picked, sentence])
        if picked and len(candidate) > SUMMARY_MAX_CHARS:
            break
        picked.append(sentence)
    return _WHITESPACE.sub(" ", " ".join(picked)).strip()


def build_learning_objective(skill_name: str, cp_code: str) -> str:
    if not skill_name:
        return ""
    suffix = f" ({cp_code})" if cp_code else ""
    return f"Peserta didik dapat {_lower_first(skill_name)}{suffix}."


def build_draft(
    text: str,
    *,
    heading: str = "",
    skill_name: str = "",
    cp_code: str = "",
    phase_code: str = "",
    grade_code: str = "",
) -> MaterialDraft:
    """Pure, deterministic draft. Same input always yields the same output."""
    readability = analyse(text, phase_code=phase_code, grade_code=grade_code)
    return MaterialDraft(
        title=build_title(heading, text, skill_name),
        summary=build_summary(text),
        learning_objective=build_learning_objective(skill_name, cp_code),
        difficulty_hint=difficulty_hint(readability),
        readability=readability,
        content_hash=material_hash(text),
    )


def quality_checks(
    draft: MaterialDraft,
    *,
    skill_id: int | None,
    cp_id: int | None,
    confidence: float,
    duplicate_of: int | None,
) -> list[QaIssue]:
    """Automatic checks (§40). Blocking issues stop a candidate from being 'clean'."""
    issues: list[QaIssue] = []
    if skill_id is None:
        issues.append(QaIssue("missing_skill", "no curriculum skill mapped", True))
    if cp_id is None:
        issues.append(QaIssue("missing_cp_mapping", "no Capaian Pembelajaran mapped", True))
    if duplicate_of is not None:
        issues.append(QaIssue("duplicate", f"same text as material #{duplicate_of}", True))
    if not draft.summary:
        issues.append(QaIssue("missing_summary", "no extractable summary", True))
    if not draft.learning_objective:
        issues.append(QaIssue("missing_objective", "no learning objective could be built", True))
    if draft.readability.word_count < MIN_WORDS:
        issues.append(QaIssue("too_short", f"only {draft.readability.word_count} words", False))
    if draft.readability.too_long:
        issues.append(QaIssue("text_too_long", f"{draft.readability.word_count} words", False))
    if confidence < REVIEW_CONFIDENCE:
        issues.append(QaIssue("low_confidence", f"mapping confidence {confidence:.2f}", False))
    return issues


def has_blocking(issues: list[QaIssue]) -> bool:
    return any(issue.blocking for issue in issues)


@dataclass(frozen=True)
class MaterialSummary:
    source_code: str
    generated: int
    drafts: int
    needs_review: int
    duplicates: int


@dataclass(frozen=True)
class _Pending:
    chunk_id: int
    document_id: int
    source_id: int
    source_code: str
    skill_id: int | None
    cp_id: int | None
    phase_code: str
    grade_code: str
    confidence: float
    heading: str
    text: str
    language: str


class MaterialBuilder:
    def __init__(self, db: Database) -> None:
        self._db = db

    def run(
        self,
        *,
        codes: list[str] | None = None,
        limit: int | None = None,
        force: bool = False,
    ) -> list[MaterialSummary]:
        taxonomy = load_taxonomy(self._db)
        skills = {skill.id: skill for skill in taxonomy.skills}

        counters: dict[str, dict[str, int]] = {}
        for pending in self._pending(codes, limit, force):
            counts = counters.setdefault(
                pending.source_code,
                {"generated": 0, "drafts": 0, "needs_review": 0, "duplicates": 0},
            )
            skill = skills.get(pending.skill_id) if pending.skill_id is not None else None
            draft = build_draft(
                pending.text,
                heading=pending.heading,
                skill_name=skill.name if skill else "",
                cp_code=skill.cp_code if skill else "",
                phase_code=pending.phase_code,
                grade_code=pending.grade_code,
            )
            duplicate_of = self._duplicate_of(draft.content_hash, pending.chunk_id)
            issues = quality_checks(
                draft,
                skill_id=pending.skill_id,
                cp_id=pending.cp_id,
                confidence=pending.confidence,
                duplicate_of=duplicate_of,
            )
            status = "needs_review" if has_blocking(issues) else "draft"
            self._store(pending, taxonomy, draft, issues, status, duplicate_of)

            counts["generated"] += 1
            counts["needs_review" if status == "needs_review" else "drafts"] += 1
            if duplicate_of is not None:
                counts["duplicates"] += 1

        return [MaterialSummary(source_code=code, **counters[code]) for code in sorted(counters)]

    # ----------------------------------------------------------------- internal

    def _pending(self, codes: list[str] | None, limit: int | None, force: bool) -> list[_Pending]:
        conditions = ["mc.status = 'candidate'"]
        params: list[object] = []
        if codes:
            conditions.append("s.code = ANY(%s)")
            params.append(codes)
        if not force:
            conditions.append(
                "NOT EXISTS (SELECT 1 FROM content_staging.material_candidates m "
                "WHERE m.chunk_id = mc.chunk_id)"
            )
        where = " AND ".join(conditions)
        limit_sql = ""
        if limit:
            limit_sql = " LIMIT %s"
            params.append(limit)

        sql = f"""
            SELECT mc.chunk_id, mc.document_id, mc.source_id, s.code,
                   mc.skill_id, mc.cp_id, mc.phase_code, mc.grade_code, mc.confidence,
                   c.heading, c.text, d.language
            FROM content_staging.mapping_candidates mc
            JOIN content_staging.document_chunks c ON c.id = mc.chunk_id
            JOIN content_staging.source_documents d ON d.id = mc.document_id
            JOIN content_staging.source_registry s ON s.id = mc.source_id
            WHERE {where}
            ORDER BY mc.id
            {limit_sql}
        """
        with self._db.transaction() as cur:
            cur.execute(sql, params)
            rows = cur.fetchall()
        return [
            _Pending(
                chunk_id=int(row[0]),
                document_id=int(row[1]),
                source_id=int(row[2]),
                source_code=str(row[3]),
                skill_id=int(row[4]) if row[4] is not None else None,
                cp_id=int(row[5]) if row[5] is not None else None,
                phase_code=str(row[6]),
                grade_code=str(row[7]),
                confidence=float(row[8]),
                heading=str(row[9]),
                text=str(row[10]),
                language=str(row[11]),
            )
            for row in rows
        ]

    def _duplicate_of(self, content_hash: str, chunk_id: int) -> int | None:
        with self._db.transaction() as cur:
            cur.execute(
                """
                SELECT id FROM content_staging.material_candidates
                WHERE content_hash = %s AND chunk_id <> %s
                ORDER BY id LIMIT 1
                """,
                (content_hash, chunk_id),
            )
            row = cur.fetchone()
        return int(row[0]) if row is not None else None

    def _store(
        self,
        pending: _Pending,
        taxonomy: Taxonomy,
        draft: MaterialDraft,
        issues: list[QaIssue],
        status: str,
        duplicate_of: int | None,
    ) -> None:
        readability = draft.readability
        with self._db.transaction() as cur:
            cur.execute(
                """
                INSERT INTO content_staging.material_candidates
                    (document_id, chunk_id, source_id, curriculum_version_id, skill_id, cp_id,
                     phase_code, grade_code, title, summary, learning_objective, difficulty_hint,
                     language, age_band, reading_level, complexity, word_count, sentence_count,
                     avg_sentence_length, visual_required, audio_required, content_hash,
                     duplicate_of, confidence, qa_issues, status)
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s,
                        %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
                ON CONFLICT (chunk_id) DO UPDATE SET
                    curriculum_version_id = EXCLUDED.curriculum_version_id,
                    skill_id = EXCLUDED.skill_id,
                    cp_id = EXCLUDED.cp_id,
                    phase_code = EXCLUDED.phase_code,
                    grade_code = EXCLUDED.grade_code,
                    title = EXCLUDED.title,
                    summary = EXCLUDED.summary,
                    learning_objective = EXCLUDED.learning_objective,
                    difficulty_hint = EXCLUDED.difficulty_hint,
                    language = EXCLUDED.language,
                    age_band = EXCLUDED.age_band,
                    reading_level = EXCLUDED.reading_level,
                    complexity = EXCLUDED.complexity,
                    word_count = EXCLUDED.word_count,
                    sentence_count = EXCLUDED.sentence_count,
                    avg_sentence_length = EXCLUDED.avg_sentence_length,
                    visual_required = EXCLUDED.visual_required,
                    audio_required = EXCLUDED.audio_required,
                    content_hash = EXCLUDED.content_hash,
                    duplicate_of = EXCLUDED.duplicate_of,
                    confidence = EXCLUDED.confidence,
                    qa_issues = EXCLUDED.qa_issues,
                    status = EXCLUDED.status,
                    updated_at = now()
                """,
                (
                    pending.document_id, pending.chunk_id, pending.source_id,
                    taxonomy.curriculum_version_id or None, pending.skill_id, pending.cp_id,
                    pending.phase_code, pending.grade_code, draft.title, draft.summary,
                    draft.learning_objective, draft.difficulty_hint, pending.language,
                    readability.age_band, readability.reading_level, readability.complexity,
                    readability.word_count, readability.sentence_count,
                    readability.avg_sentence_length, readability.visual_required,
                    readability.audio_required, draft.content_hash, duplicate_of,
                    pending.confidence, json.dumps([issue.as_dict() for issue in issues]),
                    status,
                ),
            )
