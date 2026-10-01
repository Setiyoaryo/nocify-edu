"""Embedding store and builder (M6, §17).

Vectors are cached in ``content_staging.embeddings`` keyed by
``(owner_kind, owner_id, model)`` with the sha256 of the embedded text, so a
re-run only pays for text that changed. The store is a rebuildable cache, never
authoritative.
"""

from __future__ import annotations

import hashlib
import logging
import re
from collections.abc import Sequence
from dataclasses import dataclass
from typing import Protocol, runtime_checkable

from content_pipeline.db import Database
from content_pipeline.provider import Provider, ProviderError
from content_pipeline.taxonomy import SkillRef, load_taxonomy

log = logging.getLogger("content_pipeline.embeddings")

OWNER_CHUNK = "chunk"
OWNER_SKILL = "skill"

_WHITESPACE = re.compile(r"\s+")


def normalise_for_embedding(text: str) -> str:
    """Collapse whitespace so formatting noise does not change the vector."""
    return _WHITESPACE.sub(" ", text).strip()


def embedding_text_hash(text: str) -> str:
    return hashlib.sha256(normalise_for_embedding(text).encode("utf-8")).hexdigest()


def skill_embedding_text(skill: SkillRef) -> str:
    parts = [skill.name, skill.description, skill.chapter_name, skill.cp_code]
    return ". ".join(part for part in parts if part)


@dataclass(frozen=True)
class EmbedOutcome:
    vectors: dict[int, list[float]]
    embedded: int
    cached: int


@runtime_checkable
class VectorStore(Protocol):
    """The slice of :class:`EmbeddingStore` the embedder needs (fakeable in tests)."""

    def states(self, owner_kind: str, model: str) -> dict[int, tuple[str, list[float]]]: ...

    def upsert(
        self,
        owner_kind: str,
        owner_id: int,
        *,
        model: str,
        provider: str,
        vector: list[float],
        text_hash: str,
    ) -> None: ...


class EmbeddingStore:
    """Reads and writes cached vectors. No provider involvement."""

    def __init__(self, db: Database) -> None:
        self._db = db

    def states(self, owner_kind: str, model: str) -> dict[int, tuple[str, list[float]]]:
        """Return ``owner_id -> (text_hash, vector)`` for one kind and model."""
        with self._db.transaction() as cur:
            cur.execute(
                "SELECT owner_id, text_hash, vector FROM content_staging.embeddings "
                "WHERE owner_kind = %s AND model = %s",
                (owner_kind, model),
            )
            rows = cur.fetchall()
        return {int(r[0]): (str(r[1]), [float(x) for x in r[2]]) for r in rows}

    def get_many(self, owner_kind: str, model: str) -> dict[int, list[float]]:
        return {owner_id: vector for owner_id, (_, vector) in self.states(owner_kind, model).items()}

    def upsert(
        self,
        owner_kind: str,
        owner_id: int,
        *,
        model: str,
        provider: str,
        vector: list[float],
        text_hash: str,
    ) -> None:
        with self._db.transaction() as cur:
            cur.execute(
                """
                INSERT INTO content_staging.embeddings
                    (owner_kind, owner_id, model, provider, dim, vector, text_hash)
                VALUES (%s, %s, %s, %s, %s, %s, %s)
                ON CONFLICT (owner_kind, owner_id, model) DO UPDATE SET
                    provider = EXCLUDED.provider,
                    dim = EXCLUDED.dim,
                    vector = EXCLUDED.vector,
                    text_hash = EXCLUDED.text_hash,
                    updated_at = now()
                """,
                (owner_kind, owner_id, model, provider, len(vector), vector, text_hash),
            )


