# Authenticated Scanning for StageFlow

## Objective

Make StageFlow scan beyond marketing homepages by adding session-aware authentication to the scanner runtime. Today every scanner receives a fresh Playwright `BrowserContext` with no cookies or storage state, so protected routes (`/profile`, `/applications`, `/settings/*`) collapse onto a login redirect and produce useless audits. After this work a developer can capture a session interactively or describe a deterministic form login, and every existing scanner (axe, Lighthouse, security-headers, ai-navigator) will run against the post-login surface with zero per-scanner code changes. The work also retires `agent-harness-legacy/` from the repository, porting the one idea worth keeping (failure-policy guards) into `ai-navigator` first. Design context lives in `docs/design/authenticated-scanning.md`.

## Implementation Plan

- [ ] 1. Extend the Provenance contract with an optional auth block
  - Add an optional `auth` field to the JSON schema in `libs/contracts/report/` and the Provenance shape in `services/scanner-runner/src/core/types.ts`. The field is a discriminated union of `storage_state` (artifact reference) and `form` (recipe with login URL, steps, success strategy).
  - Rationale: every downstream change reads from Provenance, so this is the single seam that lets one configuration flow through scanner-runner, orchestrator, and CLI without coupling them.
  - Affects: `libs/contracts/`, regenerated Go and TypeScript types, `services/scanner-runner/src/core/types.ts`.
  - Dependencies: none, foundational step.

- [ ] 2. Add a from_env value resolver to PreScanAction
  - Extend the action value type so a step value can be either a literal string or a typed `from_env` reference. Resolve references at execution time inside the scanner-runner pod against an explicit allow-list derived from the recipe.
  - Rationale: keeps credentials out of stored Provenance, the unified report, and every wire payload, while letting recipes describe a deterministic login that CI can run.
  - Affects: `services/scanner-runner/src/core/types.ts`, the executor in `services/scanner-runner/src/core/browser-manager.ts:executePreScanActions`.
  - Dependencies: task 1.

- [ ] 3. Thread storageStatePath through BrowserManager.createContext
  - Add an optional `storageStatePath` parameter to `BrowserManager.createContext` and forward it directly to Playwright's context creation. No other context concern changes.
  - Rationale: this is the smallest possible seam in the runtime that lets a hydrated session apply to every scanner that consumes the shared context.
  - Affects: `services/scanner-runner/src/core/browser-manager.ts`, the call sites in `services/scanner-runner/src/core/scanner-base.ts`.
  - Dependencies: task 1.

- [ ] 4. Add a hydrateAuth helper in PageIterator
  - Run once per scanner, before page iteration. For `storage_state`: download the artifact via the existing `StorageProvider` and feed the path into context creation. For `form`: open an internal page, replay steps via `executePreScanActions`, wait for the recipe's success strategy, then discard the page. Emit an `auth_hydrated` event with method and post-login URL to the scan stage log.
  - Rationale: hydrating once per scanner means every page reuses the authenticated context, and every scanner inherits it without code changes.
  - Affects: `services/scanner-runner/src/core/page-iterator.ts`, the existing stage log writer.
  - Dependencies: tasks 1, 2, 3.

- [ ] 5. Surface a typed AuthHydrationError as a critical scan failure
  - When storage state is missing or expired, when a form recipe's success selector never appears, or when a from_env reference is unresolved, emit a structured `auth-hydration-failed` issue at `severity: critical` in the unified report and skip downstream pages for that scanner.
  - Rationale: a broken auth recipe must fail fast and loud. Producing a clean axe report against the login page is the worst outcome the system can have.
  - Affects: `services/scanner-runner/src/core/page-iterator.ts`, the issue-id catalog under `libs/contracts/report/`.
  - Dependencies: task 4.

- [ ] 6. Add a stageflow auth capture CLI subcommand
  - New subcommand in `clients/cli` that launches a non-headless Chromium, lets the developer log in interactively, then calls Playwright's storageState method on exit and writes the result to a developer-specified path with restrictive default file permissions.
  - Rationale: this is the only flow that ever sees a real password, and it runs locally on the developer's machine. Captured state is the path of least resistance for one-off scans of a personal app.
  - Affects: `clients/cli/`, the CLI README.
  - Dependencies: none, can land in parallel with tasks 1 through 5.

