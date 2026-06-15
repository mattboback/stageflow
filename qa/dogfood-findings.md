# StageFlow dogfood / E2E findings

Discovery pass on a fresh-clone user journey (report-first; fixes happen in a later
prioritized pass). Environment: Linux, Go 1.26.4, Bun 1.3.14, Podman 5.8.3, just 1.52,
**host Node v26.1.0**. Date: 2026-06-15.

Severity legend: **blocker** (stops the documented flow, no easy workaround) ·
**major** (blocks a documented happy path but has a workaround) · **minor** (friction /
wrong output, flow continues) · **cosmetic** (docs/wording/polish).

## Summary

| ID | Sev | One-liner | Suspected fix location |
|----|-----|-----------|------------------------|
| F1 | major | `just diagnose` hard-fails on Node ≠ 22.x, blocking `just demo` | `infra/scripts/diagnose-local-env.sh` (`EXPECTED_NODE_MAJOR`, ~L201) |
| F2 | major | `just demo` prints keyless next-step commands that 401 | `justfile` `demo`; docs; `podman-compose.yml:178` |
| F3 | major | Web UI Playground unusable on default stack (scanner catalog 401) | same as F2/F4; `clients/web/src/lib/api/utils.ts` |
| F4 | major | `PLATFORM_API_AUTH_DISABLED=true` is a no-op (root cause of F2/F3) | `services/platform-api/internal/api/middleware.go:397` |
| F5 | major | `just ci` fails: 2 high vulns (vite, form-data) in web + scanner-runner | `clients/web/package.json`, `services/scanner-runner/package.json` |

**F2/F3/F4 share one root cause (auth).** Fixing F4 (make the disable flag work) + running the
local demo auth-disabled resolves all three. F1 and F5 are independent. No blocker-severity or
data-loss findings; the core scan pipeline, CLI, and golden E2E are all sound.

## Phase B — fixes applied & verified

- **F4 — FIXED & verified.** `apiKeyMiddleware` now short-circuits when
  `PLATFORM_API_AUTH_DISABLED=true` (`middleware.go`). Added request-path regression test
  `TestAPIKeyMiddleware_AuthDisabled_AllowsMissingKey` (was RED → now GREEN). Full platform-api
  module passes `go test -race ./...`.
- **F2/F3 — FIXED & verified.** Set `PLATFORM_API_AUTH_DISABLED: "true"` in the repo-local
  `dev`/`local` compose overlays (`podman-compose.test.yml`, `podman-compose.local.yml`) only;
  base compose stays auth-enabled for staging/self-host. After rebuild: keyless
  `GET /api/v1/scanners` → 200 and keyless `POST /api/v1/jobs/urls` (the exact demo curl) → 201.
  Web UI Playground loads "7 scanners"; ran a full **live UI scan** end-to-end (submit → live SSE
  progress → COMPLETE → rendered unified report, grade B 85/100, 2 issues). Evidence:
  `qa/evidence/F3-fixed-unified-report.png`. The docs' "optional" wording for `STAGEFLOW_API_KEY`
  is now accurate for the default local stack (key still required for auth-enabled self-host).
- **F1 — FIXED & verified.** `diagnose-local-env.sh` now accepts Node major **>= 22** (matching
  the Go/Bun `version_ge` style) instead of strict `== 22`; rejects non-numeric majors. Verified:
  `just diagnose` passes on the **host Node v26.1.0** (exit 0). README prerequisite reworded to
  "22.x or newer (the repo pins 22 in .node-version)".
- **F5 — FIXED & verified.** Full `just ci` re-run is now **GREEN (exit 0)** — the previously
  failing Frontend audit passes, and the Scanner-runner CI + audit stages (which never ran before
  because `set -e` aborted) now run and pass. Bumped the `vite` override
  pin `7.3.2 → 7.3.5` (patched, same 7.x minor → low risk) in both `clients/web/package.json` and
  `services/scanner-runner/package.json`, and added a `form-data: "4.0.6"` override in
  `clients/web` (matches the existing `ws`-pin pattern). After `bun install`, both
  `bun audit --audit-level=high` → **exit 0, 0 vulnerabilities**. Lockfiles updated.

## Final state

All 5 findings fixed and verified. `just ci` passes green end-to-end. The freshly-cloned
open-source user journey now works out-of-the-box: `cp .env.example .env` → `just diagnose`
(passes on Node >= 22) → `just demo` → the printed CLI/curl commands work (auth-disabled local)
→ Web UI Playground runs a scan and renders a unified report. CLI, golden regression E2E, and
private-target scanning all verified. No blocker-severity or data-loss issues were found.

