# Decisions and deviations from the spec

Each entry says what differs from the spec, why, and when to revisit.

1. **Hand-written pgx queries instead of sqlc.** The query surface is small and lives next to
   its handler. Revisit if queries grow or get duplicated; `sqlc` can be introduced module by module.
2. **Plain Svelte components instead of shadcn-svelte/Bits UI.** The child UI needs bespoke
   large-target controls, and the few parent/admin widgets (tabs, dialog, radios) are native
   HTML with ARIA. Revisit when complex widgets (combobox, date picker) are needed.
3. **`assessments` is the child's session (the spec's AssessmentAttempt).** Sessions are adaptive
   and generated per child, so there is no reusable assessment template to separate from attempts.
   The `assessment_attempts` table from §19 is intentionally absent.
4. **Question options live inside `question_versions.content` (JSONB), no `question_options` table.**
   Options are part of an immutable version, and each renderer owns its payload shape.
   Validation is enforced in Go (`questions.Validate`) and documented in `packages/question-schema`.
5. **Extra tables:** `sessions`, `question_reviews` (quality-gate evidence), `payment_checkouts`,
   `payment_events` (webhook inbox for idempotency), `analytics_events`.
6. **Audio via Web Speech synthesis** (instructions) and a Web Audio chime (correct answers).
   No audio files ship, so there is nothing to license. Recorded voice-over can replace it through
   the asset library once audio passes the lifecycle.
7. **No emoji anywhere.** (Superseded the MVP's interim emoji.) Pictured things (question objects,
   avatars, badges, rewards, subject icons) are Phosphor Icons (MIT), one consistent family. Questions
   reference approved icon names (`questions/figures.go`), validated server-side, and a test checks the
   web renders every name. Illustrations are unDraw (brand-recoloured, provenance in `assets/registry.json`).
8. **Tap-to-order instead of drag-and-drop** for `sequence`/`matching`: more reliable for small
   fingers and fully keyboard/screen-reader operable. A drag interaction can be added later
   as an enhancement on the same renderer.
9. **S3-compatible object storage** (`media.S3Storage`, minio-go client) in staging/production, SeaweedFS in the
   dev stack (MinIO stopped publishing public images), local disk only for tests. The bucket stays private and files are streamed through the
   API, so the rule "only published assets are public" lives in one place. Production refuses local storage.
10. **In-memory rate limiting.** Correct for the single-binary MVP; move to Redis if the API scales out.
11. **Free-tier limits** (1 child, 3 challenges/day, teaser progress) and the price (Rp49.000) are
    configuration (`FREE_*`, `MONTHLY_PRICE_IDR`), because §4.2 says pricing is an experiment.
12. **Midtrans Snap** is the payment provider. Webhooks are checked by signature and then re-read from
    the Midtrans Status API; idempotency key is `transaction_id:status`. The HMAC mock provider remains
    for automated tests/CI only and is refused in production.
13. **Monthly plan is pay-per-month, no auto-renewal** (auto-renew deferred by the product owner).
    Each successful payment adds one month from the later of now or the current end date; after the end
    date there is a 3-day grace, then the account falls back to the free plan without losing data.
14. **Typography: Plus Jakarta Sans only** (Indonesian typeface, OFL), replacing Fredoka + Nunito after the
    redesign toward the Ruangguru/Zenius reference. §29 allows "similar" families; one family keeps it consistent.
15. **Missions and skill-focused sessions** (added to close the parent→child loop of §13): a parent turns a
    recommendation into a mission; the child sees it first; focused sessions draw from that skill only.
16. **Soft pastel gradient cards** (`.pastel-*`, `FunCard`) on the parent and child experience, at the
    product owner's request (reference: Ruangguru product cards). §27.2 warns against *too many* gradients.
    The rule kept is one gentle two-stop pastel wash per card, from a fixed palette of 7 tones, with ink
    text (≥12:1), a white icon disc, and no glossy/neon or glassmorphism. No arrow glyphs in UI copy.
17. **Owner operations (admin CMS) are deferred**: bank soal layout, asset library and audit log keep
    working but get their redesign after the parent/student experience.
