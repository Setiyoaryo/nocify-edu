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
9. **S3-compatible object storage** (`media.S3Storage`, minio-go) in staging/production, MinIO in the
   dev stack, local disk only for tests. The bucket stays private and files are streamed through the
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
