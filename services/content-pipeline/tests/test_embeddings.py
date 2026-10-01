from __future__ import annotations

from collections.abc import Sequence

from content_pipeline.embeddings import (
    OWNER_CHUNK,
    Embedder,
    embedding_text_hash,
    normalise_for_embedding,
    skill_embedding_text,
)
from content_pipeline.provider import ChatResult
from content_pipeline.taxonomy import SkillRef


class FakeProvider:
    def __init__(self) -> None:
        self.calls = 0
        self.name = "fake"
        self.embedding_model = "fake-emb"
        self.chat_model = "fake-chat"

    def embed(self, texts: Sequence[str]) -> list[list[float]]:
        self.calls += 1
        return [[float(len(text)), 1.0] for text in texts]

    def complete(self, *, system: str, user: str, max_tokens: int) -> ChatResult:
        return ChatResult("{}", "fake-chat", 0, 0, 0)

    def close(self) -> None:
        return None


class FakeStore:
    def __init__(self) -> None:
        self.data: dict[tuple[str, int, str], tuple[str, list[float]]] = {}

    def states(self, owner_kind: str, model: str) -> dict[int, tuple[str, list[float]]]:
        return {
            owner_id: value
            for (kind, owner_id, owner_model), value in self.data.items()
            if kind == owner_kind and owner_model == model
        }

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
        self.data[(owner_kind, owner_id, model)] = (text_hash, list(vector))


def _skill() -> SkillRef:
    return SkillRef(
        id=1,
        code="sd1-penjumlahan-1-20",
        name="Penjumlahan 1–20",
        description="Menjumlahkan bilangan sampai 20.",
        chapter_name="Penjumlahan & Pengurangan",
        subject_code="matematika",
        grade_code="sd-1",
        cp_id=1,
        cp_code="CP-A-MAT-BILANGAN",
        phase_code="a",
    )


def test_normalise_collapses_whitespace() -> None:
    assert normalise_for_embedding("  a \n b\t c ") == "a b c"
    assert embedding_text_hash("a  b\n c") == embedding_text_hash("a b c")


def test_skill_embedding_text_includes_name_and_cp() -> None:
    text = skill_embedding_text(_skill())
    assert "Penjumlahan 1–20" in text
    assert "CP-A-MAT-BILANGAN" in text


def test_embedder_dedupes_identical_texts_in_one_call() -> None:
    provider = FakeProvider()
    outcome = Embedder(provider, FakeStore()).embed(
        OWNER_CHUNK, [(1, "same"), (2, "same"), (3, "other")], model="m"
    )
    assert provider.calls == 1
    assert outcome.embedded == 3
    assert outcome.vectors[1] == outcome.vectors[2]
    assert outcome.vectors[3] != outcome.vectors[1]


def test_embedder_skips_cached_rows_without_calling_the_provider() -> None:
    provider = FakeProvider()
    store = FakeStore()
    store.upsert(
        OWNER_CHUNK,
        1,
        model="m",
        provider="fake",
        vector=[9.0],
        text_hash=embedding_text_hash("cached"),
    )
    outcome = Embedder(provider, store).embed(OWNER_CHUNK, [(1, "cached")], model="m")
    assert (outcome.cached, outcome.embedded, provider.calls) == (1, 0, 0)
    assert outcome.vectors[1] == [9.0]


def test_embedder_reembeds_when_the_text_changed() -> None:
    provider = FakeProvider()
    store = FakeStore()
    store.upsert(
        OWNER_CHUNK,
        1,
        model="m",
        provider="fake",
        vector=[9.0],
        text_hash=embedding_text_hash("old text"),
    )
    outcome = Embedder(provider, store).embed(OWNER_CHUNK, [(1, "new text")], model="m")
    assert (outcome.cached, outcome.embedded, provider.calls) == (0, 1, 1)


def test_embedder_force_reembeds_even_when_cached() -> None:
    provider = FakeProvider()
    store = FakeStore()
    store.upsert(
        OWNER_CHUNK,
        1,
        model="m",
        provider="fake",
        vector=[9.0],
        text_hash=embedding_text_hash("cached"),
    )
    outcome = Embedder(provider, store).embed(OWNER_CHUNK, [(1, "cached")], model="m", force=True)
    assert (outcome.cached, outcome.embedded, provider.calls) == (0, 1, 1)


def test_embedder_returns_empty_for_no_items() -> None:
    provider = FakeProvider()
    outcome = Embedder(provider, FakeStore()).embed(OWNER_CHUNK, [], model="m")
    assert (outcome.vectors, outcome.embedded, outcome.cached) == ({}, 0, 0)
    assert provider.calls == 0
