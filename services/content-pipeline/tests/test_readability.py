from content_pipeline.readability import (
    MAX_WORDS,
    analyse,
    band_for,
    count_syllables,
    difficulty_hint,
    sentences,
    words,
)


def test_words_ignores_punctuation() -> None:
    assert words("Anak, belajar 2 buku!") == ["Anak", "belajar", "2", "buku"]


def test_sentences_splits_on_terminators() -> None:
    assert sentences("Satu. Dua? Tiga!") == ["Satu", "Dua", "Tiga"]


def test_sentences_empty_when_no_text() -> None:
    assert sentences("") == []
    assert sentences("   ") == []


def test_count_syllables_counts_vowel_groups() -> None:
    assert count_syllables("buku") == 2
    assert count_syllables("sekolah") == 3
    assert count_syllables("ibu") == 2


def test_count_syllables_never_zero() -> None:
    assert count_syllables("hmm") == 1


def test_band_for_fondasi_is_visual_and_audio_first() -> None:
    band = band_for("fondasi", "tk-a")
    assert band is not None
    assert band.age_band == "4-5"
    assert band.reading_level == "emerging"
    assert band.visual_required is True
    assert band.audio_required is True


def test_band_for_uses_phase_when_grade_unknown() -> None:
    band = band_for("b", "")
    assert band is not None
    assert band.age_band == "9-10"
    assert band.reading_level == "developing"


def test_band_for_unknown_phase_is_none() -> None:
    assert band_for("", "") is None
    assert band_for("z", "sd-1") is None


def test_analyse_measures_text_and_uses_phase_band() -> None:
    result = analyse("Anak belajar membaca buku.", phase_code="fondasi", grade_code="tk-b")
    assert result.word_count == 4
    assert result.sentence_count == 1
    assert result.avg_sentence_length == 4.0
    assert result.avg_syllables_per_word > 0
    assert result.age_band == "5-6"
    assert result.reading_level == "emerging"
    assert result.complexity == "very_simple"
    assert result.visual_required is True
    assert result.audio_required is True


def test_analyse_falls_back_to_length_when_phase_unknown() -> None:
    short = analyse("Anak bermain.", phase_code="")
    assert short.reading_level == "emerging"

    long_sentence = " ".join(["kata"] * 20) + "."
    hard = analyse(long_sentence, phase_code="")
    assert hard.avg_sentence_length == 20.0
    assert hard.reading_level == "fluent"
    assert hard.complexity == "advanced"


def test_analyse_counts_at_least_one_sentence() -> None:
    result = analyse("kata kata kata")
    assert result.sentence_count == 1


def test_too_long_flag() -> None:
    text = " ".join(["kata"] * (MAX_WORDS + 1)) + "."
    assert analyse(text).too_long is True
    assert analyse("kata pendek.").too_long is False


def test_difficulty_hint_follows_band() -> None:
    assert difficulty_hint(analyse("Anak bermain.", phase_code="fondasi")) == 1
    assert difficulty_hint(analyse("Anak bermain.", phase_code="c")) == 4


def test_difficulty_hint_rises_for_long_sentences() -> None:
    text = " ".join(["kata"] * 25) + "."
    # 25-word sentence nudges Fase A (2) up to 3.
    assert difficulty_hint(analyse(text, phase_code="a")) == 3


def test_difficulty_hint_is_clamped() -> None:
    text = " ".join(["kata"] * 25) + "."
    assert difficulty_hint(analyse(text, phase_code="c")) == 5
