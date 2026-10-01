"""Parsers turn raw bytes into a normalised document (§12, §14)."""

from __future__ import annotations

from content_pipeline.parsers.base import ParsedDocument, Section
from content_pipeline.parsers.detect import UnsupportedDocumentError, detect_document_type
from content_pipeline.parsers.docx import parse_docx
from content_pipeline.parsers.html import parse_html
from content_pipeline.parsers.pdf import parse_pdf
from content_pipeline.parsers.text import parse_text

__all__ = [
    "ParsedDocument",
    "Section",
    "UnsupportedDocumentError",
    "detect_document_type",
    "parse",
]


def parse(body: bytes, content_type: str) -> ParsedDocument:
    """Detect the document type and parse *body* into sections."""
    document_type = detect_document_type(content_type, body)
    if document_type == "html":
        return parse_html(body)
    if document_type == "pdf":
        return parse_pdf(body)
    if document_type == "docx":
        return parse_docx(body)
    if document_type == "markdown":
        return parse_text(body, markdown=True)
    return parse_text(body, markdown=False)
