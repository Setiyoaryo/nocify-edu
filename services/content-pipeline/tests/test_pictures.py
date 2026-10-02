"""Picture-based question templates for TK/Fase A (M10, §22)."""

from content_pipeline.figures import load_vocab
from content_pipeline.questions import (
    COUNT_TEMPLATE,
    PICK_TEMPLATE,
    build_count_question,
    build_pick_question,
    find_count,
    question_quality_checks,
)
from content_pipeline.taxonomy import SkillRef
from content_pipeline.validation import QuestionSchema

SCHEMA = QuestionSchema.load()
VOCAB = load_vocab()

COUNT_TEXT = "Ada tiga jeruk di meja."
PICK_TEXT = "Ibu membeli jeruk di pasar."

SKILL_COUNT = SkillRef(
    id=5, code="TK-A-01", name="Berhitung", description="Menghitung benda di sekitar",
    chapter_name="Bilangan", subject_code="matematika", grade_code="tk-a",
    cp_id=1, cp_code="CP-001", phase_code="fondasi",
)
SKILL_RECOGNISE = SkillRef(
    id=6, code="TK-A-02", name="Mengenal buah", description="Mengenal dan memilih gambar buah",
    chapter_name="Diri sendiri", subject_code="", grade_code="tk-a",
    cp_id=1, cp_code="CP-001", phase_code="fondasi",
)


# ------------------------------------------------------------------- find_count


def test_find_count_reads_digits_and_words() -> None:
    assert find_count("Ada 3 jeruk.") == (3, "3")
    assert find_count("Ada tiga jeruk.") == (3, "tiga")
    assert find_count("Tidak ada angka di sini.") is None


def test_find_count_ignores_out_of_range_numbers() -> None:
    assert find_count("Ada 25 jeruk.") is None
    assert find_count("Ada 0 jeruk.") is None


# -------------------------------------------------------------- count_objects


def test_build_count_question_is_grounded() -> None:
    draft = build_count_question(
        COUNT_TEXT, source_code="test-src", skill=SKILL_COUNT, phase_code="fondasi", grade_code="tk-a", vocab=VOCAB
    )
    assert draft is not None
    assert draft.template == COUNT_TEMPLATE
    assert draft.type == "tap_count"
    content = draft.payload["content"]
    assert content["symbol"] == "orange"
    assert content["count"] == 3
    assert content["color"] == VOCAB.default_color("orange")
    assert 3 in content["choices"]
    assert 2 <= len(content["choices"]) <= 5
    assert draft.payload["answer"] == {"value": 3}
    assert SCHEMA.validate(draft.payload).ok


def test_build_count_question_needs_a_number() -> None:
    assert build_count_question("Ada jeruk di meja.", source_code="s", skill=SKILL_COUNT, vocab=VOCAB) is None


def test_build_count_question_needs_an_object() -> None:
    assert build_count_question("Ada tiga benda di sini.", source_code="s", skill=SKILL_COUNT, vocab=VOCAB) is None


def test_build_count_question_needs_a_counting_skill() -> None:
    assert build_count_question(COUNT_TEXT, source_code="s", skill=SKILL_RECOGNISE, vocab=VOCAB) is None


# ---------------------------------------------------------------- pick_object


def test_build_pick_question_is_grounded() -> None:
    draft = build_pick_question(
        PICK_TEXT, source_code="test-src", skill=SKILL_RECOGNISE, phase_code="fondasi", grade_code="tk-a", vocab=VOCAB
    )
    assert draft is not None
    assert draft.template == PICK_TEMPLATE
    assert draft.type == "image_choice"
    options = draft.payload["content"]["options"]
    assert len(options) == 4
    # No visible label: the child recognises the picture, not the word.
    assert all(option["label"] == "" for option in options)
    answer_id = draft.payload["answer"]["option_id"]
    ids = [option["id"] for option in options]
    assert answer_id in ids
    correct = next(option for option in options if option["id"] == answer_id)
    assert correct["figures"][0]["icon"] == "orange"
    assert SCHEMA.validate(draft.payload).ok


def test_build_pick_question_is_deterministic() -> None:
    first = build_pick_question(PICK_TEXT, source_code="s", skill=SKILL_RECOGNISE, vocab=VOCAB)
    second = build_pick_question(PICK_TEXT, source_code="s", skill=SKILL_RECOGNISE, vocab=VOCAB)
    assert first is not None and second is not None
    assert first.payload == second.payload


def test_build_pick_question_needs_an_object() -> None:
    assert build_pick_question("Pilih gambar yang tepat.", source_code="s", skill=SKILL_RECOGNISE, vocab=VOCAB) is None


def test_build_pick_question_needs_a_recognition_skill() -> None:
    assert build_pick_question(PICK_TEXT, source_code="s", skill=SKILL_COUNT, vocab=VOCAB) is None


# ------------------------------------------------------------- quality checks


def test_picture_drafts_pass_quality_checks() -> None:
    drafts = [
        (COUNT_TEXT, build_count_question(COUNT_TEXT, source_code="s", skill=SKILL_COUNT, vocab=VOCAB)),
        (PICK_TEXT, build_pick_question(PICK_TEXT, source_code="s", skill=SKILL_RECOGNISE, vocab=VOCAB)),
    ]
    for text, draft in drafts:
        assert draft is not None
        validation = SCHEMA.validate(draft.payload)
        assert validation.ok
        issues = question_quality_checks(
            draft, validation, skill_id=1, cp_id=1, confidence=0.9, duplicate_of=None, source_text=text
        )
        codes = {issue.code for issue in issues}
        assert "answer_mismatch" not in codes
        assert "empty_answer" not in codes
        assert "invalid_options" not in codes