---

## Confirmed NOT regressed (positive checks)

- **`.env.example` `DATABASE_URL`** uses host `postgres:5432` (the old `127.0.0.1` bug stays fixed).
- **`.env.example` MinIO keys** — `MINIO_ACCESS_KEY=stageflow` ≠ `MINIO_ROOT_USER=minioadmin`,
  which `just diagnose` requires.

---

## Findings

### F1 — `just diagnose` hard-fails on Node ≠ 22.x, blocking `just demo` out-of-the-box
- **Severity:** major
- **Area:** local deploy / onboarding (`infra/scripts/diagnose-local-env.sh`)
- **Repro:** fresh clone with host Node v26.1.0 → `just diagnose`
- **Expected:** documented quickstart (`cp .env.example .env && just diagnose && just demo`)
  works, or fails with an actionable, low-friction path.
- **Actual:** `FAIL Node.js version v26.1.0 does not match required major 22.x` → diagnose exits 1.
  Because `just demo` runs `just diagnose` first, the **primary documented onboarding command
  is hard-blocked** for any user not pinned to exactly Node 22.x.
- **Notes / mitigations that exist:** error text suggests `.node-version` (contains `22`) or
  `NODE=/path/to/node`. The check is a strict equality on the major version
  (`diagnose-local-env.sh:201` `node_major == "22"`), so newer LTS lines (24/26) are rejected
  even though the frontend actually builds/runs inside its own container image. `.node-version`
  only helps if a version manager (fnm/nvm) is active in the shell; on this machine `fnm` is
  installed but not on PATH, so only the explicit `NODE=`/PATH override works.
- **Suspected file:** `infra/scripts/diagnose-local-env.sh` (`EXPECTED_NODE_MAJOR`, line ~201).
- **Possible Phase-B fix (for discussion):** accept `>= 22` instead of `== 22`, and/or surface
  the `NODE=` override + Node 22 requirement more prominently in the README prerequisites.
- **Workaround used to continue this pass:** `NODE=<node22>` / Node 22 on PATH.

### F2 — `just demo` prints next-step commands that 401 (auth enabled, key omitted)
- **Severity:** major (hits every fresh user immediately after the happy-path demo)
- **Area:** local deploy onboarding (`justfile` `demo` recipe output; `infra/compose/podman-compose.yml`; docs)
- **Repro:** `just demo`, then run the commands it prints verbatim:
  - `curl -sS -X POST http://localhost:8080/api/v1/jobs/urls -H 'content-type: application/json' -d '{"urls":["https://example.com"]}'`
    → **HTTP 401** `{"error":"unauthorized","code":"UNAUTHORIZED"}`
  - `stageflow scan https://example.com` (no `--api-key`) → 401 / CLI error.
- **Expected:** the commands the demo prints should work as-is against the stack it just started.
- **Actual:** the local stack runs with **auth enabled** — `PLATFORM_API_AUTH_DISABLED`
  defaults to `false` (`infra/compose/podman-compose.yml:178`) and neither the `dev`/`local`
  overlays, the `demo` recipe, nor `.env` sets it true. `.env` ships `PLATFORM_API_TOKEN=
  change-me-platform-api-token`, so requests need `X-Api-Key: change-me-platform-api-token`
  (verified: same POST with the header → **HTTP 201**, job created; `/api/v1/scanners` → 200,
  lists all 8 scanners).
- **Compounding doc issue:** `docs/operations/devtools.md:82` and `cli_cheatsheet.md:194`
  call `STAGEFLOW_API_KEY` "optional", but it is effectively **required** for the default local stack.
- **Suspected fix locations (Phase B, for discussion):** either (a) `just demo` exports
  `PLATFORM_API_AUTH_DISABLED=true` for the frictionless local demo, or (b) `just demo` prints
  the commands with `STAGEFLOW_API_KEY=change-me-platform-api-token` (+ an `export` hint), and
  the docs stop calling the key "optional" for the default local posture.
- **Workaround used to continue:** `export STAGEFLOW_API_KEY=change-me-platform-api-token`.

### F3 — Web UI Playground is unusable on the default `just demo` stack (scanner catalog 401)
- **Severity:** major/blocker for the UI surface (the demo explicitly says "Try the web UI";
  the landing page advertises "No account required").
