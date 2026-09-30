-- Content pipeline: chunk → curriculum mapping candidates (M3).
--
-- One candidate per chunk (the best mapping). Canonical ids are copied as plain
-- integers, not foreign keys: canonical content must never depend on staging,
-- which can be pruned and rebuilt (see docs/content-pipeline-roadmap.md, D4).
-- Owner: the Python content-pipeline service.

CREATE TABLE IF NOT EXISTS content_staging.mapping_candidates (
    id                    bigserial PRIMARY KEY,
    document_id           bigint NOT NULL REFERENCES content_staging.source_documents (id) ON DELETE CASCADE,
    chunk_id              bigint NOT NULL REFERENCES content_staging.document_chunks (id) ON DELETE CASCADE,
    source_id             int NOT NULL,
    -- canonical identifiers, copied at mapping time (no cross-schema FK)
    curriculum_version_id int,
    skill_id              int,
    cp_id                 int,
    phase_code            text NOT NULL DEFAULT '',
    grade_code            text NOT NULL DEFAULT '',
    subject_code          text NOT NULL DEFAULT '',
    method                text NOT NULL DEFAULT 'lexical',
    confidence            numeric(4,3) NOT NULL DEFAULT 0,
    reason                text NOT NULL DEFAULT '',
    status                text NOT NULL DEFAULT 'candidate'
                          CHECK (status IN ('candidate', 'approved', 'rejected')),
    created_at            timestamptz NOT NULL DEFAULT now(),
    UNIQUE (chunk_id)
);
CREATE INDEX IF NOT EXISTS mapping_candidates_source_idx
    ON content_staging.mapping_candidates (source_id, confidence DESC);
CREATE INDEX IF NOT EXISTS mapping_candidates_status_idx
    ON content_staging.mapping_candidates (status);
