# Shared Go Libraries (`libs/go`)

Small, single-purpose modules shared by the services and the CLI. Each is its
own Go module (managed via the root `go.work`); none of them import a service.

| Module            | Purpose                                                                  |
| ----------------- | ------------------------------------------------------------------------ |
| `bootstrap`       | Shared service startup helpers (logging, NATS and MinIO clients)         |
| `config`          | Environment-variable configuration loading                               |
| `diff`            | Issue-level diffs between two scan reports (baseline regression engine)  |
| `domain`          | Job domain logic: the job state machine and transitions                  |
| `events`          | Typed NATS event payloads and envelope validation                        |
| `httputil`        | Standardized HTTP response/error helpers                                 |
| `logging`         | Structured logging (slog) conventions for all services                   |
| `messaging`       | NATS JetStream client, consumers, and publish envelopes                  |
| `models`          | Shared data structures for scan jobs                                     |
| `provenance`      | Go-side mirror of the provenance contract (per-job scan input)           |
| `scannercatalog`  | Embedded built-in scanner manifests and lookup                           |
| `scannerregistry` | Scanner enablement/resource config resolution (`scanners.yaml`)          |
| `storage`         | Storage interfaces (MinIO/S3 object access) shared across services       |

Types generated from [`libs/contracts`](../contracts) land in these modules
during `just setup` / `just generate-contracts`; generated files are not
committed.
