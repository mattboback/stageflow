# Web Client (`clients/web`)

This is the **StageFlow web app** — the browser UI served at
[stageflow.org](https://stageflow.org) and in local self-host stacks.

It is responsible for:

- Submitting scans (URL or static-site ZIP) against a StageFlow API.
- Streaming live job status over SSE, with polling fallback.
- Rendering the unified report: overview dashboard, issue triage with evidence
  and remediation, visual review with screenshot overlays, and artifact
  downloads.

It is **not** responsible for:

- Running scans or talking to scanners (handled by the Orchestrator and
  scanner-runner behind the Platform API).
- Persisting anything — it is a stateless SPA over the HTTP + SSE API.

## Stack

React Router 7 in SPA mode (`ssr: false`; `/` and `/playground` are
prerendered to static HTML), Vite, TypeScript, Bun. Design tokens are
canonical in `app/styles/instrument.css` — see
[docs/design.md](../../docs/design.md) for the visual system and
[docs/product.md](../../docs/product.md) for who the app serves.

Report types under `app/lib/types/` follow the v2 unified-report contract in
[`libs/contracts/report`](../../libs/contracts/report); the committed
all-scans fixture there is the canonical example of a full report.

## Layout

| Path                     | Purpose                                                  |
| ------------------------ | -------------------------------------------------------- |
| `app/routes/`            | Pages: home, playground, scan submission, status, report |
| `app/components/report/` | Report UI: overview, issue list, evidence, visual review |
| `app/lib/api/`           | Fetch + SSE clients over the platform API                |
| `app/lib/report/`        | Pure report logic: severity, grouping, filters, scoring  |
| `app/lib/hooks/`         | Scan status/report monitors (SSE with polling fallback)  |
| `e2e/`                   | Playwright smoke against a preview build with mocked API |

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
