"""Curriculum document → CP statement diff (§27, §28, §44).

Pure, deterministic and DB-free. The watcher parses a fetched curriculum
document into text, extracts its CP statements, and diffs them against the
canonical `curriculum_cp` statements for that curriculum version.

**Text contract (M7).** Official Kemendikdasmen documents have no stable
machine-readable format, so `extract_cp_statements` implements a deliberately
simple contract: a statement begins on a line whose first token is a CP code
(``CP-…``) followed by a separator — ``:``, an en/em dash, or a *spaced* hyphen
(`` - ``); following non-empty lines without a code are continuation lines.
First occurrence per code wins. The hyphen must be spaced so that hyphens inside
a CP code are never mistaken for the separator. A source-specific PDF/HTML
parser can replace the body of `extract_cp_statements` later behind the same
signature.
"""

from __future__ import annotations

import hashlib
import re
import unicodedata
from collections.abc import Mapping
from dataclasses import dataclass

ADDED = "added"
CHANGED = "changed"
REMOVED = "removed"

_CP_HEADER = re.compile(r"^\s*(CP-[A-Za-z0-9-]+?)(?:\s*[:\u2013\u2014]\s*|\s+-\s+)(.*)$")
_WHITESPACE = re.compile(r"\s+")


def normalise_statement(statement: str) -> str:
    """Normalise a CP statement for comparison: NFKC, whitespace, case."""
    text = unicodedata.normalize("NFKC", statement).replace("\u00a0", " ")
    return _WHITESPACE.sub(" ", text).strip().casefold()


def statement_hash(statement: str) -> str:
    return hashlib.sha256(normalise_statement(statement).encode()).hexdigest()


def extract_cp_statements(text: str) -> dict[str, str]:
    """Extract ``{cp_code: statement}`` from a curriculum document's text.

    See the module docstring for the text contract. Malformed lines are ignored;
    a blank line ends the current statement.
    """
    statements: dict[str, str] = {}
    current: str | None = None
    for raw_line in text.splitlines():
        line = _WHITESPACE.sub(" ", raw_line).strip()
        if not line:
            current = None
            continue
        match = _CP_HEADER.match(line)
        if match is not None:
            code = match.group(1).upper()
            statement = match.group(2).strip()
            if not statement or code in statements:
                # A bare code is not a statement; first occurrence wins.
                current = None
                continue
            statements[code] = statement
            current = code
            continue
        if current is not None and current in statements:
            statements[current] = f"{statements[current]} {line}".strip()
    return statements


@dataclass(frozen=True)
class CpChange:
    code: str
    change_type: str
    old_statement: str
    new_statement: str
    statement_hash: str


def diff_cp_statements(
    previous: Mapping[str, str], current: Mapping[str, str]
) -> tuple[CpChange, ...]:
    """Classify added/changed/removed CPs, sorted by code.

    Statements are compared after `normalise_statement`, so formatting-only
    differences are not reported.
    """
    changes: list[CpChange] = []
    for code in sorted(set(previous) | set(current)):
        old = previous.get(code, "")
        new = current.get(code, "")
        if code not in previous:
            change_type = ADDED
        elif code not in current:
            change_type = REMOVED
        elif normalise_statement(old) != normalise_statement(new):
            change_type = CHANGED
        else:
            continue
        changes.append(
            CpChange(
                code=code,
                change_type=change_type,
                old_statement=old,
                new_statement=new,
                statement_hash=statement_hash(new or old),
            )
        )
    return tuple(changes)
