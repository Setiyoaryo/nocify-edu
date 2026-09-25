# EduPlay

Playful assessment platform for TK–SD children, with skill-level progress for parents.
The source of truth is [`eduplay-mvp-product-technical-spec.md`](eduplay-mvp-product-technical-spec.md);
its Appendix A records how this implementation deviates from the spec and why.

## Quick start

```bash
make up          # Traefik + SvelteKit + Go API + worker + PostgreSQL on http://localhost:8000
make seed        # curriculum, rewards, licenses, 92 questions (published: dev only)
make staff EMAIL=admin@eduplay.local PASSWORD=admin12345 ROLE=admin
```

- Parents: http://localhost:8000 → *Coba gratis*
- CMS: sign in with the staff account → `/app/admin`
- Payments go through **Midtrans Snap**. Put a sandbox Server Key in `infra/compose/.env`
  (`MIDTRANS_SERVER_KEY=SB-Mid-server-…`), and expose the webhook (e.g. with a tunnel) as the Midtrans Notification URL.
  For automated tests: `PAYMENT_PROVIDER=mock make up` (never allowed in production).
- File storage is S3-compatible: SeaweedFS runs in the dev stack as the S3 server (production uses any S3 provider).

Local development without Docker for the apps:

```bash
docker compose -f infra/compose/compose.yaml up -d postgres
cd apps/api && DATABASE_URL=postgres://eduplay:eduplay@localhost:55433/eduplay?sslmode=disable go run ./cmd/eduplay serve
cd apps/api && DATABASE_URL=... go run ./cmd/eduplay seed --publish
cd apps/web && npm i && API_URL=http://localhost:8080 npm run dev     # http://localhost:5173
```

## Tests

| Command | What |
|---|---|
| `make test-api` | Go unit tests plus DB-backed API E2E (critical flow §48, CSRF, RBAC, webhook idempotency). **Resets the schema** of `TEST_DATABASE_URL`. |
| `make test-web` | svelte-check + Vitest |
| `make e2e` | Playwright browser flow at 360px and 1280px against the running stack (register → child → challenge → report → mission → subscribe), plus an overflow check on every marketing page. Set `E2E_PAYMENTS=mock` when the stack runs the test payment provider. |

## Repository

```
apps/api            Go modular monolith: chi, pgx, one binary (serve | worker | migrate | seed | user | recompute)
apps/web            SvelteKit + TypeScript + Tailwind v4 (marketing prerendered; app is client-rendered)
packages/contracts  OpenAPI 3.1 contract
packages/question-schema  JSON Schema for authored questions
packages/design-tokens    Colour/type tokens (starting tokens, pending brand review)
content/            Curriculum, rewards and question banks (data, not code)
assets/             Brand assets + license registry (no production artwork approved yet)
infra/              Dockerfiles, compose (dev + prod with Traefik/Let's Encrypt), backups, monitoring
docs/               Architecture, decisions, operations, privacy, brand-asset governance
```

## Definition of Done (§56): status

| Area | Status |
|---|---|
| Product | Register, child profiles, challenges (7 question types), feedback/hint/retry, completion, progress, parent dashboard, subscription (mock provider), CMS create → review → publish: **done and E2E-tested** |
| Design | Redesigned in the style of Indonesian edtech (Ruangguru/Zenius reference): Plus Jakarta Sans, indigo bands, yellow pill CTAs, card UI. unDraw illustrations with provenance; Phosphor icons; **no emoji, no AI imagery**. **Open:** logo/mascot by a designer (§26.2). |
| Technical | Go, PostgreSQL, Docker, auth, RBAC, migrations, backups, structured logs, Prometheus metrics, CI: **done**. HTTPS via Traefik/Let's Encrypt is configured in `compose.prod.yaml` but not yet exercised on a real domain. |
| Mobile | Verified at 320/360/1280 in Playwright; **open:** real-device performance testing |
| Payments & storage | Midtrans Snap + verified webhooks; S3-compatible storage (SeaweedFS in dev). **Open:** production Midtrans keys, bucket, and domain. |
| Before public launch | Legal review of child-data handling (UU PDP / PP 17/2025), academic review of the seed question bank, transactional email (deferred), auto-renewal (deferred) |
