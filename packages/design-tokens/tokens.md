# Design tokens v2

Source: `tokens.css`. Direction taken after reviewing Ruangguru and Zenius: bright indigo brand
bands, sunshine-yellow pill CTAs, white cards with soft shadows on a cool canvas, and one
geometric sans (Plus Jakarta Sans, an Indonesian typeface, SIL OFL 1.1, self-hosted).

| Token | Hex | Role | Contrast |
|---|---|---|---|
| ink | #1B1D3A | Text | 16.4:1 on white |
| ink-soft | #5C6079 | Secondary text | 6.2:1 on white (5.7:1 on canvas) |
| primary | #5B5CE2 | Brand, primary buttons | white text 5.2:1 |
| night | #1E1F57 | Deep bands, footer | white text 15:1 |
| sun | #FFC857 | Marketing CTA, stars | ink text 10.6:1 |
| leaf-deep | #1F7A4D | Success text | 5.3:1 on white |
| tangerine-deep | #A84514 | Warning text | 6.0:1 on white |
| ember | #C9501C | Bahasa Indonesia subject card | white text 4.5:1 |
| forest | #15804A | IPAS subject card | white text 5.0:1 |
| error | #C0262D | Errors | 5.9:1 on white |

Rules
- No emoji anywhere. Icons and pictured objects come from Phosphor (MIT), in one style: "fill" for objects and avatars, "bold" for UI.
- Illustrations: unDraw only, recoloured to the brand palette, and recorded in `assets/registry.json`.
- Meaning never rests on colour alone: correctness and mastery always pair colour with an icon or a word.
- Cards: white, radius 20px, `--shadow-card`. Buttons are pills.

Motion (all in `apps/web/src/app.css`, no animation library)
- Only `transform`/`opacity`, 0.3–0.7s, ease-out. Everything is switched off by `prefers-reduced-motion`.
- `anim-rise` / `anim-rise-scale`: content entering (page load, next question), staggered with `--d`.
- `reveal` / `reveal-stagger` + `use:reveal`: marketing sections entering the viewport, once. Hidden only when JS runs (`html.js`).
- `anim-pop` + `<Burst>`: positive moments only (correct answer, mini reward, stars, unlocks, payment confirmed).
- `anim-nudge`: a gentle shake of the feedback bar for "try again", never on the child's answer.
- `anim-float`: slow idle bob, only for the landing proof cards and illustrations.
- `bar-fill`, `<CountUp>`: progress bars fill and numbers count up on first paint.
- `lift` (hover, pointer devices) and `press` (tap). Hover uses the `translate` property so it never conflicts with entrance animations.
- `skeleton`: shimmering placeholder while data loads.
Ratios computed with the WCAG 2.x relative-luminance formula.
