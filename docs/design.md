---
name: StageFlow - Calibrated Instrument
description: Light-only, measurement-oriented interface language for a frontend quality platform.
colors:
  ground: "oklch(0.976 0.004 235)"
  ground-deep: "oklch(0.955 0.005 235)"
  surface: "oklch(1 0 0)"
  surface-sunk: "oklch(0.968 0.004 235)"
  ink-strong: "oklch(0.22 0.012 250)"
  ink: "oklch(0.31 0.012 250)"
  ink-muted: "oklch(0.46 0.013 250)"
  ink-faint: "oklch(0.56 0.012 250)"
  line: "oklch(0.90 0.005 240)"
  line-strong: "oklch(0.82 0.006 240)"
  tick: "oklch(0.86 0.005 240)"
  signal: "oklch(0.58 0.094 210)"
  signal-strong: "oklch(0.50 0.092 211)"
  signal-press: "oklch(0.44 0.085 212)"
  signal-ink: "oklch(0.34 0.07 213)"
  signal-wash: "oklch(0.96 0.018 210)"
  signal-edge: "oklch(0.86 0.045 208)"
  severity-critical: "oklch(0.52 0.20 27)"
  severity-serious: "oklch(0.57 0.16 47)"
  severity-moderate: "oklch(0.58 0.12 80)"
  severity-minor: "oklch(0.52 0.12 135)"
  severity-pass: "oklch(0.56 0.10 165)"
  severity-critical-wash: "oklch(0.96 0.03 27)"
  severity-serious-wash: "oklch(0.965 0.03 47)"
  severity-moderate-wash: "oklch(0.965 0.035 85)"
  severity-minor-wash: "oklch(0.965 0.03 135)"
typography:
  sans: "Archivo, system-ui, -apple-system, Segoe UI, sans-serif"
  mono: "JetBrains Mono, ui-monospace, SFMono-Regular, Menlo, monospace"
  body:
    fontFamily: "Archivo, system-ui, -apple-system, Segoe UI, sans-serif"
    fontSize: "16px"
    fontWeight: 400
    lineHeight: 1.55
  label:
    fontFamily: "JetBrains Mono, ui-monospace, SFMono-Regular, Menlo, monospace"
    fontSize: "0.6875rem"
    fontWeight: 500
    letterSpacing: "0.09em"
  readout:
    fontFamily: "JetBrains Mono, ui-monospace, SFMono-Regular, Menlo, monospace"
    fontSize: "1.6rem"
    fontWeight: 700
    lineHeight: 1
    letterSpacing: "-0.01em"
rounded:
  xs: "2px"
  sm: "4px"
  md: "6px"
  lg: "8px"
  pill: "999px"
spacing:
  wrap-inline: "1.75rem"
  maxw: "76rem"
  panel-head: "0.85rem 1.1rem"
  panel-body: "1.1rem"
components:
  button-primary:
    backgroundColor: "{colors.signal-strong}"
    textColor: "#ffffff"
    rounded: "{rounded.md}"
    padding: "0.7rem 1.1rem"
  button-primary-hover:
    backgroundColor: "{colors.signal-press}"
  button-ghost:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.ink-strong}"
    rounded: "{rounded.md}"
    padding: "0.7rem 1.1rem"
  button-sm:
    padding: "0.5rem 0.8rem"
  input:
    backgroundColor: "{colors.surface-sunk}"
    textColor: "{colors.ink-strong}"
    rounded: "{rounded.md}"
    padding: "0.7rem 0.85rem"
  panel:
    backgroundColor: "{colors.surface}"
    rounded: "{rounded.lg}"
  pill:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.ink-muted}"
    rounded: "{rounded.pill}"
    padding: "0.28rem 0.55rem"
  severity-badge:
    textColor: "{colors.surface}"
    rounded: "{rounded.pill}"
    padding: "0.16rem 0.5rem"
  severity-chip:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.ink-muted}"
    rounded: "{rounded.pill}"
    padding: "0.22rem 0.55rem"
