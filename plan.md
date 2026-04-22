Implementation Plan — StageFlow Open-Source Readiness Polish

Problem Statement:

StageFlow is a well-built distributed scanning platform being prepared for open-source as a portfolio piece. A comprehensive audit identified
correctness bugs (race conditions), structural debt (flat CLI package, dead code), UI defects (duplicate screenshots), and normalization gaps
(arbitrary scoring, incomplete dedup) that a senior reviewer would catch in a 15-minute skim.

Requirements:

- Fix correctness/race condition bugs in NATS client and orchestrator DB layer
- Restructure the CLI from a flat 38-file package main into proper internal packages
- Fix the duplicate screenshot rendering in the issue detail modal
- Document and improve the scoring formula and cross-scanner dedup
- Clean up dead code, entry point logging, and repo hygiene
- Refactor the Lighthouse scanner's scanPage method and remove dead delegation methods
- All changes must pass existing CI (go test -race ./..., golangci-lint run, bun run ci)

Background:

Based on a full codebase audit covering all services, libs, clients, infra, and docs. The codebase already has strong CI, good test coverage, proper
security patterns (SSRF, constant-time auth, rootless containers), and clean architecture. The issues found are polish-level except for two genuine
race conditions.

Proposed Solution:

9 tasks organized by severity: correctness fixes first, then structural refactors, then UI/normalization fixes, then polish. Each task is
independently mergeable and leaves CI green.

─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

Task Breakdown:

Task 1: Fix race condition in `RecordScannerCompletion` — missing `SELECT ... FOR UPDATE`

- Objective: Prevent lost updates when two scanner completion events arrive concurrently for the same job.
- Current bug: RecordScannerCompletion in services/orchestrator/internal/adapters/repository/job_scanners.go begins a transaction and reads
  completed_scanners without row-level locking. Two concurrent handlers can both read completed_scanners = ["axe"], both append their scanner, and one
  write overwrites the other.
- Implementation:
  - Change the SELECT inside RecordScannerCompletion to SELECT ... FOR UPDATE so the row is locked for the duration of the transaction.
  - The query at ~line 68 (SELECT expected_scanners, completed_scanners, scanner_results FROM jobs WHERE id = ?) becomes SELECT expected_scanners,
    completed_scanners, scanner_results FROM jobs WHERE id = ? FOR UPDATE.
  - Note: bindPostgresParams converts ? to $1 etc., so the FOR UPDATE clause just appends to the existing query string.

- Test requirements:
  - Add a test in services/orchestrator/internal/adapters/repository/jobs_scanners_test.go that launches two goroutines calling
    RecordScannerCompletion concurrently for different scanner types on the same job. Assert both scanners appear in completed_scanners and allComplete
    is true only on the second call.
  - Existing go test -race ./services/orchestrator/... must pass.

- Demo: Run go test -race -run TestConcurrentScannerCompletion ./services/orchestrator/internal/adapters/repository/... — both scanners recorded, no
  lost update.

Task 2: Fix data race in NATS `Client.Close()` vs `ensureReady()`/`Publish()`

- Objective: Eliminate the TOCTOU race where ensureReady() reads c.nc/c.js without the mutex while Close() nils them under the mutex.
- Current bug: In libs/go/messaging/client.go, ensureReady() (line ~117) checks c.nc and c.js without holding c.mu. Close() sets them to nil under
  c.mu. A concurrent Publish() can pass ensureReady(), then Close() runs, then Publish dereferences nil c.js.
- Implementation:
  - Add a closed boolean field to Client protected by c.mu.
  - In Close(), set c.closed = true under the mutex before stopping consumers and closing the connection.
  - In ensureReady(), acquire c.mu.Lock(), check c.closed, then release. This is a brief lock — no contention concern since ensureReady is called
    per-publish and the lock body is just a bool check.
  - Alternative: use atomic.Bool for closed to avoid mutex contention entirely.

