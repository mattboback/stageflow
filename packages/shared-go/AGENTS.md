# AGENTS.md — packages/shared-go

Shared Go library imported by all platform services. No service-specific logic here.

## PACKAGES

| Package | Role |
|---------|------|
| `models/` | Canonical `Job`, `JobStatus`, `ScannerResult`, artifact, screenshot types |
| `events/` | NATS event payload structs + `Validate()` methods |
| `messaging/` | JetStream client: `PublishEvent`, `SubscribeTyped`, `EnsureStreams` |
| `storage/` | `Uploader`/`Downloader`/`Presigner` interfaces + MinIO implementation |
| `scannercatalog/` | Scanner catalog loader (reads `manifests/*/manifest.json`) |
| `scannerregistry/` | Scanner registry (ids, aliases, categories, defaults) |
| `httputil/` | `RespondStructuredError`, `RespondJSON`, `New*Error` constructors |
| `logging/` | `slog`-based helpers + context keys (`RequestID`, `RunID`) |
| `config/` | Env/file config loaders + validation helpers |
| `bootstrap/` | Shared service startup helpers (MinIO client, NATS connect) |

## WHERE TO LOOK

| Task | File |
|------|------|
| Add/change event payload | `events/types.go` |
| Add NATS stream/subject | `messaging/nats.go` |
| Job/result model change | `models/job.go` |
| Storage interface change | `storage/client.go` |
| Scanner manifest loading | `scannercatalog/catalog.go` |
| HTTP error constructors | `httputil/errors.go` |
| Log context key | `logging/logger.go` |

## CONVENTIONS

- Interfaces in `storage/` and `messaging/` are the seam for mocking in tests.
- All event payloads in `events/types.go` must implement `Validate() error`.
- Use `logging.RequestID(ctx)` / `logging.RunID(ctx)` for correlation in structured logs.
- Timestamps always UTC — enforced in `events/envelope.go`.
- This module is a `go.work` participant — changes affect all services; test all dependents.

## NOTES

- `EnsureStreams()` creates required JetStream streams at service startup.
- Built-in scanner manifests live under `scannercatalog/manifests/` — one dir per scanner.
- `httputil.RespondStructuredError` sets `Retry-After` header when present in error detail.
