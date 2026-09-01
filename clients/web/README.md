# Web Client (`clients/web`)

This is the **StageFlow web app** — the browser UI served at
[stageflow.org](https://stageflow.org) and in local self-host stacks.

It is responsible for:

- Submitting scans (URL or static-site ZIP) against a StageFlow API.
- Streaming live job status over SSE, with polling fallback.
- Rendering the unified report: a Review workspace with screenshot overlays and
  human decisions, a searchable Findings view with evidence and remediation,
  and Artifacts downloads.
- Browser-local named projects (`/projects`) with baseline promotion, report
  diffs, and export/import. Project metadata stays in IndexedDB with no
  StageFlow account or cloud sync.
- The employer-facing `/demo` report, driven by the committed unified-report
  fixture rather than a live scan.

It is **not** responsible for:

- Running scans or talking to scanners (handled by the Orchestrator and
  scanner-runner behind the Platform API).
- Persisting server-owned scan, job, or report data, which remains API-owned.
  Human-review decisions are the exception: they are stored in browser
  `localStorage`, scoped per job, so reviewers can resume locally.

## Stack

React Router 8 in SPA mode (`ssr: false`; `/` and `/playground` are
prerendered to static HTML), Vite, TypeScript, Bun. Design tokens are
canonical in `app/styles/instrument.css` — see
[docs/design.md](../../docs/design.md) for the visual system and
[docs/product.md](../../docs/product.md) for who the app serves.

Report types under `app/lib/types/` follow the v2 unified-report contract in
[`libs/contracts/report`](../../libs/contracts/report); the committed
all-scans fixture there is the canonical example of a full report.

## Layout

| Path                     | Purpose                                                                   |
| ------------------------ | ------------------------------------------------------------------------- |
| `app/routes/`            | Pages: home, projects, playground, demo, scan, report, privacy            |
| `app/components/report/` | Report UI: Review, Findings, Artifacts, job actions, and evidence         |
| `app/lib/api/`           | Fetch + SSE clients over the platform API                                 |
| `app/lib/report/`        | Pure report logic: severity, grouping, filters, scoring                   |
| `app/lib/hooks/`         | Scan status/report monitors (SSE with polling fallback)                   |
| `e2e/`                   | Playwright smoke against a preview build with mocked API                  |

## Develop

```bash
bun install
bun run dev        # dev server on :3020
```

The app expects a StageFlow API (local stack or `stageflow.org`) for real
scans; the report screens can be exercised without one via the tests below.

## Test

```bash
bun run test       # vitest: pure lib functions + report component rendering
bun run test:e2e   # Playwright: production preview build, API mocked from the contract fixture
bun run ci         # lint + typecheck + test + build (the CI gate)
```

Unit and component tests are driven by the committed contract fixture rather
than hand-rolled objects, so they stay honest against the v2 report schema.
The e2e smoke builds the app, serves it with `vite preview`, and mocks the
jobs API with `page.route()` — no backend or containers required.
