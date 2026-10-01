from __future__ import annotations

from content_pipeline.chunking import (
    MAX_TOKENS,
    TARGET_TOKENS,
    approximate_tokens,
    chunk_sections,
)
from content_pipeline.parsers.base import Section


def _words(n: int) -> str:
    return " ".join(f"kata{i}" for i in range(n))


def test_short_section_becomes_one_chunk() -> None:
    sections = [Section(heading="Bilangan", text="Peserta didik membandingkan bilangan.", page=3)]
    chunks = chunk_sections(sections)
    assert len(chunks) == 1
    assert chunks[0].heading == "Bilangan"
    assert chunks[0].page_start == 3
    assert chunks[0].sequence == 0


def test_paragraphs_are_packed_up_to_target() -> None:
    paragraph = _words(TARGET_TOKENS // 2)
    sections = [Section(heading="Bab", text=f"{paragraph}\n\n{paragraph}\n\n{paragraph}")]
    chunks = chunk_sections(sections)
    assert len(chunks) >= 2
    assert all(chunk.token_count <= TARGET_TOKENS + 1 for chunk in chunks)
    assert [chunk.sequence for chunk in chunks] == list(range(len(chunks)))


def test_long_paragraph_is_split_at_sentences_with_overlap() -> None:
    # 200 sentences ~= 800 tokens, above MAX_TOKENS, so the semantic fallback kicks in.
    sentences = " ".join(f"Kalimat {i} berisi penjelasan." for i in range(200))
    sections = [Section(heading="Panjang", text=sentences)]
    chunks = chunk_sections(sections)
    assert len(chunks) > 1
    assert all(approximate_tokens(chunk.text) <= MAX_TOKENS for chunk in chunks)
    # Consecutive chunks share the overlap sentence.
    tail = chunks[0].text.split()[-3:]
    assert " ".join(tail) in chunks[1].text


def test_empty_sections_are_skipped() -> None:
    sections = [Section(heading="Kosong", text="   "), Section(heading="Isi", text="Ada isi.")]
    chunks = chunk_sections(sections)
    assert len(chunks) == 1
    assert chunks[0].heading == "Isi"


def test_sequence_is_continuous_across_sections() -> None:
    sections = [
        Section(heading="A", text="Isi A."),
        Section(heading="B", text="Isi B."),
    ]
    chunks = chunk_sections(sections)
    assert [chunk.sequence for chunk in chunks] == [0, 1]
