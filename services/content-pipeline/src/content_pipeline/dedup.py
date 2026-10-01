"""Near-duplicate detection (§29).

Signals: canonical URL + SHA-256 (already handled by the builders) plus semantic
similarity over cached chunk embeddings. Produces the §29 statuses
``exact`` / ``near_duplicate`` / ``related`` / ``unique`` per chunk, then copies
the outcome onto material/question candidates so the reviewer CMS shows it.
"""

from __future__ import annotations

import json
import logging
from dataclasses import dataclass

from content_pipeline.db import Database
from content_pipeline.embeddings import OWNER_CHUNK, EmbeddingStore
from content_pipeline.qa import QaIssue, has_blocking
from content_pipeline.similarity import classify_similarity, cosine

log = logging.getLogger("content_pipeline.dedup")

MATERIAL = "material"
QUESTION = "question"

DEDUP_CODES = frozenset({"near_duplicate", "related"})

_TABLE = {MATERIAL: "material_candidates", QUESTION: "question_candidates"}


class MissingEmbeddingsError(RuntimeError):
    """Raised when dedup is asked to run before any chunk embeddings exist."""


def relation_status(hash_a: str, hash_b: str, similarity: float) -> str:
    """§29 status for a pair: an exact hash match always wins over similarity."""
    if hash_a and hash_a == hash_b:
        return "exact"
    return classify_similarity(similarity)


def dedup_issues(status: str, related_id: int | None) -> list[QaIssue]:
    """QA issues for a chunk's relation. Near-duplicates block; related do not."""
    where = f" (candidate #{related_id})" if related_id else ""
    if status == "near_duplicate":
        return [QaIssue("near_duplicate", f"Sangat mirip dengan kandidat lain{where} (§29).", True)]
    if status == "related":
        return [QaIssue("related", f"Terkait dengan kandidat lain{where} (§29).", False)]
    return []


@dataclass(frozen=True)
class DedupReport:
    chunks_checked: int
    exact: int
    near_duplicate: int
    related: int
    unique: int
    candidates_flagged: int


@dataclass(frozen=True)
class _ChunkRow:
    chunk_id: int
    hash: str


