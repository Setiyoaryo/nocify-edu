-- Content pipeline: extracted documents and chunks (M2).
--
-- Raw bytes stay in object storage (crawl_items.storage_key); these tables hold
-- the normalised text so extraction can be re-run without re-fetching (§11, §51).
-- Owner: the Python content-pipeline service.

CREATE TABLE IF NOT EXISTS content_staging.source_documents (
    id                bigserial PRIMARY KEY,
    source_id         int NOT NULL REFERENCES content_staging.source_registry (id) ON DELETE CASCADE,
    crawl_item_id     bigint NOT NULL REFERENCES content_staging.crawl_items (id) ON DELETE CASCADE,
    title             text NOT NULL DEFAULT '',
    canonical_url     text NOT NULL,
    document_type     text NOT NULL,               -- html | pdf | docx | text | markdown
    mime_type         text NOT NULL DEFAULT '',
    content_hash      text NOT NULL,
    storage_key       text NOT NULL DEFAULT '',
    language          text NOT NULL DEFAULT '',
    license           text NOT NULL DEFAULT '',
    published_at      timestamptz,
    retrieved_at      timestamptz NOT NULL DEFAULT now(),
    extraction_status text NOT NULL DEFAULT 'extracted'
                      CHECK (extraction_status IN ('extracted', 'needs_ocr', 'skipped', 'failed')),
    error             text NOT NULL DEFAULT '',
    UNIQUE (crawl_item_id)
);
CREATE INDEX IF NOT EXISTS source_documents_source_idx
    ON content_staging.source_documents (source_id, retrieved_at DESC);
CREATE INDEX IF NOT EXISTS source_documents_hash_idx
    ON content_staging.source_documents (content_hash);

CREATE TABLE IF NOT EXISTS content_staging.document_chunks (
    id          bigserial PRIMARY KEY,
    document_id bigint NOT NULL REFERENCES content_staging.source_documents (id) ON DELETE CASCADE,
    heading     text NOT NULL DEFAULT '',
    page_start  int,
    page_end    int,
    sequence    int NOT NULL,
    text        text NOT NULL,
    token_count int NOT NULL DEFAULT 0,
    hash        text NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (document_id, sequence)
);
CREATE INDEX IF NOT EXISTS document_chunks_document_idx
    ON content_staging.document_chunks (document_id, sequence);
CREATE INDEX IF NOT EXISTS document_chunks_hash_idx
    ON content_staging.document_chunks (hash);
