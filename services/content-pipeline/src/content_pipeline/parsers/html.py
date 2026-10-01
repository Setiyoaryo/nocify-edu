"""HTML parsing: headings define sections (§12, §15)."""

from __future__ import annotations

from bs4 import BeautifulSoup, Tag

from content_pipeline.parsers.base import ParsedDocument, Section

_DROP = ("script", "style", "noscript", "nav", "footer", "header", "aside", "form", "svg")
_BLOCKS = ("h1", "h2", "h3", "h4", "h5", "h6", "p", "li", "table", "pre", "blockquote")


def _heading_level(name: str) -> int | None:
    if len(name) == 2 and name[0] == "h" and name[1].isdigit():
        return int(name[1])
    return None


def _table_text(table: Tag) -> str:
    rows: list[str] = []
    for tr in table.find_all("tr"):
        cells = [c.get_text(" ", strip=True) for c in tr.find_all(["td", "th"])]
        if any(cells):
            rows.append(" | ".join(cells))
    return "\n".join(rows)


def parse_html(body: bytes) -> ParsedDocument:
    soup = BeautifulSoup(body.decode("utf-8", errors="replace"), "html.parser")
    for tag in soup(_DROP):
        tag.decompose()

    title = soup.title.get_text(strip=True) if soup.title else ""
    language = ""
    if soup.html and soup.html.get("lang"):
        language = str(soup.html["lang"]).split("-", 1)[0].lower()

    sections: list[Section] = []
    heading = ""
    level = 0
    buffer: list[str] = []

    def flush() -> None:
        nonlocal buffer
        text = "\n".join(part for part in buffer if part.strip())
        if text.strip():
            sections.append(Section(heading=heading, text=text.strip(), level=level))
        buffer = []

    for element in soup.find_all(_BLOCKS):
        if not isinstance(element, Tag):
            continue
        heading_level = _heading_level(element.name)
        if heading_level is not None:
            flush()
            heading = element.get_text(" ", strip=True)
            level = heading_level
            continue
        if element.name == "table":
            table = _table_text(element)
            if table:
                buffer.append(table)
            continue
        text = element.get_text(" ", strip=True)
        if text:
            buffer.append(text)
    flush()

    if not title and sections:
        title = sections[0].heading
    return ParsedDocument(
        title=title, language=language, document_type="html", sections=tuple(sections)
    )
