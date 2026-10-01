from dataclasses import replace

from content_pipeline.qa import QaIssue
from content_pipeline.questions import (
    BLANK,
    QuestionDraft,
    blank_out,
    build_question,
    content_words,
    pick_sentence,
    pick_term,
    question_hash,
    question_quality_checks,
    vocabulary,
)
from content_pipeline.taxonomy import SkillRef
from content_pipeline.validation import QuestionSchema, SchemaValidation

SCHEMA = QuestionSchema.load()

SKILL_B = SkillRef(
    id=14, code="SD-B-01", name="Perkalian", description="Perkalian bilangan cacah",
    chapter_name="Bilangan", subject_code="matematika", grade_code="sd-4",
    cp_id=2, cp_code="CP-002", phase_code="b",
)
SKILL_FONDASI = SkillRef(
    id=3, code="TK-B-01", name="Bentuk dasar", description="Mengenal bentuk dasar",
    chapter_name="Bentuk dan Warna", subject_code="", grade_code="tk-b",
    cp_id=1, cp_code="CP-001", phase_code="fondasi",
)

SHORT_TEXT = (
    "Perkalian adalah penjumlahan berulang dari bilangan yang sama. "
    "Siswa dapat menghitung hasilnya dengan cepat."
)
FONDASI_TEXT = "Mengenal bentuk geometri sederhana melalui permainan warna cerah."


def test_question_hash_is_stable_and_normalised() -> None:
    assert question_hash("Lengkapi   ini?", "Buku") == question_hash("lengkapi ini?", "buku")
    assert question_hash("A", "B") != question_hash("A", "C")


def test_content_words_filters_short_and_stopwords() -> None:
    assert content_words("dan yang Perkalian buku") == ["Perkalian"]


def test_content_words_dedupes_case_insensitively() -> None:
    assert content_words("Bilangan bilangan BILANGAN") == ["Bilangan"]


def test_vocabulary_from_skill() -> None:
    vocab = vocabulary(SKILL_B)
    assert "perkalian" in vocab
    assert vocabulary(None) == set()


def test_pick_term_prefers_vocabulary() -> None:
    sentence = "Perkalian adalah penjumlahan berulang."
    # "penjumlahan" is longer, but "Perkalian" is a curriculum term.
    assert pick_term(sentence, vocabulary(SKILL_B)) == "Perkalian"


def test_pick_term_falls_back_to_longest_content_word() -> None:
    sentence = "Membaca kalimat sederhana dengan lancar."
    assert pick_term(sentence, set()) == "sederhana"


def test_blank_out_replaces_first_occurrence_only() -> None:
    assert blank_out("buku dan buku", "buku") == f"{BLANK} dan buku"


def test_blank_out_returns_none_when_absent() -> None:
    assert blank_out("buku dan meja", "pensil") is None


def test_pick_sentence_returns_none_when_too_short() -> None:
    assert pick_sentence("Halo dunia.", set()) is None


def test_pick_sentence_returns_none_without_content_words() -> None:
    assert pick_sentence("Ini itu apa dan siapa.", set()) is None


def test_pick_sentence_prefers_curriculum_term() -> None:
    picked = pick_sentence(SHORT_TEXT, vocabulary(SKILL_B))
    assert picked is not None
    sentence, term = picked
    assert term == "Perkalian"
    assert sentence.startswith("Perkalian")


def test_build_question_short_answer_for_upper_band() -> None:
    draft = build_question(SHORT_TEXT, source_code="test-source", skill=SKILL_B,
                           phase_code="b", grade_code="sd-4")
    assert draft is not None
    assert draft.type == "short_answer"
    assert draft.payload["answer"] == {"accepted": ["Perkalian"]}
    assert draft.payload["skill"] == "SD-B-01"
    assert BLANK in draft.prompt
    assert SCHEMA.validate(draft.payload).ok is True


