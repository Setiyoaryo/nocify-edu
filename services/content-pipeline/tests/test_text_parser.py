from __future__ import annotations

from content_pipeline.parsers.text import parse_text


def test_markdown_headings_become_sections() -> None:
    body = b"# Judul\n\nParagraf pertama.\n\n## Sub\n\nParagraf kedua.\n"
    document = parse_text(body, markdown=True)
    assert document.document_type == "markdown"
    assert [s.heading for s in document.sections] == ["Judul", "Sub"]
    assert [s.level for s in document.sections] == [1, 2]
    assert document.sections[0].text == "Paragraf pertama."


def test_plain_text_is_one_section() -> None:
    document = parse_text(b"Baris satu.\n\nBaris dua.", markdown=False)
    assert document.document_type == "text"
    assert len(document.sections) == 1
    assert document.sections[0].heading == ""
    assert "Baris dua." in document.sections[0].text


def test_empty_text_has_no_sections() -> None:
    document = parse_text(b"   \n  ", markdown=False)
    assert document.sections == ()
    assert document.is_empty
