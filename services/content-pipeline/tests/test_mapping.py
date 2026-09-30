from __future__ import annotations

from content_pipeline.mapping import (
    best_mapping,
    confidence_band,
    score_skill,
    tokens,
)
from content_pipeline.taxonomy import SkillRef


def _skill(
    skill_id: int,
    code: str,
    name: str,
    description: str,
    *,
    chapter: str = "",
    subject: str = "matematika",
    grade: str = "sd-1",
    cp_id: int | None = 1,
    cp_code: str = "CP-A-MAT-BILANGAN",
    phase: str = "a",
) -> SkillRef:
    return SkillRef(
        id=skill_id,
        code=code,
        name=name,
        description=description,
        chapter_name=chapter,
        subject_code=subject,
        grade_code=grade,
        cp_id=cp_id,
        cp_code=cp_code,
        phase_code=phase,
    )


PENJUMLAHAN = _skill(
    1, "sd1-penjumlahan-1-20", "Penjumlahan 1–20", "Menjumlahkan bilangan sampai 20.",
    chapter="Penjumlahan & Pengurangan",
)
PENGURANGAN = _skill(
    2, "sd1-pengurangan-1-20", "Pengurangan 1–20", "Mengurangkan bilangan sampai 20.",
    chapter="Penjumlahan & Pengurangan",
)
WUJUD = _skill(
    3, "sd3-wujud-benda", "Wujud benda", "Membedakan benda padat, cair, dan gas.",
    chapter="Tumbuhan & Wujud Benda", subject="ipas", grade="sd-3",
    cp_id=4, cp_code="CP-B-IPAS-ZAT", phase="b",
)
SKILLS = (PENJUMLAHAN, PENGURANGAN, WUJUD)


def test_tokens_drops_stopwords_and_short_words() -> None:
    assert tokens("Peserta didik dan 20 benda") == {"benda"}


def test_best_mapping_prefers_the_matching_skill() -> None:
    mapping = best_mapping("Peserta didik menjumlahkan dua bilangan sampai 20.", "Penjumlahan", SKILLS)
    assert mapping is not None
    assert mapping.skill.code == "sd1-penjumlahan-1-20"
    assert mapping.confidence > 0


def test_heading_matching_the_skill_name_boosts_confidence() -> None:
    with_heading, _ = score_skill("bilangan sampai 20", "Penjumlahan", PENJUMLAHAN)
    without_heading, _ = score_skill("bilangan sampai 20", "", PENJUMLAHAN)
    assert with_heading > without_heading


def test_best_mapping_returns_none_when_nothing_matches() -> None:
    assert best_mapping("Cuaca dan musim di Indonesia.", "Cuaca", SKILLS) is None


def test_best_mapping_can_select_a_different_subject() -> None:
    mapping = best_mapping("Benda padat, cair, dan gas memiliki wujud berbeda.", "Wujud benda", SKILLS)
    assert mapping is not None
    assert mapping.skill.code == "sd3-wujud-benda"
    assert mapping.skill.cp_code == "CP-B-IPAS-ZAT"


def test_confidence_bands_follow_the_plan() -> None:
    assert confidence_band(0.95) == "high"
    assert confidence_band(0.80) == "review"
    assert confidence_band(0.40) == "manual"


def test_score_is_zero_for_unrelated_text() -> None:
    score, reason = score_skill("Cuaca cerah hari ini.", "", PENJUMLAHAN)
    assert score == 0
    assert reason == ""