- Test requirements:
  - Add a test in libs/go/messaging/nats_client_test.go that calls Publish and Close concurrently from multiple goroutines. Run with -race. Assert no
    race detected and Publish after Close returns ErrNotConnected.
  - All existing messaging tests must pass.

- Demo: go test -race ./libs/go/messaging/... passes with no race warnings.

Task 3: Restructure CLI from flat `package main` into internal packages

- Objective: Move the 38-file flat package main into domain-oriented internal packages so the CLI demonstrates clean Go project structure.
- Current state: clients/cli/ has 38 non-test .go files (6047 lines) all in package main. Only 3 small packages exist in internal/.
- Implementation — extract these packages into clients/cli/internal/:
  - internal/apiclient/ — client.go, client_projects.go, types.go (HTTP client and API types)
  - internal/sse/ — sse.go (SSE streaming and polling logic)
  - internal/reportfmt/ — report_output.go, report_output_markdown.go, report_flags.go, scanners_output.go, scan_output.go, filter.go,
    output_format.go (all output formatting)
  - internal/projectmode/ — project_bootstrap.go, project_config.go, project_doctor.go, project_init.go, project_root.go, project_run.go (project
    mode domain logic)
  - internal/devstack/ — dev_stack.go (local dev stack management)
  - Move corresponding \*\_test.go files with each package.
  - cobra\_\*.go files stay in package main — they're thin wiring that calls into internal packages.
  - main.go, run.go, constants.go, version.go, cli_errors.go, cli_helpers.go, time_helpers.go stay in package main.

- Approach: Move files one package at a time. After each move, run go build ./clients/cli/... and go test ./clients/cli/... to catch import issues
  immediately. Export only what the cobra files need.
- Test requirements:
  - go build ./clients/cli/... succeeds.
  - go test -race ./clients/cli/... passes (all existing tests).
  - bash devtools/scripts/tests/cli-install.test.sh passes.
  - Each new internal package has at least one test file.

- Demo: find clients/cli -maxdepth 1 -name '_.go' -not -name '_\_test.go' | wc -l returns ≤15 (down from 38). go test -race ./clients/cli/... green.

Task 4: Refactor Lighthouse scanner — remove dead code and extract `scanPage` timeout wrappers

- Objective: Remove 6 dead private delegation methods and extract inline timeout wrappers from scanPage to reduce index.ts from 389 to ~250 lines.
- Dead methods to remove (never called — confirmed by grep):
  - getAuditNodeCount, extractAuditNodes, enrichNodesWithContext, getAuditCategory, mapScoreToSeverity, getHelpUrl

- scanPage refactor:
  - Extract the enrichment-with-timeout pattern into a private enrichWithTimeout(page, issues) method.
  - Extract the screenshot-with-timeout pattern into a private captureScreenshotWithTimeout(page, violations, dir, pageId) method.
  - Extract the re-navigation-after-lighthouse block into a private renavigateAfterLighthouse(page, url) method.

- Test requirements:
  - bun run ci in services/scanner-runner passes unchanged — no test edits needed.
  - wc -l services/scanner-runner/src/scanners/lighthouse/index.ts returns ≤260.

- Demo: bun run ci green in services/scanner-runner. File is under 260 lines.

Task 5: Fix duplicate screenshot rendering in issue detail modal

- Objective: When an issue is selected, show the bounding-box screenshot once, not twice with different visual contexts.
- Current bug: IssueEvidenceSection.svelte renders the full-page overview with bounding boxes for all occurrences. Then each
  IssueOccurrenceCard.svelte renders a cropped version of the same screenshot for each individual occurrence. The user sees the same element
  highlighted twice.
