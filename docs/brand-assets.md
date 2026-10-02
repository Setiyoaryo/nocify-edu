# Brand asset governance (§25–28, §44–46)

## Current state
- **Illustrations:** 12 unDraw illustrations, recoloured to the brand palette (§28): hero, parents, onboarding,
  progress, missions, result, empty states, login, payment. Originals are in
  `assets/illustrations/undraw/original/`, and production copies are in `apps/web/src/lib/illustrations/`. Provenance is in
  `assets/registry.json` and the CMS asset library (seeded).
- **Icons and pictured objects:** Phosphor Icons (MIT), "fill" for objects/avatars and "duotone"/"bold" for UI.
- **Logo:** a text wordmark with a simple geometric mark, pending a proper brand identity.
- **No emoji. No AI-generated imagery.**

## Enforced in software
- Every asset has provenance: source, creator, licence and licence URL, download date, original file, modification notes, where used.
- Lifecycle: `imported → license_verified → design_approved → performance_checked → published` (or `rejected`).
  - Licence verification requires source, creator, licence and licence URL.
  - AI-generated assets cannot be design-approved; a DB constraint also forbids publishing them.
  - The performance check enforces size budgets (icons ≤50 KB, stickers ≤100 KB, illustrations ≤300 KB, animation/sound ≤500 KB).
  - Changing provenance sends the asset back to `imported`; replacing the file sends it back to `license_verified`.
- Only published assets are publicly served (`/api/v1/media/:id`, SVG served with a script-blocking CSP).
- A question cannot be published while it references an unpublished asset.
- Seeded licences: `assets/licenses/licenses.json`. Keep a copy of each licence text in `assets/licenses/`.

## Question visuals (M10)
TK/Fase A questions carry a pictured stimulus, proposed by the pipeline and approved by a reviewer.
Two forms, both licensed or drawn from the approved vocabulary:

- **Icon figures** — the approved pictured objects live in `packages/figures/figures.json`, the single
  source of truth shared by the Go API (the `Icons` allowlist), the web client (Phosphor map) and the
  Python pipeline. One figure may repeat at most 12 times and a question may show at most 24 pictured
  objects in total.
- **Context illustrations** — licensed illustrations from the asset library, matched to a question by
  **tag** and served as `/api/v1/media/<uuid>`. They must be `published` before the question can be
  published; a question that links an unpublished asset is blocked at publish time.

Rules:
- **No AI-generated imagery, ever.** Enforced by a DB constraint, a Go gate and this document.
- **No answer leak.** A visual must never depict the answer; a leaking visual is a blocking QA issue.
- **Grounding.** Only objects/concepts that appear in the source are pictured; an ungrounded visual is
  an advisory QA issue, not a silent invention.
- **Preferred licences:** DrawKit free (no attribution) for illustrations; Phosphor (MIT) for icons.
  Storyset (Freepik) requires attribution — use only when necessary and add the credit to the credits
  page. unDraw stays marketing-only.

### Tagging assets for questions
Tag each question-usable asset with the Indonesian concepts it depicts (comma-separated in the CMS,
e.g. `buah, jeruk, hitung`). The pipeline matches a question's source concepts against these tags to
propose a context illustration. The tag vocabulary is open — there is no fixed list — but keep tags
lower-case, singular where sensible, and reuse existing tags rather than inventing near-duplicates.
Tagging is metadata only: changing tags never resets an asset's review status.

## Next (with a designer)
Reference collection → moodboard → brand direction → asset shortlist (DrawKit primary; unDraw for
marketing only) → mascot with the 6 expressions (§26.2) → subject motifs → mobile review.
Then import each file through the CMS asset library.

## Original work
For commissioned artwork, choose the licence "EduPlay Original Work" and record the contract reference
in *modification notes*.
