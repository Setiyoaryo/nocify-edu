-- Pipeline worker heartbeats (M11 operator monitoring).
--
-- Records worker instance heartbeats, host/process info, and current execution state.
-- The Go API and admin UI read this to show real-time worker health (Active / Idle / Stopped).

CREATE TABLE IF NOT EXISTS content_staging.worker_heartbeats (
    worker_id      text PRIMARY KEY,
    hostname       text NOT NULL DEFAULT '',
    pid            int NOT NULL DEFAULT 0,
    status         text NOT NULL DEFAULT 'idle',
    current_job_id bigint,
    last_seen_at   timestamptz NOT NULL DEFAULT now(),
    started_at     timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS worker_heartbeats_seen_idx
    ON content_staging.worker_heartbeats (last_seen_at DESC);