def test_build_question_single_choice_for_early_band() -> None:
    draft = build_question(FONDASI_TEXT, source_code="test-source", skill=SKILL_FONDASI,
                           phase_code="fondasi", grade_code="tk-b")
    assert draft is not None
    assert draft.type == "single_choice"
    options = draft.payload["content"]["options"]
    assert len(options) == 4
    assert len({option["id"] for option in options}) == 4
    correct = next(o for o in options if o["id"] == draft.payload["answer"]["option_id"])
    assert correct["label"] == draft.term
    assert SCHEMA.validate(draft.payload).ok is True


def test_build_question_is_deterministic() -> None:
    first = build_question(FONDASI_TEXT, source_code="test-source", skill=SKILL_FONDASI,
                           phase_code="fondasi", grade_code="tk-b")
    second = build_question(FONDASI_TEXT, source_code="test-source", skill=SKILL_FONDASI,
                            phase_code="fondasi", grade_code="tk-b")
    assert first == second


def test_build_question_code_matches_schema_pattern() -> None:
    draft = build_question(SHORT_TEXT, source_code="kemendikdasmen-kurikulum", skill=SKILL_B,
                           phase_code="b", grade_code="sd-4")
    assert draft is not None
    assert draft.payload["code"].startswith("Q-KEMENDIKDASMEN-KURIKULUM-")


def test_build_question_returns_none_without_grounding() -> None:
    assert build_question("Ini itu apa.", source_code="s", skill=SKILL_B) is None


def _draft() -> QuestionDraft:
    draft = build_question(SHORT_TEXT, source_code="test-source", skill=SKILL_B,
                           phase_code="b", grade_code="sd-4")
    assert draft is not None
    return draft


def _checks(
    draft: QuestionDraft,
    *,
    skill_id: int | None = 14,
    cp_id: int | None = 2,
    confidence: float = 0.95,
    duplicate_of: int | None = None,
    source_text: str = SHORT_TEXT,
    validation: SchemaValidation | None = None,
) -> list[QaIssue]:
    return question_quality_checks(
        draft,
        validation or SchemaValidation(ok=True, errors=()),
        skill_id=skill_id,
        cp_id=cp_id,
        confidence=confidence,
        duplicate_of=duplicate_of,
        source_text=source_text,
    )


def test_quality_checks_clean_candidate() -> None:
    assert _checks(_draft()) == []


def test_quality_checks_flags_missing_mapping() -> None:
    codes = {issue.code for issue in _checks(_draft(), skill_id=None, cp_id=None)}
    assert codes == {"missing_skill", "missing_cp_mapping"}


def test_quality_checks_flags_duplicate() -> None:
    issues = _checks(_draft(), duplicate_of=9)
    assert [issue.code for issue in issues] == ["duplicate"]
    assert issues[0].blocking is True


def test_quality_checks_flags_schema_invalid() -> None:
    issues = _checks(_draft(), validation=SchemaValidation(ok=False, errors=("boom",)))
    assert [issue.code for issue in issues] == ["schema_invalid"]


def test_quality_checks_flags_answer_mismatch() -> None:
    draft = replace(_draft(), answer_text="TidakAda")
    issues = _checks(draft)
    assert [issue.code for issue in issues] == ["answer_mismatch"]


def test_quality_checks_flags_empty_answer() -> None:
    draft = replace(_draft(), payload={**_draft().payload, "answer": {"accepted": []}})
    issues = _checks(draft)
    assert "empty_answer" in {issue.code for issue in issues}


def test_quality_checks_flags_invalid_options() -> None:
    base = _draft()
    payload = {
        **base.payload,
        "type": "single_choice",
        "content": {"prompt": base.prompt, "options": [
            {"id": "a", "label": "x"}, {"id": "a", "label": "y"}]},
        "answer": {"option_id": "z"},
    }
    draft = replace(base, type="single_choice", payload=payload)
    assert "invalid_options" in {issue.code for issue in _checks(draft)}


def test_quality_checks_flags_long_prompt_and_low_confidence() -> None:
    long_prompt = "Lengkapi kalimat berikut: " + " ".join(["kata"] * 60)
    draft = replace(_draft(), prompt=long_prompt)
    codes = {issue.code for issue in _checks(draft, confidence=0.1)}
    assert codes == {"prompt_too_long", "low_confidence"}
