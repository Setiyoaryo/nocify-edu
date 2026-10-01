from __future__ import annotations

import pytest

from content_pipeline.mapping import best_hybrid_mapping, blend_confidence
from content_pipeline.taxonomy import SkillRef


def _skill(skill_id: int, code: str, name: str, description: str) -> SkillRef:
    return SkillRef(
        id=skill_id,
        code=code,
        name=name,
        description=description,
        chapter_name="Bab",
        subject_code="matematika",
        grade_code="sd-1",
        cp_id=1,
        cp_code="CP-A-MAT-BILANGAN",
        phase_code="a",
    )


PENJUMLAHAN = _skill(1, "sd1-penjumlahan-1-20", "Penjumlahan 1–20", "Menjumlahkan bilangan sampai 20.")
WUJUD = _skill(3, "sd3-wujud-benda", "Wujud benda", "Membedakan benda padat, cair, dan gas.")


def test_blend_never_lowers_the_lexical_score() -> None:
    assert blend_confidence(0.8, 0.0) == 0.8
    assert blend_confidence(0.8, 0.1) == 0.8


def test_blend_lifts_when_semantic_exceeds_lexical() -> None:
    assert blend_confidence(0.4, 1.0) == pytest.approx(0.7)
    assert blend_confidence(0.6, 0.9) == pytest.approx(0.75)


def test_hybrid_mapping_finds_a_paraphrase_via_embeddings() -> None:
    # No keyword overlap at all, but the chunk vector matches the skill vector.
    mapping = best_hybrid_mapping(
        "teks tanpa kata kunci yang cocok", "", [1.0, 0.0], (PENJUMLAHAN, WUJUD), {1: [1.0, 0.0], 3: [0.0, 1.0]}
    )
    assert mapping is not None
    assert mapping.skill.id == PENJUMLAHAN.id
    assert mapping.method == "hybrid"


def test_hybrid_mapping_falls_back_to_lexical_when_semantic_is_weak() -> None:
    mapping = best_hybrid_mapping(
        "Peserta didik menjumlahkan dua bilangan sampai 20.",
        "Penjumlahan",
        [0.0, 1.0],
        (PENJUMLAHAN,),
        {1: [1.0, 0.0]},
    )
    assert mapping is not None
    assert mapping.skill.id == PENJUMLAHAN.id
    assert mapping.method == "lexical"


def test_hybrid_mapping_returns_none_without_any_signal() -> None:
    assert best_hybrid_mapping("cuaca cerah hari ini", "", None, (PENJUMLAHAN,), {}) is None
