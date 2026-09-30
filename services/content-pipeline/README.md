# content-pipeline

Python **content intelligence** service for Nocify EduPlay (M1: source registry + official crawler).

This service owns the `content_staging` PostgreSQL schema. It never writes canonical
content — approved candidates are handed to the Go API through the publish contract.
See [`docs/content-pipeline-roadmap.md`](../../docs/content-pipeline-roadmap.md).

## What M1 does

- **Source registry** (§4–5): data-driven sources with a governance lifecycle
  (`discovered → verified → approved → active / paused / retired`). New sources start
  **disabled**, so nothing is crawled until a human activates it.
- **SSRF-safe fetcher** (§42): allowlist + public-IP-only resolution (blocks loopback,
  private, link-local, cloud metadata), manual redirect validation, size/timeout caps,
  MIME allowlist.
- **robots.txt** (§43): respected through the same fetcher; a source that disallows
  crawling is **paused**, never bypassed.
- **Conditional fetch** (§8): `ETag` / `Last-Modified` / `SHA-256`; a 304 is recorded as
  `not_modified` and skips reprocessing.
- **Raw archive** (§11): changed bodies go to S3 under `raw/<source>/<YYYY>/<MM>/<hash>.<ext>`.
- **Crawl bookkeeping** (§9–10): every run and item is recorded.

## Setup

```bash
cd services/content-pipeline
uv sync --group dev
cp .env.example .env        # adjust DATABASE_URL / S3 settings
```

Apply the staging schema (idempotent, tracked in `content_staging.schema_migrations`):

```bash
uv run content-pipeline migrate
```

For production, run `sql/roles_grants.sql` once as a DB owner to create the
least-privilege `content_pipeline` role (see the file for details).

## Usage

```bash
# Register the official allowlist (inserted disabled)
uv run content-pipeline sources seed-defaults
uv run content-pipeline sources list

# Add a source manually
uv run content-pipeline sources add \
  --code my-source --name "My Source" \
  --base-url https://example.go.id/ --source-type official_curriculum --tier 1

# Preview links without fetching or writing (works before activation)
uv run content-pipeline crawl --source my-source --dry-run --limit 20

# Activate and crawl
uv run content-pipeline sources status --code my-source --status active --enable
uv run content-pipeline crawl --source my-source --limit 50
```

## Layout

```
src/content_pipeline/
  cli.py        command-line entry point
  config.py     environment-driven Settings
  db.py         psycopg3 access
  migrate.py    forward-only SQL migration runner
  safety.py     canonicalisation, allowlist, SSRF protection
  robots.py     robots.txt parsing
  fetcher.py    SSRF-safe conditional HTTP fetcher
  links.py      same-host link discovery (pure function)
  registry.py   source registry CRUD
  storage.py    raw archive (S3)
  crawler.py    crawl orchestration
sql/            content_staging migrations + roles/grants
tests/          unit tests (no database required)
```

## Checks

```bash
uv run ruff check .
uv run mypy
uv run pytest
```

## Not in M1 (next milestones)

Extraction/normalisation and `source_documents` (M2), curriculum taxonomy + CP mapping (M3),
candidates (M4), reviewer CMS + publish (M5). The scheduler is a plain cron/loop for now (§34).
