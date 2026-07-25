# Product

> Strategic context for StageFlow's design work. Describes the **web client**
> (`clients/web`, the React Router app served at stageflow.org). The visual
> system lives in [design.md](design.md); tokens are canonical in
> `clients/web/app/styles/instrument.css`.

## Register

product

## Users

**Primary — first-time visitors to stageflow.org.** A developer or site owner who
arrives from search, a shared link, or the README demo, wants to scan a URL and see
a real report immediately — no StageFlow account on the hosted demo, low commitment. They are evaluating whether
the tool is worth trusting and adopting. They may be on mobile, and may not know what
"axe critical" or a "Lighthouse score" mean. The whole web experience is judged on
how well it serves this first minute.

**Secondary — teams reading reports.** Returning users who explore findings, evidence
(screenshots, exact rules), and remediation to actually fix frontend issues. They care
about depth, accuracy, and being able to trust a score enough to act on it.

**Tertiary — developers in CI and self-hosters.** Primarily CLI users who gate builds
on regressions against a per-project baseline. The web UI is a visual companion to the
terminal/JSON workflow, not their main surface.

## Product Purpose

StageFlow is a self-hostable **frontend quality platform**. It ships eight built-in
scanners — accessibility, performance, SEO, links, security headers, social metadata,
content quality, and agent-driven navigation — behind a single report contract, and
remembers a **baseline per project** so CI can answer the question that matters:
_did this change make the frontend worse?_

The web app is the platform's public face: submit a scan (URL or static-site ZIP), watch
live progress over SSE, and explore one unified, evidence-rich report with severity,
proof, and remediation side by side.

**Success** is a first-time visitor who runs a scan and understands their frontend's
quality — and what to fix — within a minute, with enough trust to adopt the tool or
self-host it.

## Brand Personality

**Calm · trustworthy · understated.** The voice is plain, exact, and never alarmist —
even when reporting critical issues. It states findings with evidence and gets out of
the way. Confidence comes from precision and transparency, not from loud color, urgency,
or persuasion. A first-time visitor should come away feeling the tool is _serious, honest,
and on my side_ — not that they are being marketed to.

## Anti-references

What StageFlow's web UI must **not** look like:

- Generic SaaS-cream dashboard template; gradient-accented hero-metric cards.
- Identical icon-card grids ("eight feature cards: icon + heading + blurb").
- Alarmist security-scanner aesthetics — red-everywhere, scare-tactic "THREATS DETECTED"
  urgency, gamified scores, confetti.
- Fake browser chrome; brand-colored scanner logos used as decoration.
- Cool-slate "developer-tool dark dashboard." StageFlow is light-only cool graphite;
  dark is reserved for the intentional terminal/automation islands only.
- Heavy marketing-page reveal choreography on a tool that should feel instant and honest.

## Design Principles

1. **Show the real report, don't pitch it.** The product sells itself by letting a
   stranger run an actual scan and explore real findings. Demonstrate; don't advertise.
2. **Calm under criticality.** Even critical findings are presented with composure —
   evidence, severity, and remediation side by side. Never alarmist, never gamified.
3. **Practice what you preach.** A tool that scans for accessibility and quality must be
   exemplary itself: WCAG AA floor, AAA contrast where practical, fast, semantic,
   keyboard-navigable. The product is its own best demo.
4. **Trust through transparency.** Every score and claim is backed by evidence
   (screenshots, exact rules, source). No black-box numbers without the "why."
5. **Earn the first minute.** Optimize the path from first-time visitor to "I understand
   my report": minimal friction, no demo account, sensible defaults, legible live progress.

## Accessibility & Inclusion

- **WCAG 2.1 AA** is the hard floor across primitives, checked three ways:
  the web app CI path (`cd clients/web && bun run ci`), the separate Playwright/axe
  E2E suite (`clients/web/e2e/accessibility.spec.ts`, which asserts zero
  serious-or-critical axe violations on five surfaces at two viewports), and
  StageFlow scanning its own running UI — the `Scan StageFlow's own UI with
  StageFlow` step in `.github/workflows/golden-regression.yml`, which runs the real
  CLI against the frontend container and fails on `serious`.
- **Prefer AAA contrast where practical** on body and UI text; tokens target cool
  near-black ink on white/near-white surfaces (`--ink` / `--ink-strong` on
  `--surface` / `--ground`).
- Contrast floors: `--ink-muted` and body text ≥ 4.5:1 (AA); decorative large or
  non-essential text may use fainter ink only when it still meets the applicable ratio.
- **Severity is never conveyed by color alone** (color-blind safe): pair every hue with a
  label, icon, or text.
- **Reduced motion** is honored on every animation; no entrance fades on form fields (axe
  races partial opacity).
- Light-only by design; keyboard and screen-reader support are first-class — the product
  is itself an accessibility scanner.
