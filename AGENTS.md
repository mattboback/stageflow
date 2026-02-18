# AGENTS.md

## OVERVIEW

StageFlow: Podman-native web accessibility + quality scanning platform. URLs or ZIP archives → containerized scanners (axe, lighthouse, SEO, security-headers, link-checker, ai-navigator) → unified report via SSE. Polyglot monorepo: Go 1.25 (API/orchestrator/extractor) + TypeScript/Bun/Playwright (scanner-runner) + SvelteKit 5 (frontend). Messaging: NATS JetStream. Storage: MinIO + PostgreSQL. Containers: Podman.

## STRUCTURE

```
stageflow/
├── platform/
│   ├── api/            # Go REST API + SSE (public intake, SSRF validation)
│   ├── orchestrator/   # Go job FSM + Podman pod management + report aggregation
│   ├── extractor/      # Go ZIP extraction service (runs inside job pods)
│   └── scanner-runner/ # TypeScript/Bun/Playwright scanner worker (plugin system)
├── frontend/           # SvelteKit 5 SPA (runes, Tailwind v4)
├── packages/
│   ├── contracts/      # JSON Schema → generated Go+TS types (report, scanner-manifest)
│   └── shared-go/      # Shared Go: models, events, messaging, storage, httputil, logging
├── infra/              # Compose files, Caddy, Quadlet templates, Grafana
├── tools/              # job-status-cli (ops), suite-runner (integration tests)
├── tests/e2e/          # Go e2e tests against running stack
└── scripts/            # build-images.sh, quadlet-install.sh
```

## WHERE TO LOOK

| Task | Location |
|------|----------|
| API route registration | `platform/api/internal/api/router.go` |
| Job intake validation | `platform/api/internal/api/handlers_jobs_*.go` |
| SSRF/URL security | `platform/api/internal/api/security.go` |
| Job FSM transitions | `platform/orchestrator/internal/fsm/state.go` |
| Orchestrator event handlers | `platform/orchestrator/internal/orchestrator/events.go` |
| Scanner startup + limits | `platform/orchestrator/internal/orchestrator/scanning.go` |
| Report aggregation + dedup | `platform/orchestrator/internal/orchestrator/report_aggregator_*.go` |
| ZIP extraction safety | `platform/extractor/internal/extractor/extractor.go` |
| Scanner plugin entry | `platform/scanner-runner/src/worker.ts` |
| Scanner base lifecycle | `platform/scanner-runner/src/core/scanner-base.ts` |
| Plugin discovery | `platform/scanner-runner/src/core/plugins/` |
| Scanner manifests | `packages/shared-go/scannercatalog/manifests/*/manifest.json` |
| Shared NATS event types | `packages/shared-go/events/types.go` |
| NATS client wrapper | `packages/shared-go/messaging/nats.go` |
| HTTP error helpers | `packages/shared-go/httputil/errors.go` |
| Frontend SSE + scan store | `frontend/src/lib/stores/scan-status.svelte.ts` |
| Frontend API client | `frontend/src/lib/api/client.ts` |

## COMMANDS

```bash
just setup            # One-time: Podman network + go work sync + bun install (frozen)
just dev up           # Local stack (NATS, MinIO, Postgres, services, frontend)
just dev init         # Init MinIO buckets (run after dev up)
just images           # Build all container images
just ci               # Full CI: go build/lint/test + bun lint/typecheck/test:coverage
just run api          # Run API service locally
just run orchestrator # Run orchestrator locally
just run frontend     # Run frontend dev server
just deploy full      # Build images + restart prod Quadlets
just prod health      # Check production service states
```

## CONVENTIONS

### All
- Use `just` from repo root for all workflows.
- Assume other agents/humans may commit concurrently — ignore unrelated diffs.
- `Fail fast → Guard clauses → Validate at boundaries → Make illegal states unrepresentable`

### TypeScript (scanner-runner, frontend)
- Extend `tsconfig.strict.json`; all strict flags enforced including `noUncheckedIndexedAccess`.
- `unknown` + narrowing over `any`. Runtime validation (AJV/schema) over `as` casts.
- scanner-runner: Bun-native APIs preferred. Build produces `dist/` via `tsc`.
- scanner-runner: run `bun run prepare:contracts` before build/test.

### Go (api, orchestrator, extractor, shared-go, tools, tests)
- Always handle errors — never `_ = err`. Always pass `context.Context` through call chains.
- Wrap with `fmt.Errorf("%s: %w", msg, err)`.
- HTTP error responses via `httputil.RespondStructuredError` + `httputil.New*Error` constructors.
- Timestamps always UTC.
- `go.work` lists all modules; `just ci` iterates each.

### Svelte 5 (frontend)
- Runes only: `$state`, `$derived`, `$derived.by()`, `$effect`. No new Svelte writable stores.
- Factory stores (`.svelte.ts`) for cross-component lifecycle + async state.
- **Verify Svelte 5 docs before implementing new patterns** — API changes frequently.

## ANTI-PATTERNS

- Never `_ = err` in Go. Never `any` or `as` casts in TypeScript.
- Never disable lint rules to pass CI — fix root cause.
- Never edit `packages/contracts/*/generated/**` directly — regenerate from schema (`make`).
- Do not add Svelte writable stores where runes work.
- `WriteTimeout = 0` on API HTTP server is **intentional** (SSE). Do not change.
- Orchestrator mounts Podman socket — intentional for pod management. Do not remove.

## DEPLOYMENT

Production: systemd user services + Podman Quadlets (`infra/quadlets/templates/`).
If a shared reverse proxy exists, route to StageFlow services on loopback — do not bind a second proxy to 80/443.

## NOTES

- **SSRF**: URL submissions block loopback, private, link-local, metadata IPs.
- **scanner-runner plugins**: `dist/scanners` → `/plugins` (volume) → `$HOME/.stageflow/plugins` → `PLUGIN_PATHS`. `SCANNER_OPTIONS` validated against manifest `configSchema` (strict in prod).
- **SSE**: `WriteTimeout = 0` on API server. Per-handler timeouts in middleware.
- **Coverage thresholds** (~50%) enforced by vitest in CI.
- **Contracts regen**: `cd packages/contracts/<name> && make` regenerates Go+TS from JSON Schema.
