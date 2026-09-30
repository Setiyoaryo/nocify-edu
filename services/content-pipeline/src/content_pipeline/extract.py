"""Extraction orchestration (§12–§15).

Reads raw documents back from the archive, parses them into sections, chunks the
text and stores ``source_documents`` + ``document_chunks``. Re-running is safe:
each crawl item maps to one document and its chunks are replaced.
"""

from __future__ import annotations

import hashlib
import logging
from dataclasses import dataclass

from content_pipeline.chunking import Chunk, chunk_sections
from content_pipeline.db import Database
from content_pipeline.parsers import UnsupportedDocumentError, parse
from content_pipeline.storage import RawArchive

log = logging.getLogger("content_pipeline.extract")


@dataclass(frozen=True)
class ExtractSummary:
    source_code: str
    documents: int
    chunks: int
    needs_ocr: int
    skipped: int
    failed: int


@dataclass(frozen=True)
class _Pending:
    crawl_item_id: int
    source_id: int
    source_code: str
    canonical_url: str
    content_type: str
    content_hash: str
    storage_key: str
    license: str


class Extractor:
    def __init__(self, db: Database, archive: RawArchive) -> None:
        self._db = db
        self._archive = archive

    def run(
        self,
        *,
        codes: list[str] | None = None,
        limit: int | None = None,
        force: bool = False,
    ) -> list[ExtractSummary]:
        pending = self._pending(codes, limit, force)
        counters: dict[str, dict[str, int]] = {}
        for item in pending:
            counts = counters.setdefault(
                item.source_code,
                {"documents": 0, "chunks": 0, "needs_ocr": 0, "skipped": 0, "failed": 0},
            )
            status, chunk_count = self._extract_one(item)
            if status == "failed":
                counts["failed"] += 1
                continue
            counts["documents"] += 1
            counts["chunks"] += chunk_count
            if status == "needs_ocr":
                counts["needs_ocr"] += 1
            elif status == "skipped":
                counts["skipped"] += 1
        return [
            ExtractSummary(source_code=code, **counters[code])
            for code in sorted(counters)
        ]

    # ----------------------------------------------------------------- internal

    def _extract_one(self, item: _Pending) -> tuple[str, int]:
        try:
            body = self._archive.load(item.storage_key)
        except Exception as exc:  # noqa: BLE001 - a missing object must not stop the batch
            log.warning("cannot read raw archive for %s: %s", item.canonical_url, exc)
            return "failed", 0

        try:
            document = parse(body, item.content_type)
        except UnsupportedDocumentError as exc:
            self._store(item, document_type="unknown", title="", language="", status="skipped",
                        error=str(exc), chunks=[])
            return "skipped", 0
        except Exception as exc:  # noqa: BLE001 - a broken file is recorded, not fatal
            log.warning("parse failed for %s: %s", item.canonical_url, exc)
            self._store(item, document_type="unknown", title="", language="", status="failed",
                        error=str(exc), chunks=[])
            return "failed", 0

        if document.needs_ocr:
            self._store(item, document_type=document.document_type, title=document.title,
                        language=document.language, status="needs_ocr",
                        error="no text layer; OCR required", chunks=[])
            return "needs_ocr", 0

        chunks = chunk_sections(document.sections)
        if not chunks:
            self._store(item, document_type=document.document_type, title=document.title,
                        language=document.language, status="skipped",
                        error="no extractable text", chunks=[])
            return "skipped", 0

        self._store(item, document_type=document.document_type, title=document.title,
                    language=document.language, status="extracted", error="", chunks=chunks)
        return "extracted", len(chunks)

    def _pending(
        self, codes: list[str] | None, limit: int | None, force: bool
    ) -> list[_Pending]:
        conditions = ["ci.status = 'fetched'", "ci.storage_key <> ''"]
        params: list[object] = []
        if codes:
            conditions.append("s.code = ANY(%s)")
            params.append(codes)
        if not force:
            conditions.append(
                "NOT EXISTS (SELECT 1 FROM content_staging.source_documents d "
                "WHERE d.crawl_item_id = ci.id)"
            )
        where = " AND ".join(conditions)
        limit_sql = ""
        if limit:
            limit_sql = " LIMIT %s"
            params.append(limit)

        sql = f"""
            SELECT DISTINCT ON (ci.source_id, ci.content_hash)
                   ci.id, ci.source_id, s.code, ci.canonical_url, ci.content_type,
                   ci.content_hash, ci.storage_key, s.license_type
            FROM content_staging.crawl_items ci
            JOIN content_staging.source_registry s ON s.id = ci.source_id
            WHERE {where}
            ORDER BY ci.source_id, ci.content_hash, ci.fetched_at DESC
            {limit_sql}
        """
        with self._db.transaction() as cur:
            cur.execute(sql, params)
            rows = cur.fetchall()
        return [
            _Pending(
                crawl_item_id=int(row[0]),
                source_id=int(row[1]),
                source_code=str(row[2]),
                canonical_url=str(row[3]),
                content_type=str(row[4]),
                content_hash=str(row[5]),
                storage_key=str(row[6]),
                license=str(row[7]),
            )
            for row in rows
        ]

    def _store(
        self,
        item: _Pending,
        *,
        document_type: str,
        title: str,
        language: str,
        status: str,
        error: str,
        chunks: list[Chunk],
    ) -> None:
        with self._db.transaction() as cur:
            cur.execute(
                """
                INSERT INTO content_staging.source_documents
                    (source_id, crawl_item_id, title, canonical_url, document_type, mime_type,
                     content_hash, storage_key, language, license, extraction_status, error)
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
                ON CONFLICT (crawl_item_id) DO UPDATE SET
                    title = EXCLUDED.title,
                    document_type = EXCLUDED.document_type,
                    mime_type = EXCLUDED.mime_type,
                    content_hash = EXCLUDED.content_hash,
                    storage_key = EXCLUDED.storage_key,
                    language = EXCLUDED.language,
                    license = EXCLUDED.license,
                    extraction_status = EXCLUDED.extraction_status,
                    error = EXCLUDED.error,
                    retrieved_at = now()
                RETURNING id
                """,
                (
                    item.source_id, item.crawl_item_id, title, item.canonical_url,
                    document_type, item.content_type.split(";", 1)[0].strip(),
                    item.content_hash, item.storage_key, language, item.license, status, error,
                ),
            )
            row = cur.fetchone()
            assert row is not None
            document_id = int(row[0])

            cur.execute(
                "DELETE FROM content_staging.document_chunks WHERE document_id = %s",
                (document_id,),
            )
            for chunk in chunks:
                cur.execute(
                    """
                    INSERT INTO content_staging.document_chunks
                        (document_id, heading, page_start, page_end, sequence, text, token_count, hash)
                    VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
                    """,
                    (
                        document_id, chunk.heading, chunk.page_start, chunk.page_end,
                        chunk.sequence, chunk.text, chunk.token_count, chunk_hash(chunk.text),
                    ),
                )


def chunk_hash(text: str) -> str:
    return hashlib.sha256(text.strip().encode("utf-8")).hexdigest()
