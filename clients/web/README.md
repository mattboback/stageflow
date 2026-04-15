# StageFlow Web App

The web app is the SvelteKit 5 frontend for StageFlow.

It is responsible for:

- scan submission from the browser
- live job progress over SSE
- unified report exploration with issue evidence and screenshots
- scanner selection, presets, and AI Navigator configuration
- the public-facing landing and playground experience

It is not responsible for:

- executing scanners
- persisting jobs or reports
- orchestrating containers or NATS consumers
- normalizing raw scanner output into the canonical report contract

Those responsibilities live in the Platform API, Orchestrator, Scanner Runner, and shared contracts.

## How it fits into the system

The web app talks to the Platform API over HTTP:

- `POST /api/v1/jobs/urls` and `POST /api/v1/jobs/zip` submit scans
- `GET /api/v1/scanners` loads available scanners and capabilities
- `GET /api/v1/jobs/:id/stream` drives live status updates over SSE
- `GET /api/v1/jobs/:id`, `/report`, `/results`, and `/diff` back the report view and project diff UX

The browser never talks directly to the Orchestrator or Scanner Runner. It consumes the API boundary and renders the normalized report shape produced by the backend.

## Routes worth inspecting

| Route                                      | Purpose                                                  |
| ------------------------------------------ | -------------------------------------------------------- |
| `src/routes/+page.svelte`                  | Marketing/landing page and product framing               |
| `src/routes/playground/+page.svelte`       | Main scan configuration flow for URL and ZIP jobs        |
| `src/routes/scan/[id]/+page.svelte`        | Live job status shell                                    |
| `src/routes/scan/[id]/report/+page.svelte` | Unified report view with evidence and remediation detail |

## Key directories

| Path                             | What is there                                                          |
| -------------------------------- | ---------------------------------------------------------------------- |
| `src/lib/api/`                   | Browser API client, URL helpers, and SSE plumbing                      |
| `src/lib/components/playground/` | Scan configuration UI, presets, validation, and ZIP upload flow        |
| `src/lib/components/report/`     | Report rendering, issue detail, evidence, and summary components       |
| `src/lib/stores/`                | Realtime scan/report stores and job-stream state management            |
| `src/lib/report/`                | Filtering, grouping, severity, screenshots, and virtualization helpers |
| `tests/unit/api/`                | API client and request/response behavior tests                         |
| `tests/unit/components/`         | Component-level tests for playground and report surfaces               |
| `.storybook/`                    | Storybook stories used by interaction and accessibility checks         |

## Local commands

From `clients/web/`:

```bash
bun install --frozen-lockfile
bun run dev
```

Useful workspace commands:

```bash
bun run type-check
bun run lint
bun run test
bun run storybook
bun run ci
```

`bun run ci` is the main verification path. It runs formatting checks, strict linting, type checks, and coverage-backed tests.

## What to look at first

If you only inspect a few files, start here:

- `src/routes/playground/+page.svelte` — the main scan submission experience
- `src/lib/components/playground/PlaygroundPage.svelte` — form state, presets, AI config, submission flow
- `src/lib/stores/scan-report.svelte.ts` and `src/lib/stores/scan-status.svelte.ts` — job streaming and report hydration
- `src/lib/components/report/` — report UX and evidence rendering
- `tests/unit/components/playground/PlaygroundPage.test.ts` — broad UI behavior coverage for the main flow
- `tests/unit/api/client.test.ts` — browser/API contract behavior and error handling

## Relationship to reports

The frontend assumes the backend already did the hard normalization work. It renders the canonical report contract rather than branching on scanner-specific payloads. That is why most report logic lives in filtering, grouping, screenshot, and presentation helpers instead of per-scanner adapters.
