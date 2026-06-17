---
version: "stageflow-print-room"
name: "StageFlow Design System"
identity: "Print Room"
description: >
  StageFlow's design language is "Print Room" (rebuilt 2026-06): a warm-paper,
  editorial, light-only aesthetic. Serif display headlines, monospace for every
  number/ID/status, hairline borders and data tables instead of icon-card grids,
  and near-zero shadows (elevation comes from borders, not blur). The single
  source of truth for every token is the Tailwind v4 `@theme` block in
  clients/web/src/app.css — values below mirror it for reference only.
token-source: "clients/web/src/app.css"
colors:
  paper: "#faf9f7"          # page background — warm paper, not cool slate
  surface: "#ffffff"        # card backgrounds
  surface-muted: "#f4f2ee"  # subtle inset areas, chart tracks
  paper-deep: "#e8e5df"     # deepest paper tone (decorative fills only)
  ink-strong: "#1a1714"     # h1/h2, strongest text
  ink: "#2d2a26"            # body text
  ink-muted: "#5c5751"      # secondary copy
  ink-faint: "#6f6961"      # faint copy — clears WCAG AA (4.9:1) on paper
  ink-ghost: "#877f74"      # decorative ghost numerals — clears AA large-text (3.75:1)
  line: "#ddd9d0"           # all hairline borders and dividers
  accent: "#1b5c5e"         # single deep-teal spot colour
  accent-hover: "#154a4c"
  accent-soft: "#e6efef"
  accent-ink: "#0e3638"
  accent-mist: "#f0f7f7"
typography:
  display:
    fontFamily: "Source Serif 4 Variable"
    usage: "h1, h2, .font-display only — editorial serif headlines"
    tracking: "-0.015em (h1), -0.01em (h2)"
  body:
    fontFamily: "Inter Variable"
    usage: "body text, h3 and below"
  mono:
    fontFamily: "JetBrains Mono Variable"
    usage: "every number, ID, status, step number, code, and stat"