---

# Design System: StageFlow - Calibrated Instrument

Canonical tokens and shared primitives:
`clients/web/app/styles/instrument.css`. Report severity surfaces:
`clients/web/app/styles/report.css` (selected via
`clients/web/app/lib/report/severity.ts`). Update CSS first; update this guide
when the design language changes.

## 1. Overview

**Creative North Star: "Calibrated Instrument"**

StageFlow is a precision measuring instrument for frontend quality: exact, quiet,
and evidence-heavy. The interface favors linework, mono readouts, compact controls,
and stable report surfaces over marketing decoration. Density is operational —
built for scanning scores and issues — without becoming a dark developer console.

The product is its own proof point. A first-time visitor should run a scan, read
live progress, and inspect a report without feeling sold to or scared. Calm under
criticality: severity is a calibrated scale, never a siren. Personality from
PRODUCT.md: **calm · trustworthy · understated**.

Explicitly rejects: SaaS-cream dashboard templates and gradient-accented
hero-metric cards; identical icon-card grids; alarmist "THREATS DETECTED"
urgency, gamified scores, and confetti; fake browser chrome; cool-slate dark
dashboards; heavy marketing-page reveal choreography.

**Key Characteristics:**

- Light-only cool graphite/white surfaces with one teal-cyan signal color
- Archivo for prose and UI; JetBrains Mono for IDs, scores, status, timings, counts
- Borders and tonal steps for structure; shadows only for real overlays
- Severity always color plus text, count, shape, or position
- Rounded corners small and consistent: 2px, 4px, 6px, or 8px (pills 999px)
- Restrained and exact controls — signal rare and meaningful
- Semantic z-index scale (sticky → dropdown → backdrop → modal → toast → tooltip)

## 2. Colors

Measurement panel, not a themed dashboard. **Restrained** strategy: cool tinted
neutrals plus one accent used sparingly for primary action, live activity, and links.

### Primary

- **Signal Teal** (`oklch(0.58 0.094 210)` / `--signal`): live activity, links,
  gauge arcs, brand mark, current-state emphasis
- **Signal Strong** (`oklch(0.50 0.092 211)` / `--signal-strong`): primary button
  fill (white text); hover **Signal Press** (`oklch(0.44 0.085 212)`)
- **Signal Ink / Wash / Edge**: text-on-light signal, faint fills, tinted borders
  for live pills and focus rings

### Neutral

- **Ground** (`oklch(0.976 0.004 235)`): page background — cool near-white graphite,
  never warm cream
- **Ground Deep** (`oklch(0.955 0.005 235)`): footer, hover wells
- **Surface** (`oklch(1 0 0)`): panels, cards, default controls
- **Surface Sunk** (`oklch(0.968 0.004 235)`): inputs, inset wells, meter tracks
- **Ink Strong / Ink / Ink Muted / Ink Faint**: text hierarchy; body and muted
  target AA (prefer AAA where practical); faint ink is not body prose
- **Line / Line Strong / Tick**: hairline structure and calibration marks

### Severity (calibrated scale)

| Role | Token | Value |
|------|--------|--------|
| Critical | `--sev-critical` | `oklch(0.52 0.20 27)` |
| Serious | `--sev-serious` | `oklch(0.57 0.16 47)` |
| Moderate | `--sev-moderate` | `oklch(0.58 0.12 80)` |
| Minor | `--sev-minor` | `oklch(0.52 0.12 135)` |
| Pass / info | `--sev-pass` | `oklch(0.56 0.10 165)` |

Wash variants (`--sev-*-wash`) back issue containers. Info maps onto pass in
`report.css`. Overlay markers use `color-mix` 10% fills with 2px severity borders.

**The One Signal Rule.** Signal teal appears on ≤10% of any screen — primary
actions, live state, links, and brand mark. Never as decorative wash across large
regions.

