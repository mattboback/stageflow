# StageFlow Portfolio Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Leave StageFlow accurate, current, fully verified, release-ready, and presented as a professional senior full-stack portfolio project, with `main` as the only local and remote branch at completion.

**Architecture:** Preserve the existing product architecture and Calibrated Instrument interface. Make small evidence-driven dependency, copy, documentation, test, and metadata corrections; verify each independently; then merge through the protected branch workflow, publish CLI v0.4.0, and remove all non-`main` branches.

**Tech Stack:** Go 1.26.5, Node.js 22.x, Bun 1.3.8, TypeScript, React Router 7, Playwright, Vitest, Podman, GitHub Actions, GitHub CLI.

## Global Constraints

- Keep Node.js pinned to `22.x`; do not adopt Node 26 type packages.
- Keep the React Router runtime and `@react-router/dev` on the same major version; do not mix React Router 7 and 8.
- Do not adopt TypeScript 7 unless both contract packages generate successfully and the full repository gate passes; otherwise retain the current TypeScript 6 line.
- Podman with Compose support is the supported self-host runtime; do not claim Docker support.
- Only claim artifact formats that the current implementation emits; StageFlow currently documents HTML and JSON reports, not SARIF.
- Preserve the existing Calibrated Instrument visual language; targeted corrections are in scope, a redesign is not.
- Employer-facing material should position the project for senior full-stack roles using verifiable evidence, not unsupported marketing claims.
- The final release tag is `clients/cli/v0.4.0`.
- Final repository state must have a clean, up-to-date local `main`, remote `main`, no open pull requests, and no local or remote branches except `main`.

---

### Task 1: Dependency PR Triage And Consolidation

**Files:**
- Possible modify: `clients/web/package.json`
- Possible modify: `clients/web/bun.lock`
- Possible modify: `services/scanner-runner/package.json`
- Possible modify: `services/scanner-runner/bun.lock`
- Possible modify: `libs/contracts/report/package.json`
- Possible modify: `libs/contracts/report/bun.lock`
- Possible modify: `libs/contracts/scanner-manifest/package.json`
- Possible modify: `libs/contracts/scanner-manifest/bun.lock`

**Interfaces:**
- Consumes: the six open Dependabot PRs numbered 32 through 37 and current `origin/main`.
- Produces: compatible dependency updates committed on the cleanup branch, factual closure comments on superseded/incompatible PRs, and no remaining Dependabot branches.

- [ ] **Step 1: Record the current dependency and PR state**

Run `gh pr list --state open --json number,title,headRefName,url` and record the result in the task report.

- [ ] **Step 2: Evaluate each proposed update against the global constraints**

Use package release notes and peer/engine metadata to classify PRs 32-37. React Router 8 tooling paired with runtime 7 and Node 26 types paired with Node 22 are incompatible by policy. TypeScript 7 must satisfy generation plus full package checks before acceptance.

- [ ] **Step 3: Trial compatible candidates one at a time**

Regenerate the affected Bun lockfile without changing unrelated workspaces, then run the affected workspace gate. For web changes run `cd clients/web && bun run ci`; for scanner-runner changes run `cd services/scanner-runner && bun run ci`; for contract changes run the repository contract-generation command plus the package's own scripts.

- [ ] **Step 4: Keep only green, policy-compatible updates**

Commit retained package and lockfile changes. Restore rejected trial changes before closing the corresponding PR.

- [ ] **Step 5: Close superseded or incompatible Dependabot PRs and delete their branches**

Use a concise closure comment that states the exact pin or compatibility reason. Confirm `gh pr list --state open` has no Dependabot PRs and `git ls-remote --heads origin` has no `dependabot/` heads.

- [ ] **Step 6: Commit retained updates**

Commit message: `chore(deps): consolidate compatible updates`.

---

### Task 2: Product Truth And Portfolio Documentation

