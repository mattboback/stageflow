---
target: stageflow.org landing + playground
total_score: 23
p0_count: 0
p1_count: 4
p2_count: 4
timestamp: 2026-07-07T05-54-39Z
slug: stageflow-org-landing-playground
---
# StageFlow: Landing Page + Playground Critique

**Method: dual-agent (A: `a0304ba122a5a5f86` · B: `a4175e54e57a63a6b`)**

Two surfaces, two registers: the **landing page** (stageflow.org/) is brand-register. The **playground** (stageflow.org/playground) is product-register.

## Design Health Score

| # | Heuristic | Score | Key Issue |
|---|-----------|-------|-----------|
| 1 | Visibility of System Status | 3 | Playground pills solid; nothing async on landing to fault. |
| 2 | Match System / Real World | 3 | Instrument metaphor coherent but needs acclimation. |
| 3 | User Control and Freedom | 2 | No reset-to-defaults, no bulk arm/disarm, no undo. |
| 4 | Consistency and Standards | 2 | Two different toggle-switch skins on one page. |
| 5 | Error Prevention | 2 | Landing URL field has zero validation before navigating. |
| 6 | Recognition Rather Than Recall | 3 | Labels visible throughout, no icon-only nav. |
| 7 | Flexibility and Efficiency | 1 | No bulk actions, no keyboard shortcuts, no presets. |
| 8 | Aesthetic and Minimalist Design | 3 | Low visual clutter; genericness is the issue, not noise. |
| 9 | Error Recovery | 2 | Error copy plain-language but detached from source field. |
| 10 | Help and Documentation | 2 | Docs link only in header/footer. |
| **Total** | | **23/40** | **Acceptable** |

## Anti-Patterns Verdict

LLM assessment: no gradient text, no fake testimonials, demo numbers honestly labeled "Sample." Fails brand.md's inverse test: the macro-structure (hero+input, feature list, dark stats band, 3-step, CTA) fits the modal SaaS-devtool landing page template.

Deterministic scan: `detect.mjs` returned zero findings (tool scope limitation — doesn't check heading skips or per-section class repetition). Both assessments independently found:
- `.eyetag` kicker reused 5x on landing (`home.tsx:71,151,188,230,263`) — the named banned eyebrow pattern.
- Heading skip on playground: h1 (`playground.tsx:215`) → h3 (`PlaygroundAuthConfig.tsx:23`, `PlaygroundAiConfig.tsx:55`), no h2 anywhere.

False positive: numbered `01/02/03` workflow steps are a legitimate single sequence, not a ban violation.

## Priority Issues

- **[P1]** Repeated `.eyetag` kicker on every section — banned AI-scaffolding pattern. Fix: keep one kicker max, vary other section intros structurally. → `/impeccable quieter` or `/impeccable typeset`
- **[P1]** Landing defaults to Restrained color + modal SaaS structure on a brand-register surface. Fix: commit further in at least one moment (fuller dark/drenched hero, asymmetric break). → `/impeccable bolder`
- **[P1]** Heading hierarchy skip on playground (h1→h3, no h2). Fix: promote Target/Channels/Auth/AI Navigator to a consistent level. → `/impeccable audit`
- **[P1]** `--ink-faint` used on real informative copy, violating the token file's own documented rule (`instrument.css:22`). Hits: `.drop__hint`, `.runhint`, `.cta__meta`. Fix: swap to `--ink-muted`. → `/impeccable audit`
- **[P2]** Alarmist hero headline ("worse" in accent color) contradicts PRODUCT.md's "calm... never alarmist" personality. → `/impeccable clarify`
- **[P2]** Two incompatible toggle-switch skins (teal vs. near-black "on" state) on playground. → `/impeccable extract` then `/impeccable polish`
- **[P2]** Zero client-side validation on landing's scan submit despite playground already having the helpers. Direct fix.
- **[P2]** Validation errors surface detached from source field (dock note vs. offending URL row). → `/impeccable clarify` or `/impeccable harden`

## Persona Red Flags

- Jordan (landing): "Operation" kicker unclear; DevOps jargon assumed early.
- Riley (landing): garbage URL still navigates into playground with no validation.
- Alex (playground): no bulk arm/disarm or shortcuts on 8-channel rack.
- Sam (playground): broken heading outline + likely-failing contrast on `--ink-faint` microcopy.

## Minor Observations

- Hard-coded inline colors in `home.tsx` JSX bypass the token system.
- Header logo's ping animation runs continuously on the playground console — decorative motion not tied to state.
- `.pauth__label`/`.pai__label` duplicate the shared `.label` treatment instead of reusing it.

## Questions to Consider

1. What if the landing page committed to one genuinely arresting "moment" instead of an evenly-paced features-then-CTA scroll?
2. What if the repeated kicker were replaced by a live/rotating reading in the header's calibration-tick band?
3. What if the 8-channel rack defaulted to one "recommended" preset with one-click alternates instead of 8 raw toggles?