- Implementation:
  - In IssueEvidenceSection.svelte: when the full-page overview is shown with bounding boxes, pass a prop showOccurrenceThumbnails={false} (or
    equivalent) to signal that occurrence cards should NOT render their own cropped screenshots.
  - In IssueDetailModal.svelte: pass a hideOccurrenceThumbnails prop to the occurrences block when the full-page evidence section is visible and
    rendering the page overview.
  - In IssueOccurrenceCard.svelte: accept and respect a hideThumbnail prop. When true, skip the SVG crop rendering entirely — show only the text
    details (selector, HTML, failure summary).
  - This way: full-page overview shows all bounding boxes in one view, and occurrence cards show only text details. Clicking a bounding box in the
    overview scrolls to the matching occurrence card (existing behavior preserved).

- Test requirements:
  - Existing Storybook interaction tests pass (bun run test-storybook:ci in clients/web).
  - Manual verification: open an issue with multiple occurrences → see ONE screenshot with bounding boxes, not duplicated per occurrence.

- Demo: Open a report, click an issue → single screenshot with bounding boxes, occurrence cards show text only.

Task 6: Document scoring formula and make weights configurable

- Objective: Replace magic-number severity weights with named constants and add a doc comment explaining the rationale.
- Current state: calculateAccessibilityScore in services/orchestrator/internal/adapters/storage/report_aggregator_utils.go uses critical*10 +
  serious*5 + moderate*2 + minor*1 with scaled = 20*log10(penalty+1) + penalty*0.3. No documentation.
- Implementation:
  - Extract weights into named constants: scorePenaltyCritical = 10, scorePenaltySerious = 5, scorePenaltyModerate = 2, scorePenaltyMinor = 1.
  - Add a doc comment on calculateAccessibilityScore explaining: the formula is a logarithmic penalty curve that compresses high issue counts (so 100
    minor issues don't score worse than 5 critical), the weights reflect WCAG impact tiers, and the grade thresholds use standard academic grading.
  - Add a comment noting this is a heuristic, not a WCAG-defined metric, and link to the WCAG severity definitions for context.

- Test requirements:
  - Existing go test ./services/orchestrator/... passes.
  - Add a table-driven test for calculateAccessibilityScore with edge cases: zero issues → 100/A+, 1 critical → verify score, 100 minor → verify
    score is higher than 5 critical.

- Demo: go test -run TestCalculateAccessibilityScore ./services/orchestrator/internal/adapters/storage/... passes with documented edge cases.

Task 7: Expand cross-scanner deduplication mapping and add test coverage

- Objective: Add missing rule equivalences and add test coverage for the dedup logic.
- Current state: ruleEquivalences in rule_deduplication.go covers ~15 canonical rules. Lighthouse accessibility audits not in the map still duplicate
  with axe.
- Implementation:
  - Audit the full list of Lighthouse accessibility audits (from issue-mapper.ts accessibilityAudits map — 30+ rules) against axe-core rule IDs. Add
    missing equivalences for: aria-allowed-attr, aria-hidden-body, aria-hidden-focus, aria-required-attr, aria-roles, aria-valid-attr-value,
    aria-valid-attr, bypass, duplicate-id-aria, form-field-multiple-labels, html-lang-valid, input-image-alt, list, listitem, object-alt, tabindex,
    td-headers-attr, th-has-data-cells, valid-lang, video-caption.
  - Add a comment at the top of the equivalences map explaining the maintenance contract: when adding a new scanner that overlaps with
    axe/lighthouse, add entries here.

- Test requirements:
  - Add a test TestDeduplicateIssues_CrossScanner that creates issues from both axe and lighthouse for the same rule on the same page, runs
    deduplicateIssues, and asserts only the axe version survives (higher priority) with alsoDetectedBy: ["lighthouse"].
  - Add a test TestDeduplicateIssues_NoFalsePositive that creates issues from different scanners with different rule IDs and asserts no dedup occurs.
  - Existing tests pass.

- Demo: go test -run TestDeduplicateIssues ./services/orchestrator/internal/adapters/storage/... passes.

Task 8: Fix `os.Getenv` per-event in orchestrator `newService()` and clean up entry point logging

- Objective: Read env vars once at startup instead of per-event, and route the scanner-runner entry point through the project logger.
- Implementation:
  - In services/orchestrator/internal/orchestrator/service_adapters.go: the newService() method reads os.Getenv("OPENROUTER_API_KEY") etc. on every
    call. Instead, store these values as fields on the Orchestrator struct (set once in NewOrchestrator from the Config). The config already has these
    values passed from main.go.
  - In services/scanner-runner/src/index.ts: replace console.error(...) on line 11 with createLogger('main').error(...) to match the project's
    logging convention.

- Test requirements:
  - go test -race ./services/orchestrator/... passes.
  - bun run ci in services/scanner-runner passes.

- Demo: grep -n 'os.Getenv' services/orchestrator/internal/orchestrator/service_adapters.go returns no matches. grep -n 'console\.'
  services/scanner-runner/src/index.ts returns no matches.

Task 9: Repo hygiene — remove `plan/`, clean `DEFAULT_GITHUB_URL`, final verification

- Objective: Remove local-only artifacts and hardcoded references before going public.
- Implementation:
  - Delete the plan/ directory from disk (it's already gitignored, so this is just local cleanup — but confirm it's not tracked with git status
    plan/).
  - In clients/web/src/lib/config/site.ts: the DEFAULT_GITHUB_URL constant on line 3 hardcodes https://github.com/mattboback/stageflow. This is
    actually correct for the public repo URL — it's the project's real GitHub URL, not a leak. However, the normalizeGithubUrl function is overly
    defensive for what it does. Simplify: if VITE_GITHUB_URL is set and valid, use it; otherwise use the default. Remove the try/catch URL parsing since
    the default is a known-good URL.
  - Verify all doc links in README resolve (spot-check the 10 links in the docs table).
  - Verify the evaluator's guide still matches current code after all refactors (specifically: CLI section should mention the new internal package
    structure, Lighthouse section should reflect the cleaned-up file).