**The Severity Pairing Rule.** Never rely on severity hue alone. Pair every
severity color with label, count, badge text, grouped position, or overlay geometry.

## 3. Typography

**Display/Body Font:** Archivo (system-ui, -apple-system, Segoe UI, sans-serif)
**Label/Mono Font:** JetBrains Mono (ui-monospace, SFMono-Regular, Menlo, monospace)

**Character:** Archivo carries calm UI prose; mono is a co-primary data voice for
scores, IDs, job states, scanner names, timings, and counts — the instrument's
readout face. Body sets `font-feature-settings: 'cv05' 1, 'ss01' 1`; mono readouts
enable `'zero' 1`.

### Hierarchy

- **Body** (400, 16px, line-height 1.55): default page text in Archivo
- **UI / controls** (600 on buttons, ~0.9rem): compact tool surfaces; steady type,
  not fluid clamp scales on product chrome
- **Label** (`.label`: mono, 0.6875rem / 11px, weight 500, letter-spacing 0.09em,
  uppercase): field annotations and instrument captions
- **Readout** (`.readout__val`: mono, weight 700, 1.6rem, letter-spacing -0.01em):
  quantitative values
- **Gauge caption** (mono, 0.625rem, letter-spacing 0.12em, uppercase)
- **Eyetag** (mono, 0.7rem, letter-spacing 0.14em, uppercase, signal-ink): rare
  section annotation — not an eyebrow on every section
- **Hero / report titles**: larger type only on landing and major report headings;
  panels, filters, and rows stay dense and steady

### Named Rules

**The Mono-for-Truth Rule.** Quantitative truth — scores, IDs, timestamps,
severities, command-like labels — uses JetBrains Mono. Prose stays Archivo.

**The No Decorative Display Rule.** No serif display faces, no alternate display
fonts, no aggressive negative letter-spacing on compact tool surfaces (readout
allows −0.01em only).

## 4. Elevation

Depth is structural, not theatrical. Surfaces are flat at rest. Structure comes
from full borders (`--line` / `--line-strong`), tonal steps (`--surface` vs
`--surface-sunk` vs `--ground`), and calibration ticks (`.rule-ticks`). Shadows
appear only when something truly floats above the page.

### Shadow Vocabulary

- **sm** (`0 1px 2px oklch(0.22 0.01 250 / 0.06)`): subtle edge on small overlays
- **md** (`0 2px 10px oklch(0.22 0.01 250 / 0.07)`): sticky bars / modest popovers
- **pop** (`0 8px 30px oklch(0.22 0.01 250 / 0.12)`): modals and elevated dialogs

Sticky header uses frosted ground (`backdrop-filter: saturate(1.4) blur(8px)`)
plus a bottom hairline — purposeful stacking, not decorative glass cards.

### z-index scale

`--z-base` 0 → `--z-sticky` 100 → `--z-dropdown` 200 → `--z-backdrop` 300 →
`--z-modal` 400 → `--z-toast` 500 → `--z-tooltip` 600. Never arbitrary 999 values.

**The Flat-By-Default Rule.** Surfaces are flat at rest. Shadows respond to
stacking (sticky, popover, modal, tooltip), never decoration on static cards.

## 5. Components

Components are restrained and exact: near-flat, compact, signal reserved for
primary and live states. Shared primitives live in `instrument.css`; report
severity surfaces in `report.css`.

### Buttons

- **Shape:** gently rectilinear (`6px` / `--r-md`)
- **Primary:** `--signal-strong` fill, white text, padding `0.7rem 1.1rem`; hover
  `--signal-press`; active presses `translateY(1px)`
- **Ghost:** surface fill, strong line border, ink-strong text; hover deepens
  border and ground-deep fill
- **sm:** tighter padding (`0.5rem 0.8rem`, `0.82rem` type) for header actions
- **Focus:** global `:focus-visible` — `2.5px solid --signal`, offset 2px
- Optional arrow (`.ar`) nudges `translateX(3px)` on hover

