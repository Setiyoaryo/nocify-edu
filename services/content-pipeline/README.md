# content-pipeline

Python **content intelligence** service for Nocify EduPlay (M1: source registry + official crawler; M2: extraction + chunks; M3: curriculum mapping; M4: material/question candidates; M5: reviewer CMS + publish; M6: embeddings, semantic mapping/dedup, AI-assisted generation).

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

## What M4 does

- **Material candidates** (§21): a mapped chunk becomes a reviewable teaching snippet with a
  deterministic `title`, `summary` and `learning_objective` (no AI yet — §23 templates first),
  plus a `difficulty_hint` (1–5). Stored in `material_candidates`, one per chunk, re-runnable.
- **Child readability layer** (§30): `age_band`, `reading_level`, `complexity`, word/sentence
  counts and `visual_required` / `audio_required`, derived from the curriculum phase/grade and
  the measured text. A heuristic, not a psychometric score.
- **Automatic quality checks** (§40): missing skill/CP mapping, duplicate text, missing
  summary/objective, too short/long, low mapping confidence. Blocking issues move a candidate to
  `needs_review`; otherwise it starts as `draft`. A reviewer is still the final authority (§24).
- **Duplicate detection** (§29, exact step): a SHA-256 text fingerprint; an identical earlier
  candidate is linked through `duplicate_of`.
- **Question candidates** (§22–§23): a mapped chunk becomes a reviewable question via a
  deterministic **source-grounded cloze** template (no AI). A sentence loses its most salient
  term (a curriculum term when one is present, else the longest content word); early bands get a
  `single_choice` with distractors from the same chunk, upper bands a `short_answer`. Stored in
  `question_candidates`, one per template per chunk, re-runnable.
- **Shared-schema validation**: the generated payload is checked against
  `packages/question-schema/question.schema.json` *before* it is stored (`validation.py`); a
  payload that fails never enters staging. The Go API validates the same payload again at publish
  (roadmap D2).
- **Question QA** (§40): missing skill/CP, duplicate, schema-invalid, empty answer, invalid
  options, answer mismatch (the answer must occur in the source chunk), long prompt, low
  confidence. Blocking issues → `needs_review`.
- **Contracts**: `packages/contracts/material-candidate.schema.json` and
  `packages/contracts/question-candidate.schema.json` document the envelopes the Go reviewer CMS
  will consume in M5.

## What M5 does (Go side)

- **Reviewer CMS + publish**: the Go API gains `/admin/candidates` (queue + detail + decision) and
  `eduplay publish-candidates`. The publish queue is the canonical ledger `candidate_publications`
  (not a staging table — Go is read-only on `content_staging`). Approving a question queues it;
  `publish-candidates` re-validates with the canonical schema and writes `questions`,
  `question_provenance`, `question_reviews` and `audit_logs`. This service only *writes candidates*;
  it never publishes (roadmap D3).

## What M6 does

- **Embeddings** (§17): cached vectors in `content_staging.embeddings`, keyed by
  `(owner_kind, owner_id, model)` with the sha256 of the embedded text, so a re-run only pays for
  text that changed. One polymorphic store for both consumers: `chunk` (near-dup) and `skill`
  (semantic mapping). Vectors are `double precision[]` — no pgvector; cosine is computed in Python.
- **Provider abstraction** (`provider.py`): one interface for embeddings + chat, implemented by
  `OpenAICompatibleProvider` (any OpenAI-shaped endpoint) and a deterministic offline `MockProvider`
  (no key — mirrors the Go `PAYMENT_PROVIDER=mock`). Configure with `AI_PROVIDER` (`""`/`mock`/
  `openai`), `AI_BASE_URL`, `AI_API_KEY`, `AI_EMBEDDING_MODEL`, `AI_CHAT_MODEL`,
  `AI_TIMEOUT_SECONDS`, `AI_MAX_OUTPUT_TOKENS`, `AI_GENERATION_CAP`,
  `AI_EMBEDDING_BATCH_SIZE`, `AI_PROMPT_VERSION`. `DATABASE_URL` stays the only required variable.
- **Semantic mapping** (§18): `map --semantic` blends the lexical score with embedding similarity:
  `confidence = max(lexical, 0.5·(lexical+semantic))`. The semantic signal may only *lift* the
  deterministic score, never lower it, so existing results and the §18 bands (0.90 / 0.75) are
  preserved. Confidence is a signal, never a publish gate.
- **Near-duplicate dedup** (§29): `dedup` compares cached chunk vectors and assigns each chunk
  `exact` / `near_duplicate` (cosine ≥ 0.92, blocking) / `related` (0.80–0.92, advisory) /
  `unique`, storing the result in `duplicate_relations` and copying it onto material/question
  candidates (`duplicate_of` + `qa_issues`) without clobbering a human decision.
- **AI-assisted generation** (§23–§24): `generate` produces *scenario* and *comprehension*
  candidates. The server — not the model — assembles the payload: the model supplies free text
  only, `code`/`skill` come from canonical ids, and the answer type is fixed to `short_answer` so
  grounding is checkable. The source is passed as delimited untrusted data (`<SUMBER>…</SUMBER>`),
  the answer must occur in the source (`answer_mismatch`) and the prompt must reuse the source's
  vocabulary (`ungrounded`). Every AI candidate is marked `ai_generated` and carries a blocking
  `ai_generated_unverified`, so it can never leave `needs_review` without a human (§24).
