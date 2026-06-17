# StageFlow — Agent Guide

## Design Context

StageFlow's design work is governed by two root docs. Read them before touching UI:

- **[PRODUCT.md](PRODUCT.md)** — strategic "who/what/why". Register: **`product`**. Primary
  user: **first-time visitors** running their first scan. Brand personality:
  **calm · trustworthy · understated**.
- **[DESIGN.md](DESIGN.md)** — the visual system ("Print Room"). Sidecar machine data lives
  in `.impeccable/design.json`.

**The canonical token source is `clients/web/src/app.css`** (the Tailwind v4 `@theme` block).
`DESIGN.md` and `docs/design-system.md` mirror it for reference — never redefine tokens
elsewhere. `docs/design-system.md` is the long-form prose reference.

Load-bearing guardrails (full rules in DESIGN.md):

- **Light-only, warm paper + warm ink.** No `slate-*`, no cool grey, no dark mode. The scan
  terminal and automation panel are the only sanctioned dark islands.
- **One deep-teal spot colour** (`#1b5c5e`), ≤10% of any view — links, primary actions, one
  emphasis per region.
- **Serif (Source Serif 4) for h1/h2 only**; Inter for body/h3; **JetBrains Mono for every
  number, ID, status, score, and step** (`tabular-nums`).
- **Hairline borders + data tables** over cards and icon-card grids. **Elevation from borders,
  not shadows** (hover darkens the border; never a shadow-lift or `translateY`).
- **Radius ≤ 8px.** Severity is **never colour alone** — always paired with label/icon/shape.
- WCAG 2.1 AA, gated by `cd clients/web && bun run test-storybook`.
  `--color-ink-faint` ≥ 4.5:1; decorative `--color-ink-ghost` ≥ 3:1 on paper.

Iterate visually with `/impeccable live` (dev server on `:5173`). Other impeccable commands
(`critique`, `audit`, `polish`, …) read PRODUCT.md + DESIGN.md automatically.
