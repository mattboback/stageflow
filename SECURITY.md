# Security and privacy

[Overview](README.md) · [Self-hosting](docs/self-hosting.md) · [Architecture](docs/architecture.md)

## Reporting a vulnerability

Report vulnerabilities privately through [GitHub Security Advisories](https://github.com/mattboback/stageflow/security/advisories/new). Do not open a public issue for a vulnerability.

## Hosted data and retention

The hosted demo is a no-account service for public URLs. A finished scan's job record and files are deleted 24 hours after it completes; the deletion endpoint removes them immediately. Job IDs are unguessable bearer-style references, so anyone who receives a job or report URL can retrieve it until expiry. Daily database backups are retained for 14 days, so a job record can outlive the 24-hour window in a backup. Browser-local projects and saved baseline files never leave your device.

## Trust boundaries

The security model has five nested boundaries:

- **Edge proxy:** terminates TLS, routes `/api/*` to the API, and injects no credentials.
- **API intake:** http(s) only, SSRF classification of every resolved address, request and URL limits, literal-only authentication values, rate limits and timeouts.
- **Archive extraction:** entry-count, per-entry and total size limits, compression-ratio checks and path-traversal prevention in the job's own workspace.
- **Job containers:** unprivileged users, all Linux capabilities dropped, `no-new-privileges`, read-only root filesystem, capped CPU/memory and a disk-backed `/tmp` removed with the pod.
- **Job network:** job pods join a dedicated Podman network with `isolate=strict`; they reach the internet and no other Podman network.

Residual risks remain. URL classification is not connection-level DNS pinning, so DNS rebinding is not fully prevented. Chromium runs with `--no-sandbox`; the container is its sandbox. Public deployments should add host egress policy.

## Authenticated scanning

A job can carry a form-login recipe or Playwright storage state. Every caller sends literal values; the API and orchestrator reject environment references, so nothing from a server's environment reaches a scanner. The CLI resolves recipe references locally.

```bash
stageflow auth capture https://app.example.com/login --output ./auth/state.json
stageflow scan https://app.example.com/profile --auth-state ./auth/state.json
```

Capture files are written with mode `0600`. Storage state is mounted only into the scanners that need it and is deleted when the job ends. Credential values stay in the live job row until terminal cleanup, which scrubs them from job and scanner errors. Public provenance, reports and screenshots of sensitive controls are redacted. A failed login becomes a critical `auth-hydration-failed` finding instead of a silent logged-out scan. Never submit a production or reused credential to the hosted demo; use a controlled self-hosted deployment and a throwaway test account.
