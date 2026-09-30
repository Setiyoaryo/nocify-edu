-- M5: reviewer decisions + publish ledger, and canonical question provenance.
--
-- The content pipeline writes candidates into the separate `content_staging`
-- schema. Go owns canonical content and is the only side allowed to write it, so
-- the reviewer's decision and the publish state live here (not in staging): Go
-- keeps read-only access to staging (roadmap D2/D3).
--
-- `candidate_id` and `source_id` are staging ids copied by value — canonical
-- content must never depend on staging, which can be pruned and rebuilt (D4).

CREATE TABLE candidate_publications (
    id                    bigserial PRIMARY KEY,
    kind                  text NOT NULL CHECK (kind IN ('material', 'question')),
    candidate_id          bigint NOT NULL,           -- content_staging.<kind>_candidates.id
    source_id             int    NOT NULL,           -- content_staging.source_registry.id
    decision              text NOT NULL CHECK (decision IN ('approved', 'rejected', 'changes_requested')),
    decided_by            uuid REFERENCES users (id) ON DELETE SET NULL,
    decided_at            timestamptz NOT NULL DEFAULT now(),
    notes                 text NOT NULL DEFAULT '',
    checklist             jsonb NOT NULL DEFAULT '{}',
    -- Lifecycle: approved questions become 'queued'; materials are review-only
    -- ('not_applicable'); rejected/change-requested rows keep their outcome.
    status                text NOT NULL
                          CHECK (status IN ('queued', 'publishing', 'published', 'failed',
                                            'not_applicable', 'rejected', 'changes_requested')),
    canonical_question_id uuid REFERENCES questions (id) ON DELETE SET NULL,
    attempts              int  NOT NULL DEFAULT 0,
    last_error            text NOT NULL DEFAULT '',
    published_at          timestamptz,
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now(),
    UNIQUE (kind, candidate_id)
);
-- Queue scan: the publisher claims queued rows oldest-first.
CREATE INDEX candidate_publications_queue_idx
    ON candidate_publications (updated_at, id) WHERE status = 'queued';
CREATE INDEX candidate_publications_kind_status_idx
    ON candidate_publications (kind, status, decided_at DESC);
CREATE INDEX candidate_publications_source_idx ON candidate_publications (source_id);
CREATE INDEX candidate_publications_question_idx ON candidate_publications (canonical_question_id);

-- Provenance of a published question: which source document/chunk it came from.
-- Copied by value so it survives pruning of `content_staging` (D4, §20, §38).
CREATE TABLE question_provenance (
    question_id        uuid PRIMARY KEY REFERENCES questions (id) ON DELETE CASCADE,
    cp_id              int  REFERENCES curriculum_cp (id),
    skill_id           int  NOT NULL REFERENCES skills (id),
    source_code        text NOT NULL,
    source_document_id bigint,                       -- staging id, by value (no FK)
    source_chunk_id    bigint,                       -- staging id, by value (no FK)
    original_url       text NOT NULL DEFAULT '',
    source_title       text NOT NULL DEFAULT '',
    retrieved_at       timestamptz,
    content_hash       text NOT NULL DEFAULT '',
    created_at         timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX question_provenance_skill_idx ON question_provenance (skill_id);
CREATE INDEX question_provenance_hash_idx ON question_provenance (content_hash);
