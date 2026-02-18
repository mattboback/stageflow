# StageFlow Operations Runbook

Runbook for local, staging, and production environments.

## Command Surface

- Local dev stack: `just dev [up|down|restart|logs|init]`
- Staging stack: `just staging [up|down|restart|logs|init|ps]`
- Production quadlets: `just prod [install|up|down|restart|logs|ps|health]`
- Production deploy: `just deploy [full|quick]`

## Startup and Shutdown

### Local

```bash
just setup
just dev up
just dev init
just images
```

Shutdown:

```bash
just dev down
```

### Staging

```bash
just staging up
just staging init
```

Shutdown:

```bash
just staging down
```

### Production

```bash
just prod install
just prod up
```

Shutdown:

```bash
just prod down
```

## Health Verification

### Local and Staging

- `just dev logs` or `just staging logs` shows service readiness and errors.
- UI responds at expected frontend host/port.
- API responds on configured API host/port.
- Buckets initialized (`just dev init` or `just staging init`).

### Production

```bash
just prod health
just prod ps
```

Healthy indicators:

- Core units active (`nats`, `minio`, `postgres`, `orchestrator`, `platform-api`, `frontend`, `grafana`).
- No crash loop in service logs.
- Job submissions move through states without stalls.

## Incident Triage

1. Identify affected jobs and states.
2. Determine failing stage (intake, extraction, scanning, aggregation).
3. Pull service logs for failing window.
4. Inspect job event timeline with `job-status-cli`.
5. Validate storage artifacts for failed jobs.

## Job Debugging Workflow

```bash
# Build once
go build -o tools/job-status-cli/job-status-cli ./tools/job-status-cli

# Inspect jobs
./tools/job-status-cli/job-status-cli jobs

# Inspect one job timeline
./tools/job-status-cli/job-status-cli events <job-id> -payload

# Inspect orchestrator summary
./tools/job-status-cli/job-status-cli status
```

## Recovery Playbooks

### Orchestrator unavailable

1. Confirm NATS and Postgres are healthy.
2. Restart orchestrator service (`just dev restart`, `just staging restart`, or `just prod restart`).
3. Re-check pending/scanning jobs and event consumption.

### API unavailable

1. Confirm API service status and logs.
2. Verify API environment configuration.
3. Confirm upstream dependencies (orchestrator status source) are reachable.

### MinIO issues

1. Verify MinIO service health.
2. Re-run bucket initialization for environment.
3. Check artifact upload/download failures in logs.

### Scanner failures

1. Validate scanner list and manifest expectations.
2. Verify scanner options format and schema compatibility.
3. Check container runtime errors and page-level failures.

## Deployment and Rollback

### Full deployment

```bash
just deploy full
```

### Quick restart deployment

```bash
just deploy quick
```

### Rollback guidance

Rollback strategy depends on image/version management in your environment. At minimum:

1. Restore previously known-good images/config.
2. Restart services in dependency order.
3. Re-run health checks and smoke test a job flow.

## Post-Incident Notes

Capture and retain:

- Incident window and impacted jobs.
- Root cause and blast radius.
- Mitigations and permanent fixes.
- Follow-up tasks for reliability hardening.

## Related Docs

- [README.md](README.md)
- [ARCHITECTURE.md](ARCHITECTURE.md)
- [CONFIGURATION.md](CONFIGURATION.md)
- [SUPPORT.md](SUPPORT.md)
