"""DOCX parsing: paragraph styles define sections."""

from __future__ import annotations

import io

import docx
from docx.document import Document as DocxDocument
from docx.table import Table
from docx.text.paragraph import Paragraph

from content_pipeline.parsers.base import ParsedDocument, Section


def _heading_level(style_name: str) -> int | None:
    name = style_name.strip().lower()
    if not name.startswith("heading"):
        return None
    suffix = name.removeprefix("heading").strip()
    return int(suffix) if suffix.isdigit() else 1


def _table_text(table: Table) -> str:
    rows: list[str] = []
    for row in table.rows:
        cells = [cell.text.strip() for cell in row.cells]
        if any(cells):
            rows.append(" | ".join(cells))
    return "\n".join(rows)


def parse_docx(body: bytes) -> ParsedDocument:
    document: DocxDocument = docx.Document(io.BytesIO(body))

    sections: list[Section] = []
    heading = ""
    level = 0
    buffer: list[str] = []

    def flush() -> None:
        nonlocal buffer
        text = "\n".join(buffer).strip()
        if text:
            sections.append(Section(heading=heading, text=text, level=level))
        buffer = []

    for block in document.iter_inner_content():
        if isinstance(block, Paragraph):
            text = block.text.strip()
            if not text:
                continue
            style_name = block.style.name if block.style is not None else ""
            heading_level = _heading_level(style_name)
            if heading_level is not None:
                flush()
                heading = text
                level = heading_level
                continue
            buffer.append(text)
        elif isinstance(block, Table):
            table = _table_text(block)
            if table:
                buffer.append(table)
    flush()

    title = sections[0].heading if sections else ""
    return ParsedDocument(
        title=title, language="", document_type="docx", sections=tuple(sections)
    )
