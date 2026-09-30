from content_pipeline.materials import (
    MIN_WORDS,
    QaIssue,
    build_draft,
    build_learning_objective,
    build_summary,
    build_title,
    has_blocking,
    material_hash,
    quality_checks,
)
from content_pipeline.readability import MAX_WORDS

LONG_TEXT = (
    "Anak belajar mengenal huruf dan angka di sekolah bersama guru. "
    "Mereka membaca buku bergambar lalu menulis nama sendiri. "
    "Kegiatan ini melatih kemampuan dasar literasi dan numerasi."
)


def test_material_hash_is_stable_and_normalised() -> None:
    assert material_hash("Halo   Dunia") == material_hash("halo dunia")
    assert material_hash("Halo Dunia") != material_hash("Halo Dunia!")


def test_build_title_prefers_heading() -> None:
    assert build_title("  Bab   Satu ", "Isi.", "Skill") == "Bab Satu"


def test_build_title_falls_back_to_first_sentence() -> None:
    assert build_title("", "Belajar berhitung. Lalu bermain.", "Skill") == "Belajar berhitung"


def test_build_title_falls_back_to_skill_then_default() -> None:
    assert build_title("", "", "Mengenal huruf") == "Mengenal huruf"
    assert build_title("", "", "") == "Materi"


def test_build_title_truncates_long_headings() -> None:
    title = build_title("x" * 200, "", "")
    assert len(title) == 81
    assert title.endswith("…")


def test_build_summary_takes_first_sentences() -> None:
    assert build_summary("Satu. Dua. Tiga.") == "Satu Dua"


def test_build_summary_stops_at_char_budget() -> None:
    long_first = "a" * 218 + "."
    assert build_summary(f"{long_first} Dua.") == "a" * 218


def test_build_learning_objective_with_and_without_cp() -> None:
    assert build_learning_objective("Mengenal huruf", "CP-001") == (
        "Peserta didik dapat mengenal huruf (CP-001)."
    )
    assert build_learning_objective("Mengenal huruf", "") == "Peserta didik dapat mengenal huruf."
    assert build_learning_objective("", "CP-001") == ""


def test_build_draft_is_deterministic() -> None:
    first = build_draft(LONG_TEXT, heading="Huruf", skill_name="Mengenal huruf",
                        cp_code="CP-001", phase_code="a", grade_code="sd-1")
    second = build_draft(LONG_TEXT, heading="Huruf", skill_name="Mengenal huruf",
                         cp_code="CP-001", phase_code="a", grade_code="sd-1")
    assert first == second
    assert first.readability.age_band == "6-7"
    assert first.readability.word_count == len(LONG_TEXT.split())


def _clean_issues(
    *,
    skill_id: int | None = 1,
    cp_id: int | None = 2,
    confidence: float = 0.95,
    duplicate_of: int | None = None,
) -> list[QaIssue]:
    draft = build_draft(LONG_TEXT, heading="Huruf", skill_name="Mengenal huruf",
                        cp_code="CP-001", phase_code="a", grade_code="sd-1")
    return quality_checks(
        draft,
        skill_id=skill_id,
        cp_id=cp_id,
        confidence=confidence,
        duplicate_of=duplicate_of,
    )


def test_quality_checks_clean_candidate_has_no_issues() -> None:
    assert _clean_issues() == []
    assert has_blocking([]) is False


def test_quality_checks_flags_missing_mapping_as_blocking() -> None:
    issues = _clean_issues(skill_id=None, cp_id=None)
    codes = {issue.code for issue in issues}
    assert codes == {"missing_skill", "missing_cp_mapping"}
    assert has_blocking(issues) is True


def test_quality_checks_flags_duplicate_as_blocking() -> None:
    issues = _clean_issues(duplicate_of=7)
    assert [issue.code for issue in issues] == ["duplicate"]
    assert issues[0].blocking is True
    assert "7" in issues[0].message


def test_quality_checks_flags_short_text() -> None:
    draft = build_draft("Dua kata.", heading="", skill_name="Skill",
                        cp_code="CP-1", phase_code="a")
    assert draft.readability.word_count < MIN_WORDS
    issues = quality_checks(draft, skill_id=1, cp_id=2, confidence=0.9, duplicate_of=None)
    assert [issue.code for issue in issues] == ["too_short"]
    assert has_blocking(issues) is False


def test_quality_checks_flags_long_text_and_low_confidence() -> None:
    long_text = " ".join(["kata"] * (MAX_WORDS + 1)) + "."
    draft = build_draft(long_text, heading="Bab", skill_name="Skill",
                        cp_code="CP-1", phase_code="c")
    issues = quality_checks(draft, skill_id=1, cp_id=2, confidence=0.4, duplicate_of=None)
    codes = {issue.code for issue in issues}
    assert codes == {"text_too_long", "low_confidence"}
    assert has_blocking(issues) is False


def test_quality_checks_flags_missing_objective() -> None:
    draft = build_draft(LONG_TEXT, heading="Huruf", skill_name="", cp_code="", phase_code="a")
    issues = quality_checks(draft, skill_id=None, cp_id=None, confidence=0.9, duplicate_of=None)
    assert "missing_objective" in {issue.code for issue in issues}


def test_qa_issue_serialises() -> None:
    issues = _clean_issues(duplicate_of=3)
    assert issues[0].as_dict() == {
        "code": "duplicate",
        "message": "same text as material #3",
        "blocking": True,
    }
