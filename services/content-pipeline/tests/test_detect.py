from __future__ import annotations

import pytest

from content_pipeline.parsers.detect import UnsupportedDocumentError, detect_document_type


def test_pdf_magic_wins_over_wrong_content_type() -> None:
    assert detect_document_type("text/html", b"%PDF-1.7\n...") == "pdf"


def test_pdf_by_content_type() -> None:
    assert detect_document_type("application/pdf", b"whatever") == "pdf"


def test_docx_from_zip_magic_and_type() -> None:
    content_type = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
    assert detect_document_type(content_type, b"PK\x03\x04rest") == "docx"


def test_spreadsheet_zip_is_unsupported() -> None:
    content_type = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
    with pytest.raises(UnsupportedDocumentError):
        detect_document_type(content_type, b"PK\x03\x04rest")


def test_html_and_markdown_and_text() -> None:
    assert detect_document_type("text/html; charset=utf-8", b"<html></html>") == "html"
    assert detect_document_type("text/markdown", b"# hi") == "markdown"
    assert detect_document_type("text/plain", b"hello") == "text"


def test_unknown_binary_is_unsupported() -> None:
    with pytest.raises(UnsupportedDocumentError):
        detect_document_type("application/octet-stream", b"\x00\x01\x02\x03")


def test_octet_stream_with_text_content_is_text() -> None:
    assert detect_document_type("application/octet-stream", b"just text") == "text"


def test_unknown_content_type_is_unsupported() -> None:
    with pytest.raises(UnsupportedDocumentError):
        detect_document_type("image/png", b"\x89PNG")