**Files:**
- Modify: `clients/web/app/routes/home.tsx:309-349`
- Modify: `clients/web/app/root.tsx:45-53`
- Modify: `clients/web/app/routes/not-found.tsx`
- Modify: `clients/web/README.md:14-26`
- Modify: `README.md:13-21,65-81`
- Modify: `docs/design.md:31-49,127-135,182-214`
- Modify: `docs/evaluators-guide.md`

**Interfaces:**
- Consumes: current web behavior, report navigation labels, emitted artifact formats, and canonical CSS tokens in `clients/web/app/styles/instrument.css`.
- Produces: one consistent, evidence-based senior full-stack project narrative across the site and repository.

- [ ] **Step 1: Add or update assertions for employer-facing copy**

Cover the homepage runtime/artifact claims and 404 title in Playwright tests before changing the copy. Assertions must reject the stale phrases `Docker-based`, `SARIF`, and `Channel not found`.

- [ ] **Step 2: Correct homepage infrastructure and artifact claims**

Describe rootless Podman isolation and HTML/JSON report ownership. Replace `run anywhere` with a statement scoped to supported self-host infrastructure.

- [ ] **Step 3: Correct error terminology**

Use `Page not found` for 404 responses in both the route metadata and root error boundary. Keep non-404 wording calm and product-appropriate.

- [ ] **Step 4: Update web-client persistence documentation**

State that server data remains API-owned while human-review decisions are stored locally in browser `localStorage` per job.

- [ ] **Step 5: Align README and evaluator guidance with current report navigation**

Replace obsolete Pages/Overview tab instructions with the current Review, Issues, and Artifacts workflow while retaining valid schema-level `pageOverview` terminology where it names a contract field.

- [ ] **Step 6: Align the design guide with canonical CSS**

Document Gabarito as the display face, Source Sans 3 as the body/interface face, and JetBrains Mono as the data face, matching `instrument.css` and `root.tsx`.

- [ ] **Step 7: Run focused checks**

Run the new Playwright specs plus `cd clients/web && bun run ci`.

- [ ] **Step 8: Commit**

Commit message: `docs: align portfolio story with current product`.

---

### Task 3: Stale-Claim Regression Guards

**Files:**
- Modify: `devtools/scripts/check-stale-vocab.sh`
- Create: `devtools/scripts/tests/stale-vocab.test.sh`
- Modify: `justfile`
- Modify: `.github/workflows/ci.yml` only if the existing shell-test entry point does not already execute the new test

**Interfaces:**
- Consumes: active Markdown, TypeScript, TSX, CSS, YAML, shell, Go, TOML, and JSON source files.
- Produces: a deterministic stale-vocabulary check that catches the corrected active-product claims without rejecting intentional migration history in `CHANGELOG.md`.

- [ ] **Step 1: Write a failing shell regression test**

The test must prove a clean fixture passes and fixtures containing `Docker-based, rootless containers`, `Artifacts you own: HTML, JSON, SARIF`, or `Channel not found` fail with a `STALE:` diagnostic.

- [ ] **Step 2: Run the shell test and confirm it fails before implementation**

Run `bash devtools/scripts/tests/stale-vocab.test.sh`; expected result is non-zero because the checker cannot yet scan the fixture/source types required by the test.

- [ ] **Step 3: Extend the checker minimally**

Add TypeScript, TSX, and CSS to active-source scanning, support an explicit test root, and exclude historical changelog prose from new active-product phrase rules.

- [ ] **Step 4: Wire the regression test into the existing shell-test entry point**

Use the existing `just shell-tests` convention rather than creating a parallel CI path.

- [ ] **Step 5: Verify**

Run `bash devtools/scripts/tests/stale-vocab.test.sh`, `./devtools/scripts/check-stale-vocab.sh`, and `just shell-tests`; all must pass.

- [ ] **Step 6: Commit**

Commit message: `test: guard employer-facing product claims`.

---

### Task 4: Browser QA And Targeted UI Hardening

**Files:**
- Modify: only `clients/web/app/**` files required by reproducible findings
- Test: `clients/web/e2e/*.spec.ts`
- Evidence: ignored files under `output/portfolio-qa/`