- **Cost control**: one question per `(chunk, template)`, a strict cache in `generation_ledger`
  keyed by `sha256(chunk_hash|template|model|prompt_version)` (a re-run reuses the stored output
  without re-billing), a per-run `--cap`, and `--dry-run` that makes **no provider call and writes
  nothing**. Sources with an `unknown` licence are skipped unless `--allow-unknown-license` (§25).
  A provider error is recorded per item (`failed` + `attempts`) and the run aborts after three
  consecutive failures; it resumes from the cache.
- **AI-origin visibility**: `ai_generated` is added to both candidate tables and surfaces through
  the Go detail endpoint and the reviewer CMS badge ("Dibuat AI — wajib review").

> **Mock caveat:** `MockProvider` embeds by sha256, so distinct texts are orthogonal — on the dev
> corpus `map --semantic` legitimately falls back to `lexical` and `dedup` reports `unique`. The
> classification logic itself is covered by unit tests with crafted vectors; real near-duplicate
> detection needs a real embedding model (`AI_PROVIDER=openai`).

## What M7 does

- **Curriculum change detection** (§27, §28, §44): `watch` polls the canonical curriculum source
  documents, hashes the fetched bytes, and — only when the hash changed — diffs their CP statements
  against canonical `curriculum_cp`, then records a frozen impact snapshot. **M7 is detect + review
  only: it never mutates canonical curriculum.** The running baseline lives in staging
  (`curriculum_watch_state`); the canonical `document_hash` stays the untouched approved anchor.
- **Text contract, not a bespoke parser**: `curriculum_diff.extract_cp_statements` reads a CP
  statement from a line whose first token is a `CP-…` code followed by a separator (`:`, en/em dash,
  or a *spaced* hyphen); following non-empty lines continue it. The hyphen must be spaced so code
  hyphens are never the separator. Statements are compared after `normalise_statement` (NFKC, NBSP,
  whitespace, case), so formatting-only edits are not reported.
- **Impact = questions, not just CPs** (§44): a changed CP affects every question on a skill that
  points at it (`skills.cp_id`) and every question linked directly through
  `question_provenance.cp_id`. The snapshot is frozen in `curriculum_change_impacts` +
  `curriculum_change_questions`, so the reviewer queue does not drift if the taxonomy later changes.
- **Reviewer decision is canonical (Go-owned)**: `watch` writes only `content_staging`; the
  `curriculum_change_reviews` ledger and the reviewer queue/detail screens live in the Go API + web.
- **Operations**: an advisory lock (`727275`, distinct from the migration lock `727274`) makes
  concurrent runs safe; `--due` respects each document's `check_frequency` (`daily`/`weekly`/
  `monthly`/`manual`) for an external cron; `--dry-run` fetches and diffs without writing anything;
  `--from-file` replays a local document offline.
- **Grants**: the pipeline role was granted `SELECT` on `public.curriculum_source_documents`,
  `public.questions` and `public.question_provenance` (read-only, same as the rest of canonical).

## What M10 does

- **Pictured stimuli for TK/Fase A** (§22, §30): `visuals` proposes a visual for early-band question
  candidates and `questions` attaches one at generation time. Two forms, both *suggestions* a reviewer
  approves: **icon figures** (an Indonesian noun in the source that is also an approved figure) and
  **licensed context illustrations** (a `published`, non-AI asset whose tags match the source,
  referenced by `/api/v1/media/<uuid>` — never a fabricated id).
- **One vocabulary, no drift**: `packages/figures/figures.json` is the single source of truth. The Go
  API generates its `Icons` allowlist from it, the web client maps each name to a Phosphor component,
  and `figures.py` loads it here (override with `FIGURES_PATH`). Parity tests cover all three sides.
- **Two invariants**: **no answer leak** (a visual must not depict `answer_text` — `visual_answer_leak`
  is blocking) and **grounding** (only concepts present in the source; `visual_ungrounded` is advisory).
- **New picture templates**: `count_objects → tap_count` and `pick_object → image_choice`, using the
  Go types unchanged. They emit only when the skill/objective is about counting/recognising objects
  *and* the number/object is grounded in the source. `QuestionBuilder` now runs several templates per
  chunk (unique key `(chunk_id, template)`).
- **Grants**: matching an illustration needs read-only access to `public.assets` / `public.asset_licenses`
  (see `sql/roles_grants.sql`). Without it the pipeline fails closed and proposes icons only.

## What M11 does

- **Runs the pipeline automatically, from the CMS.** The admin UI (Go) writes *commands* into
  `content_staging.pipeline_jobs`; `content-pipeline worker` claims them (`FOR UPDATE SKIP LOCKED`) and
  runs the stage builders, writing per-stage progress back. The CMS page `/app/admin/pipeline` lists
  sources and jobs with live progress.
