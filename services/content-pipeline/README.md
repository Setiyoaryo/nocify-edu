# content-pipeline

Python **content intelligence** service for Nocify EduPlay (M1: source registry + official crawler; M2: extraction + chunks; M3: curriculum mapping).

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

## What M2 does

- **Parsers** (§12–13): HTML (headings → sections), Markdown/plain text, PDF via PyMuPDF
  (per-page, heading detection by font size; a PDF with no text layer is flagged
  `needs_ocr` rather than OCR'd by default), DOCX via python-docx.
- **File-type detection**: magic bytes win over the declared content type.
- **Chunking** (§15): section → paragraph → sentence-boundary fallback with overlap,
  not a blind character count.
- **Documents** (§11, §14): `source_documents` (title, type, hash, page provenance) and
  `document_chunks` (heading, page, sequence, tokens, hash). Re-running replaces a
  document's chunks; `--force` re-extracts from the raw archive (§51).

## What M3 does

- **Reads the canonical taxonomy read-only**: the active curriculum version, its skills and
  their Capaian Pembelajaran (`taxonomy.py`).
- **Maps chunks to skills/CP** (§16–§18) with the deterministic + lexical layers: a chunk is
  scored against each skill's terms (name, description, chapter) and the best match is stored
  in `mapping_candidates` with a confidence. Embeddings and LLM assistance are M6; a reviewer
  stays the final authority.
- Confidence bands (§18): `>= 0.90` high, `0.75–0.89` review, `< 0.75` manual.
- Canonical ids are copied as plain integers — **no cross-schema foreign keys** (roadmap D4).

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

# Extract text + chunks from archived documents (needs S3_BUCKET)
uv run content-pipeline extract --source my-source --limit 20
uv run content-pipeline extract --source my-source --force   # re-parse from the raw archive

# Map chunks onto the curriculum (CP + skills)
uv run content-pipeline map --source my-source --limit 50
uv run content-pipeline map --source my-source --force       # re-map
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
  parsers/      html / pdf / docx / text + file-type detection
  chunking.py   heading -> section -> paragraph -> sentence fallback
  extract.py    extraction orchestration (documents + chunks)
  taxonomy.py   read-only view of the canonical curriculum + CP
  mapping.py    chunk -> skill/CP scoring and confidence
sql/            content_staging migrations + roles/grants
tests/          unit tests (no database required)
```

## Checks

```bash
uv run ruff check .
uv run mypy
uv run pytest
```

## Not yet (next milestones)

Curriculum taxonomy + CP mapping (M3), candidates (M4), reviewer CMS + publish (M5).
The scheduler is a plain cron/loop for now (§34).