- [ ] 7. Wire stageflow scan to accept auth-state and auth-recipe flags
  - Extend the existing scan subcommand with `--auth-state <path>` and `--auth-recipe <path>`. The CLI uploads the storage-state artifact under the job's MinIO prefix and inlines a recipe directly into the Provenance the orchestrator persists.
  - Rationale: closes the loop between the developer's captured session or recipe file and the runtime that hydrates it.
  - Affects: `clients/cli/`, the CLI's intake-payload builder, the platform-api handler that accepts auth artifacts.
  - Dependencies: tasks 1, 6.

- [ ] 8. Forward allow-listed credential env vars from orchestrator to scanner-runner
  - When launching the scanner-runner pod, the orchestrator inspects `Provenance.auth.form.steps` for `from_env` references and injects exactly those env vars (and only those) from its own host environment into the pod. Build the allow-list from the recipe; arbitrary passthrough is forbidden.
  - Rationale: keeps the credential boundary on the orchestrator host, never on the platform-api wire or in any persisted artifact.
  - Affects: `services/orchestrator/internal/orchestrator/`, the pod-launch builder.
  - Dependencies: tasks 2, 4.

- [ ] 9. Port failure-policy guards from legacy harness into ai-navigator
  - Add two inline guards to the loop in `services/scanner-runner/src/scanners/ai-navigator/agent.ts`: stop after N consecutive failed action attempts, and stop when N successive successful turns produced no observable URL or DOM signature change. Thresholds configurable on `AgentGoal`. Aim for under 50 added lines, no new abstractions.
  - Rationale: this is the single useful idea from `agent-harness-legacy/runtimes/job-automation/agent-harness/internal/engine/turn.go` that maps cleanly onto `ai-navigator`. Land it in the live codebase before deleting the snapshot so the idea outlives the directory.
  - Affects: `services/scanner-runner/src/scanners/ai-navigator/agent.ts`, the agent goal type.
  - Dependencies: none, independent improvement.

- [ ] 10. Add integration tests for both auth modes
  - Add tests under `services/scanner-runner/tests` that run the full pipeline against a fixture login app for both `storage_state` and `form` modes. Assert axe and Lighthouse run against the post-login DOM, not the login redirect, and assert no env var values appear in the stage log or unified report.
  - Rationale: locks in the behavior and makes credential leakage a build-failure rather than a review-time concern.
  - Affects: `services/scanner-runner/tests/`, a new fixture login app.
  - Dependencies: tasks 4, 5, 8.

- [ ] 11. Document the auth flow and threat model
  - Add a section to `docs/architecture/system.md` covering the contract change, the trust boundaries, and the storage-state retention rules. Update `clients/cli/README.md` with the capture and recipe workflows. Cross-link from `docs/design/authenticated-scanning.md`.
  - Rationale: the trust boundary only holds if it is documented in the place reviewers actually read, not only in a design note.
  - Affects: `docs/architecture/system.md`, `clients/cli/README.md`, `docs/design/authenticated-scanning.md`.
  - Dependencies: tasks 1 through 10.

## Verification Criteria

- A scan submitted with `--auth-state` against a target whose homepage redirects unauthenticated visitors produces an axe report containing at least one finding from the post-login DOM, distinct from any finding produced by the unauthenticated scan of the same target.
- A scan submitted with `--auth-recipe` referencing two env vars succeeds in CI, and a grep for either env var value across stored Provenance, unified report, scan stage log, and NATS event payloads returns zero matches.
- A scan whose recipe success strategy never matches finishes with an `auth-hydration-failed` issue at `severity: critical`, downstream pages for that scanner are skipped, and the post-login URL captured at failure is included in the issue metadata.
- The orchestrator log for an auth-enabled run shows exactly the env var names declared in the recipe being injected into the scanner-runner pod, and no other env vars from the orchestrator host.
- `bun run ci` passes in `services/scanner-runner` after all changes, including the new fixture-driven integration tests.
- Running an unauthenticated scan still works unchanged when `Provenance.auth` is absent, with no new fields appearing in stored Provenance for those runs.
- The `agent-harness-legacy/` directory no longer exists in the tree, and no CI job, script, or workspace file references it.

## Potential Risks and Mitigations

