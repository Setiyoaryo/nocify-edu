-- Content pipeline: semantic layer (M6, §17/§18/§23/§29).
--
-- Three additive concerns, all owned by the Python service:
--   1. embeddings          -- cached vectors for chunks (near-dup) and skills (mapping)
--   2. duplicate_relations -- the §29 outcome per chunk (exact/near/related/unique)
--   3. generation_ledger   -- AI-assisted generation cache + audit (§23/§24)
--
-- Vectors are stored as plain double precision[] because the dev Postgres has no
-- pgvector; cosine is computed in Python. Embeddings are a rebuildable cache, not
-- authoritative data, so there are no cross-schema FKs (roadmap D4).

CREATE TABLE IF NOT EXISTS content_staging.embeddings (
    id         bigserial PRIMARY KEY,
    -- polymorphic owner: document_chunks.id (chunk) or canonical skills.id (skill)
    owner_kind text NOT NULL CHECK (owner_kind IN ('chunk', 'skill')),
    owner_id   bigint NOT NULL,
    -- vectors are model-specific (dimensions differ); never compare across models
    model      text NOT NULL,
    provider   text NOT NULL DEFAULT '',
    dim        int NOT NULL CHECK (dim > 0),
    vector     double precision[] NOT NULL,
    -- sha256 of the normalised embedded text; a change invalidates the cached vector
    text_hash  text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (owner_kind, owner_id, model)
);
CREATE INDEX IF NOT EXISTS embeddings_owner_idx
    ON content_staging.embeddings (owner_kind, model);

-- The §29 deduplication outcome for each chunk: its best relation against an
-- earlier chunk. dedup.py copies the result onto material/question candidates
-- (duplicate_of + qa_issues) so the reviewer CMS shows it without a join.
CREATE TABLE IF NOT EXISTS content_staging.duplicate_relations (
    id                 bigserial PRIMARY KEY,
    chunk_id           bigint NOT NULL REFERENCES content_staging.document_chunks (id) ON DELETE CASCADE,
    duplicate_chunk_id bigint REFERENCES content_staging.document_chunks (id) ON DELETE SET NULL,
    status             text NOT NULL
                       CHECK (status IN ('exact', 'near_duplicate', 'related', 'unique')),
    similarity         numeric(4,3) NOT NULL DEFAULT 0,
    signal             text NOT NULL DEFAULT 'semantic' CHECK (signal IN ('semantic', 'hash')),
    model              text NOT NULL DEFAULT '',
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),
    -- one canonical best relation per chunk, so re-runs replace in place
    UNIQUE (chunk_id)
);
CREATE INDEX IF NOT EXISTS duplicate_relations_status_idx
    ON content_staging.duplicate_relations (status);

-- Assisted-generation ledger (§23/§24). One row per cache key, so a re-run with
-- the same (chunk_hash, template, model, prompt_version) never re-bills. Also the
-- audit trail: model, prompt version, token usage, raw model output, outcome.
CREATE TABLE IF NOT EXISTS content_staging.generation_ledger (
    id                    bigserial PRIMARY KEY,
    chunk_id              bigint NOT NULL REFERENCES content_staging.document_chunks (id) ON DELETE CASCADE,
    source_id             int NOT NULL,
    template              text NOT NULL,
    model                 text NOT NULL,
    provider              text NOT NULL DEFAULT '',
    prompt_version        text NOT NULL,
    chunk_hash            text NOT NULL,
    -- sha256(chunk_hash|template|model|prompt_version): the strict-cache identity
    cache_key             text NOT NULL,
    status                text NOT NULL DEFAULT 'pending'
                          CHECK (status IN ('pending', 'generated', 'failed', 'skipped', 'rejected')),
    request_tokens        int NOT NULL DEFAULT 0,
    response_tokens       int NOT NULL DEFAULT 0,
    total_tokens          int NOT NULL DEFAULT 0,
    attempts              int NOT NULL DEFAULT 0,
    last_error            text NOT NULL DEFAULT '',
    raw_output            jsonb,
    question_candidate_id bigint REFERENCES content_staging.question_candidates (id) ON DELETE SET NULL,
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now(),
    UNIQUE (cache_key)
);
CREATE INDEX IF NOT EXISTS generation_ledger_chunk_idx
    ON content_staging.generation_ledger (chunk_id, template);
CREATE INDEX IF NOT EXISTS generation_ledger_status_idx
    ON content_staging.generation_ledger (status);

-- Provenance of origin: AI-assisted candidates must be visibly marked so the
-- reviewer knows they need extra scrutiny (§24: AI -> Production is forbidden).
ALTER TABLE content_staging.question_candidates
    ADD COLUMN IF NOT EXISTS ai_generated boolean NOT NULL DEFAULT false;
ALTER TABLE content_staging.material_candidates
    ADD COLUMN IF NOT EXISTS ai_generated boolean NOT NULL DEFAULT false;
