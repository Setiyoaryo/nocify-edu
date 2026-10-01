"""Chunk → curriculum mapping (§16–§18).

M3 implements the deterministic and lexical layers only: a chunk is scored
against each skill's terms (name, description, chapter) and the best match is
recorded with a confidence. Embeddings and LLM assistance are M6; a reviewer
remains the final authority (§17, §24).

Confidence is a signal, never a publish decision (§18):
    >= 0.90  high
    0.75–0.89 needs review
    < 0.75    manual review
"""

from __future__ import annotations

import logging
import re
from dataclasses import dataclass

from content_pipeline.db import Database
from content_pipeline.taxonomy import SkillRef, Taxonomy, load_taxonomy

log = logging.getLogger("content_pipeline.mapping")

HIGH_CONFIDENCE = 0.90
REVIEW_CONFIDENCE = 0.75

_WORD = re.compile(r"[a-z0-9]+")

# Small Indonesian stopword list; enough to stop glue words from dominating.
STOPWORDS = frozenset(
    {
        "dan", "atau", "yang", "untuk", "pada", "dengan", "dari", "ke", "di",
        "se", "sebagai", "adalah", "ialah", "ini", "itu", "tidak", "bukan",
        "akan", "dapat", "dapatkan", "serta", "juga", "karena", "agar",
        "supaya", "dalam", "oleh", "para", "siswa", "peserta", "didik", "anak",
        "murid", "kelas", "melalui", "secara", "tentang",
    }
)


def tokens(text: str) -> set[str]:
    return {
        word
        for word in _WORD.findall(text.lower())
        if len(word) >= 3 and word not in STOPWORDS
    }


def skill_terms(skill: SkillRef) -> set[str]:
    return tokens(f"{skill.name} {skill.description} {skill.chapter_name}")


def _f1(left: set[str], right: set[str]) -> float:
    if not left or not right:
        return 0.0
    shared = len(left & right)
    if not shared:
        return 0.0
    precision = shared / len(left)
    recall = shared / len(right)
    return 2 * precision * recall / (precision + recall)


@dataclass(frozen=True)
class Mapping:
    skill: SkillRef
    confidence: float
    reason: str


def confidence_band(confidence: float) -> str:
    if confidence >= HIGH_CONFIDENCE:
        return "high"
    if confidence >= REVIEW_CONFIDENCE:
        return "review"
    return "manual"


def score_skill(chunk_text: str, heading: str, skill: SkillRef) -> tuple[float, str]:
    """Score one skill against a chunk. Pure function, easy to test and tune."""
    chunk_tokens = tokens(chunk_text)
    heading_tokens = tokens(heading)
    terms = skill_terms(skill)

    body = _f1(chunk_tokens, terms)
    head = _f1(heading_tokens, terms)
    score = 0.75 * body + 0.25 * head

    reasons: list[str] = []
    if body:
        reasons.append(f"body_f1={body:.2f}")
    if head:
        reasons.append(f"heading_f1={head:.2f}")
    name_tokens = tokens(skill.name)
    if heading_tokens and name_tokens and (heading_tokens <= name_tokens or name_tokens <= heading_tokens):
        score = min(1.0, score + 0.25)
        reasons.append("heading matches skill name")

    return round(score, 3), "; ".join(reasons)


def best_mapping(chunk_text: str, heading: str, skills: tuple[SkillRef, ...]) -> Mapping | None:
    """Return the highest-scoring skill, or None when nothing matches."""
    best: Mapping | None = None
    for skill in skills:
        score, reason = score_skill(chunk_text, heading, skill)
        if score <= 0:
            continue
        if best is None or score > best.confidence:
            best = Mapping(skill=skill, confidence=score, reason=reason)
    return best


@dataclass(frozen=True)
class MapSummary:
    source_code: str
    mapped: int
    unmapped: int
    high: int
    review: int
    manual: int


@dataclass(frozen=True)
class _PendingChunk:
    chunk_id: int
    document_id: int
    source_id: int
    source_code: str
    heading: str
    text: str


class Mapper:
    def __init__(self, db: Database) -> None:
        self._db = db

    def run(
        self,
        *,
        codes: list[str] | None = None,
        limit: int | None = None,
        force: bool = False,
    ) -> list[MapSummary]:
        taxonomy = load_taxonomy(self._db)
        if not taxonomy.skills:
            log.warning("no active curriculum skills; nothing to map onto")
            return []

        counters: dict[str, dict[str, int]] = {}
        for chunk in self._pending(codes, limit, force):
            counts = counters.setdefault(
                chunk.source_code,
                {"mapped": 0, "unmapped": 0, "high": 0, "review": 0, "manual": 0},
            )
            mapping = best_mapping(chunk.text, chunk.heading, taxonomy.skills)
            if mapping is None:
                counts["unmapped"] += 1
                continue
            self._store(chunk, taxonomy, mapping)
            counts["mapped"] += 1
            counts[confidence_band(mapping.confidence)] += 1

        return [MapSummary(source_code=code, **counters[code]) for code in sorted(counters)]

    def _pending(
        self, codes: list[str] | None, limit: int | None, force: bool
    ) -> list[_PendingChunk]:
        conditions = ["d.extraction_status = 'extracted'"]
        params: list[object] = []
        if codes:
            conditions.append("s.code = ANY(%s)")
            params.append(codes)
        if not force:
            conditions.append(
                "NOT EXISTS (SELECT 1 FROM content_staging.mapping_candidates m "
                "WHERE m.chunk_id = c.id)"
            )
        where = " AND ".join(conditions)
        limit_sql = ""
        if limit:
            limit_sql = " LIMIT %s"
            params.append(limit)

        sql = f"""
            SELECT c.id, c.document_id, d.source_id, s.code, c.heading, c.text
            FROM content_staging.document_chunks c
            JOIN content_staging.source_documents d ON d.id = c.document_id
            JOIN content_staging.source_registry s ON s.id = d.source_id
            WHERE {where}
            ORDER BY c.document_id, c.sequence
            {limit_sql}
        """
        with self._db.transaction() as cur:
            cur.execute(sql, params)
            rows = cur.fetchall()
        return [
            _PendingChunk(
                chunk_id=int(r[0]),
                document_id=int(r[1]),
                source_id=int(r[2]),
                source_code=str(r[3]),
                heading=str(r[4]),
                text=str(r[5]),
            )
            for r in rows
        ]

    def _store(self, chunk: _PendingChunk, taxonomy: Taxonomy, mapping: Mapping) -> None:
        skill = mapping.skill
        with self._db.transaction() as cur:
            cur.execute(
                """
                INSERT INTO content_staging.mapping_candidates
                    (document_id, chunk_id, source_id, curriculum_version_id, skill_id, cp_id,
                     phase_code, grade_code, subject_code, method, confidence, reason)
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, 'lexical', %s, %s)
                ON CONFLICT (chunk_id) DO UPDATE SET
                    skill_id = EXCLUDED.skill_id,
                    cp_id = EXCLUDED.cp_id,
                    phase_code = EXCLUDED.phase_code,
                    grade_code = EXCLUDED.grade_code,
                    subject_code = EXCLUDED.subject_code,
                    method = EXCLUDED.method,
                    confidence = EXCLUDED.confidence,
                    reason = EXCLUDED.reason,
                    status = 'candidate',
                    created_at = now()
                """,
                (
                    chunk.document_id, chunk.chunk_id, chunk.source_id,
                    taxonomy.curriculum_version_id or None, skill.id, skill.cp_id,
                    skill.phase_code, skill.grade_code, skill.subject_code,
                    mapping.confidence, mapping.reason,
                ),
            )
