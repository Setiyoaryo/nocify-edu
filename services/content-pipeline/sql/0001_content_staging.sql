-- Content pipeline staging schema (M1).
--
-- Owner: the Python content-pipeline service. The Go API owns canonical
-- content in the default schema and only reads these tables for review/publish.
-- See docs/content-pipeline-roadmap.md (D2, D5).

CREATE SCHEMA IF NOT EXISTS content_staging;

-- ---------------------------------------------------------------- sources
CREATE TABLE IF NOT EXISTS content_staging.source_registry (
    id              serial PRIMARY KEY,
    code            text NOT NULL UNIQUE,
    name            text NOT NULL,
    publisher       text NOT NULL DEFAULT '',
    base_url        text NOT NULL,
    -- official_curriculum | institution | licensed | public | community
    source_type     text NOT NULL,
    tier            smallint NOT NULL CHECK (tier BETWEEN 1 AND 5),
    license_type    text NOT NULL DEFAULT 'unknown',
    license_url     text NOT NULL DEFAULT '',
    terms_url       text NOT NULL DEFAULT '',
    -- respect = honour robots.txt; skip = source is known not to use robots.txt
    robots_policy   text NOT NULL DEFAULT 'respect' CHECK (robots_policy IN ('respect', 'skip')),
    crawl_frequency text NOT NULL DEFAULT 'weekly'
                    CHECK (crawl_frequency IN ('daily', 'weekly', 'monthly', 'manual')),
    priority        smallint NOT NULL DEFAULT 100,
    -- enabled is a kill switch; status is the governance lifecycle (§5).
    enabled         boolean NOT NULL DEFAULT false,
    status          text NOT NULL DEFAULT 'discovered'
                    CHECK (status IN ('discovered', 'verified', 'approved', 'active', 'paused', 'retired')),
    max_pages       int NOT NULL DEFAULT 50 CHECK (max_pages BETWEEN 1 AND 1000),
    rate_limit_ms   int NOT NULL DEFAULT 1000 CHECK (rate_limit_ms >= 0),
    last_crawled_at timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);

-- ---------------------------------------------------------------- crawl runs
CREATE TABLE IF NOT EXISTS content_staging.crawl_runs (
    id                bigserial PRIMARY KEY,
    source_id         int NOT NULL REFERENCES content_staging.source_registry (id) ON DELETE CASCADE,
    started_at        timestamptz NOT NULL DEFAULT now(),
    finished_at       timestamptz,
    status            text NOT NULL DEFAULT 'queued'
                      CHECK (status IN ('queued', 'running', 'completed', 'partial', 'failed', 'cancelled')),
    pages_discovered  int NOT NULL DEFAULT 0,
    pages_fetched     int NOT NULL DEFAULT 0,
    pages_changed     int NOT NULL DEFAULT 0,
    documents_created int NOT NULL DEFAULT 0,
    errors            int NOT NULL DEFAULT 0,
    message           text NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS crawl_runs_source_idx
    ON content_staging.crawl_runs (source_id, started_at DESC);

-- ---------------------------------------------------------------- crawl items
CREATE TABLE IF NOT EXISTS content_staging.crawl_items (
    id             bigserial PRIMARY KEY,
    crawl_run_id   bigint NOT NULL REFERENCES content_staging.crawl_runs (id) ON DELETE CASCADE,
    source_id      int NOT NULL REFERENCES content_staging.source_registry (id) ON DELETE CASCADE,
    url            text NOT NULL,
    canonical_url  text NOT NULL,
    http_status    int,
    content_type   text NOT NULL DEFAULT '',
    etag           text NOT NULL DEFAULT '',
    last_modified  text NOT NULL DEFAULT '',
    content_hash   text NOT NULL DEFAULT '',
    content_length bigint,
    -- Raw archive location in object storage (empty when storage is disabled).
    storage_key    text NOT NULL DEFAULT '',
    language       text NOT NULL DEFAULT '',
    discovered_at  timestamptz NOT NULL DEFAULT now(),
    fetched_at     timestamptz,
    status         text NOT NULL DEFAULT 'discovered'
                   CHECK (status IN ('discovered', 'fetched', 'not_modified', 'skipped', 'failed')),
    error_code     text NOT NULL DEFAULT '',
    error_message  text NOT NULL DEFAULT '',
    UNIQUE (crawl_run_id, canonical_url)
);
-- Cross-run lookup for conditional fetch: newest fetch of a canonical URL.
CREATE INDEX IF NOT EXISTS crawl_items_source_canonical_idx
    ON content_staging.crawl_items (source_id, canonical_url, fetched_at DESC);
CREATE INDEX IF NOT EXISTS crawl_items_hash_idx
    ON content_staging.crawl_items (content_hash) WHERE content_hash <> '';