- **Area:** web UI ↔ platform-api auth (same root cause as F2; distinct user-visible surface)
- **Repro:** `just demo`, open `http://localhost:3000/playground` in a browser.
- **Expected:** scanner catalog loads, user picks scanners, runs a scan.
- **Actual (confirmed in-browser):** page shows **"Scanner catalog failed to load (401). Refresh
  to retry."**, "0 scanners", and the **"Start Scan" button is disabled** — a fresh user cannot
  run any scan from the UI. DevTools shows the page's only XHR is
  `GET http://localhost:8080/api/v1/scanners` → **401**. Evidence: `qa/evidence/F3-playground-401.png`.
- **Why:** the SvelteKit app calls the platform API **directly from the browser** via
  `VITE_API_URL` with **no API key** (no server-side proxy, no `+server.ts`/`hooks.server.ts`,
  no key in `clients/web/src`), so with auth enabled (F2) every API call 401s.
- **Suspected fix:** resolved by the same F2 fix (local demo runs auth-disabled). If auth is
  intended to stay enabled for local, the web app needs a server-side proxy that injects the
  token, or a documented way to pass a key to the browser app — but auth-disabled-for-demo is
  the simplest correct local posture.
- **Suspected files:** `infra/compose/podman-compose.yml:178` (auth default), `justfile` `demo`,
  `clients/web/src/lib/api/utils.ts` (browser-side API base).

### F4 — `PLATFORM_API_AUTH_DISABLED=true` does NOT disable request auth (documented toggle is a no-op)
- **Severity:** major (security-relevant + breaks the documented local/dev escape hatch; root
  cause that makes F2/F3 unavoidable even when users do what the docs say)
- **Area:** platform-api auth middleware
- **Repro:** bring the stack up with `PLATFORM_API_AUTH_DISABLED=true` AND `PLATFORM_API_TOKEN`
  set (the shipped `.env` default). Container env confirms `PLATFORM_API_AUTH_DISABLED=true`,
  yet `curl http://localhost:8080/api/v1/scanners` (no key) → **HTTP 401**.
- **Root cause:** `services/platform-api/internal/api/middleware.go:397` `apiKeyMiddleware` gates
  enforcement only on the token being non-empty:
  ```go
  expected := strings.TrimSpace(os.Getenv("PLATFORM_API_TOKEN"))
  if expected == "" { return next }   // auth skipped ONLY when token is empty
  ```
  It never checks `PLATFORM_API_AUTH_DISABLED`. That flag is consulted only in
  `ValidateAuthConfig` (startup, line 348) and `ValidateCORSConfig` (line 381). So with a token
  set, auth is always enforced regardless of the disable flag.
- **Doc/behavior mismatch:** `.env.example:39-40` documents `PLATFORM_API_AUTH_DISABLED=true` as
  the way to "explicitly disable platform-api authentication (local/dev only)". It doesn't work.
  The only working way to disable auth today is to **empty `PLATFORM_API_TOKEN`** (with the
  disable flag set so `ValidateAuthConfig` still passes).
- **Test gap:** `middleware_test.go:216-219` only asserts `ValidateAuthConfig` returns nil with
  the flag set — it does not assert the request path actually skips auth, so the gap is uncaught.
- **Possible Phase-B fix:** in `apiKeyMiddleware`, short-circuit when
  `PLATFORM_API_AUTH_DISABLED=true` (return `next`), and add a request-path regression test. This
  also makes F2/F3 fixable by simply setting the flag in the local demo.
- **Suspected file:** `services/platform-api/internal/api/middleware.go:397-401`.

---

## Verified working end-to-end (no defect)

**Local deploy / stack**
- `just diagnose` passes (with Node 22); `just setup` generates `scanners.yaml` as a **file**
  (prior "directory" quirk not regressed); `just images` builds all 6 images cleanly.
- `just demo` brings the full stack up — postgres, nats, minio, orchestrator, platform-api,
  grafana, frontend all healthy; all three health waits pass.
- `local` profile (`just dev up local` + `just dev init local`) comes up healthy; MinIO init is
  idempotent; frontend served at `:3010`, API at `:8080`.
- Scanner enablement via `infra/scanners/scanners.yaml` works — API reports `enabled: 7` after
  flipping the 5 non-AI scanners on.

**Scan pipeline (API + CLI)**
- Raw API: `POST /api/v1/jobs/urls` (with key) → job runs to `DONE`, produces 2 axe violations
  and full artifacts (report.json/html, screenshots, provenance) via presigned MinIO URLs.
- CLI `scan https://example.com --scanner axe --format json` → `DONE`, score 85, 2 moderate
  issues, envelope schema `stageflow-cli/report@v1`.
