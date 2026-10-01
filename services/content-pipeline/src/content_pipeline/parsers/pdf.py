"""PDF parsing (§13).

Text-layer PDFs are read with PyMuPDF. A PDF with no extractable text is almost
always a scan: we flag it ``needs_ocr`` and leave OCR to a later, explicit queue
rather than making it the default path.
"""

from __future__ import annotations

import statistics
from typing import Any

import pymupdf

from content_pipeline.parsers.base import ParsedDocument, Section

_HEADING_RATIO = 1.15
_MAX_HEADING_CHARS = 120


def _page_lines(page: Any) -> list[tuple[str, float]]:
    # PyMuPDF ships incomplete stubs, so the document objects are handled as Any.
    lines: list[tuple[str, float]] = []
    data = page.get_text("dict")
    for block in data.get("blocks", []):
        for line in block.get("lines", []):
            spans = line.get("spans", [])
            text = "".join(str(span.get("text", "")) for span in spans).strip()
            if not text:
                continue
            size = max((float(span.get("size", 0.0)) for span in spans), default=0.0)
            lines.append((text, size))
    return lines


def parse_pdf(body: bytes) -> ParsedDocument:
    document: Any = pymupdf.open(stream=body, filetype="pdf")
    try:
        pages = [_page_lines(page) for page in document]
        sizes = [size for lines in pages for _, size in lines]
        if not sizes:
            return ParsedDocument(
                title="", language="", document_type="pdf", sections=(), needs_ocr=True
            )
        threshold = statistics.median(sizes) * _HEADING_RATIO

        sections: list[Section] = []
        heading = ""
        level = 0
        heading_page: int | None = None
        buffer: list[str] = []

        def flush() -> None:
            nonlocal buffer
            text = "\n".join(buffer).strip()
            if text:
                sections.append(Section(heading=heading, text=text, page=heading_page, level=level))
            buffer = []

        for page_number, lines in enumerate(pages, start=1):
            for text, size in lines:
                if size >= threshold and len(text) <= _MAX_HEADING_CHARS:
                    flush()
                    heading = text
                    level = 1
                    heading_page = page_number
                    continue
                if heading_page is None:
                    heading_page = page_number
                buffer.append(text)
        flush()

        title = sections[0].heading if sections else ""
        return ParsedDocument(
            title=title, language="", document_type="pdf", sections=tuple(sections)
        )
    finally:
        document.close()
