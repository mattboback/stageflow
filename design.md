---
version: "stageflow-nexus-adapted"
name: "StageFlow Design System"
source-inspiration: "Nexus Analytics Dashboard by Sourasith Phomhome (@madebysourasith)"
description: >
  StageFlow's design system adapted from the Nexus Analytics Dashboard aesthetic.
  Clean information density, modular panels, and dashboard interface rhythm — applied
  to StageFlow's self-hosted scanning platform with the teal brand identity preserved.
colors:
  primary: "#0d5c63"
  primary-hover: "#094a50"
  primary-deep: "#052e32"
  primary-soft: "#e0f2f1"
  primary-mist: "#f0faf9"
  primary-subtle: "#b2dfdb"
  background: "#f8fafc"
  surface: "#ffffff"
  surface-muted: "#f1f5f9"
  surface-deep: "#e2e8f0"
  text-primary: "#0f172a"
  text-secondary: "#1e293b"
  text-muted: "#475569"
  text-faint: "#64748b"
  border: "#e2e8f0"
typography:
  display:
    fontFamily: "Inter Variable"
    fontWeight: "600-700"
    letterSpacing: "-0.02em"
    note: "Inter for all headings — clean, modern, high information density"
  body:
    fontFamily: "Inter Variable"
    fontWeight: "400-500"
    lineHeight: "1.6"
  label:
    fontFamily: "JetBrains Mono Variable"
    fontSize: "11-12px"
    fontWeight: "500-600"
    note: "Mono for stats, step numbers, code, and technical metadata"
shadows:
  xs: "0 2px 8px rgb(0, 0, 0, 0.04)"
  sm: "0 4px 16px rgb(0, 0, 0, 0.03)"
  md: "0 8px 30px rgb(0, 0, 0, 0.04)"
  card-hover: "0 8px 32px rgb(0, 0, 0, 0.08)"
  lg: "0 16px 40px rgb(0, 0, 0, 0.06)"
  float: "0 20px 48px rgb(0, 0, 0, 0.10)"
  note: "Nexus ultra-subtle shadows — nearly imperceptible at rest, perceptible on hover"
rounded:
  card: "1rem"
  card-lg: "1.5rem"
  control: "0.625rem"
  pill: "9999px"
spacing:
  base: "8px"
  gap: "16px"
  card-padding: "24px"
  section-padding: "80px (5rem)"
components:
  card:
    background: "surface (#ffffff)"
    border: "1px solid border (#e2e8f0)"
    shadow: "shadow-sm at rest, shadow-card-hover on hover"
    radius: "card radius (1rem) or card-lg (1.5rem) for hero panels"
    transition: "box-shadow + translateY(-2px) on hover"
  button-primary:
    background: "primary (#0d5c63)"
    text: "white"
    radius: "pill (9999px) or control (0.625rem)"
  badge:
    background: "surface-muted or primary/10"
    radius: "pill"
    font: "JetBrains Mono"
  section-kicker:
    color: "primary (#0d5c63)"
    font: "Inter, uppercase, letter-spacing 0.08em, text-xs"
    note: "No italic — clean dashboard aesthetic replaces editorial serif italic"
guardrails:
  - "Do not use warm tones (ochre, stone, sepia). Palette is cool slate."
  - "Shadow intensity must stay subtle — cards should appear to float, not cast."
  - "Inter is the sole typeface family. JetBrains Mono only for monospaced contexts."
  - "Teal (#0d5c63) remains the single brand accent. No indigo substitution."
  - "Card backgrounds are white (#ffffff) on slate frame (#f8fafc) — not reversed."
  - "Bento grid: asymmetric columns are preferred (7-col/5-col) over uniform grids."
---

# StageFlow Design System

Adapted from **Nexus Analytics Dashboard** by Sourasith Phomhome — a modern analytics
interface with clear information density, modular panels, and clean interface rhythm.

## Palette

| Token | Value | Usage |
|---|---|---|
| `primary` | `#0d5c63` | Teal brand accent, active states, links |
| `primary-deep` | `#052e32` | CTA dark backgrounds |
| `background` | `#f8fafc` | Page frame / shell |
| `surface` | `#ffffff` | Card backgrounds |
| `surface-muted` | `#f1f5f9` | Subtle inset areas |
| `text-primary` | `#0f172a` | Headings, strong text |
| `text-secondary` | `#1e293b` | Body text |
| `text-muted` | `#475569` | Secondary copy |
| `border` | `#e2e8f0` | All dividers and card borders |

## Typography

- **Display / Headings**: Inter Variable, weight 600–700, tracking −0.02em
- **Body**: Inter Variable, weight 400–500, line-height 1.6
- **Labels / Stats**: JetBrains Mono Variable, weight 500–600

## Shadows

Cards at rest should feel elevated but not heavy. Nexus-style ultra-subtle:

```
sm:         0 4px 16px rgb(0,0,0,0.03)
md:         0 8px 30px rgb(0,0,0,0.04)
card-hover: 0 8px 32px rgb(0,0,0,0.08)
float:      0 20px 48px rgb(0,0,0,0.10)
```

## Layout Principles

- **Bento grid**: Asymmetric column splits (7-col / 5-col) preferred over uniform grids
- **Section padding**: 80px (5rem) vertical; 24px card padding
- **Max-width**: 72rem (1152px) container with 2rem side padding
- **Stack order**: Hero with bento → Capabilities grid → Workflow → Why → CTA

## Component Notes

### Cards
White (`#ffffff`) on slate frame (`#f8fafc`). Subtle `1px solid #e2e8f0` border.
Hover: `translateY(-2px)` + increased shadow. Transition 200ms ease-out.

### Badges / Pills
Background: `primary/10` teal tint or `surface-muted`. Pill radius (`9999px`).
Monospace font for numeric badges (step numbers, counts).

### Section Kicker
Uppercase, tracked, `text-xs`, teal color. **No italic** (Inter is not a serif).
