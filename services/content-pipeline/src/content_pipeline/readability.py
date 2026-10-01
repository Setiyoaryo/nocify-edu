"""Child readability layer (§30).

Knowledge extraction is not enough: material must fit the reader's age. This
module is deliberately a *heuristic*, not a psychometric instrument. It derives a
baseline from the curriculum phase/grade and then measures the text, so a
reviewer sees both what the curriculum expects and what the text actually is.

Bands follow §30:

    Fase Fondasi (TK)  visual-first, very short instruction, audio-friendly
    Fase A (SD 1–2)    short text
    Fase B (SD 3–4)    moderate reading
    Fase C (SD 5–6)    more reasoning and contextual text
"""

from __future__ import annotations

import re
from dataclasses import dataclass

VOWELS = frozenset("aeiou")
_WORD = re.compile(r"[A-Za-z0-9]+")
_SENTENCE_SPLIT = re.compile(r"[.!?]+")

# A material longer than this is flagged by automatic QA (§40, "Text too long").
MAX_WORDS = 220


@dataclass(frozen=True)
class Band:
    age_band: str
    reading_level: str
    complexity: str
    visual_required: bool
    audio_required: bool


# Phase code → band. Grade codes refine the age band within a phase.
PHASE_BANDS: dict[str, Band] = {
    "fondasi": Band("4-6", "emerging", "very_simple", True, True),
    "a": Band("7-8", "early", "simple", True, False),
    "b": Band("9-10", "developing", "moderate", False, False),
    "c": Band("11-12", "fluent", "advanced", False, False),
}

GRADE_AGE: dict[str, str] = {
    "tk-a": "4-5",
    "tk-b": "5-6",
    "sd-1": "6-7",
    "sd-2": "7-8",
    "sd-3": "8-9",
    "sd-4": "9-10",
    "sd-5": "10-11",
    "sd-6": "11-12",
}

# Fallback when no phase is known: sentence length alone decides the band.
_LENGTH_BANDS: tuple[tuple[float, Band], ...] = (
    (8.0, PHASE_BANDS["fondasi"]),
    (12.0, PHASE_BANDS["a"]),
    (16.0, PHASE_BANDS["b"]),
)


@dataclass(frozen=True)
class Readability:
    age_band: str
    reading_level: str
    complexity: str
    word_count: int
    sentence_count: int
    avg_sentence_length: float
    avg_syllables_per_word: float
    visual_required: bool
    audio_required: bool

    @property
    def too_long(self) -> bool:
        return self.word_count > MAX_WORDS


def words(text: str) -> list[str]:
    return _WORD.findall(text)


def sentences(text: str) -> list[str]:
    parts = [p.strip() for p in _SENTENCE_SPLIT.split(text)]
    return [p for p in parts if p]


def count_syllables(word: str) -> int:
    """Count vowel groups; good enough for Indonesian, where diphthongs are few."""
    groups = 0
    previous_vowel = False
    for char in word.lower():
        is_vowel = char in VOWELS
        if is_vowel and not previous_vowel:
            groups += 1
        previous_vowel = is_vowel
    return groups or 1


def band_for(phase_code: str, grade_code: str) -> Band | None:
    band = PHASE_BANDS.get(phase_code)
    if band is None:
        return None
    if grade_code in GRADE_AGE:
        return Band(
            age_band=GRADE_AGE[grade_code],
            reading_level=band.reading_level,
            complexity=band.complexity,
            visual_required=band.visual_required,
            audio_required=band.audio_required,
        )
    return band


def _band_from_length(avg_sentence_length: float) -> Band:
    for threshold, band in _LENGTH_BANDS:
        if avg_sentence_length <= threshold:
            return band
    return PHASE_BANDS["c"]


def analyse(text: str, *, phase_code: str = "", grade_code: str = "") -> Readability:
    """Measure a chunk and pick its readability band. Pure and deterministic."""
    word_list = words(text)
    sentence_list = sentences(text)
    word_count = len(word_list)
    sentence_count = max(1, len(sentence_list))
    avg_sentence_length = round(word_count / sentence_count, 2)
    total_syllables = sum(count_syllables(word) for word in word_list)
    avg_syllables = round(total_syllables / word_count, 2) if word_count else 0.0

    band = band_for(phase_code, grade_code) or _band_from_length(avg_sentence_length)
    return Readability(
        age_band=band.age_band,
        reading_level=band.reading_level,
        complexity=band.complexity,
        word_count=word_count,
        sentence_count=sentence_count,
        avg_sentence_length=avg_sentence_length,
        avg_syllables_per_word=avg_syllables,
        visual_required=band.visual_required,
        audio_required=band.audio_required,
    )


# Reading level → difficulty hint (1 easiest … 5 hardest), §21 difficulty_hint.
DIFFICULTY_BY_LEVEL: dict[str, int] = {
    "emerging": 1,
    "early": 2,
    "developing": 3,
    "fluent": 4,
}


def difficulty_hint(readability: Readability) -> int:
    """Base hint from the band, nudged up when sentences run long. Clamped to 1–5."""
    hint = DIFFICULTY_BY_LEVEL.get(readability.reading_level, 3)
    if readability.avg_sentence_length > 20:
        hint += 1
    return max(1, min(5, hint))