class Deduplicator:
    """Computes chunk relations and propagates them onto candidate rows."""

    def __init__(self, db: Database, *, model: str) -> None:
        self._db = db
        self._model = model
        self._store = EmbeddingStore(db)

    def run(
        self,
        *,
        kinds: tuple[str, ...] = (MATERIAL, QUESTION),
        codes: list[str] | None = None,
        limit: int | None = None,
    ) -> DedupReport:
        vectors = self._store.get_many(OWNER_CHUNK, self._model)
        if not vectors:
            raise MissingEmbeddingsError(
                f"no chunk embeddings for model {self._model!r}; run `embed` first"
            )

        relations: dict[int, tuple[str, int | None, float]] = {}
        counts = {"exact": 0, "near_duplicate": 0, "related": 0, "unique": 0}
        seen: list[_ChunkRow] = []
        for row in self._chunks(codes, limit):
            if row.chunk_id not in vectors:
                continue
            status, target, similarity = self._classify(row, seen, vectors)
            seen.append(row)
            relations[row.chunk_id] = (status, target, similarity)
            counts[status] += 1
            self._store_relation(row.chunk_id, status, target, similarity)

        flagged = 0
        for kind in kinds:
            flagged += self._propagate(kind, relations)

        return DedupReport(
            chunks_checked=len(relations),
            exact=counts["exact"],
            near_duplicate=counts["near_duplicate"],
            related=counts["related"],
            unique=counts["unique"],
            candidates_flagged=flagged,
        )

    def _classify(
        self,
        row: _ChunkRow,
        seen: list[_ChunkRow],
        vectors: dict[int, list[float]],
    ) -> tuple[str, int | None, float]:
        twin = next((prev for prev in seen if prev.hash == row.hash), None)
        if twin is not None:
            return "exact", twin.chunk_id, 1.0
        best: _ChunkRow | None = None
        best_score = 0.0
        for prev in seen:
            score = cosine(vectors[row.chunk_id], vectors[prev.chunk_id])
            if best is None or score > best_score:
                best, best_score = prev, score
        if best is None:
            return "unique", None, 0.0
        status = classify_similarity(best_score)
        return status, (best.chunk_id if status != "unique" else None), best_score

    def _chunks(self, codes: list[str] | None, limit: int | None) -> list[_ChunkRow]:
        conditions = ["d.extraction_status = 'extracted'"]
        params: list[object] = []
        if codes:
            conditions.append("s.code = ANY(%s)")
            params.append(codes)
        where = " AND ".join(conditions)
        limit_sql = ""
        if limit:
            limit_sql = " LIMIT %s"
            params.append(limit)
        sql = f"""
            SELECT c.id, c.hash
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
        return [_ChunkRow(chunk_id=int(r[0]), hash=str(r[1])) for r in rows]

    def _store_relation(
        self, chunk_id: int, status: str, target: int | None, similarity: float
    ) -> None:
        signal = "hash" if status == "exact" else "semantic"
        with self._db.transaction() as cur:
            cur.execute(
                """
                INSERT INTO content_staging.duplicate_relations
                    (chunk_id, duplicate_chunk_id, status, similarity, signal, model)
                VALUES (%s, %s, %s, %s, %s, %s)
                ON CONFLICT (chunk_id) DO UPDATE SET
                    duplicate_chunk_id = EXCLUDED.duplicate_chunk_id,
                    status = EXCLUDED.status,
                    similarity = EXCLUDED.similarity,
                    signal = EXCLUDED.signal,
                    model = EXCLUDED.model,
                    updated_at = now()
                """,
                (chunk_id, target, status, round(similarity, 3), signal, self._model),
            )

    def _propagate(self, kind: str, relations: dict[int, tuple[str, int | None, float]]) -> int:
        table = _TABLE[kind]
        if not relations:
            return 0
        with self._db.transaction() as cur:
            cur.execute(
                f"SELECT id, chunk_id, status, qa_issues FROM content_staging.{table} "
                "WHERE chunk_id = ANY(%s)",
                (list(relations.keys()),),
            )
            rows = cur.fetchall()

        flagged = 0
        for row in rows:
            candidate_id = int(row[0])
            chunk_id = int(row[1])
            current_status = str(row[2])
            existing = [dict(issue) for issue in (row[3] or [])]
            status, target, _ = relations[chunk_id]
            if status not in DEDUP_CODES:
                # Re-running after a chunk became unique: drop stale dedup codes.
                if not any(issue.get("code") in DEDUP_CODES for issue in existing):
                    continue
                kept = [issue for issue in existing if issue.get("code") not in DEDUP_CODES]
                self._update(candidate_id, table, None, kept, current_status, touch_duplicate_of=False)
                continue

            duplicate_of = self._earliest_candidate(table, target) if target is not None else None
            issues = dedup_issues(status, duplicate_of)
            kept = [issue for issue in existing if issue.get("code") not in DEDUP_CODES]
            merged = kept + [issue.as_dict() for issue in issues]
            self._update(candidate_id, table, duplicate_of, merged, current_status, touch_duplicate_of=True)
            flagged += 1
        return flagged

    def _earliest_candidate(self, table: str, chunk_id: int) -> int | None:
        with self._db.transaction() as cur:
            cur.execute(
                f"SELECT id FROM content_staging.{table} WHERE chunk_id = %s ORDER BY id LIMIT 1",
                (chunk_id,),
            )
            row = cur.fetchone()
        return int(row[0]) if row is not None else None

    def _update(
        self,
        candidate_id: int,
        table: str,
        duplicate_of: int | None,
        issues: list[dict[str, object]],
        current_status: str,
        *,
        touch_duplicate_of: bool,
    ) -> None:
        parsed = [
            QaIssue(
                code=str(issue.get("code", "")),
                message=str(issue.get("message", "")),
                blocking=bool(issue.get("blocking", False)),
            )
            for issue in issues
        ]
        # Only draft/needs_review are pipeline-owned; never clobber a human decision.
        status = current_status
        if current_status in ("draft", "needs_review"):
            status = "needs_review" if has_blocking(parsed) else "draft"
        with self._db.transaction() as cur:
            if touch_duplicate_of:
                cur.execute(
                    f"UPDATE content_staging.{table} "
                    "SET duplicate_of = %s, qa_issues = %s::jsonb, status = %s, updated_at = now() "
                    "WHERE id = %s",
                    (duplicate_of, json.dumps(issues), status, candidate_id),
                )
            else:
                cur.execute(
                    f"UPDATE content_staging.{table} "
                    "SET qa_issues = %s::jsonb, status = %s, updated_at = now() WHERE id = %s",
                    (json.dumps(issues), status, candidate_id),
                )