class Embedder:
    """Embeds texts through a provider, reusing cached vectors and batching calls."""

    def __init__(self, provider: Provider, store: VectorStore, *, batch_size: int = 64) -> None:
        self._provider = provider
        self._store = store
        self._batch_size = max(1, batch_size)

    def embed(
        self,
        owner_kind: str,
        items: Sequence[tuple[int, str]],
        *,
        model: str,
        force: bool = False,
    ) -> EmbedOutcome:
        if not items:
            return EmbedOutcome(vectors={}, embedded=0, cached=0)

        states = self._store.states(owner_kind, model)
        vectors: dict[int, list[float]] = {}
        # One provider call per unique text, even when several owners share it.
        pending: dict[str, list[int]] = {}
        texts: dict[str, str] = {}
        cached = 0
        for owner_id, text in items:
            text_hash = embedding_text_hash(text)
            state = states.get(owner_id)
            if not force and state is not None and state[0] == text_hash:
                vectors[owner_id] = state[1]
                cached += 1
                continue
            pending.setdefault(text_hash, []).append(owner_id)
            texts[text_hash] = normalise_for_embedding(text)

        embedded = 0
        hashes = list(pending.keys())
        for start in range(0, len(hashes), self._batch_size):
            batch = hashes[start : start + self._batch_size]
            results = self._provider.embed([texts[h] for h in batch])
            if len(results) != len(batch):
                raise ProviderError(f"provider returned {len(results)} vectors for {len(batch)} texts")
            for text_hash, vector in zip(batch, results, strict=True):
                for owner_id in pending[text_hash]:
                    vectors[owner_id] = vector
                    self._store.upsert(
                        owner_kind,
                        owner_id,
                        model=model,
                        provider=self._provider.name,
                        vector=vector,
                        text_hash=text_hash,
                    )
                    embedded += 1
        return EmbedOutcome(vectors=vectors, embedded=embedded, cached=cached)


@dataclass(frozen=True)
class EmbedSummary:
    source_code: str
    chunks_embedded: int
    chunks_cached: int


@dataclass(frozen=True)
class EmbedReport:
    sources: list[EmbedSummary]
    skills_embedded: int
    skills_cached: int

    @property
    def chunks_embedded(self) -> int:
        return sum(s.chunks_embedded for s in self.sources)

    @property
    def chunks_cached(self) -> int:
        return sum(s.chunks_cached for s in self.sources)


@dataclass(frozen=True)
class _PendingChunk:
    chunk_id: int
    source_code: str
    text: str


class EmbeddingBuilder:
    """Embeds the active skills and the extracted chunks the pipeline knows about."""

    def __init__(self, db: Database, provider: Provider, *, batch_size: int = 64) -> None:
        self._db = db
        self._provider = provider
        self._store = EmbeddingStore(db)
        self._embedder = Embedder(provider, self._store, batch_size=batch_size)
        self._model = provider.embedding_model

    def run(
        self,
        *,
        codes: list[str] | None = None,
        limit: int | None = None,
        force: bool = False,
        kinds: Sequence[str] = (OWNER_CHUNK, OWNER_SKILL),
    ) -> EmbedReport:
        skills_embedded = skills_cached = 0
        if OWNER_SKILL in kinds:
            skills = load_taxonomy(self._db).skills
            outcome = self._embedder.embed(
                OWNER_SKILL,
                [(skill.id, skill_embedding_text(skill)) for skill in skills],
                model=self._model,
                force=force,
            )
            skills_embedded, skills_cached = outcome.embedded, outcome.cached

        sources: list[EmbedSummary] = []
        if OWNER_CHUNK in kinds:
            per_source: dict[str, list[tuple[int, str]]] = {}
            for chunk in self._pending_chunks(codes, limit):
                per_source.setdefault(chunk.source_code, []).append((chunk.chunk_id, chunk.text))
            for code in sorted(per_source):
                outcome = self._embedder.embed(
                    OWNER_CHUNK, per_source[code], model=self._model, force=force
                )
                sources.append(
                    EmbedSummary(
                        source_code=code,
                        chunks_embedded=outcome.embedded,
                        chunks_cached=outcome.cached,
                    )
                )
        return EmbedReport(sources=sources, skills_embedded=skills_embedded, skills_cached=skills_cached)

    def _pending_chunks(self, codes: list[str] | None, limit: int | None) -> list[_PendingChunk]:
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
            SELECT c.id, s.code, c.text
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
            _PendingChunk(chunk_id=int(r[0]), source_code=str(r[1]), text=str(r[2])) for r in rows
        ]
