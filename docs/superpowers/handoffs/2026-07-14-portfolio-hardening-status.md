# StageFlow Portfolio Hardening Status

## Fast Start

Worktree: `/tmp/opencode/stageflow-portfolio-hardening`

Branch: `cleanup/portfolio-hardening`

Base: `e8e42bd` (`origin/main`)

Detailed plan: `docs/superpowers/plans/2026-07-14-portfolio-hardening.md`

Use `git status --short --branch` for the current ahead count; this handoff is
updated in the release-preparation commit, so a hard-coded count would become
stale immediately.

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
- Follow-up commit `d045308` makes scan/tool errors fatal, adds `.txt` coverage,
  prevents legacy line filters from suppressing product-copy findings, fixes
  fixture cleanup, and updates contributor guidance.
- Verification: focused tests, repository scan, Bash syntax, ShellCheck, and
  `just shell-tests` passed.
- Task review: approved after the follow-up fixes.

### 4. Browser and accessibility QA

- Production-preview Playwright passed 9/9 on desktop and mobile.
- ESLint, typecheck, Vitest 46/46, and the production build passed serially.
- Manual desktop/mobile checks covered homepage, playground, 404, validation,
  keyboard focus, reduced motion, console/page errors, network requests, and
  narrow viewport overflow.
- Report Review, Findings, Artifacts, review decisions, and cross-tab behavior
  were exercised through the repository's mocked-API Playwright suite.
- No reproducible source defects were found; ignored evidence is under
  `output/portfolio-qa/`.

### 5. CLI v0.4.0 preparation

- `CHANGELOG.md` now keeps a fresh Unreleased section and records 0.4.0 on
  2026-07-14.
- Active Pages-tab wording in the release entry and GIF recorder was updated to
  Review/finding terminology.
- No package version bump is needed; the release workflow stamps the CLI from
  the `clients/cli/v0.4.0` tag.

## Current Repository State

- Open pull requests: none.
- Remote branches: `main` and `feat/web-daylight-gauge-ui`.
- The feature branch is fully represented by merged PR #39 and is safe to delete after the cleanup branch is merged.
- `origin/main` is `e8e42bd` and has green CI plus Golden Regression.
- Local cleanup worktree should be clean after the release-preparation commit.

## Remaining Findings

### Important

1. The cleanup commits are local only. Push and merge them before deleting the cleanup branch.
2. GitHub's latest CLI release is 0.2.0; publish 0.4.0 only from the verified post-merge `main` SHA.
3. `main` rules prevent deletion and force pushes but do not require existing CI jobs before merge.

### Minor

1. React Router builds emit existing v8 future-flag warnings.
2. GitHub still has the stale `sveltekit` topic.

## Minimum Completion Path

### 1. Review and verify the current diff

```bash
cd /tmp/opencode/stageflow-portfolio-hardening
git diff --check origin/main..HEAD
git diff --stat origin/main..HEAD
just ci
just project-golden
```

Expected: both commands exit 0. If local Podman prevents the golden test, require the GitHub Golden Regression workflow before merge.

### 2. Update GitHub presentation and protection

```bash
gh repo edit --remove-topic sveltekit --add-topic react-router --add-topic typescript
```

Update the `Protect main` ruleset to require the existing CI jobs plus `Golden
Baseline Promote Diff` before pull-request merges. Preserve deletion protection,
non-fast-forward protection, and the pull-request requirement.

### 3. Push and merge

```bash
git push -u origin cleanup/portfolio-hardening
gh pr create --base main --head cleanup/portfolio-hardening \
  --title "chore: harden StageFlow portfolio and dependencies" \
  --body-file docs/superpowers/handoffs/2026-07-14-portfolio-hardening-status.md
gh pr checks --watch
```

Confirm CI and the follow-on Golden Regression run are green for the exact PR
head SHA, then merge and delete the PR branch:

```bash
gh pr merge --merge --delete-branch
```

### 4. Verify merged main and publish CLI 0.4.0

Fetch `main`, record the exact merged SHA, and wait for CI plus Golden Regression
to succeed on that SHA. Then create the tag explicitly against it:

```bash
git fetch origin main
release_sha="$(git rev-parse origin/main)"
if git ls-remote --exit-code --tags origin refs/tags/clients/cli/v0.4.0 >/dev/null; then
  echo "clients/cli/v0.4.0 already exists" >&2
  exit 1
fi
git tag -a clients/cli/v0.4.0 "$release_sha" -m "StageFlow CLI v0.4.0"
git push origin clients/cli/v0.4.0
gh release view clients/cli/v0.4.0
```

Watch the `Release StageFlow CLI` run whose `headSha` is `release_sha`. Verify
five platform archives plus `checksums.txt`, downloaded checksums, the binary's
stamped version/commit, and release notes that surface the 0.3.0 breaking CLI,
config, and JSON changes.

### 5. Production smoke and final branch cleanup

Smoke-test `https://stageflow.org` on desktop and mobile, including one
no-account scan/report path. Only after the smoke test and release are complete:

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

### 6. Prove completion

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
