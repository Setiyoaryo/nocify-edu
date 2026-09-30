"""File-type detection (§12).

Magic bytes win over the declared content type: a server that mislabels a PDF
as ``text/html`` must not send us down the wrong parser.
"""

from __future__ import annotations

_PDF_MAGIC = b"%PDF-"
_ZIP_MAGIC = b"PK\x03\x04"  # docx/xlsx/pptx are all zip containers

_TEXT_TYPES = ("text/", "application/json", "application/xml")


class UnsupportedDocumentError(ValueError):
    """The document type is not supported by the extraction pipeline."""


def _zip_document_type(content_type: str) -> str:
    base = content_type.split(";", 1)[0].strip().lower()
    if "wordprocessingml" in base or "msword" in base:
        return "docx"
    if "spreadsheetml" in base or "presentationml" in base:
        raise UnsupportedDocumentError(f"unsupported office document: {base}")
    # Ambiguous zip: assume docx and let the parser reject it if it is not.
    return "docx"


def detect_document_type(content_type: str, body: bytes) -> str:
    base = content_type.split(";", 1)[0].strip().lower()

    if body.startswith(_PDF_MAGIC) or base == "application/pdf":
        return "pdf"
    if body.startswith(_ZIP_MAGIC):
        return _zip_document_type(content_type)
    if base == "text/html" or base == "application/xhtml+xml":
        return "html"
    if base == "text/markdown":
        return "markdown"
    if base in {"application/octet-stream", ""} and not _looks_like_text(body):
        # Sniff: binary content that is not text is unsupported.
        raise UnsupportedDocumentError("binary content with no known type")
    if base.startswith(_TEXT_TYPES) or base in {"application/octet-stream", ""}:
        return "text"
    raise UnsupportedDocumentError(f"unsupported content type: {base!r}")


def _looks_like_text(body: bytes) -> bool:
    sample = body[:4096]
    if b"\x00" in sample:
        return False
    try:
        sample.decode("utf-8")
    except UnicodeDecodeError:
        return False
    return True
