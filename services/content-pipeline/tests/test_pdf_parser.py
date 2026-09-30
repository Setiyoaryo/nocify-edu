from __future__ import annotations

from typing import Any

import pymupdf

from content_pipeline.parsers.pdf import parse_pdf


def _pdf_with_heading_and_body() -> bytes:
    document: Any = pymupdf.open()
    page = document.new_page()
    page.insert_text((72, 72), "Bilangan", fontsize=24)
    page.insert_text((72, 110), "Peserta didik membandingkan bilangan sampai 100.", fontsize=10)
    data = bytes(document.tobytes())
    document.close()
    return data


def test_pdf_extracts_heading_and_body() -> None:
    parsed = parse_pdf(_pdf_with_heading_and_body())
    assert parsed.document_type == "pdf"
    assert not parsed.needs_ocr
    headings = [s.heading for s in parsed.sections]
    assert "Bilangan" in headings
    body = "\n".join(s.text for s in parsed.sections)
    assert "membandingkan bilangan" in body


def test_pdf_records_page_number() -> None:
    parsed = parse_pdf(_pdf_with_heading_and_body())
    assert all(s.page == 1 for s in parsed.sections)


def test_scanned_pdf_is_flagged_for_ocr() -> None:
    document: Any = pymupdf.open()
    document.new_page()  # empty page: no text layer
    data = bytes(document.tobytes())
    document.close()
    parsed = parse_pdf(data)
    assert parsed.needs_ocr
    assert parsed.sections == ()