- **Scheduled + manual.** The worker also crawls active sources on their `crawl_frequency` (advisory
  lock 727276) and can be triggered by a "Jalankan sekarang" button. Human review stays mandatory: the
  pipeline stops at candidates; a reviewer approves and a one-click **Terbitkan** publishes.
- **One control-table exception.** The Go role gets `INSERT, UPDATE` on `pipeline_jobs` only — never a
  pipeline *data* table (source enable/disable is itself a job). Python still never writes canonical.
- **Resilient.** Stages are idempotent; a 30-minute reaper re-queues stuck jobs; SIGTERM re-queues the
  current job and exits between stages. Retry is manual (UI `/retry`).

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

# Build material candidates from mapped chunks
uv run content-pipeline materials --source my-source --limit 50
uv run content-pipeline materials --source my-source --force # rebuild in place

# Build question candidates (validated against packages/question-schema)
uv run content-pipeline questions --source my-source --limit 50
uv run content-pipeline questions --source my-source --force # rebuild in place

# M6 — cache embeddings for chunks and skills (needs AI_PROVIDER; mock needs no key)
uv run content-pipeline embed --source my-source
uv run content-pipeline embed --kind skill --force             # re-embed skills only

# Map with the semantic blend (run `embed` first; falls back to lexical per chunk)
uv run content-pipeline map --source my-source --semantic --force

# Near-duplicate detection over cached chunk vectors (§29)
uv run content-pipeline dedup --source my-source

# AI-assisted scenario/comprehension questions (§23) — cached, capped, never auto-approved
uv run content-pipeline generate --dry-run                     # report only: no calls, no writes
uv run content-pipeline generate --allow-unknown-license --cap 20
uv run content-pipeline generate --template scenario --force   # re-generate (re-bills)

# M7 — detect curriculum source-document changes (§44); detect + review only
uv run content-pipeline watch register --document-code KEMENDIKDASMEN-KURIKULUM \
  --url https://example.go.id/kurikulum.pdf --frequency weekly
uv run content-pipeline watch --dry-run                        # fetch + diff, write nothing
uv run content-pipeline watch --due                            # only documents whose frequency is due
uv run content-pipeline watch --from-file ./kurikulum.txt --document-code KEMENDIKDASMEN-KURIKULUM
uv run content-pipeline changes --status analyzed --limit 20   # list detected changes

# M10 — propose pictured stimuli for early-band (TK/Fase A) questions
uv run content-pipeline visuals --source my-source             # propose icons / illustrations
uv run content-pipeline visuals --source my-source --dry-run   # report only: no writes
uv run content-pipeline visuals --source my-source --force     # also revisit questions that already have one

# M11 — worker daemon (claims UI-queued jobs + crawls due sources)
uv run content-pipeline worker                                 # run forever (long-lived service)
uv run content-pipeline worker --once                          # process at most one queued job and exit
```

The Go reviewer API + web screens for M7 (`/api/curriculum-changes`, the reviewer queue) are part
of the same milestone but live outside this service.

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
  readability.py child readability bands and text metrics (§30)
  materials.py  material candidates, QA checks and duplicate detection (§21/§40)
  qa.py         shared quality-issue primitives (§40)
  validation.py shared question-schema loading and payload validation
  questions.py  deterministic question candidates + QA (§22/§23/§40)
  figures.py    shared figure vocabulary loader (packages/figures/figures.json, M10)
  visuals.py    visual-need decisions + grounded visual proposals (M10)
  provider.py   AI provider abstraction: OpenAI-compatible + offline mock (M6)
  similarity.py cosine + semantic confidence and dedup thresholds (M6)
  embeddings.py embedding cache store, batcher and builder (M6)
  dedup.py      near-duplicate detection over chunk vectors (§29, M6)
  assisted.py   AI-assisted scenario/comprehension generation + grounding (M6)
  curriculum_diff.py  pure CP-statement extraction + diff (M7)
  watch.py      curriculum change watcher, impacts and advisory lock (M7)
  worker.py     job-queue worker daemon: claim/dispatch/schedule/reap (M11)
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

**M8 creator/revenue attribution is not a pipeline milestone** — it adds no staging tables and no
Python code. The data is canonical/product-side (questions, attempts, sessions), so it lives in the
Go API + web (`internal/creators`, `/app/admin/creators`). The pipeline's only touchpoint is that
publish attributes pipeline content to the canonical system creator via `question_provenance`.

The scheduler is a plain cron/loop for now (§34) — for M7 run
`content-pipeline watch --due` from cron; the advisory lock makes overlapping runs safe.

The deterministic cloze template is intentionally narrow; scenario/comprehension items come from
M6 assisted generation, and every candidate — deterministic or AI — is still reviewed by a human
(§24). With `AI_PROVIDER=mock` the semantic layer is deliberately inert (orthogonal vectors); use a
real embedding model for genuine near-duplicate detection.

**M10 deferrals:** the optional semantic match between source terms and asset tags (a `visual_tag`
embedding owner) is not implemented — exact tag intersection is enough for the first cut. Importing
the curated DrawKit illustrations is an operator task through the CMS, not a pipeline step.
