"""Validate a generated question payload against the shared question schema (§23).

The authoritative runtime validation lives in the Go API
(``apps/api/internal/questions.Validate``). This module checks the same payload
against ``packages/question-schema/question.schema.json`` at generation time, so
a malformed candidate never reaches ``content_staging`` — the cheap, early gate
that mirrors the last gate at publish (roadmap D2).
"""

from __future__ import annotations

import json
import os
from dataclasses import dataclass
from pathlib import Path
from typing import Any

import jsonschema

SCHEMA_RELATIVE = Path("packages/question-schema/question.schema.json")


@dataclass(frozen=True)
class SchemaValidation:
    ok: bool
    errors: tuple[str, ...]


def find_question_schema() -> Path | None:
    """Locate the shared schema from an env override, the package, or the repo root."""
    override = os.environ.get("QUESTION_SCHEMA_PATH", "").strip()
    if override:
        candidate = Path(override)
        return candidate if candidate.is_file() else None
    for base in (Path(__file__).resolve(), Path.cwd().resolve()):
        for parent in (base, *base.parents):
            candidate = parent / SCHEMA_RELATIVE
            if candidate.is_file():
                return candidate
    return None


class QuestionSchema:
    def __init__(self, schema: dict[str, Any]) -> None:
        self._validator = jsonschema.Draft202012Validator(schema)

    @classmethod
    def load(cls, path: Path | None = None) -> QuestionSchema:
        resolved = path or find_question_schema()
        if resolved is None:
            raise FileNotFoundError(
                "question schema not found; set QUESTION_SCHEMA_PATH to "
                "packages/question-schema/question.schema.json"
            )
        return cls(json.loads(resolved.read_text(encoding="utf-8")))

    def validate(self, payload: dict[str, Any]) -> SchemaValidation:
        errors = sorted(
            self._validator.iter_errors(payload),
            key=lambda err: [str(part) for part in err.absolute_path],
        )
        if not errors:
            return SchemaValidation(ok=True, errors=())
        messages = tuple(
            f"{'/'.join(str(part) for part in err.absolute_path) or '<root>'}: {err.message}"
            for err in errors
        )
        return SchemaValidation(ok=False, errors=messages)
