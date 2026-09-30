"""Shared parser types (§14)."""

from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True)
class Section:
    """A heading and the text that follows it, up to the next heading."""

    heading: str
    text: str
    page: int | None = None
    level: int = 0


@dataclass(frozen=True)
class ParsedDocument:
    title: str
    language: str
    document_type: str
    sections: tuple[Section, ...]
    needs_ocr: bool = False

    @property
    def is_empty(self) -> bool:
        return not any(section.text.strip() for section in self.sections)
