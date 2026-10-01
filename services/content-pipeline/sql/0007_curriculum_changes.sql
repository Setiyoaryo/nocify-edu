-- Content pipeline: curriculum change detection (M7, §27/§28/§44).
--
-- The watcher polls canonical curriculum source documents, hashes them, and —
-- only when the hash changed — diffs their CP statements against canonical
-- `curriculum_cp` and records a frozen impact snapshot here.
--
-- M7 is *detect + review only*: it never mutates canonical curriculum. The
-- reviewer decision lives in the canonical ledger `curriculum_change_reviews`
-- (Go-owned, like `candidate_publications` in M5). Canonical ids are copied by
-- value — there are no cross-schema FKs (roadmap D4).

-- Operational watch cursor: one row per watched canonical source document.
-- Canonical `curriculum_source_documents.document_hash` is the approved anchor
-- and is never advanced by M7, so the running baseline lives here (seeded from
-- canonical on registration) together with the conditional-fetch state.
CREATE TABLE IF NOT EXISTS content_staging.curriculum_watch_state (
    id                    bigserial PRIMARY KEY,
    -- canonical curriculum_source_documents.id, copied by value (no FK)
    source_document_id    uuid NOT NULL,
    source_document_code  text NOT NULL,
    curriculum_version_id int,
    -- staging override; the canonical source_url wins when it is set
    url                   text NOT NULL DEFAULT '',
    baseline_hash         text NOT NULL DEFAULT '',
    baseline_source       text NOT NULL DEFAULT 'canonical'
                          CHECK (baseline_source IN ('canonical', 'observed')),
    etag                  text NOT NULL DEFAULT '',
    last_modified         text NOT NULL DEFAULT '',
    check_frequency       text NOT NULL DEFAULT 'weekly'
                          CHECK (check_frequency IN ('daily', 'weekly', 'monthly', 'manual')),
    last_checked_at       timestamptz,
    last_changed_at       timestamptz,
    last_status           text NOT NULL DEFAULT 'pending'
                          CHECK (last_status IN ('pending', 'unchanged', 'not_modified', 'changed',
                                                 'skipped', 'failed')),
    last_error            text NOT NULL DEFAULT '',
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now(),
    UNIQUE (source_document_id)
);
CREATE INDEX IF NOT EXISTS curriculum_watch_state_status_idx
    ON content_staging.curriculum_watch_state (last_status, last_checked_at);

-- One row per detected document change (a new observed hash != the baseline).
CREATE TABLE IF NOT EXISTS content_staging.curriculum_change_events (
    id                    bigserial PRIMARY KEY,
    source_document_id    uuid NOT NULL,
    source_document_code  text NOT NULL,
    curriculum_version_id int,
    url                   text NOT NULL,
    baseline_hash         text NOT NULL,
    observed_hash         text NOT NULL,
    http_status           int,
    etag                  text NOT NULL DEFAULT '',
    last_modified         text NOT NULL DEFAULT '',
    detection             text NOT NULL DEFAULT 'content_hash'
                          CHECK (detection IN ('content_hash', 'etag', 'last_modified', 'first_seen')),
    document_type         text NOT NULL DEFAULT '',
    storage_key           text NOT NULL DEFAULT '',
    cp_added              int NOT NULL DEFAULT 0,
    cp_changed            int NOT NULL DEFAULT 0,
    cp_removed            int NOT NULL DEFAULT 0,
    status                text NOT NULL DEFAULT 'detected'
                          CHECK (status IN ('detected', 'analyzed', 'failed', 'superseded')),
    error                 text NOT NULL DEFAULT '',
    retrieved_at          timestamptz NOT NULL DEFAULT now(),
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS curriculum_change_events_doc_idx
    ON content_staging.curriculum_change_events (source_document_id, created_at DESC);
CREATE INDEX IF NOT EXISTS curriculum_change_events_status_idx
    ON content_staging.curriculum_change_events (status, created_at DESC);
-- Dedup only *open* events, so a failed analysis can be retried without
-- duplicating the row while a later genuine re-change is still recordable.
CREATE UNIQUE INDEX IF NOT EXISTS curriculum_change_events_open_uidx
    ON content_staging.curriculum_change_events (source_document_id, observed_hash)
    WHERE status IN ('detected', 'failed');

-- One row per changed CP. cp_id is NULL for an `added` CP (not yet canonical).
CREATE TABLE IF NOT EXISTS content_staging.curriculum_change_impacts (
    id                 bigserial PRIMARY KEY,
    change_event_id    bigint NOT NULL
                       REFERENCES content_staging.curriculum_change_events (id) ON DELETE CASCADE,
    cp_id              int,
    cp_code            text NOT NULL,
    change_type        text NOT NULL CHECK (change_type IN ('added', 'changed', 'removed')),
    old_statement      text NOT NULL DEFAULT '',
    new_statement      text NOT NULL DEFAULT '',
    statement_hash     text NOT NULL DEFAULT '',
    affected_skills    int NOT NULL DEFAULT 0,
    affected_questions int NOT NULL DEFAULT 0,
    UNIQUE (change_event_id, cp_code)
);
CREATE INDEX IF NOT EXISTS curriculum_change_impacts_event_idx
    ON content_staging.curriculum_change_impacts (change_event_id);
CREATE INDEX IF NOT EXISTS curriculum_change_impacts_cp_idx
    ON content_staging.curriculum_change_impacts (cp_id);

-- Frozen snapshot of which canonical questions were affected, and how.
CREATE TABLE IF NOT EXISTS content_staging.curriculum_change_questions (
    id            bigserial PRIMARY KEY,
    impact_id     bigint NOT NULL
                  REFERENCES content_staging.curriculum_change_impacts (id) ON DELETE CASCADE,
    question_id   uuid NOT NULL,
    question_code text NOT NULL DEFAULT '',
    skill_id      int NOT NULL,
    match         text NOT NULL CHECK (match IN ('skill', 'provenance')),
    status        text NOT NULL DEFAULT '',
    UNIQUE (impact_id, question_id)
);
CREATE INDEX IF NOT EXISTS curriculum_change_questions_impact_idx
    ON content_staging.curriculum_change_questions (impact_id);
CREATE INDEX IF NOT EXISTS curriculum_change_questions_question_idx
    ON content_staging.curriculum_change_questions (question_id);
