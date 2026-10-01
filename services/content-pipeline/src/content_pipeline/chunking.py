"""Chunking (§15).

Priority is document → heading → section → paragraph → semantic fallback, not a
blind character count:

1. a section that fits the token target becomes one chunk;
2. otherwise paragraphs are packed up to the target;
3. a paragraph that is itself too large is split at sentence boundaries with a
   small overlap so a sentence is not orphaned across chunks.
"""

from __future__ import annotations

import re
from collections.abc import Sequence
from dataclasses import dataclass

from content_pipeline.parsers.base import Section

TARGET_TOKENS = 350
MAX_TOKENS = 500
OVERLAP_TOKENS = 60

_SENTENCE_BOUNDARY = re.compile(r"(?<=[.!?])\s+")
_PARAGRAPH_BOUNDARY = re.compile(r"\n\s*\n")


@dataclass(frozen=True)
class Chunk:
    heading: str
    page_start: int | None
    page_end: int | None
    sequence: int
    text: str
    token_count: int


def approximate_tokens(text: str) -> int:
    """Cheap token estimate; a real tokenizer is not worth the dependency here."""
    return len(text.split())


def _paragraphs(text: str) -> list[str]:
    return [p.strip() for p in _PARAGRAPH_BOUNDARY.split(text) if p.strip()]


def _sentences(paragraph: str) -> list[str]:
    return [s.strip() for s in _SENTENCE_BOUNDARY.split(paragraph) if s.strip()]


def _hard_split(text: str, target: int) -> list[str]:
    words = text.split()
    return [" ".join(words[i:i + target]) for i in range(0, len(words), target)]


def _split_long_paragraph(paragraph: str) -> list[str]:
    pieces: list[str] = []
    current: list[str] = []
    count = 0
    for sentence in _sentences(paragraph) or [paragraph]:
        if approximate_tokens(sentence) > TARGET_TOKENS:
            if current:
                pieces.append(" ".join(current))
                current, count = [], 0
            pieces.extend(_hard_split(sentence, TARGET_TOKENS))
            continue
        sentence_tokens = approximate_tokens(sentence)
        if current and count + sentence_tokens > TARGET_TOKENS:
            pieces.append(" ".join(current))
            overlap: list[str] = []
            overlap_count = 0
            for previous in reversed(current):
                previous_tokens = approximate_tokens(previous)
                if overlap_count + previous_tokens > OVERLAP_TOKENS:
                    break
                overlap.insert(0, previous)
                overlap_count += previous_tokens
            current, count = overlap, overlap_count
        current.append(sentence)
        count += sentence_tokens
    if current:
        pieces.append(" ".join(current))
    return pieces


def _make(section: Section, units: Sequence[str], sequence: int) -> Chunk:
    text = "\n\n".join(units)
    return Chunk(
        heading=section.heading,
        page_start=section.page,
        page_end=section.page,
        sequence=sequence,
        text=text,
        token_count=approximate_tokens(text),
    )


def chunk_sections(sections: Sequence[Section]) -> list[Chunk]:
    chunks: list[Chunk] = []
    sequence = 0
    for section in sections:
        if not section.text.strip():
            continue
        units: list[str] = []
        for paragraph in _paragraphs(section.text):
            if approximate_tokens(paragraph) > MAX_TOKENS:
                units.extend(_split_long_paragraph(paragraph))
            else:
                units.append(paragraph)

        current: list[str] = []
        count = 0
        for unit in units:
            unit_tokens = approximate_tokens(unit)
            if current and count + unit_tokens > TARGET_TOKENS:
                chunks.append(_make(section, current, sequence))
                sequence += 1
                current, count = [], 0
            current.append(unit)
            count += unit_tokens
        if current:
            chunks.append(_make(section, current, sequence))
            sequence += 1
    return chunks
