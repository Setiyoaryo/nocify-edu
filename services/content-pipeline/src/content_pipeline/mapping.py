"""Chunk → curriculum mapping (§16–§18).

M3 implements the deterministic and lexical layers; M6 adds an optional
semantic layer: a chunk is scored against each skill's terms (name, description,
chapter) and, when embeddings exist, against the skill's embedding. The two are
blended so the semantic signal can only *lift* the lexical score, never lower it.
A reviewer remains the final authority (§17, §24).

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
from content_pipeline.embeddings import OWNER_CHUNK, OWNER_SKILL, EmbeddingStore
from content_pipeline.similarity import cosine, semantic_confidence
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
    method: str = "lexical"


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


def blend_confidence(lexical: float, semantic: float) -> float:
    """Combine the deterministic and semantic scores.

    The semantic signal may only lift the lexical score, never lower it: a
    genuine paraphrase (lexical ~0.5, semantic ~0.9) reaches the review band,
    while a semantic-only match (lexical 0, semantic 1.0) stays honestly manual.
    """
    return round(max(lexical, 0.5 * (lexical + semantic)), 3)


def best_hybrid_mapping(
    chunk_text: str,
    heading: str,
    chunk_vector: list[float] | None,
    skills: tuple[SkillRef, ...],
    skill_vectors: dict[int, list[float]],
) -> Mapping | None:
    """Lexical + semantic mapping. Falls back to lexical when a vector is missing."""
    best: Mapping | None = None
    for skill in skills:
        lexical, reason = score_skill(chunk_text, heading, skill)
        skill_vector = skill_vectors.get(skill.id)
        similarity = cosine(chunk_vector, skill_vector) if chunk_vector and skill_vector else 0.0
        semantic = semantic_confidence(similarity)
        combined = blend_confidence(lexical, semantic)
        if combined <= 0:
            continue
        if best is None or combined > best.confidence:
            parts = [reason] if reason else []
            method = "lexical"
            if semantic > 0:
                parts.append(f"semantic_cos={similarity:.2f}; semantic_conf={semantic:.2f}")
                if combined > lexical:
                    method = "hybrid"
            best = Mapping(
                skill=skill, confidence=combined, reason="; ".join(parts), method=method
            )
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
    def __init__(
        self,
        db: Database,
        *,
        embeddings: EmbeddingStore | None = None,
        semantic_model: str = "",
    ) -> None:
        self._db = db
        self._embeddings = embeddings
        self._semantic_model = semantic_model

    def run(
        self,
        *,
        codes: list[str] | None = None,
        limit: int | None = None,
        force: bool = False,
        semantic: bool = False,
    ) -> list[MapSummary]:
        taxonomy = load_taxonomy(self._db)
        if not taxonomy.skills:
            log.warning("no active curriculum skills; nothing to map onto")
            return []

        chunk_vectors: dict[int, list[float]] = {}
        skill_vectors: dict[int, list[float]] = {}
        if semantic:
            if self._embeddings is None:
                raise ValueError("semantic mapping requires an EmbeddingStore")
            chunk_vectors = self._embeddings.get_many(OWNER_CHUNK, self._semantic_model)
            skill_vectors = self._embeddings.get_many(OWNER_SKILL, self._semantic_model)

        counters: dict[str, dict[str, int]] = {}
        for chunk in self._pending(codes, limit, force):
            counts = counters.setdefault(
                chunk.source_code,
                {"mapped": 0, "unmapped": 0, "high": 0, "review": 0, "manual": 0},
            )
            if semantic:
                mapping = best_hybrid_mapping(
                    chunk.text,
                    chunk.heading,
                    chunk_vectors.get(chunk.chunk_id),
                    taxonomy.skills,
                    skill_vectors,
                )
            else:
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
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
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
                    mapping.method, mapping.confidence, mapping.reason,
                ),
            )
