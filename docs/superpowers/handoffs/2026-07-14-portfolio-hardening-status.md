# StageFlow Portfolio Hardening Status

## Fast Start

Worktree: `/tmp/opencode/stageflow-portfolio-hardening`

Branch: `cleanup/portfolio-hardening`

Base: `e8e42bd` (`origin/main`)

Detailed plan: `docs/superpowers/plans/2026-07-14-portfolio-hardening.md`

Current branch is clean and four commits ahead of `origin/main`.

## Completed Work

### 1. Dependency cleanup

Commit: `1079b9b chore(deps): consolidate compatible updates`

- Accepted `sharp` 0.35.3.
- Accepted TypeScript 7.0.2 in both contract packages after generation, package checks, and `just ci` passed.
- Rejected Node 26 types because the repository targets Node 22.
- Rejected React Router 8 tooling because the runtime remains React Router 7.
- Closed Dependabot PRs 32 through 37 with disposition comments.
- Deleted all six Dependabot remote branches.
- Verified under Node 22.23.1 and Bun 1.3.8.
- Task review: approved after full-gate evidence was added.

### 2. Employer-facing accuracy

Commit: `56dae7e docs: align portfolio story with current product`

- Replaced unsupported Docker claims with rootless Podman wording.
- Removed the unsupported SARIF claim; retained HTML and JSON reports.
- Replaced stale `Channel not found` copy with `Page not found`.
- Documented per-job review decisions stored in browser `localStorage`.
- Updated README and evaluator guidance to Review, Findings, and Artifacts.
- Aligned `docs/design.md` with Gabarito, Source Sans 3, and JetBrains Mono.
- Added Playwright assertions for homepage claims and 404 terminology.
- Verification: Playwright 9/9, Vitest 46/46, lint, typecheck, and build passed.
- Task review: approved.

### 3. Regression guard

Commit: `257ff57 test: guard employer-facing product claims`

- Extended stale-vocabulary scanning to TypeScript, TSX, and CSS.
- Added fixture-based shell tests for the corrected stale phrases.
- Added the test to `just shell-tests`.
- Changed CI to invoke the canonical `just shell-tests` entry point.
- Verification: shell tests, ShellCheck, Bash syntax, and diff checks passed.
- Task review: not completed because execution was intentionally stopped to reduce token use.

## Current Repository State

- Open pull requests: none.
- Remote branches: `main` and `feat/web-daylight-gauge-ui`.
- The feature branch is fully represented by merged PR #39 and is safe to delete after the cleanup branch is merged.
- `origin/main` is `e8e42bd` and has green CI plus Golden Regression.
- Local cleanup worktree is clean.

## Remaining Findings

### Important

1. The three completed commits are local only. Push and merge them before deleting the cleanup branch.
2. Task 3 still needs a quick review of `56dae7e..257ff57`.
3. `CHANGELOG.md` records 0.3.0 while GitHub's latest CLI release is 0.2.0.
4. `main` rules prevent deletion and force pushes but do not require existing CI jobs before merge.

### Minor

1. React Router builds emit existing v8 future-flag warnings.
2. `clients/web/qa/record-report-gif.mjs` has a legacy Pages-tab comment.
3. GitHub still has the stale `sveltekit` topic.

## Minimum Completion Path

### 1. Review the current diff

```bash
cd /tmp/opencode/stageflow-portfolio-hardening
git diff --check origin/main..HEAD
git diff --stat origin/main..HEAD
git show --stat 257ff57
```

Inspect these Task 3 files:

- `devtools/scripts/check-stale-vocab.sh`
- `devtools/scripts/tests/stale-vocab.test.sh`
- `justfile`
- `.github/workflows/ci.yml`

### 2. Run final local gates

```bash
cd /tmp/opencode/stageflow-portfolio-hardening
just ci
just project-golden
```

Expected: both commands exit 0. If local Podman prevents the golden test, require the GitHub Golden Regression workflow before merge.

### 3. Prepare CLI 0.4.0

Edit `CHANGELOG.md`:

- Keep a fresh `## [Unreleased]` section.
- Move current unreleased entries under `## [0.4.0] - 2026-07-14`.
- Commit as `chore(release): prepare CLI v0.4.0`.

### 4. Push and merge

```bash
git push -u origin cleanup/portfolio-hardening
gh pr create --base main --head cleanup/portfolio-hardening \
  --title "chore: harden StageFlow portfolio and dependencies" \
  --body-file docs/superpowers/handoffs/2026-07-14-portfolio-hardening-status.md
gh pr checks --watch
```

Confirm CI and Golden Regression are green, then merge and delete the PR branch:

```bash
gh pr merge --merge --delete-branch
```

### 5. Update GitHub presentation

```bash
gh repo edit --remove-topic sveltekit --add-topic react-router --add-topic typescript
```

Update the `Protect main` ruleset to require the existing CI jobs before pull-request merges. Preserve deletion protection, non-fast-forward protection, and the pull-request requirement.

### 6. Publish CLI 0.4.0

From updated `main`:

```bash
git tag -a clients/cli/v0.4.0 -m "StageFlow CLI v0.4.0"
git push origin clients/cli/v0.4.0
gh run list --workflow "Release StageFlow CLI" --limit 1
gh release view clients/cli/v0.4.0
```

Verify five platform archives plus `checksums.txt`.

### 7. Final branch cleanup

Only after the cleanup PR and release are complete:

```bash
git push origin --delete feat/web-daylight-gauge-ui
git fetch --prune origin
```

Remove the temporary worktree from the primary checkout:

```bash
git worktree remove /tmp/opencode/stageflow-portfolio-hardening
git switch main
git pull --ff-only origin main
git branch -d feat/web-daylight-gauge-ui
git branch -d cleanup/portfolio-hardening
```

### 8. Prove completion

```bash
git status --short --branch
git branch --all
git ls-remote --heads origin
gh pr list --state open
gh release view clients/cli/v0.4.0
```

Expected final state:

- Clean local `main` matching `origin/main`.
- No open pull requests.
- No local or remote branches except `main`.
- CLI v0.4.0 release available with archives and checksums.
