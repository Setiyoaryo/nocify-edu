-- Content pipeline: UI-driven job queue (M11).
--
-- The admin UI needs to run the pipeline (crawl → extract → map → candidates →
-- visuals) and schedule it, without a CLI. The Go API owns the UI, so it writes
-- *commands* into this queue; the Python worker daemon claims and executes them.
--
-- This is the single, deliberate control-table exception to "Go is read-only on
-- content_staging" (roadmap D5): Go may INSERT/UPDATE this one table, never a
-- pipeline *data* table (source_registry, document_chunks, *_candidates). Source
-- enable/disable is expressed as a `set_source_status` job so Go never writes
-- source_registry. Python still owns the schema and every data table, and still
-- never writes canonical (D2/D5). Canonical ids (requested_by) are copied by
-- value — no cross-schema FK (D4).

CREATE TABLE IF NOT EXISTS content_staging.pipeline_jobs (
    id           bigserial PRIMARY KEY,
    -- crawl: fetch sources only. run_pipeline: crawl + the downstream stages,
    -- stopping at review candidates. set_source_status: apply a governance
    -- change. watch: curriculum change detection (M7).
    kind         text NOT NULL
                 CHECK (kind IN ('crawl', 'run_pipeline', 'set_source_status', 'watch')),
    params       jsonb NOT NULL DEFAULT '{}',
    status       text NOT NULL DEFAULT 'queued'
                 CHECK (status IN ('queued', 'running', 'done', 'failed', 'cancelled')),
    -- per-stage counters, merged as each stage finishes (e.g. {"crawl": {...}})
    progress     jsonb NOT NULL DEFAULT '{}',
    error        text NOT NULL DEFAULT '',
    attempts     int NOT NULL DEFAULT 0,
    max_attempts int NOT NULL DEFAULT 3,
    -- public.users.id, copied by value (no FK)
    requested_by uuid,
    created_at   timestamptz NOT NULL DEFAULT now(),
    started_at   timestamptz,
    finished_at  timestamptz,
    updated_at   timestamptz NOT NULL DEFAULT now()
);

-- Claim path: oldest queued job, taken with FOR UPDATE SKIP LOCKED.
CREATE INDEX IF NOT EXISTS pipeline_jobs_claim_idx
    ON content_staging.pipeline_jobs (created_at, id) WHERE status = 'queued';
-- Reaper path: running jobs older than the lease.
CREATE INDEX IF NOT EXISTS pipeline_jobs_running_idx
    ON content_staging.pipeline_jobs (started_at) WHERE status = 'running';
-- UI list: most recent first.
CREATE INDEX IF NOT EXISTS pipeline_jobs_recent_idx
    ON content_staging.pipeline_jobs (created_at DESC);
