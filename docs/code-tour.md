# Code Tour

This path gives reviewers a concrete tour of StageFlow without repeating the architecture narrative.

## Two Minutes

1. Run a scan at [stageflow.org](https://stageflow.org).
2. Inspect screenshot evidence in Review, search the complete issue list in Findings, and open the owned outputs in Artifacts.
3. Open the [full report fixture](../libs/contracts/report/fixtures/unified-report.v2.all-scans.json).
4. Skim the [platform shape](architecture.md#platform-shape) and [security model](architecture.md#security-model).

## Fifteen Minutes

| Question | Start here |
| --- | --- |
| How does intake reject unsafe targets? | `services/platform-api/internal/api/security.go` and its tests |
| How does a job move between states? | `services/orchestrator/internal/domain/jobs/transitions.go` |
| How is completion decided? | `services/orchestrator/internal/domain/jobs/completion_policy.go` |
| How do scanners share one output model? | `libs/contracts/report/schema/unified-report.v2.schema.json` |
| How are scanner plugins resolved? | `services/scanner-runner/src/scanners/registry.ts` |
| How does the CLI stream and gate results? | `clients/cli/internal/command/scan_job.go`, `clients/cli/internal/jobstream/`, `clients/cli/internal/render/` |
| How does the report support human review? | `clients/web/app/components/report/` and `clients/web/e2e/report.spec.ts` |
| How are regressions proven end to end? | `qa/e2e/project-scan-golden.sh` and `qa/fixtures/project-golden/` |

## Design Boundaries

- The Platform API owns public HTTP intake, status projection, projects, baselines, and diffs.
- The orchestrator owns the job state machine, Podman lifecycle, aggregation, and durable job history.
- Scanner containers emit one schema-defined report shape; clients do not branch on scanner identity.
- Rootless per-job pods contain scanner failures and untrusted archive content.
- Stable content-derived issue IDs make baseline diffs meaningful.
- SSE fits one-way progress delivery without WebSocket state or protocol upgrades.

## Quality Gates

- `.github/workflows/ci.yml` runs workflow, secret, Go, web, scanner, dead-code, vulnerability, and image checks.
- `just ci` is the local repository gate.
- `just project-golden` exercises create → scan → promote → regress → diff and asserts CLI exit behavior.
- The committed report fixture drives contract, component, and browser tests.

Component READMEs contain local commands and implementation details. The [architecture](architecture.md) remains the canonical explanation of system decisions.