1. **Credential leakage through stored artifacts**
   Mitigation: never store resolved credential values in Provenance, the unified report, the scan stage log, or any NATS event. Only `from_env` references are persisted. Add an explicit redaction test on the stage log writer that fails the build if any allow-listed env var value appears in serialized output.

2. **Storage state outliving its intended scope**
   Mitigation: write captured storage state under the job's MinIO prefix subject to the existing scan artifact retention policy, never expose it through the Web UI's signed-URL surface, and document the lifecycle in the architecture doc with a retention test.

3. **Recipe drift breaks scans silently**
   Mitigation: treat hydration failure as a critical scan issue with a structured id, never as a soft skip. Surface the post-login URL in the auth_hydrated and failure events so a wrong success selector that lands on the login page is diagnosable from the log alone.

4. **Failure-policy port grows beyond its budget**
   Mitigation: cap the port at 50 added lines in `agent.ts` and reject any version that introduces new modules or abstractions. If the port exceeds the budget, drop it and revisit failure handling separately. The surrounding work does not depend on this port.

5. **Auth hydration becomes a flaky CI dependency**
   Mitigation: ship a fixture login app for the integration tests so CI never depends on a real authenticated target. Make the form-mode success wait strategy configurable per recipe so each project can tune for its own login latency.

6. **Deleting agent-harness-legacy removes context someone wanted**
   Mitigation: the directory is fully self-contained with no live imports, no CI step, and no workspace entry. Git history retains the snapshot, and the design doc summarizes what was evaluated and why each piece was kept or dropped.

## Alternative Approaches

1. **Per-page pre_scan_actions only**: keep the existing per-page action mechanism and instruct users to log in inside every page entry. Rejected because it duplicates the login N times per scanner, never shares cookies across scanners, and forces every scanner module to know about authentication.

2. **Pull the legacy Go agent harness into a Podman scanner pod**: would deliver authenticated browsing but add a second runtime, a process-boundary protocol, and a domain-specific tool model with no other consumer in the repo. Rejected; the same value lands with three small TypeScript changes inside the existing scanner runner.

3. **Punt to the ai-navigator scanner**: ask the LLM to log in. Rejected because authentication is a deterministic, reproducible problem that should not pay the latency, cost, or non-determinism tax of an agent loop. The agent stays reserved for open-ended exploration of the post-login app.

4. **Generic auth-wall detector first**: before authenticated scanning, port a generic detector that flags scans that hit a login redirect. Worth doing eventually, but provides diagnosis without resolution; users still cannot scan their app. Deferred to a follow-up plan once the deterministic auth path exists.

## Assumptions

- The platform-api already exposes a job-scoped artifact upload path for the storage-state file, or the existing intake payload upload route can be extended without a contract break.
- The orchestrator host stores credentials destined for scanner runs in its own environment, accessible at pod-launch time. If a more formal secrets backend is introduced later, the allow-list mechanism in task 8 forwards cleanly to it.
- Playwright remains the runtime for the scanner runner. The design relies on Playwright's native storage-state seam.

## Dependencies

- Existing `Provenance` contract pipeline in `libs/contracts/` and the generator under `devtools/scripts/precommit/`.
- Existing `StorageProvider` and MinIO integration in `libs/go/storage` and the scanner-runner's storage adapter.
- Existing `executePreScanActions` plumbing in `services/scanner-runner/src/core/browser-manager.ts`, which is the substrate the form-mode replay extends.

## Notes

- The plan deliberately omits the LinkedIn and Indeed auth-wall detector and the multi-target labeled-screenshot port from `agent-harness-legacy/`. Both are domain-specific to AlchemizeCV's job-board targets and add noise without a present consumer in StageFlow.
- The deletion of `agent-harness-legacy/` lands together with the failure-policy port (task 9). The port is the only piece of the legacy snapshot that survives, and it survives in the live codebase rather than as a frozen reference.
- A follow-up plan covers a generic auth-wall detector (login-redirect heuristics, captcha presence, generic 401 and 403 patterns) at [`docs/plans/2026-05-23-auth-wall-detector-v1.md`](./2026-05-23-auth-wall-detector-v1.md), with design context at [`docs/design/auth-wall-detector.md`](../design/auth-wall-detector.md).
