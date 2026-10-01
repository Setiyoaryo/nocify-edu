# Operations

## Environments (§54)
`development` (compose.yaml, mock payments, `seed --publish` allowed) · `staging` and `production`
(compose.prod.yaml with `APP_ENV` set accordingly). Secrets live in `infra/compose/.env` on the host,
never in git. The API refuses to start in production without `SESSION_SECRET`, without `PAYMENT_SECRET`,
or with the mock provider.

## Payments (Midtrans)
1. Midtrans dashboard → Settings → Access Keys: copy the Server Key into `MIDTRANS_SERVER_KEY`
   (sandbox `SB-Mid-server-…` with `MIDTRANS_PRODUCTION=false` for dev and staging).
2. Settings → Payment → Notification URL: `https://<domain>/api/v1/webhooks/payment`.
3. Finish redirect is set per transaction (`/app/parent/subscription?checkout=<ref>`); that page polls
   until the webhook has activated the plan.
4. Watch `eduplay_payment_webhooks_total{outcome}` and Admin → Ringkasan "Webhook error".
   Refunds are not applied automatically; handle them manually and record in the audit log.

## Object storage
Any S3-compatible bucket (`S3_ENDPOINT`, `S3_BUCKET`, `STORAGE_ACCESS_KEY`, `STORAGE_SECRET_KEY`).
Keep the bucket private. Enable provider-side versioning/lifecycle for backups of uploaded assets.

## Deploy
CI builds `eduplay-api` and `eduplay-web` images on `main` and pushes them to GHCR. On the server:
```bash
cd infra/compose && docker compose -f compose.prod.yaml pull && docker compose -f compose.prod.yaml up -d
docker compose -f compose.prod.yaml exec api eduplay user --email ops@… --password … --role admin   # first time
docker compose -f compose.prod.yaml exec api eduplay seed      # questions land in "review", never auto-published
```
Migrations run automatically at API start, serialised with an advisory lock.

## Backups
The `backup` service runs `pg_dump -Fc` daily and keeps 14 days. **Copy `/backups` off-host.**
Restore: `infra/scripts/restore.sh`. Practise the restore on staging every month.

## Monitoring
`--profile monitoring` starts Prometheus, Loki/Promtail and Grafana (bound to 127.0.0.1:3001; use an SSH tunnel).
API metrics: `eduplay_http_request_duration_seconds{route,status}`, `eduplay_assessments_started_total`,
`eduplay_assessments_completed_total`, `eduplay_question_attempts_total{correct}`,
`eduplay_payment_webhooks_total{outcome}`, plus Go/process metrics. Logs are JSON to stdout, with
route patterns rather than raw paths, so child ids never land in metric labels.
Admin → Ringkasan shows the business KPIs of §51.

## Worker jobs (every minute, idempotent)
active/trial past period end → `past_due`; `past_due` beyond the 3-day grace, or `cancelled` at
period end → `expired`; pending checkouts older than 24h → `expired`; expired sessions purged;
sessions in progress for over a day → `abandoned`.

## Content pipeline (Python, `--profile pipeline`)
The pipeline is a separate container and never runs in the product request path; the product still
`make up` without it. Migrations are forward-only SQL tracked in `content_staging.schema_migrations`.

- **Publish (M5):** reviewers approve in the CMS; `eduplay publish-candidates` (Go) drains the
  canonical `candidate_publications` queue into canonical questions. Run it from cron after review.
- **Curriculum watch (M7):** `content-pipeline watch --due` polls the registered canonical source
  documents and records a change event + frozen impact snapshot only when a document's hash changed.
  Run it from an external cron (e.g. hourly); `--due` respects each document's `check_frequency`
  (`daily`/`weekly`/`monthly`/`manual`). Overlapping runs are safe — an advisory lock (727275)
  makes a second run a no-op. **M7 never writes canonical curriculum**; a reviewer decides in the
  CMS (`/app/admin/curriculum-changes`), and the decision is recorded in the canonical
  `curriculum_change_reviews` ledger.
- **Secrets/env:** `DATABASE_URL` is the only required variable; AI steps need `AI_PROVIDER`
  (`mock` needs no key). Sources with an `unknown` licence are skipped unless `--allow-unknown-license`.

## Creator attribution (M8)
`eduplay recompute-attribution` rebuilds `question_attribution` (usage/completion/quality per question)
from raw assessment evidence — `--question <uuid>`, `--creator <uuid>`, or all questions by default.
The rollup is also refreshed incrementally when a child finishes a session, so the command is a safety
net after a data repair or a quality-formula change. It never touches canonical content; run it from
cron if a periodic full rebuild is wanted.

Erasure is handled inline: deleting a child (`DELETE /children/{id}`) or an account (`DELETE /me`)
captures the affected questions first and rebuilds their rollup **in the same transaction**, so no
stale row is left behind. `recompute-attribution` therefore only needs to be run by hand after a
data repair or a formula change — not after routine erasure (it remains safe and idempotent if you do).