**Interfaces:**
- Consumes: the production-like preview build, mocked report fixture, and StageFlow's existing visual language.
- Produces: verified desktop/mobile homepage, playground, scan, report, and 404 flows with tests for every source fix.

- [ ] **Step 1: Build and serve the production preview**

Run the app through the existing Playwright/Vite preview setup with the committed contract fixture and mocked API.

- [ ] **Step 2: Audit desktop and mobile flows**

Inspect homepage, scan configuration, running status, report Review/Issues/Artifacts, modal review decisions, and 404 behavior. Capture screenshots and record console errors, page errors, and failed requests under `output/portfolio-qa/`.

- [ ] **Step 3: Audit accessibility and keyboard behavior**

Check semantic headings, form labels/errors, focus order, visible focus, modal focus containment/escape, mobile touch targets, and reduced-motion behavior.

- [ ] **Step 4: Add a failing test for each reproducible source defect**

Do not change source for subjective preferences or redesign opportunities. Every source change must correspond to a functional, accessibility, responsive, console, or factual defect reproduced in the preview.

- [ ] **Step 5: Implement minimal fixes and rerun focused tests**

Preserve existing tokens, components, density, and responsive conventions.

- [ ] **Step 6: Run the complete web gate**

Run `cd clients/web && bun run ci && bun run test:e2e`.

- [ ] **Step 7: Commit if source changes were required**

Commit message: `fix(web): harden portfolio-facing flows`.

---

### Task 5: CLI v0.4.0 Release Preparation And Full Verification

**Files:**
- Modify: `CHANGELOG.md`
- Verify: `.github/workflows/release-stageflow-cli.yml`
- Verify: all files changed by Tasks 1-4

**Interfaces:**
- Consumes: all prior task commits and the release workflow's `clients/cli/v*` tag contract.
- Produces: release-ready `main` content for `clients/cli/v0.4.0` and complete local verification evidence.

- [ ] **Step 1: Convert current Unreleased CLI work into a 0.4.0 entry**

Add `## [0.4.0] - 2026-07-14`, preserve the existing Added/Changed/Fixed/Removed content under it, and leave a fresh empty `## [Unreleased]` section above it.

- [ ] **Step 2: Verify release workflow semantics**

Confirm the workflow builds Linux and macOS amd64/arm64 plus Windows amd64 archives, stamps `v0.4.0`, generates checksums, and creates a GitHub release from `clients/cli/v0.4.0` without source changes unless a defect is found.

- [ ] **Step 3: Run repository quality gates**

Run `just ci`. Run `just project-golden` if the local Podman prerequisites are available; otherwise the task is blocked until the golden regression is executed successfully in GitHub Actions.

- [ ] **Step 4: Confirm repository cleanliness**

Run `git status --short`, inspect the complete branch diff, and verify no generated or QA evidence files are tracked.

- [ ] **Step 5: Commit**

Commit message: `chore(release): prepare CLI v0.4.0`.

---

## Post-Review Integration And Repository Completion

After all five tasks pass task review and the broad whole-branch review is clean:

1. Push `cleanup/portfolio-hardening`, open a focused PR, wait for all CI and Golden Regression checks, and merge it.
2. Update GitHub topics by removing `sveltekit` and adding `react-router` and `typescript` while retaining the accurate existing topics.
3. Update the `Protect main` ruleset so all existing CI jobs required for pull requests must pass, while preserving deletion and non-fast-forward protection.
4. Create and push the annotated tag `clients/cli/v0.4.0` from the verified merge commit.
5. Wait for the release workflow and verify five platform archives, `checksums.txt`, the stamped release version, and successful workflow conclusion.
6. Smoke-test `https://stageflow.org` on desktop and mobile after deployment, including one no-account scan/report path.
7. Close any remaining pull requests, delete every remote branch except `main`, prune remote-tracking refs, remove the temporary worktree, switch the primary checkout to `main`, fast-forward it, and delete every local branch except `main`.
8. Prove completion with `git status --short --branch`, `git branch --all`, `git ls-remote --heads origin`, `gh pr list --state open`, and `gh release view clients/cli/v0.4.0`.