### Status pills

- Mono uppercase, LED dot, pill radius `999px`
- Variants: default, live (signal wash + blink), done (minor/pass), queued, error
- State language, not decoration

### Cards / Panels

- **Corner:** `8px` (`--r-lg`)
- **Background:** `--surface` with `1px solid --line`
- **Shadow:** none at rest
- **Internal padding:** head `0.85rem 1.1rem`; body `1.1rem`
- Cards only for repeated items, report panels, modals, tool surfaces — never
  nested cards

### Inputs / Fields

- **Style:** `--surface-sunk` fill, `--line-strong` border, `--r-md`, padding
  `0.7rem 0.85rem`, `0.95rem` type
- **Placeholder:** `--ink-faint` (must remain readable; never pure decorative gray)
- **Focus:** signal border + `0 0 0 3px --signal-wash`, surface background
- Label via `.label` mono annotation voice

### Navigation

- Sticky header (60px), frosted ground, brand mono wordmark + signal mark
- Nav links: muted ink, soft hover wash; collapse text links under 640px
- Primary "Run a scan" remains visible; calibration tick-rule sits under the bar

### Severity surfaces (report)

- **Container:** full border + wash fill (no left-stripe accents)
- **Dot:** ~8px circle for row headers / counts
- **Badge:** solid mono uppercase pill with severity fill + surface text
- **Border:** highlighted selection border + wash
- **Overlay:** 2px severity border on screenshot markers, 10% fill via `color-mix`
- **Filter chips:** toggleable mono pills; inactive shows severity-tinted border/text;
  active uses solid severity fill + surface text
- Helpers: `getSeverity*Class` in `app/lib/report/severity.ts`

### Signature: Score gauge & severity scale

- Arc gauge (132px dial): track `--line`, arc `--signal`, mono center value
- Severity scale: compact rows with square 9px dots (not color alone) + mono counts

### Motion (folded here)

- Durations: 120ms / 190ms / 280ms; ease-out-quint-ish (`cubic-bezier(0.22, 1, 0.36, 1)`)
- State and feedback only — no orchestrated page-load reveals
- Brand mark ping and live LED blink are the rare continuous cues; both honor
  `prefers-reduced-motion: reduce` (global collapse of animation/transition)

## 6. Do's and Don'ts

### Do:

- **Do** use linework, ticks, score arcs, and tables to reinforce measurement.
- **Do** keep the signal color rare and meaningful (The One Signal Rule).
- **Do** pair every severity hue with label, icon, count, or position.
- **Do** preserve contrast floors (AA hard floor; AAA on body/UI where practical)
  and visible keyboard focus (`2.5px` signal outline).
- **Do** keep report evidence next to the issue it explains.
- **Do** use the semantic z-index scale for stacking contexts.
- **Do** update `instrument.css` / `report.css` first; this file second.
- **Do** test UI changes by rendering the relevant route, not just reading CSS.

### Don't:

- **Don't** ship generic SaaS-cream dashboard templates or gradient-accented
  hero-metric cards.
- **Don't** use identical icon-card grids ("eight feature cards: icon + heading +
  blurb").
- **Don't** use alarmist security-scanner aesthetics — red-everywhere, scare-tactic
  "THREATS DETECTED" urgency, gamified scores, or confetti.
- **Don't** add fake browser chrome or brand-colored scanner logos as decoration.
- **Don't** turn the interface into a cool-slate dark developer console; dark
  surfaces are reserved for terminal-like output only.
- **Don't** add heavy marketing-page reveal choreography; the tool should feel
  instant and honest.
- **Don't** use side-stripe accent borders (`border-left` / `border-right` > 1px
  as colored accent), gradient text, or glassmorphism as default card treatment.
- **Don't** invent z-index values like 999 / 9999.
- **Don't** redefine design tokens outside `clients/web/app/styles/instrument.css`
  without a clear reason.