18. **Creator attribution is decoupled from auth (M8).** `questions.created_by` conflates the author
    with whoever pressed publish — the pipeline publisher sets it to the approving reviewer, and seeded
    questions have none — so attribution lives in a new `creators` table (optional `user_id` link) with
    `questions.creator_id`. Pipeline-published content is attributed to the seeded **system creator**.
    `usage_count`/`completion_count`/`quality_score` are a stored rollup (`question_attribution`)
    rebuilt from `assessment_items` the same way `skill_progress` is rebuilt, and adaptation lineage
    (`adaptation_type`, `adapted_by`, `reviewed_by`) extends `question_provenance`. M8 is readiness
    only: it records the inputs for revenue share and computes no money (that stays in Phase 5, #6).
    The rollup lives in a leaf package (`internal/attribution`, importing only `db`) so the erasure
    paths — which are below `creators` in the import graph — can refresh it: child/account deletion
    captures the affected questions and rebuilds their rollup in the same transaction, and the
    recompute drops rows whose evidence is gone. That refresh is **best-effort** (run in a savepoint),
    not atomic with the erasure: a derived, rebuildable rollup must never deny a data-subject deletion
    (UU PDP), so a recompute failure is logged and the delete still commits — `eduplay
    recompute-attribution` reconciles the drift. The session-complete refresh uses the same helper, so
    a rollup error can never abort a child's session.
19. **Content Studio is a read-only, per-lesson view over existing tables (M9).** The flat review queue
    (M5) makes it hard to see what the pipeline produced for a given lesson, so M9 adds a browse path
    `grade → subject → chapter → skill` and a lesson detail page. "Lesson" is the existing `skills` row —
    there is no separate lesson table. Material↔question lineage is the shared `chunk_id`; questions
    without a matching chunk surface as `orphan_questions`, and candidates with a NULL `skill_id` go to
    an `unmapped` bucket. The page shows pipeline candidates and already-published canonical questions
    side by side but separated (`published` joins `questions.published_version_id`, the live version),
    and inline approve/reject reuses `POST /admin/candidates/{kind}/{id}/decision` unchanged, so the
    11-item checklist and `ValidateDecision` are the same as the queue. Materials stay review-only.
    No migration: the two new endpoints are pure reads. Auto-generating figures/images is a separate
    follow-up milestone — M9 only surfaces the existing `visual_required`/`has_visual` signals so a
    reviewer can see which questions are still text-only.
20. **Early-band visuals are proposed, grounded, licensed and reviewer-approved (M10).** TK/Fase A
    questions need a pictured stimulus, but the pipeline must not invent imagery. Two mechanisms, both
    suggestions: (a) **icon figures** — an Indonesian noun in the source that is also an approved figure
    (`packages/figures/figures.json`, the single source of truth shared by Go/web/Python) becomes a
    `content.figures` entry; (b) **licensed context illustrations** — only a `published`, non-AI asset
    whose `tags` intersect the source concepts, referenced by `/api/v1/media/<uuid>` (never a fabricated
    id). The vocabulary lives in one JSON so the Go `Icons` allowlist, the web Phosphor map and the
    Python loader cannot drift; parity tests enforce it. Assets gain an open `tags text[]` (not a join
    table — matching is read-only `tags && $1`, and `status`/`ai_generated` stay query filters, not
    tags); a tag-only edit never resets an asset's review status. Two invariants protect the learner:
    **no answer leak** (a visual must not depict `answer_text`; `visual_answer_leak` is blocking) and
    **grounding** (only concepts present in the source; `visual_ungrounded` is advisory). `tap_count`/
    `image_choice` already exist in Go, so new picture templates (`count_objects`, `pick_object`) add no
    Go type; `pick_object` options carry **empty labels** so the child recognises the picture rather than
    reads the answer. AI-generated imagery stays banned end to end (DB `assets_no_ai_in_production` +
    the Go `design_approved` gate + `docs/brand-assets.md`). Publishing a question that links an
    unpublished asset is still blocked (`linkMedia`) — the correct behaviour, not a bug. The optional
    semantic tag match (`embeddings.py`) is deferred; exact tag intersection is enough for the first cut.
