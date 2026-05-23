# Authenticated Scanning and the Future of `agent-harness-legacy`

Status: design note. Captures the verdict on the legacy directory and the
shape of the work needed to make StageFlow scan beyond marketing
homepages. The execution plan lives at
`docs/plans/2026-05-23-authenticated-scanning-v1.md`.

> **See also:** the architecture-level write-up of the contract, trust
> boundaries, and storage-state retention rules at
> [`docs/architecture/system.md#authenticated-scanning`](../architecture/system.md#authenticated-scanning),
> and the developer workflows at [`clients/cli/README.md#authenticated-scans`](../../clients/cli/README.md#authenticated-scans).

This document does two things:

1. Gives an honest verdict on whether `agent-harness-legacy/` has a place
   in StageFlow.
2. Describes the design of authenticated scanning that the plan
   implements.

---

## 1. Verdict on `agent-harness-legacy/`

### 1.1 What StageFlow already has

The legacy directory was extracted from AlchemizeCV before the production
runtime removed its dev harness, vision substrate, and parity tooling. It
carries:

- `runtimes/job-automation/agent-harness/` — Go agent core: turn loop,
  model wrapper, vision/browser tools, parity-run, queue acceptance,
  router, shell with `editf`/`readf`/`astf`, run-context fetcher.
- `runtimes/job-automation/patchright-companion/` — TypeScript Patchright
  driver: frame capture with sharp annotation, auth-wall detection,
  action executor, submit classifier, paired-gateway session.
- `runtimes/job-automation/browser-companion/` — browser extension
  companion (UI for human-in-the-loop sessions).
- `runtimes/job-automation/desktop-companion/` — desktop wrapper.
- `runtimes/job-automation/protocol/` and
  `shared/job-automation-protocol/` — wire protocol + action policy for
  AlchemizeCV's automation pipeline.

What this repo already has, independently:

- `services/scanner-runner/src/scanners/ai-navigator/` — a working
  LLM-driven browser agent built on Playwright and OpenRouter. It does
  perceive → decide → act with `AgentGoal`, `successCriteria`, screenshot
  trace per step, and trace upload to MinIO. Roughly 600 lines, fully
  integrated with the contract-driven scanner runtime.
- `services/scanner-runner/src/core/browser-manager.ts` — Playwright
  `BrowserContext` lifecycle, runtime SSRF/target-validation routing, and
  navigation with retries and wait strategies.
- `services/scanner-runner/src/core/page-iterator.ts` — single
  `BrowserContext` per scanner run, iterated across pages with
  `PreScanAction[]` support.
- `services/scanner-runner/src/scanners/{axe,lighthouse,seo,security-headers,…}` —
  the actual battery of scanners that will benefit from any auth work.

The `ai-navigator` scanner and `agent-harness` engine are architectural
twins. The legacy version is more sophisticated in some specific areas
(failure policy, no-progress detection, unsafe-text guards), but it is
written in Go, bound to AlchemizeCV's tool protocol (`exec`, `editf`,
`readf`, `browser post`), and depends on a separate `patchright-companion`
process over a WebSocket gateway. StageFlow's `ai-navigator` lives
in-process inside the scanner runner and uses Playwright directly. There
is no scenario where pulling the Go harness into StageFlow improves
anything — it would add a second runtime, a second process boundary, and
a domain-specific protocol that nothing else in this repo speaks.

### 1.2 Concrete decision

**Delete the directory.** It is fully self-contained: no CI step, no
entry in `go.work`, no script, and no live module references it. Carrying
~50 unused Go packages plus three TypeScript companions costs us in
gitleaks scans, govulncheck runs, dependabot noise, and contributor
attention every time someone wonders what it is. The AlchemizeCV history
worth preserving is preserved by Git itself.

### 1.3 Ideas worth keeping (and which ones are not)

**Worth porting, as a planned task:**

- The **failure-policy concepts** in
  `agent-harness/internal/engine/turn.go`: stop after N consecutive
  failed tool calls; stop when N successful turns produced no observable
  state change. These are 30–50 lines added directly to the existing
  loop in `services/scanner-runner/src/scanners/ai-navigator/agent.ts`.
  No new abstractions, just inline guards with configurable thresholds.
  This is captured as a task in the linked plan, executed before the
  directory is deleted so the ideas land in the codebase that will
  outlive the snapshot.

**Considered and dropped:**

- `patchright-companion/src/auth-wall.ts` — declarative auth-wall
  detection. The detection patterns are hard-coded for LinkedIn,
  LinkedIn checkpoint, Indeed, and Indeed's Cloudflare interstitial.
  Those targets are AlchemizeCV's job-board domain, not StageFlow's.
  A generic version (login redirects, generic captcha presence,
  generic 401/403 patterns) is worth doing eventually but is not part
  of this work, and porting the AlchemizeCV-specific version would add
  noise without value.
- `patchright-companion/src/frame-capture.ts` — multi-target labeled
  screenshot overlay. Nice to have for richer agent traces, but
  StageFlow's `core/screenshots.ts` already covers the current need
  (single highlight per step). Re-evaluate when there is a concrete
  consumer asking for it.
- Everything else — shell, agenttools, parityrun, queueacceptance,
  paired-gateway-session, the wire protocol, browser/desktop companions
  — is AlchemizeCV-domain or solves problems StageFlow does not have.

---

## 2. Authenticated scanning — design

### 2.1 The problem

Every scanner today receives a fresh Playwright `BrowserContext`. There
is no place to attach cookies, localStorage, an Authorization header, or
a recorded login flow. Scans of a real app collapse onto the marketing
landing page or a login redirect, both of which axe and Lighthouse
audit as if they were the real thing.

The fix is small because the existing architecture cooperates:

- `Provenance` is the single source of truth read by `PageIterator` and
  every scanner. It already carries page-level configuration like
  `pre_scan_actions`, `wait_for`, and `viewport`.
- `BrowserManager.createContext` is the single seam where a Playwright
  context is built. Playwright natively accepts `storageState` at
  context-creation time.
- `PageIterator.iteratePages` creates exactly one context per scanner
  run, then reuses it. A session hydrated once survives every page in
  the scanner, and naturally cuts across all scanners (axe, Lighthouse,
  security-headers, ai-navigator) without per-scanner code changes.

### 2.2 Contract change

Extend `Provenance` (in `libs/contracts` and the TypeScript types in
`services/scanner-runner/src/core/types.ts`) with a single new optional
field. Auth is one of two shapes:

- `storage_state` — a reference to a Playwright storage-state JSON file
  uploaded to MinIO under the job's prefix. Captured interactively by
  the developer via a new CLI subcommand.
- `form` — a declarative login recipe: a login URL, a list of steps
  reusing the existing `PreScanAction` shape, and a success wait
  strategy. Step values can be either literal strings or
  `from_env` references that resolve at execution time inside the
  scanner-runner pod.

Two important details:

- Secrets never appear in stored Provenance. Only `from_env` references
  do. Provenance can be persisted to MinIO and referenced from the
  unified report without leaking credentials.
- A pre-captured storage-state file is treated as a job-scoped secret.
  It lives at a known MinIO key under the job's prefix, is subject to
  the same retention policy as other scan artifacts, and is not exposed
  via the Web UI's signed-URL surface.

### 2.3 Code seams

Three small changes in `services/scanner-runner`:

1. **`core/types.ts`** — extend `Provenance` and `PreScanAction.value`
   to allow `from_env` references alongside string literals. Add a typed
   `ProvenanceAuth` union.
2. **`core/browser-manager.ts`** — accept an optional
   `storageStatePath` in `createContext`. Pass it straight to
   Playwright's context creation.
3. **`core/page-iterator.ts`** — after creating the context, before
   iterating pages, call a new `hydrateAuth` helper that:
   - For `storage_state`: resolves the artifact via the existing
     `StorageProvider`, downloads it to a temp file, and feeds the path
     into context creation.
   - For `form`: opens an internal page, runs the steps via the
     existing `executePreScanActions` plumbing (extended to resolve
     `from_env` references), waits for the recipe's success strategy,
     and discards the page. Cookies and localStorage now live on the
     context for every subsequent scanner page.
   - Records an `auth_hydrated` event with the auth method and
     post-login URL into the existing scan stage log. No credentials
     are logged.
   - On failure, throws a typed `AuthHydrationError` that surfaces in
     the unified report as a critical issue and skips downstream pages.

Every existing scanner inherits the authenticated context for free —
no edits to `axe`, `lighthouse`, `seo`, `security-headers`,
`link-checker`, `open-graph`, `spelling-grammar`, or `ai-navigator`.

### 2.4 CLI surfaces

Two paths, both supported:

- **Captured storage state (developer-driven, common case).** A new
  `stageflow auth capture` subcommand launches a non-headless Chromium,
  lets the developer log in by hand, and writes the resulting storage
  state to a developer-specified path. `stageflow scan ... --auth-state
  <path>` uploads that file to MinIO and inlines its key into
  Provenance. This is the only flow that ever sees a real password,
  and it runs locally on the developer's machine.
- **Declarative form recipe (CI-driven).** A YAML/JSON recipe checked
  into the project, plus `STAGEFLOW_AUTH_USER` / `STAGEFLOW_AUTH_PASSWORD`
  (or other named) environment variables. `stageflow scan ...
  --auth-recipe <path>` inlines the recipe into Provenance with
  `from_env` references; the orchestrator forwards exactly the named
  env vars (and only those) into the scanner-runner pod.

### 2.5 Trust boundaries

- **Platform API:** never receives a password. It accepts only an
  artifact upload (storage state) or a recipe whose values are
  `from_env` references.
- **Orchestrator:** when launching the scanner-runner pod, it
  optionally injects a tightly-scoped, allow-listed set of env vars
  derived from the recipe's `from_env` references. The set is built
  from the recipe; arbitrary env passthrough is not permitted.
- **Scanner runner:** treats storage state files as ephemeral. They
  are written to the job workspace, used to seed the context, and
  removed in `cleanup()`. The Web UI never receives a signed URL for
  the storage-state artifact.
- **Existing SSRF policy still applies.** Auth hydration goes through
  the same `validateTargetURLForPolicy` path as every other navigation —
  login URLs that fail SSRF validation fail fast with a clear error.

---

## 3. What this design explicitly does not do

- **It does not build a "Harness Audit Bundle" or a 3-pane Svelte
  agent-replay UI.** That belongs to a separate design once
  authenticated scanning is real and the data is worth visualizing. The
  current `ai-navigator` trace JSON + step screenshots are enough to
  start.
- **It does not adopt patchright** as a runtime. Playwright is the
  established dependency.
- **It does not propose a generic "agent runs my scan for me" feature.**
  Auth is a deterministic, reproducible problem and should be solved
  deterministically. The agent (`ai-navigator`) is the right tool for
  open-ended exploration of an authenticated app, not for getting past
  the login form in the first place.
