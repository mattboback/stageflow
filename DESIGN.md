---
name: StageFlow — Print Room
description: Warm-paper, editorial, light-only design language for a frontend quality platform.
colors:
  paper: "#faf9f7"
  surface: "#ffffff"
  surface-muted: "#f4f2ee"
  paper-deep: "#e8e5df"
  ink-strong: "#1a1714"
  ink: "#2d2a26"
  ink-muted: "#5c5751"
  ink-faint: "#6f6961"
  ink-ghost: "#877f74"
  line: "#ddd9d0"
  accent: "#1b5c5e"
  accent-hover: "#154a4c"
  accent-soft: "#e6efef"
  accent-mist: "#f0f7f7"
  accent-subtle: "#b2cecd"
  accent-ink: "#0e3638"
  accent-deep: "#0a2526"
  severity-critical: "#ef4444"
  severity-serious: "#f97316"
  severity-moderate: "#f59e0b"
  severity-minor: "#3b82f6"
  severity-info: "#a855f7"
typography:
  display:
    fontFamily: "Source Serif 4 Variable, Georgia, 'Times New Roman', serif"
    fontSize: "clamp(2.25rem, 5vw, 3.5rem)"
    fontWeight: 700
    lineHeight: 1.08
    letterSpacing: "-0.015em"
  headline:
    fontFamily: "Source Serif 4 Variable, Georgia, 'Times New Roman', serif"
    fontSize: "clamp(1.75rem, 4vw, 2.5rem)"
    fontWeight: 600
    lineHeight: 1.15
    letterSpacing: "-0.01em"
  title:
    fontFamily: "Inter Variable, system-ui, sans-serif"
    fontSize: "clamp(1.25rem, 2vw, 1.5rem)"
    fontWeight: 600
    lineHeight: 1.3
    letterSpacing: "-0.005em"
  body:
    fontFamily: "Inter Variable, system-ui, sans-serif"
    fontSize: "1rem"
    fontWeight: 400
    lineHeight: 1.6
    letterSpacing: "normal"
  mono:
    fontFamily: "JetBrains Mono Variable, ui-monospace, SFMono-Regular, Menlo, monospace"
    fontSize: "0.875rem"
    fontWeight: 500
    lineHeight: 1.4
    letterSpacing: "normal"
    fontFeature: "tnum"
rounded:
  xs: "2px"
  sm: "3px"
  md: "6px"
  lg: "8px"
components:
  button-primary:
    backgroundColor: "{colors.accent}"
    textColor: "#ffffff"
    typography: "{typography.body}"
    rounded: "{rounded.md}"
    padding: "8px 16px"
    height: "36px"
  button-primary-hover:
    backgroundColor: "{colors.accent-hover}"
    textColor: "#ffffff"
  button-outline:
    backgroundColor: "transparent"
    textColor: "{colors.ink}"
    rounded: "{rounded.md}"
    padding: "8px 16px"
    height: "36px"
  button-secondary:
    backgroundColor: "{colors.surface-muted}"
    textColor: "{colors.ink}"
    rounded: "{rounded.md}"
    padding: "8px 16px"
    height: "36px"
  input:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.ink}"
    rounded: "{rounded.md}"
    padding: "8px 12px"
    height: "40px"
  panel:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.ink}"
    rounded: "{rounded.md}"
    padding: "24px"
  chip:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.ink-muted}"
    rounded: "9999px"
    padding: "4px 12px"
  chip-active:
    backgroundColor: "{colors.ink}"
    textColor: "{colors.surface}"
---

# Design System: StageFlow — Print Room

> The machine-readable tokens above mirror the project's canonical source: the
> Tailwind v4 `@theme` block in `clients/web/src/app.css`. **`app.css` is normative.**
> `docs/design-system.md` is the long-form prose reference. When a value changes,
> change it in `app.css` first, then update both docs.

## 1. Overview

**Creative North Star: "The Proof Sheet"**

