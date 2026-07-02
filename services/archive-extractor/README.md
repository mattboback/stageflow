# Archive Extractor

The Archive Extractor prepares an uploaded static site for scanning. It is a
**short-lived, one-shot job process** — not a long-running service. The
Orchestrator launches one instance inside each ZIP-upload job pod, alongside the
Scanner Runner.

It is responsible for:

- downloading the uploaded ZIP from MinIO staging
- safely extracting it (path-traversal, absolute-path, and size/nesting checks)
- discovering the HTML pages in the extracted site
- generating a `provenance.json` describing the pages and base URL
- serving the extracted site over a local HTTP server for the Scanner Runner to scan
- publishing `extraction.ready` (or `extraction.failed`) back to NATS

It is not responsible for:

- running scanners (the Scanner Runner does)
- deciding job state (the Orchestrator does)

## Job pipeline

`cmd/server/main.go` runs the pipeline in `runExtraction`, failing fast and
publishing a terminal `extraction.failed` event at any step via
`publishFailureAndExit`:

1. connect to NATS and MinIO
2. extract the ZIP from staging into the job workspace (`internal/extractor`)
3. discover HTML pages (`internal/discovery`)
4. start the static file server bound to the workspace (`internal/server`)
5. generate provenance and upload it to the artifacts bucket (`internal/provenance`)
6. publish `extraction.ready` with the base URL and page count
7. stay alive serving the site until the scanner finishes and the pod is torn down

## Security

Untrusted user ZIPs are the core threat. The extractor sanitizes entries before
writing and rejects path traversal, absolute paths, and oversized archives. The
dedicated fixtures under `testdata/` (e.g. `path-traversal.zip`,
`absolute-path.zip`, `nested-site.zip`) exercise these guards. Extracted
files are also chmod-restricted so they are reachable only over the local HTTP
server, never world-readable.

## Configuration

The process is fully configured by environment variables (see `loadConfig` /
`Config.Validate` in `cmd/server/main.go`):

| Variable | Purpose |
| --- | --- |
| `JOB_ID` | Identifies the job; namespaces artifacts and events |
| `INPUT_PATH` | Object key of the uploaded ZIP in MinIO staging |
| `WORKSPACE` | Local directory for the extracted site (default `/workspace`) |
| `PORT` | Port for the static file server (default `8080`) |
| `NATS_*`, `MINIO_*` | Connection settings for the event bus and object storage |
| `MINIO_ARTIFACT_BUCKET` | Destination bucket for provenance/artifacts |

## Local run and test commands

From `services/archive-extractor/`:

```bash
go test ./...
go test -race ./...
```

From a fresh checkout, run `just generate-contracts` from the repo root before
running Go commands directly inside this module.

Running the binary directly requires `JOB_ID`, `INPUT_PATH`, NATS, and MinIO to
be set; in practice it runs inside an Orchestrator-launched job pod.

## Files to inspect first

- `cmd/server/main.go` — the full extraction pipeline and failure handling
- `internal/extractor/extractor.go` — safe ZIP extraction and traversal guards
- `internal/discovery/discovery.go` — HTML page discovery
- `internal/server/server.go` — the workspace-scoped static file server
- `internal/provenance/provenance.go` — provenance generation
