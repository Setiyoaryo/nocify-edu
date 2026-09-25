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

## Next (with a designer)
Reference collection → moodboard → brand direction → asset shortlist (DrawKit primary; unDraw for
marketing only) → mascot with the 6 expressions (§26.2) → subject motifs → mobile review.
Then import each file through the CMS asset library.

## Original work
For commissioned artwork, choose the licence "EduPlay Original Work" and record the contract reference
in *modification notes*.
