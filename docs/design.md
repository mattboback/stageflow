---
name: StageFlow - Calibrated Instrument
description: Light-only, measurement-oriented interface language for a frontend quality platform.
colors:
  ground: "oklch(0.976 0.004 235)"
  surface: "oklch(1 0 0)"
  surface-sunk: "oklch(0.968 0.004 235)"
  ink-strong: "oklch(0.22 0.012 250)"
  ink: "oklch(0.31 0.012 250)"
  ink-muted: "oklch(0.46 0.013 250)"
  line: "oklch(0.90 0.005 240)"
  signal: "oklch(0.58 0.094 210)"
  signal-strong: "oklch(0.50 0.092 211)"
  severity-critical: "oklch(0.52 0.20 27)"
  severity-serious: "oklch(0.57 0.16 47)"
  severity-moderate: "oklch(0.58 0.12 80)"
  severity-minor: "oklch(0.52 0.12 135)"
  severity-pass: "oklch(0.56 0.10 165)"
typography:
  sans: "Archivo, system-ui, -apple-system, Segoe UI, sans-serif"
  mono: "JetBrains Mono, ui-monospace, SFMono-Regular, Menlo, monospace"
rounded:
  xs: "2px"
  sm: "4px"
  md: "6px"
  lg: "8px"
---

# Design System: StageFlow - Calibrated Instrument

The canonical tokens and component primitives live in
`clients/web/app/styles/instrument.css`, with report-specific severity classes in
`clients/web/app/styles/report.css`. Update those CSS files first, then update
this guide when the design language changes.

## 1. Overview

StageFlow should feel like a calibrated measurement instrument for frontend
quality: exact, quiet, and evidence-heavy. The interface favors linework,
readouts, compact controls, and stable report surfaces over marketing decoration.

The product is its own proof point. A first-time visitor should be able to run a
scan, read live progress, and inspect a report without feeling sold to or scared
by exaggerated security-scanner aesthetics.

Key characteristics:

- Light-only graphite/white surfaces with one teal-cyan signal color.
- Archivo for prose and UI; JetBrains Mono for IDs, scores, status, timings, and counts.
- Borders and tonal steps for structure; shadows only for real overlays.
- Severity is always color plus text, count, shape, or position.
- Rounded corners are small and consistent: 2px, 4px, 6px, or 8px.

## 2. Color

The color model is a measurement panel, not a themed dashboard.

- **Ground** (`--ground`) is a cool near-white page background.
- **Surface** (`--surface`) is the default panel and card background.
- **Surface sunk** (`--surface-sunk`) is for inputs, wells, meter tracks, and quiet inset areas.
- **Ink strong / ink / ink muted** hold the text hierarchy.
- **Line / line strong / tick** draw the interface grid and calibration marks.
- **Signal** (`--signal`) is the only accent: active state, primary action, live activity, and links.

The severity scale is calibrated and restrained:

- `critical`: red
- `serious`: orange
- `moderate`: amber
- `minor`: green
- `info/pass`: green-teal pass tone

Never rely on severity hue alone. Pair it with the severity label, issue count,
badge text, grouped position, or screenshot overlay geometry.

## 3. Typography

- **Archivo** is the primary typeface for headings, prose, navigation, labels, and controls.
- **JetBrains Mono** is the data voice for scores, IDs, job states, scanner names, timings, counts, and compact labels.

Typography should be dense enough for repeated operational use. Hero-scale text
is reserved for the landing page and major report titles; panels, filters, rows,
and controls use smaller, steady type.

Rules:

- Use mono for quantitative truth: scores, IDs, timestamps, severities, and command-like labels.
- Keep body copy plain and direct.
- Do not introduce decorative serif text or alternate display faces.
- Do not use negative letter spacing in compact tool surfaces.

## 4. Layout And Surfaces

StageFlow separates information with rules, grids, and stable dimensions.

- Prefer full-width bands and unframed layouts for page structure.
- Use cards only for individual repeated items, report panels, modals, and tool surfaces.
- Do not nest cards inside cards.
- Keep control heights and grid dimensions stable so live status, counts, and hover states do not shift layout.
- Use small shadows only for sticky bars, popovers, modals, and tooltips.

## 5. Components

Buttons:

- Primary actions use `--signal-strong` with white text.
- Secondary and outline actions stay near-flat, using `--surface`, `--surface-sunk`, and `--line`.
- Icon buttons should use familiar icons where available and include accessible labels.

Inputs:

- Inputs use `--surface-sunk`, `--line`, and a clear `--signal` focus outline.
- Placeholder and helper text must remain readable; do not wash it out for decoration.

Report UI:

- Severity classes are centralized in `clients/web/app/styles/report.css` and
  selected through helpers in `clients/web/app/lib/report/severity.ts`.
- Screenshot overlays should be inspectable, not theatrical.
- Issue rows should optimize for scanning, comparison, and repeated triage.

## 6. Do And Do Not

Do:

- Use linework, ticks, score arcs, and tables to reinforce measurement.
- Keep the signal color rare and meaningful.
- Preserve contrast and keyboard focus visibility.
- Keep report evidence close to the issue it explains.
- Test UI changes by rendering the relevant route, not just reading CSS.

Do not:

- Add gradient blobs, fake browser chrome, generic icon-card grids, or confetti.
- Turn the interface into a dark developer console; dark surfaces are reserved for terminal-like output only.
- Use alarmist red-heavy security language.
- Add framework-specific design instructions unless the implementation actually uses that stack.
- Redefine design tokens outside `clients/web/app/styles/instrument.css` without a clear reason.