StageFlow looks like a print shop's proof sheet: serif headlines set in warm ink,
monospace data ruled into tables, hairline borders instead of boxes, and the calm of
paper rather than the glow of a screen. It is a tool that *reports* on a frontend —
accessibility, performance, SEO, security — so the interface itself behaves like a
trustworthy document: exact, legible, and never raising its voice. A first-time visitor
should feel they are reading a serious, honest assessment, not being sold a product.

Depth comes from **borders, not shadows**; emphasis from **weight and a single spot
colour**, not from saturation or motion. The one chromatic note is a deep teal, used
the way a printer drops a single Pantone spot onto an otherwise black-on-cream page —
rarely, and to mean something. Numbers, IDs, severities, and step counts are always
monospace, so quantitative truth is visually distinct from prose.

This system explicitly rejects the generic SaaS-dashboard look: no cream-tinted
template gradients, no icon-card grids, no alarmist red-everywhere security aesthetic,
no cool-slate "developer tool" dark mode, no fake browser chrome. Warmth lives in the
paper and the ink, not in decoration.

**Key Characteristics:**
- Warm paper surfaces (`#faf9f7`), warm near-black ink (`#2d2a26`), one deep-teal spot.
- Editorial serif display (Source Serif 4) + humanist sans body (Inter) + mono data (JetBrains Mono).
- Hairline borders and data tables over cards and icon grids.
- Rectilinear (radius ≤ 8px), near-flat (elevation from borders), light-only.
- Severity is always hue **plus** label/shape — never colour alone.

## 2. Colors

A warm paper-and-ink palette: surfaces are warm paper, text is warm near-black, borders
are a single warm hairline grey, and the only chromatic accent is a deep teal spot colour.

### Primary
- **Deep Teal Spot** (`#1b5c5e`): the single chromatic accent. Links, primary buttons,
  focus rings, the kicker label, active data marks. Used sparingly, like a print spot
  colour — its rarity is the point. Hover deepens to `#154a4c`; `accent-ink` (`#0e3638`)
  and `accent-deep` (`#0a2526`) are for text-on-tint and the darkest teal fills;
  `accent-soft` (`#e6efef`), `accent-mist` (`#f0f7f7`), and `accent-subtle` (`#b2cecd`)
  are the tints for selection, soft backgrounds, and quiet borders.

### Neutral
- **Warm Paper** (`#faf9f7`): the page background. Warm, never cool slate.
- **Surface** (`#ffffff`): card and panel backgrounds, lifted one notch off paper.
- **Surface Muted** (`#f4f2ee`): inset areas, chart tracks, secondary button fill.
- **Paper Deep** (`#e8e5df`): the deepest paper tone, for decorative fills only.
- **Warm Hairline** (`#ddd9d0`): every border and divider in the system.
- **Ink Strong** (`#1a1714`): h1/h2 and the strongest text.
- **Ink** (`#2d2a26`): body text.
- **Ink Muted** (`#5c5751`): secondary copy.
- **Ink Faint** (`#6f6961`): faint copy and placeholders — holds 4.9:1 on paper.
- **Ink Ghost** (`#877f74`): decorative ghost numerals only — AA large-text (3:1) only.

### Tertiary (semantic severity scale)
A five-step severity scale, shared with the report. These are the canonical primary hue
per level (Tailwind `-500`); pastel `-50` backgrounds and `-600` solid badges are tints
and shades of the same hues, defined in `src/lib/report/severity.ts`.
- **Critical** (`#ef4444`), **Serious** (`#f97316`), **Moderate** (`#f59e0b`),
  **Minor** (`#3b82f6`), **Info** (`#a855f7`).

### Named Rules
**The One Spot Rule.** The deep teal is a spot colour, not a theme colour. It carries
links, primary actions, and a single emphasis per region — never a fill that dominates a
screen. If teal covers more than ~10% of a view, it has stopped meaning "here."

**The Severity-Is-Never-Colour-Alone Rule.** Every severity hue is paired with a text
label, an icon, or a shape. A colour-blind reader must never have to distinguish critical
from minor by hue.

