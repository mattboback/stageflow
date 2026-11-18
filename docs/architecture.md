# Architecture

[Overview](../README.md) · [Contributing](../CONTRIBUTING.md) · [Security](../SECURITY.md)

StageFlow separates intake, durable job coordination and browser execution. The clients use the public HTTP API; the orchestrator owns the Podman control plane.

![Clients submit over HTTP and SSE to the Platform API, which admits jobs into a PostgreSQL queue; the orchestrator claims one job at a time, runs it in rootless Podman pods and publishes one Unified Report v2](../assets/architecture.svg)

| Component                    | Responsibility                                                                                                                     |
| ---------------------------- | ---------------------------------------------------------------------------------------------------------------------------------- |
| `services/platform-api`      | Public HTTP boundary: URL and ZIP intake, SSRF policy, queue admission, job status, SSE and report files                           |
| `services/orchestrator`      | Claims jobs, runs each in a rootless Podman pod, settles scanner outcomes, aggregates the report, enforces deadlines and retention |
| `services/archive-extractor` | For ZIP jobs, validates and extracts the archive, then serves the site to the scanners                                             |
| `services/scanner-runner`    | Node.js/TypeScript Playwright runtime; each container runs one scanner over the job's pages                                        |
| `internal/jobstore`          | PostgreSQL queue, job state machine and status snapshot shared by the API and orchestrator                                         |
| `internal/artifacts`         | Job file store on the artifact volume, confined with `os.Root` and atomic renames                                                  |
| `libs/go/scannercatalog`     | The five scanners, read from embedded manifests with their resource limits and option schemas                                      |
| `libs/contracts`             | JSON Schemas with committed Go and TypeScript types, checked for drift in CI                                                       |
| `clients/web`, `clients/cli` | The React Router app and the Go CLI                                                                                                |

A job moves through `PENDING → EXTRACTING → READY_TO_SCAN → SCANNING → COMPLETING → DONE`, with `FAILED` reachable from any active state. URL jobs skip extraction. PostgreSQL holds the queue, job state and scanner attempts; the artifact volume holds private inputs, the extracted site, screenshots and the unified report. Only one job runs at a time and five may wait.

`UnifiedReportV2` is one document per job: metadata, a summary with an optional 0–100 score, per-scanner status, per-page summaries with screenshot overlays, issues and structured errors. Each issue ID is a hash of its rule, page and element, so the same violation keeps its ID across runs for baselines and regression detection. JSON Schema files under `libs/contracts/` are the source of truth; `just generate-contracts` regenerates the committed types and CI fails on drift.

## Design decisions in the code

| Concern                        | Implementation                                                                                                               | Where to read                                                                                                                                         |
| ------------------------------ | ---------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| Recover interrupted work       | Reconcile durable job state with observed containers; deterministic names let retries adopt an existing pod or container.    | [Reconciliation](../services/orchestrator/internal/orchestrator/reconcile.go), [job runtime](../services/orchestrator/internal/podman/job_runtime.go) |
| Keep failures visible          | Reports record scanner and page coverage; comparisons check compatible coverage before claiming a fix or a comparable score. | [Comparison coverage](../libs/go/diff/coverage.go), [report aggregation](../services/orchestrator/internal/aggregate/report_aggregator.go)            |
| Share a stable report contract | JSON Schema is the source for committed Go and TypeScript types; generation checks detect drift.                             | [Report schema](../libs/contracts/report/schema/unified-report.v2.schema.json), [generator](../devtools/scripts/generate-contracts.sh)                |
| Keep job files contained       | The artifact store uses rooted filesystem operations and atomic publication.                                                 | [Artifact store](../internal/artifacts/files.go)                                                                                                      |
| Serve a small web deployment   | The React Router client prerenders public pages and hydrates dynamic scan and project-detail routes from a static shell.     | [Router configuration](../clients/web/react-router.config.ts), [routes](../clients/web/app/routes.ts)                                                 |

## Repository boundaries

- `clients/` contains user interfaces and their adapters to the API and report formats.
- `services/` contains independently built runtime processes. Each Go service keeps service-specific packages under its own `internal/` directory.
- `libs/contracts/` owns shared data formats; `libs/go/` owns shared Go behavior such as report comparison, logging and scanner metadata.
- Root `internal/` contains repository-internal storage and archive support. It is separate from a service's own internal packages.
- `infra/` wires deployments; `devtools/` builds and checks the repository; `qa/` holds full-stack regression inputs and scripts.

Bun manages dependencies and build commands for the TypeScript packages; the scanner production image executes the compiled JavaScript with Node.js.

The scanner manifests in `libs/go/scannercatalog/manifests/` are embedded by the Go catalog and copied into the scanner runtime during its build. Contract packages, the web app and the scanner runtime keep separate Bun lockfiles because they have separate install/build contexts.

Generated schemas, demo reports, screenshot assets and regression fixtures are working inputs. See [generated files and fixtures](../CONTRIBUTING.md#generated-files-and-fixtures) before changing or removing them.
