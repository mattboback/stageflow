# Provenance Schema

JSON Schema (Draft 7) for the Provenance document that flows from CLI/orchestrator
into the scanner-runner. This schema is the single source of truth for the
serialized wire format, including the optional `auth` block introduced for
authenticated scanning.

## Generated artifacts

- `../generated/typescript/provenance.ts` — TypeScript types via `json2ts`.
- `../generated/go/provenance_schema.go` — Go types via `go-jsonschema`.

Both are produced by `just generate-contracts` from the repo root.

## Auth contract

The `auth` field is optional. When absent, a Provenance document is byte-identical
to the pre-auth shape. When present, it is a discriminated union on `mode`:

- `storage_state` — `{ mode, artifact_key }`. The artifact_key references a
  Playwright storage-state JSON object stored under the job's MinIO prefix.
- `form` — `{ mode, login_url, steps, success }`. `steps` reuses the existing
  `PreScanAction` shape; any string `value` field can also be a typed
  `{ from_env: NAME }` reference. The `success` block is a `WaitStrategy` that
  signals a completed login.

The shared contract remains compatible with literal strings. Caller-authenticated
API and CLI workflows should use `{ from_env }` references. The deliberately
narrow public browser-auth route accepts literal form steps for throwaway demo
accounts, then strips auth from public Provenance and artifacts. Literal values
necessarily cross the file-backed `job.created` event and remain in live job
configuration until terminal cleanup; terminal audit records, reports, scanner
stage logs, and sensitive screenshots are redacted.
