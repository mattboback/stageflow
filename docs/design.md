# Design: Calibrated Instrument

StageFlow should feel like a precise frontend-quality instrument: calm, trustworthy, understated, and evidence-heavy. It favors clear readouts and stable report surfaces over marketing decoration.

Exact tokens and shared primitives live in `clients/web/app/styles/instrument.css`. Report severity treatments live in `clients/web/app/styles/report.css`, selected through `clients/web/app/lib/report/severity.ts`. Update CSS first; keep this document focused on intent and usage.

## Visual Language

- Use cool graphite and white surfaces with one teal signal color.
- Use borders and tonal steps for structure; reserve shadows for elements that actually float.
- Keep controls compact and exact rather than soft or decorative.
- Use rounded corners from the shared scale; do not invent local radii.
- Offer a full light and dark theme. Keep terminal-like output dark in both.
- Treat severity as a calibrated scale, not an alarm.

Avoid generic cream SaaS dashboards, gradient text, glass cards, fake browser chrome, icon-card grids, gamified scores, red-everywhere security styling, and decorative page-load choreography.

## Color Semantics

Signal teal is reserved for primary actions, links, live state, focus, brand identity, and gauge emphasis. It should remain visually scarce.

Neutral surfaces establish hierarchy:

- ground for the page;
- surface for panels and controls;
- sunk surface for inputs and inset tracks;
- progressively muted ink for secondary text;
- line tokens for structure and calibration marks.

Severity colors communicate critical, serious, moderate, minor, and pass/info states. Never rely on hue alone: pair severity with a label, count, icon, badge text, grouping, or overlay geometry.

## Typography

- Gabarito carries headings, major measurements, and gauges.
- Source Sans 3 carries prose, labels, controls, and navigation.
- JetBrains Mono carries commands, selectors, snippets, IDs, and machine-oriented values.

Do not substitute fonts for decoration. Product chrome stays compact and steady; large display type belongs only on landing and major report headings.

## Structure and Elevation

Surfaces are flat by default. Full borders, background steps, spacing, and calibration marks establish grouping. Shadows indicate real stacking: sticky elements, popovers, modals, toasts, and tooltips.

Use the semantic z-index tokens. Never introduce arbitrary values such as `999` or `9999`.

## Components

### Buttons

- Primary buttons use the strong signal token and white text.
- Ghost buttons use a surface fill and structural border.
- Active states may move by one pixel; hover should not become theatrical.
- Every control needs a visible `:focus-visible` treatment.

### Inputs

- Use the sunk surface, strong line, shared radius, and readable placeholder color.
- Keep labels explicit and sentence-cased.
- Focus combines signal border and restrained focus wash.

### Panels

- Use one border and no resting shadow.
- Avoid nested cards when spacing or a divider expresses the same hierarchy.
- Keep evidence adjacent to the issue or decision it explains.

### Status and Severity

- Status pills represent state, not decoration.
- Severity containers use a complete border and a restrained wash, not colored side stripes.
- Screenshot markers use both severity border and geometry.
- Filter chips retain readable labels in active and inactive states.

### Navigation

- Keep the primary navigation stable and compact.
- Collapse secondary text links on narrow screens while preserving the main action.
- Use sticky/frosted treatment only where persistent orientation justifies it.

## Motion

Motion communicates state or feedback. Use the shared duration and easing tokens. Continuous animation is limited to genuine live-state cues, and all motion must honor `prefers-reduced-motion`.

Do not orchestrate reveal sequences for ordinary page content.

## Accessibility Rules

- WCAG AA is the contrast floor; prefer AAA for body and interface text where practical.
- Keep keyboard focus visible.
- Do not encode severity, status, or selection through color alone.
- Keep targets usable on mobile and labels available to assistive technology.
- Test UI changes in the rendered route at desktop and mobile widths.

## Review Checklist

1. Does the screen read as an instrument rather than a marketing template?
2. Is signal color rare and meaningful?
3. Are severity and status understandable without color?
4. Is hierarchy carried by spacing, linework, and type before cards or shadows?
5. Are exact values sourced from shared CSS rather than redefined locally?
6. Does the interaction work with keyboard, reduced motion, and narrow viewports?