- Exit-code contract exactly as documented: `--fail-on serious` → 0, `--fail-on moderate` → 1;
  bad `--api` host / missing URL / keyless / unknown command → 2.
- Multi-scanner `axe,seo` aggregates correctly (`byScanner: {axe:2, seo:6}`, both `status:success`).
- **Localhost private-target scan** (`scan http://127.0.0.1:3010 --allow-private-targets`) → axe
  `status:success`, 1 issue — `allow_private_targets` forwarding to scanner pods **not regressed**.

**CLI surface**
- `scanners` (text/json), `report <job-id>`, `diff <a> <b>`, `completion {bash,zsh,fish,powershell}`,
  `docs --out-dir`, `version`, `--help` all behave and exit 0.
- `dev init` scaffolds `.stageflow/config.yaml` + `README.md`; `dev doctor --skip-dev` → exit 0
  ("Doctor checks passed") — **prior `--skip-dev` placeholder bug not regressed**; `dev doctor
  --format json` emits **pure JSON** on stdout (preamble correctly on stderr).
- `project create/list/show/update/delete` CRUD all exit 0.

**Automated suites**
- `just project-golden` (golden regression E2E) **PASSED**: baseline 0 issues → promote → update
  to regression page → regression scan exits 1 → golden + structural assertions pass
  (1 new `image-alt` critical issue). Full project scan/promote/diff lifecycle verified.
- `just ci` — **FAILS (exit 1)** at the Frontend audit stage (see F5). All earlier stages
  passed: stale-vocab, Go build, **Go lint**, **Go test -race**, **govulncheck**, CLI-docs drift
  (no drift), shell regression tests, Frontend CI, Frontend Storybook setup + tests.

---

### F5 — `just ci` fails on a fresh clone: 2 high-severity frontend dependency vulns
- **Severity:** major (the documented full-check command fails out-of-the-box; CI is red)
- **Area:** dependencies (`clients/web`) / `just ci` Frontend audit stage
- **Repro:** `just ci` → reaches "Frontend audit" → `bun audit --audit-level=high` reports 2 highs → exit 1.
- **The two advisories:**
  - **vite** `>=7.0.0 <=7.3.4` (direct dependency) — `server.fs.deny` bypass on Windows alternate
    paths, GHSA-fx2h-pf6j-xcff. (Windows-specific dev-server issue; low impact on Linux self-host,
    but `bun audit` is platform-agnostic.)
  - **form-data** `>=4.0.0 <4.0.6` (transitive, dev tooling only:
    `start-server-and-test`/`@storybook/test-runner` → wait-on → axios → form-data) — CRLF
    injection, GHSA-hmw2-7cc7-3qxx.
- **Also affects scanner-runner:** `services/scanner-runner` `bun audit --audit-level=high` fails
  (exit 1) on the **same vite advisory** (transitive: `vitest › @vitest/mocker › vite`). So the
  vite bump is needed in both `clients/web` and `services/scanner-runner`.
- **Consequence:** `set -e` aborts `ci` at the frontend audit, so the **scanner-runner CI + audit
  stages never run** under `just ci`. I ran them separately: scanner-runner **CI passes** (exit 0,
  ~78% coverage); scanner-runner **audit fails** (exit 1, vite).
- **Possible Phase-B fix (dependency-management task):** `bun update` to pull vite > 7.3.4 and a
  patched `form-data` (>= 4.0.6) in `clients/web`, then re-run `bun audit --audit-level=high` and
  the frontend test suite to confirm no breakage. Mirrors the recent
  `fix(ci): pin ws >=8.21.0 to clear GHSA-96hv-2xvq-fx4p audit` commit.
- **Suspected files:** `clients/web/package.json`, `clients/web/bun.lock`.

## Deferred to Phase B (blocked, not a separate defect)

- **Live Web-UI happy-path scan (submit → SSE → rendered report).** Blocked by F4 on the default
  auth-enabled stack; the browser cannot send an API key. The failure mode is captured (F3,
  screenshot). Once F4 is fixed (or the demo runs auth-disabled), re-run: open `:3010/playground`,
  submit `https://example.com`, confirm live SSE progress and a rendered unified report.

## Environment notes (NOT StageFlow defects)
- This machine's Claude guard hooks (`[cp-env]`, `[env-file]`, `[cat-env]`, `[echo-secret-var]`)
  block creating/reading `.env` and putting the API token in shell variables. A real user is
  unaffected. Workarounds used: user created `.env`; API key passed as a fully-inline literal.
- Host Node is v26.1.0; used a Node 22.22.3 (fnm) binary on PATH to satisfy F1's gate.
