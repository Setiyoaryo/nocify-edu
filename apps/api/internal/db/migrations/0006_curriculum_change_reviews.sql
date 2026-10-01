-- M7: canonical reviewer ledger for detected curriculum changes.
--
-- The Python watcher detects a source-document change, diffs its CP statements
-- and freezes the per-CP impact in `content_staging` (§44). Go owns canonical
-- content, so the reviewer's decision lives here (not in staging): Go keeps
-- read-only access to staging (roadmap D2/D3).
--
-- `change_event_id` is a staging id copied by value — canonical content must
-- never depend on staging, which can be pruned and rebuilt (D4). `source_document_id`
-- is a canonical id, so it keeps a real FK. The impact snapshot itself stays in
-- staging; this ledger is the durable decision record (with the counts a reviewer
-- signed off on).
--
-- M7 is detect + review only: this migration and its API never mutate
-- `curriculum_cp`, `skills` or `questions`.

CREATE TABLE curriculum_change_reviews (
    id                 bigserial PRIMARY KEY,
    change_event_id    bigint NOT NULL,              -- content_staging.curriculum_change_events.id
    source_document_id uuid REFERENCES curriculum_source_documents (id) ON DELETE SET NULL,
    document_code      text NOT NULL DEFAULT '',
    observed_hash      text NOT NULL DEFAULT '',
    cp_added           int  NOT NULL DEFAULT 0,
    cp_changed         int  NOT NULL DEFAULT 0,
    cp_removed         int  NOT NULL DEFAULT 0,
    affected_questions int  NOT NULL DEFAULT 0,
    -- 'pending' until a reviewer acts; the terminal outcomes are acknowledged
    -- (accepted, curriculum update to be planned), dismissed (no action) and
    -- deferred (revisit later). No status mutates canonical curriculum.
    status             text NOT NULL DEFAULT 'pending'
                       CHECK (status IN ('pending', 'acknowledged', 'dismissed', 'deferred')),
    decided_by         uuid REFERENCES users (id) ON DELETE SET NULL,
    decided_at         timestamptz,
    notes              text NOT NULL DEFAULT '',
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),
    -- One review per detected event; re-running the watcher must not fork it.
    UNIQUE (change_event_id)
);
-- Queue scan: the reviewer screen lists pending changes oldest-first.
CREATE INDEX curriculum_change_reviews_queue_idx
    ON curriculum_change_reviews (updated_at, id) WHERE status = 'pending';
CREATE INDEX curriculum_change_reviews_status_idx
    ON curriculum_change_reviews (status, updated_at DESC);
CREATE INDEX curriculum_change_reviews_document_idx
    ON curriculum_change_reviews (document_code);

-- The watcher resolves affected questions through `question_provenance.cp_id`
-- (§44); without this index that lookup is a sequential scan.
CREATE INDEX question_provenance_cp_idx ON question_provenance (cp_id);