- Test requirements:
  - bun run ci in clients/web passes.
  - git status shows no untracked files that should be gitignored.
  - Manual: open README in a markdown previewer, verify all links resolve.

- Demo: grep -rn 'plan/' .gitignore confirms it's ignored. All README doc links resolve. Evaluator's guide is accurate.

  Implementation Plan — StageFlow Open-Source Readiness Polish (Revised)

  Problem Statement:

  StageFlow has two genuine race conditions, a UI defect that shows duplicate screenshots, incomplete cross-scanner deduplication, an undocumented
  scoring heuristic, and scattered code-quality issues. These are the things a senior reviewer would catch in a 15-minute skim of a portfolio repo.

  Requirements:
  - Fix correctness bugs first (race conditions)
  - Fix visible UX defects (duplicate screenshots)
  - Improve normalization quality (dedup, scoring docs)
  - Clean up code-quality issues (env reads, dead wrappers, logging)
  - CLI restructure deferred to a follow-up PR — too large for a polish pass
  - All changes must leave CI green (go test -race ./..., golangci-lint run, bun run ci)

  Proposed Solution:

  8 tasks in priority order. Each is independently mergeable. Correctness → UX → normalization → code quality → hygiene.

  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

  Task 1: Fix race condition in `RecordScannerCompletion` — add `SELECT ... FOR UPDATE`
  - Objective: Prevent lost updates when two scanner completion events arrive concurrently for the same job.
  - Where: services/orchestrator/internal/adapters/repository/job_scanners.go, the SELECT at ~line 68 inside RecordScannerCompletion.
  - Bug: The transaction reads completed_scanners and scanner_results without a row lock. Two concurrent handlers can both read the same state, both
    append, and one overwrites the other.
  - Fix: Change SELECT expected_scanners, completed_scanners, scanner_results FROM jobs WHERE id = ? to SELECT expected_scanners, completed_scanners,
    scanner_results FROM jobs WHERE id = ? FOR UPDATE. This acquires a row-level lock for the transaction duration, serializing concurrent completions.
  - Test: The existing TestRecordScannerCompletionConcurrency in jobs_scanners_test.go already launches two goroutines and asserts exactly one
    allComplete and both scanners persisted. After the fix, strengthen it: run 10 iterations in a loop to increase the chance of hitting the race window.
    Verify the test was actually failing before the fix by temporarily removing FOR UPDATE and running with -count=20.
  - Completion criteria:
    - grep -n 'FOR UPDATE' services/orchestrator/internal/adapters/repository/job_scanners.go returns a match.
    - go test -race -count=20 -run TestRecordScannerCompletionConcurrency ./services/orchestrator/internal/adapters/repository/... passes all 20 runs.
    - go test -race ./services/orchestrator/... green.

  - Demo: The concurrency test passes reliably under -count=20 -race.

  Task 2: Fix data race in NATS `Client.Close()` vs `ensureReady()`/`Publish()`
  - Objective: Eliminate the TOCTOU race where ensureReady() reads c.nc/c.js without synchronization while Close() mutates and nils them.
  - Where: libs/go/messaging/client.go — ensureReady() (~line 117), Close() (~line 133), and all callers of ensureReady() in publish.go and
    subscribe.go.
  - Bug: ensureReady() checks c.nc and c.js without holding c.mu. After ensureReady() returns, Close() can nil out c.js before Publish() uses it.
  - Fix — snapshot pattern:
    - Add a closed bool field to Client.
    - Add a method func (c *Client) snapshot() (*nats.Conn, jetstream.JetStream, error) that acquires c.mu, checks c.closed, copies c.nc and c.js into
      local variables, releases the lock, and returns the copies (or ErrNotConnected).
    - Replace ensureReady() calls in Publish(), PublishEvent(), Subscribe(), and SubscribeWithContext() with snapshot(). Use the returned js/nc locals
      for the rest of the method — never touch c.js/c.nc directly after the snapshot.
    - In Close(): acquire c.mu, set c.closed = true, copy consumeContexts into a local, clear c.consumeContexts, copy c.nc into a local, set c.nc = nil
      and c.js = nil, release the lock. Then stop consume contexts and close nc outside the lock.
    - Delete ensureReady() entirely — it's replaced by snapshot().

  - Test: Add a test in libs/go/messaging/nats_client_test.go that spawns 10 goroutines calling Publish in a loop while another goroutine calls
    Close(). Run with -race. Assert no race detected and that Publish after Close returns an error.
  - Completion criteria:
    - grep -n 'ensureReady' libs/go/messaging/ returns no matches.
    - go test -race ./libs/go/messaging/... passes.
    - go test -race ./... passes (all services that import messaging).

  - Demo: go test -race -count=5 ./libs/go/messaging/... — no race warnings.

  Task 3: Fix duplicate screenshot rendering in issue detail modal
  - Objective: When viewing an issue, show the bounding-box evidence once — not twice in different visual forms.
  - Where: clients/web/src/lib/components/report/IssueDetailModal.svelte and IssueOccurrenceCard.svelte.
  - Bug: IssueDetailModal renders IssueEvidenceSection (which shows the full-page overview with highlight boxes for all occurrences), then separately
    renders IssueOccurrenceCard for each occurrence (which renders a cropped version of the same screenshot). Same evidence shown twice.
  - Fix: In IssueDetailModal.svelte, when the evidenceBlock snippet is rendered and the page overview is visible (shouldShowPageOverview &&
    pageOverviewRenderable), pass hideThumbnail={true} to each IssueOccurrenceCard in the occurrencesBlock snippet. In IssueOccurrenceCard.svelte, accept
    a hideThumbnail?: boolean prop and skip the SVG crop block when true. The text details (selector, HTML context, failure summary) still render.
  - Test: Extend the existing clients/web/tests/unit/components/report/IssueDetailModal.test.ts:
    - Add a test case that renders the modal with page overview data present, and asserts that occurrence cards do NOT contain an SVG element (the
      cropped screenshot).
    - Add a test case that renders the modal WITHOUT page overview data, and asserts occurrence cards DO contain the SVG crop (fallback behavior
      preserved).

  - Completion criteria:
    - bun run ci in clients/web passes.
    - The two new test cases pass.
    - Manual: open a report issue with multiple occurrences → one full-page screenshot with boxes, occurrence cards show text only.

  - Demo: bun run test in clients/web — new tests green.

  Task 4: Wire OpenRouter env vars through orchestrator config and clean scanner-runner entry logging
  - Objective: Read OPENROUTER\_\* env vars once at startup instead of per-event, and route the scanner-runner entry point through the project logger.
  - Where:
    - services/orchestrator/cmd/orchestrator/config.go — add OpenRouterAPIKey, OpenRouterAppTitle, OpenRouterAppReferer fields to Config.
    - services/orchestrator/cmd/orchestrator/main.go — pass the new config fields into orchestrator.Config.
    - services/orchestrator/internal/orchestrator/orchestrator.go — add corresponding fields to Orchestrator struct and Config struct, populate in
      NewOrchestrator.
    - services/orchestrator/internal/orchestrator/service_adapters.go — replace os.Getenv("OPENROUTER_API_KEY") etc. with o.openRouterAPIKey etc.
    - services/scanner-runner/src/index.ts — replace console.error(...) on line 11 with createLogger('main').error(...).

  - Test:
    - go test -race ./services/orchestrator/... passes.
    - bun run ci in services/scanner-runner passes.
    - Verify: grep -rn 'os.Getenv' services/orchestrator/internal/orchestrator/ returns no matches.

  - Completion criteria:
    - grep -n 'os.Getenv' services/orchestrator/internal/orchestrator/service_adapters.go returns no matches.
    - grep -n 'console\.' services/scanner-runner/src/index.ts returns no matches.
    - go test -race ./services/orchestrator/... green.
    - bun run ci in services/scanner-runner green.

  - Demo: Both greps return no matches. Tests green.

  Task 5: Expand cross-scanner deduplication mappings and extend existing tests
  - Objective: Add missing axe↔lighthouse rule equivalences so the dedup map covers the full overlap surface.
  - Where: services/orchestrator/internal/adapters/storage/rule_deduplication.go and rule_deduplication_test.go.
  - Current state: ruleEquivalences covers ~15 canonical rules. The accessibilityAudits map in issue-mapper.ts lists 30+ Lighthouse accessibility
    audits, most of which share IDs with axe-core.
  - Fix: Add equivalences for the remaining shared rules: aria-allowed-attr, aria-hidden-body, aria-hidden-focus, aria-required-attr, aria-roles,
    aria-valid-attr-value, aria-valid-attr, bypass, duplicate-id-aria, form-field-multiple-labels, html-lang-valid, input-image-alt, list, listitem,
    object-alt, tabindex, td-headers-attr, th-has-data-cells, valid-lang, video-caption. Add a maintenance comment at the top of the map.
  - Test: Extend the existing test suite in rule_deduplication_test.go:
    - Add TestDeduplicateIssues_AriaRuleDuplicates — axe and lighthouse both flag aria-required-attr on the same page → only axe version survives.
    - Add TestDeduplicateIssues_AllEquivalencesHaveCanonicalID — iterate ruleEquivalences, verify every value appears at least twice (i.e., every
      canonical ID has at least two scanner-prefixed entries).

  - Completion criteria:
    - go test -run TestDeduplicateIssues ./services/orchestrator/internal/adapters/storage/... passes.
    - len(ruleEquivalences) increases from current count to cover the 20 additional rules (~40 new entries).

  - Demo: New tests pass. grep -c '=>' rule_deduplication.go shows expanded mapping count.

  Task 6: Document scoring heuristic and extract named constants
  - Objective: Make the accessibility score formula reviewable by extracting magic numbers into named constants and adding a clear doc comment.
  - Where: services/orchestrator/internal/adapters/storage/report_aggregator_utils.go, calculateAccessibilityScore function.
  - Fix:
    - Extract const scorePenaltyCritical = 10, scorePenaltySerious = 5, scorePenaltyModerate = 2, scorePenaltyMinor = 1.
    - Extract const scoreLogCoefficient = 20.0, scoreLinearCoefficient = 0.3.
    - Add a doc comment explaining: this is a heuristic penalty curve, not a WCAG-defined metric. The log term compresses high issue counts so 100
      minor issues don't dominate. The weights reflect relative WCAG impact. The grade thresholds use standard academic grading. Link to WCAG severity
      definitions.
    - Do NOT make weights configurable — constants and docs are sufficient for OSS polish.

  - Test: Add TestCalculateAccessibilityScore table-driven test with cases: zero issues → (100, "A+"), 1 critical → score < 100, 100 minor issues →
    score higher than 5 critical issues, all-zero-except-info → (100, "A+").
  - Completion criteria:
    - go test -run TestCalculateAccessibilityScore ./services/orchestrator/internal/adapters/storage/... passes.
    - grep -c 'scorePenalty' services/orchestrator/internal/adapters/storage/report_aggregator_utils.go returns ≥4.
    - The function has a multi-line doc comment.

  - Demo: Test passes. Constants and doc comment visible in the file.

  Task 7: Lighthouse scanner — extract `scanPage` helpers and remove dead class wrappers
  - Objective: Reduce index.ts complexity by extracting timeout-wrapped operations from scanPage and removing private class methods that are never
    called via this..
  - Where: services/scanner-runner/src/scanners/lighthouse/index.ts.
  - Dead class wrappers (confirmed: no this. call sites): getAuditNodeCount, extractAuditNodes, getAuditCategory, mapScoreToSeverity, getHelpUrl. These
    are private methods that delegate to the same-named functions in issue-mapper.ts and result-parser.ts but are never invoked. Note:
    enrichNodesWithContext is also dead (no this.enrichNodes call). The underlying functions in the extracted modules are alive and used — only the class
    wrappers are dead.
  - scanPage extraction: extract three private methods:
    - renavigateAfterLighthouse(page, url) — the try/catch re-navigation block (~15 lines).
    - enrichWithTimeout(page, issues) — the Promise.race enrichment pattern (~15 lines).
    - captureScreenshotWithTimeout(page, violations, dir, pageId) — the Promise.race screenshot pattern (~20 lines).

  - Test: bun run ci in services/scanner-runner passes unchanged — no test edits needed. This is a pure refactor.
  - Completion criteria:
    - bun run ci in services/scanner-runner green.
    - grep -c 'private.\*Fn\b' services/scanner-runner/src/scanners/lighthouse/index.ts returns 0 (no more Fn delegation wrappers).
    - The scanPage method body is under 50 lines.

  - Demo: bun run ci green. scanPage is concise orchestration only.

  Task 8: Repo hygiene checklist
  - Objective: Final cleanup before going public.
  - Items:
    - Delete plan/ directory from disk (already gitignored — verify with git status plan/ that it's not tracked).
    - In clients/web/src/lib/config/site.ts: simplify normalizeGithubUrl — remove the try/catch URL parsing since the default is a known-good constant.
      If VITE_GITHUB_URL is set and non-empty, use it; otherwise use DEFAULT_GITHUB_URL.
    - Spot-check the 10 doc links in the README docs table — verify they resolve.
    - Update the evaluator's guide CLI section if the internal package structure changed in a follow-up PR.

  - Completion criteria:
    - ls plan/ returns "No such file or directory".
    - bun run ci in clients/web passes.
    - All 10 README doc links resolve (manual check).

  - Demo: Clean working tree, all links work.

  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────

  Does this plan look good, or would you like me to adjust anything?

▸ Credits: 13.05 • Time: 2m 31s───────────────────────────────────────────────────────────────────