severity:
  critical: "#ef4444"  # --color-severity-critical (red-500)
  serious: "#f97316"   # --color-severity-serious (orange-500)
  moderate: "#f59e0b"  # --color-severity-moderate (amber-500)
  minor: "#3b82f6"     # --color-severity-minor (blue-500)
  info: "#a855f7"      # --color-severity-info (purple-500)
  note: >
    These tokens are the canonical primary hue for severity in CSS contexts
    (severity bar segments, dots, chart tracks). SVG stroke/fill (donut, issue
    overlays) read SEVERITY_PRIMARY_RGB in src/lib/report/severity.ts, which
    mirrors these tokens (presentation attributes can't resolve CSS vars).
    Pastel backgrounds, borders, and -600 solid badges remain Tailwind-palette
    in severity.ts by design.
radius:
  note: "Rectilinear. All radii clamp to 8px max via --radius-* tokens (xl..4xl all = 8px)."
  xs: "2px"
  sm: "3px"
  md: "6px"
  lg: "8px"
shadows:
  note: "Near-zero. Elevation comes from borders, never blur. No hover shadow-lifts."
  xs: "0 1px 3px rgba(26,23,20,0.06)"
  sm: "0 1px 4px rgba(26,23,20,0.08)"
  md: "0 2px 8px rgba(26,23,20,0.08)"
  lg: "0 4px 16px rgba(26,23,20,0.1)"
guardrails:
  - "Light-only. No dark mode, no `dark:` utilities in app code (the scan terminal and automation panel are intentional dark islands; see below)."
  - "Warm paper + warm ink. No cool slate. Do not introduce slate-* utilities in app code."
  - "Source Serif 4 for h1/h2 display only; Inter for body/h3; JetBrains Mono for all numbers/IDs/statuses."
  - "Single deep-teal accent (#1b5c5e). One spot colour, used sparingly like print."
  - "Hairline borders + data tables. No fake browser chrome, no brand-coloured scanner icons, no icon-card grids."
  - "No hover shadow-lifts (border darkens via .hover-glow instead) and no entrance fades on form fields (axe races partial opacity)."
  - "Radius never exceeds 8px. --color-ink-faint must stay >=4.5:1 on paper; decorative large text uses --color-ink-ghost (>=3:1)."
  - "app.css is the only token source. Never redefine tokens elsewhere; reference them via Tailwind utilities or var(--color-*)."
---

# StageFlow Design System — "Print Room"

A warm-paper, editorial, light-only design language. Think a print shop's proof
sheet: serif headlines, monospace data, hairline rules, and ink on paper. The
**single source of truth for every token is the Tailwind v4 `@theme` block in
`clients/web/src/app.css`** — the values in this document mirror it for reference
and must be kept in sync with it, not the other way around.

## Palette

Surfaces are warm paper; text is warm near-black ink; borders are a single warm
hairline grey; the only chromatic accent is a deep teal used like a print spot
colour.

| Token | Value | Usage |
|---|---|---|
| `paper` | `#faf9f7` | Page background |
| `surface` | `#ffffff` | Card backgrounds |
| `surface-muted` | `#f4f2ee` | Inset areas, chart tracks |
| `ink-strong` | `#1a1714` | h1/h2, strongest text |
| `ink` | `#2d2a26` | Body text |
| `ink-muted` | `#5c5751` | Secondary copy |
| `ink-faint` | `#6f6961` | Faint copy (4.9:1 on paper) |
| `ink-ghost` | `#877f74` | Decorative ghost numerals (3.75:1 — AA large only) |
| `line` | `#ddd9d0` | All borders and dividers |
| `accent` | `#1b5c5e` | Deep-teal spot colour, links, primary actions |

## Typography

- **Display (h1, h2, `.font-display`)**: Source Serif 4 Variable — editorial serif headlines.
- **Body (and h3 down)**: Inter Variable.
- **Data (`.font-mono`, `code`, stats)**: JetBrains Mono Variable for every number, ID, status, and step number.

Helper classes live in `app.css` (`.h1-display`, `.h2-display`, `.h3-display`,
`.section-kicker`, `.section-tag`, `.stat-mono`).

## Elevation & shape

- **Radius**: rectilinear. `--radius-*` clamps everything to **8px max** (`xl`..`4xl` all equal 8px), so `rounded-lg`/`rounded-xl`/`rounded-2xl` all render identically.
- **Shadows**: near-zero. Elevation comes from **borders, not blur**. Interactive surfaces darken their border via `.hover-glow` — never a shadow-lift or `translateY`.

## Severity colours

Severity has a canonical primary hue per level, defined once as
`--color-severity-*` in `app.css`:

| Level | Token | Hue |
|---|---|---|
| critical | `--color-severity-critical` | `#ef4444` (red-500) |
| serious | `--color-severity-serious` | `#f97316` (orange-500) |
| moderate | `--color-severity-moderate` | `#f59e0b` (amber-500) |
| minor | `--color-severity-minor` | `#3b82f6` (blue-500) |
| info | `--color-severity-info` | `#a855f7` (purple-500) |

- **CSS contexts** (severity bar segments, dots, chart tracks) read the tokens directly via `var(--color-severity-*)`.
- **SVG contexts** (the donut, issue screenshot overlays) read `SEVERITY_PRIMARY_RGB` in `src/lib/report/severity.ts`, which mirrors the tokens — SVG presentation attributes can't resolve CSS custom properties.
- **Pastel backgrounds/borders and -600 solid badges** stay Tailwind-palette in `severity.ts` (`bg-red-50`, `bg-red-600`, …) by design; they are tints/shades of the same hues.

Keep the three in sync when changing a severity hue.

## Intentional exceptions

- **Dark islands**: `ScanTerminal.svelte` and the automation panel
  (`artifacts-sidebar/AutomationTab.svelte`) are deliberately dark, code-console
  surfaces and use `slate-*`/near-black backgrounds. This is the *only* place
  dark tones and `slate-*` are allowed.
- **`modal.ts` `dark` variant** names a darker backdrop opacity, not dark mode.
- **Status tones** (`Score.svelte`, `StatusPill.svelte`) use emerald/blue/amber/orange/red
  as a coherent semantic scale for score bands; the neutral fallback uses ink tokens.

## Layout principles

- **Container**: `max-w-6xl` (72rem) via `.container-width`, with responsive side padding.
- **Section rhythm**: `.section-padding` (`py-16 lg:py-24`).
- **Bento grid**: asymmetric splits are fine; keep gaps and padding consistent across cards.
- **Data tables and hairline rules** over icon-card grids.

## Verification gate

Primitives are gated by the Storybook axe / WCAG 2.1 AA test-runner:

```
cd clients/web && bun run test-storybook
```

Keep `--color-ink-faint` ≥ 4.5:1 and decorative large text (`--color-ink-ghost`)
≥ 3:1 on paper.
