# AGENTS.md — platform/extractor

Go service: safe ZIP extraction. Runs inside orchestrator-created job pods. Reads staged ZIP from MinIO, extracts to shared workspace volume, signals orchestrator via NATS.

## STRUCTURE

```
platform/extractor/
├── cmd/server/         # main.go — bootstrap
├── internal/
│   ├── extractor/      # Core ZIP validation + extraction logic
│   ├── discovery/      # Discovers extractable files post-extraction
│   ├── provenance/     # Generates extraction provenance artifacts
│   └── server/         # Service wiring + NATS handlers
└── testdata/           # ZIP fixtures: simple, nested, malicious, empty
```

## WHERE TO LOOK

| Task | File |
|------|------|
| ZIP validation + extraction | `internal/extractor/extractor.go` |
| File discovery | `internal/discovery/discovery.go` |
| Provenance generation | `internal/provenance/provenance.go` |
| NATS wiring + handlers | `internal/server/server.go` |
| Service bootstrap | `cmd/server/main.go` |

## CONVENTIONS

- ZIP validation enforces: entry count limit, max uncompressed size, per-entry size limit, max expansion ratio (ZIP bomb protection), path sanitization (no `../`, no absolute paths).
- No symlink traversal — discovery.go: do not include or traverse symlinks.
- `testdata/malicious/` contains traversal/bomb test cases — do not remove or rename.
- Service runs inside job pod and shares workspace volume with scanner containers.

## NOTES

- Input: staging bucket in MinIO (`INPUT_PATH` env).
- Output: extracted files to shared workspace volume; provenance JSON uploaded to MinIO.
- Signals: publishes `ExtractionReady` (with provenance artifact key) or `ExtractionFailed` to NATS.