## 3. Typography

**Display Font:** Source Serif 4 Variable (with Georgia, Times New Roman, serif)
**Body Font:** Inter Variable (with system-ui, sans-serif)
**Label/Mono Font:** JetBrains Mono Variable (with ui-monospace, monospace)

**Character:** An editorial serif against a neutral humanist sans, with a monospace face
reserved for data. The serif gives headlines the authority of print; Inter keeps prose
quiet and readable; the mono makes every number, ID, status, and step count read as a
fact rather than a sentence.

### Hierarchy
- **Display** (Source Serif 4, 700, clamp 2.25→3.5rem, line-height 1.08, tracking -0.015em):
  h1 and `.h1-display`. Page and report titles. `text-balance` on. Editorial only.
- **Headline** (Source Serif 4, 600, clamp 1.75→2.5rem, line-height 1.15, tracking -0.01em):
  h2 and `.h2-display`. Section headings.
- **Title** (Inter, 600, clamp 1.25→1.5rem, line-height 1.3, tracking -0.005em):
  h3 and below. Inter takes over here — serif stops at h2.
- **Body** (Inter, 400, 1rem, line-height 1.6): prose. Cap measure at 65–75ch.
- **Mono / Label** (JetBrains Mono, 500, 0.875rem, tabular-nums): every number, ID,
  status, score, and step number. The `.section-kicker` (Inter, 11px, tracking 0.1em,
  uppercase, teal) and `.section-tag` (mono, 11px, tracking 0.06em, uppercase) are the
  two small-label patterns.

### Named Rules
**The Mono-for-Data Rule.** Numbers, IDs, statuses, scores, and step counts are always
JetBrains Mono with `tabular-nums`. Quantitative truth must be visually distinct from prose.

**The Serif Ceiling Rule.** Source Serif 4 is for h1/h2 (and `.font-display`) only. h3 and
everything below is Inter. Never set body or UI labels in the serif.

## 4. Elevation

Near-flat by doctrine. **Elevation comes from borders, not blur.** Surfaces sit on warm
paper, separated by hairline `#ddd9d0` borders and tonal steps (paper → surface →
surface-muted), not by drop shadows. A small shadow vocabulary exists for genuinely
floating UI (skip link, sticky mobile bar, the rare overlay), but it is deliberately faint
and tinted with warm ink, never neutral black.

### Shadow Vocabulary (use sparingly)
- **xs** (`box-shadow: 0 1px 3px rgba(26,23,20,0.06)`): barely-there separation.
- **sm / md** (`0 1px 4px` / `0 2px 8px rgba(26,23,20,0.08)`): floating controls, popovers.
- **lg / xl** (`0 4px 16px` / `0 8px 24px rgba(26,23,20,0.1–0.12)`): modals, skip link.

### Named Rules
**The Border-Not-Blur Rule.** Interactive surfaces signal hover by **darkening their
border** (`.hover-glow` → `border-ink/20–25`), never by lifting with a shadow or
`translateY`. If a hover effect adds a shadow, it is wrong.

## 5. Components

Components are built with Tailwind utilities via `class-variance-authority`; the canonical
variant maps live in `clients/web/src/lib/components/ui/*.ts`. Radius defaults to `md`
(6px); the radius scale clamps everything to **8px max** (`rounded-lg` through
`rounded-3xl` all render at 8px).

### Buttons
- **Shape:** `rounded-md` (6px). Default size `h-9` (36px), `px-4 py-2`; `sm` = `h-8 px-3`,
  `lg` = `h-11 px-6`, `icon` = `h-9 w-9`.
- **Primary (default):** `bg-accent` teal, white text. Hover deepens to `bg-accent-hover`.
- **Outline:** transparent fill, `border-line`, ink text; hover `bg-surface-muted`.
- **Secondary:** `bg-surface-muted`, ink text; hover `bg-line`.
- **Ghost:** no fill, `text-ink-muted`; hover `bg-surface-muted` + `text-ink`.
- **Destructive:** `bg-red-600` (the one place red is a fill, for irreversible actions).
- **Focus:** `ring-2 ring-accent ring-offset-2 ring-offset-paper`. Transition is
  `colors` only — no transform.

