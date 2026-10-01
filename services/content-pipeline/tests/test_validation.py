from pathlib import Path

from content_pipeline.validation import (
    SCHEMA_RELATIVE,
    QuestionSchema,
    SchemaValidation,
    find_question_schema,
)

VALID_PAYLOAD = {
    "code": "Q-TEST-ABCD1234",
    "skill": "SD-B-01",
    "type": "short_answer",
    "difficulty": 3,
    "learning_objective": "Peserta didik dapat perkalian.",
    "content": {"prompt": "Lengkapi kalimat berikut: _____ adalah penjumlahan berulang."},
    "answer": {"accepted": ["Perkalian"]},
    "explanation": "Jawaban: Perkalian.",
}


def test_find_question_schema_resolves_from_repo() -> None:
    found = find_question_schema()
    assert found is not None
    assert found.name == SCHEMA_RELATIVE.name
    assert found.is_file()


def test_load_accepts_valid_payload() -> None:
    schema = QuestionSchema.load()
    result = schema.validate(VALID_PAYLOAD)
    assert result == SchemaValidation(ok=True, errors=())


def test_load_rejects_missing_required_field() -> None:
    schema = QuestionSchema.load()
    payload = {key: value for key, value in VALID_PAYLOAD.items() if key != "explanation"}
    result = schema.validate(payload)
    assert result.ok is False
    assert any("explanation" in error for error in result.errors)


def test_load_rejects_unknown_top_level_field() -> None:
    schema = QuestionSchema.load()
    payload = {**VALID_PAYLOAD, "unexpected": True}
    result = schema.validate(payload)
    assert result.ok is False
    assert any("<root>" in error for error in result.errors)


def test_load_rejects_bad_code_pattern() -> None:
    schema = QuestionSchema.load()
    payload = {**VALID_PAYLOAD, "code": "q-lower-case"}
    assert schema.validate(payload).ok is False


def test_explicit_path_is_used() -> None:
    path = find_question_schema()
    assert path is not None
    schema = QuestionSchema.load(path)
    assert schema.validate(VALID_PAYLOAD).ok is True


def test_missing_schema_raises(tmp_path: Path) -> None:
    import pytest

    with pytest.raises(FileNotFoundError):
        QuestionSchema.load(tmp_path / "nope.json")
