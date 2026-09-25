# Architecture

```
Browser ──► Traefik ──┬── /api/*  ──► Go API (modular monolith) ──► PostgreSQL
                      └── /*      ──► SvelteKit (node)              └► local/S3 storage
                                     Go worker (same binary) ──────► PostgreSQL
```

Same-origin by design: cookies are `HttpOnly; SameSite=Lax`, and state-changing API
requests must carry an allowed `Origin`. There is no CORS surface.

## Backend modules (`apps/api/internal`)

| Module | Responsibility |
|---|---|
| `auth` | Parents and staff, bcrypt, server-side sessions (HMAC-hashed tokens), RBAC middleware, account deletion |
| `children` | Child profiles scoped to the parent (`LoadOwned` is the only way to resolve a child id) |
| `curriculum` | Curriculum → level → grade → subject → chapter → skill, all data-driven |
| `questions` | Type registry: validator plus grader per type. Answers never leave the server before grading. |
| `assessment` | Session engine: adaptive selection, hint/retry, scoring, completion |
| `progress` | Mastery from raw attempts, parent views, deterministic recommendations |
| `rewards` | XP, levels, badges, avatar/sticker/scene unlocks |
| `subscriptions` | Entitlements, checkout, provider-neutral webhook inbox |
| `media` | Storage abstraction and asset-governance lifecycle |
| `admin` | CMS: question workflow and quality gate, curriculum editing, audit log, KPIs |
| `analytics`, `audit` | First-party events (no child PII) and audit trail |

## Content → engine → insight (§59)

1. **Question attempts** (`question_attempts`) are raw, append-only evidence. A DB trigger forbids updates.
2. **Skill mastery** = recency-weighted accuracy of the latest 20 *first* attempts per skill; a correct answer after a hint counts half (`progress/mastery.go`). `eduplay recompute` rebuilds it after an algorithm change.
3. **Subject/overall** = mean of practised skills.
4. **Recommendations**: deterministic rules (remediate after 3 recent misses, practice under 40%, continue at 40–70%, explore untried skills, challenge when mastered, habit nudge after 3 idle days).

## Adaptive session (§8, §11)

- Session length comes from `grades.session_size` (TK 5–6, SD1–3 8–9, SD4–6 10–12).
- Skills are ordered by need (weakest first, untried treated as 50%), and the session rotates through the top 3.
- Base difficulty per mastery band: <40 → 1, 40–70 → 2, 70–85 → 3, >85 → 4. The first item is a warm-up one step easier.
- 3 first-try-correct answers in a row raise difficulty by 1; 2 misses in a row lower it by 1.
- Each item: wrong → hint → one retry → explanation (+ correct answer). A mini reward appears after item 3.
- Score: full credit first try, half credit on retry/hint; stars 1–3 (never 0).

## Question versioning

Editing creates a new `question_versions` row and returns the question to *draft*. Children
keep receiving `published_version_id` until the new version is reviewed and published.
Attempts reference the exact version shown.
