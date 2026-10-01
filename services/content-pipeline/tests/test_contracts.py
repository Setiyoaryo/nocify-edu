from __future__ import annotations

import json
from pathlib import Path
from typing import Any, cast

import pytest
from jsonschema import Draft202012Validator
from jsonschema.exceptions import ValidationError

CONTRACTS = Path(__file__).resolve().parents[3] / "packages" / "contracts"


def _load(name: str) -> dict[str, Any]:
    text = (CONTRACTS / name).read_text(encoding="utf-8")
    return cast("dict[str, Any]", json.loads(text))


def _qa_validator(schema: dict[str, Any]) -> Draft202012Validator:
    root = {
        "$schema": "https://json-schema.org/draft/2020-12/schema",
        "$defs": schema["$defs"],
        "type": "array",
        "items": {"$ref": "#/$defs/qaIssue"},
    }
    return Draft202012Validator(root)


def test_question_contract_accepts_the_new_qa_codes() -> None:
    validator = _qa_validator(_load("question-candidate.schema.json"))
    for code in ("near_duplicate", "related", "ai_generated_unverified", "ungrounded"):
        validator.validate([{"code": code, "message": "m", "blocking": True}])


def test_material_contract_accepts_the_dedup_codes() -> None:
    validator = _qa_validator(_load("material-candidate.schema.json"))
    for code in ("near_duplicate", "related"):
        validator.validate([{"code": code, "message": "m", "blocking": False}])


def test_contracts_still_reject_unknown_qa_codes() -> None:
    validator = _qa_validator(_load("question-candidate.schema.json"))
    with pytest.raises(ValidationError):
        validator.validate([{"code": "not_a_real_code", "message": "m", "blocking": False}])


def test_assisted_templates_are_declared() -> None:
    templates = _load("question-candidate.schema.json")["properties"]["template"]["enum"]
    assert {"cloze", "scenario", "comprehension"} <= set(templates)


def test_ai_generated_is_declared_in_both_contracts() -> None:
    for name in ("question-candidate.schema.json", "material-candidate.schema.json"):
        assert _load(name)["properties"]["ai_generated"]["type"] == "boolean"