### Chips
- **Style:** `rounded-full`, `border`, `font-semibold`. Default: `border-line bg-surface
  text-ink-muted`. Sizes `xs`/`sm`/`md` from `px-2 py-0.5` to `px-3 py-1`.
- **State:** active = `border-ink bg-ink text-surface` (ink fill, not teal — teal stays the
  spot colour). Semantic tones (`success`/`warning`/`danger`) use emerald/amber/red `-50`
  tints with matching `-200` borders and `-700` text.

### Cards / Containers (Panel)
- **Corner Style:** `rounded-md` (6px).
- **Background:** `bg-surface` (`#ffffff`); `muted` variant uses `bg-surface-muted`.
- **Border:** always `border-line` — the primary means of separation.
- **Shadow Strategy:** none at rest (see Elevation). Interactive panels darken the border
  via `.hover-glow` (`hover:border-ink/20`).
- **Internal Padding:** default `lg` (`p-6` / 24px); scale `xs`–`xl` = `p-2`–`p-8`.

### Inputs / Fields
- **Style:** `h-10`, `bg-surface`, `border-line`, `rounded-md`, `px-3 py-2`, `text-sm`.
  Placeholder is `text-ink-faint` (4.9:1, not a washed-out grey).
- **Focus:** `border-accent` + `ring-2 ring-accent ring-offset-2 ring-offset-paper`.
- **Error:** `border-red-500` + `ring-red-500`, with `aria-invalid`.

### Navigation
- **Style:** quiet text links (`.nav-link`: `text-ink-muted`, hover `text-ink`) and pill
  links (`.nav-link-pill`: `rounded-md px-3.5 py-2 text-[13px]`). No underlines at rest;
  colour shift on hover. Active state carries the teal.

### Step Number (signature)
The form-step marker (`.form-step-num`): a `h-6 w-6` mono badge, `border-line bg-surface
text-ink-faint`, `rounded-md`, `text-[11px]` semibold, `tabular-nums`. It anchors the
numbered scan-configuration flow and is the clearest expression of the Mono-for-Data rule.

## 6. Do's and Don'ts

### Do:
- **Do** keep surfaces warm: paper `#faf9f7`, ink `#2d2a26`. Carry warmth in paper and ink,
  not in decoration.
- **Do** separate with hairline `#ddd9d0` borders and tonal steps; reach for a data table
  before an icon-card grid.
- **Do** set every number, ID, status, score, and step in JetBrains Mono with `tabular-nums`.
- **Do** keep the deep teal a spot colour (≤10% of any view): links, primary actions, one
  emphasis per region.
- **Do** pair every severity hue with a label, icon, or shape (colour-blind safe).
- **Do** signal hover by darkening the border (`.hover-glow`), and give every animation a
  `prefers-reduced-motion` alternative.
- **Do** keep `--color-ink-faint` ≥ 4.5:1 and decorative `--color-ink-ghost` ≥ 3:1 on paper.

### Don't:
- **Don't** introduce a cream/sand SaaS-template background, gradient hero-metric cards, or
  identical icon-card grids.
- **Don't** use an alarmist security aesthetic — no red-everywhere, no "THREATS DETECTED"
  urgency, no gamified scores or confetti.
- **Don't** add `slate-*` or any cool-grey, and **don't** ship a dark mode. The system is
  light-only; the scan terminal and automation panel are the *only* sanctioned dark islands.
- **Don't** lift surfaces with drop shadows or `translateY` on hover; elevation is borders.
- **Don't** exceed 8px radius, set body/labels in the serif, or let teal become a fill that
  dominates a screen.
- **Don't** add fake browser chrome or brand-coloured scanner logos as decoration.
- **Don't** redefine tokens anywhere but `clients/web/src/app.css`; reference them via
  Tailwind utilities or `var(--color-*)`.
