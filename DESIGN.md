# Design

StageFlow's current browser UI uses a **calibrated instrument** visual language: quiet,
high-contrast surfaces, precise linework, mono readouts, and restrained teal signal color.

The canonical implementation is `clients/web/app/styles/instrument.css`.

## Principles

1. **Readable before decorative.** Reports are dense, so layout, contrast, and spacing matter more than ornament.
2. **Data has its own voice.** Scores, counts, IDs, and status labels use the mono face.
3. **Severity is never color alone.** Every severity color is paired with labels, text, or structure.
4. **Borders over shadows.** Panels and controls separate with rules, not heavy elevation.
5. **Motion stays useful.** Animation should clarify live status or focus, not create marketing-page choreography.

## Core tokens

| Token | Purpose |
| --- | --- |
| `--ground`, `--surface`, `--surface-sunk` | Page, panel, and inset backgrounds |
| `--ink-strong`, `--ink`, `--ink-muted` | Text hierarchy |
| `--line`, `--line-strong`, `--tick` | Dividers, panel borders, calibration ticks |
| `--signal`, `--signal-strong`, `--signal-wash` | Links, active states, primary actions, live activity |
| `--sev-*` | Scanner severity scale |
| `--sans`, `--mono` | UI and data typography |
| `--r-*` | Small-radius shape scale |

## Component posture

- Navigation and page chrome should feel like a control surface, not a marketing template.
- Report pages should bias toward tables, grouped rows, evidence panels, and explicit labels.
- The primary action should be obvious, but teal should not dominate a page.
- Interactive state should be visible through border, fill, focus ring, and text changes.
- Keyboard focus must remain high contrast.

## Accessibility checks

- Keep text contrast at WCAG AA or better.
- Pair severity color with explicit severity labels.
- Preserve keyboard navigation and screen-reader labels for scan forms, report filters, dialogs, and evidence controls.
- Honor reduced-motion preferences for nonessential animation.
