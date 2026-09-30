"""Plain-text and Markdown parsing."""

from __future__ import annotations

from content_pipeline.parsers.base import ParsedDocument, Section


def _markdown_level(line: str) -> int | None:
    stripped = line.strip()
    if not stripped.startswith("#"):
        return None
    hashes = len(stripped) - len(stripped.lstrip("#"))
    if 1 <= hashes <= 6 and stripped[hashes:hashes + 1] in (" ", ""):
        return hashes
    return None


def parse_text(body: bytes, *, markdown: bool) -> ParsedDocument:
    text = body.decode("utf-8", errors="replace")
    document_type = "markdown" if markdown else "text"

    if not markdown:
        stripped = text.strip()
        only_section = (Section(heading="", text=stripped),) if stripped else ()
        return ParsedDocument(
            title="", language="", document_type=document_type, sections=only_section
        )

    sections: list[Section] = []
    heading = ""
    level = 0
    buffer: list[str] = []

    def flush() -> None:
        nonlocal buffer
        chunk = "\n".join(buffer).strip()
        if chunk:
            sections.append(Section(heading=heading, text=chunk, level=level))
        buffer = []

    for line in text.splitlines():
        markdown_level = _markdown_level(line)
        if markdown_level is not None:
            flush()
            heading = line.strip().lstrip("#").strip()
            level = markdown_level
            continue
        buffer.append(line)
    flush()

    title = sections[0].heading if sections else ""
    return ParsedDocument(
        title=title, language="", document_type=document_type, sections=tuple(sections)
    )
