"""Automatic quality-check primitives (§40).

Shared by material and question candidates so the issue shape stored in
``qa_issues`` is identical across the staging schema.
"""

from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True)
class QaIssue:
    code: str
    message: str
    blocking: bool

    def as_dict(self) -> dict[str, object]:
        return {"code": self.code, "message": self.message, "blocking": self.blocking}


def has_blocking(issues: list[QaIssue]) -> bool:
    return any(issue.blocking for issue in issues)
