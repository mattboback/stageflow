# Support

Use this guide for help, triage, and escalation.

## Getting Help

- Open a GitHub issue for bugs, feature requests, or usage questions.
- Include environment details and reproducible steps.
- For sensitive security issues, do not open a public issue; use [SECURITY.md](SECURITY.md).

## What to Include in a Support Request

- StageFlow version or commit SHA.
- Deployment mode (`dev`, `staging`, or production quadlets).
- Exact command(s) and observed output.
- Affected job ID(s), scanner module(s), and timestamps (UTC).
- Any relevant service logs.

## Quick Triage Checklist

1. Verify services are up and healthy.
2. Confirm required buckets and environment variables are present.
3. Inspect job state and timeline.
4. Check scanner artifacts and stage outputs.
5. Correlate errors across API/orchestrator/scanner logs.

## Useful Commands

```bash
# Local stack
just dev up
just dev logs
just dev init

# Staging stack
just staging ps
just staging logs

# Production state
just prod health
just prod logs
```

## Job Inspection

See `tools/README.md` for full CLI usage.

```bash
# Build utility once
go build -o tools/job-status-cli/job-status-cli ./tools/job-status-cli

# Show current jobs
./tools/job-status-cli/job-status-cli jobs

# Show event timeline for one job
./tools/job-status-cli/job-status-cli events <job-id> -payload

# Show orchestrator pod + queue metrics
./tools/job-status-cli/job-status-cli status
```

## Common Failure Patterns

- URL submission rejected: often SSRF policy or URL validation.
- ZIP job fails early: archive safety constraints or invalid structure.
- Scanner failure: plugin load/config mismatch, runtime timeout, or page instability.
- No progress updates: check SSE path and API/orchestrator reachability.

## Escalation

When opening an issue for unresolved incidents, attach:

- Job ID and state history.
- Relevant timestamps and log snippets.
- Whether failure is deterministic or intermittent.
- Any temporary mitigation already attempted.
