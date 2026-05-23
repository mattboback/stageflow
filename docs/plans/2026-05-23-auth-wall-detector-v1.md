# Generic Auth-Wall Detector for StageFlow

## Objective

Make scans that land on an authentication wall fail loudly in the unified report instead of silently auditing a login page. The deterministic auth path shipped in PRs #111–#113 lets configured scans pass through a login; this plan covers the diagnostic complement, for scans where auth was either not configured or did not stick. After this work every page navigation in the scanner runner emits an `auth-wall-detected` issue when the page shows login-redirect, 401/403, login-form, or captcha signals, with severity calibrated to whether `Provenance.auth` was configured. Design context lives in `docs/design/auth-wall-detector.md`.

## Implementation Plan

- [ ] 1. Add the auth-wall-detected issue id to the report catalog
  - Add `libs/contracts/report/issue-catalog/auth-wall-detected.json` with the same shape as `auth-hydration-failed`. Include the catalog entry for `signals`, `requested_url`, `final_url`, `auth_configured`, and `auth_mode` metadata fields.
  - Rationale: the unified report consumer renders issue descriptions from the catalog; the catalog must know about the id before the runtime emits it.
  - Affects: `libs/contracts/report/issue-catalog/`, the catalog test fixtures.
  - Dependencies: none, foundational step.

- [ ] 2. Implement the detector module
  - Add `services/scanner-runner/src/core/auth-wall.ts` exporting `detectAuthWall(page, response, request)` and the `AuthWallSignal` discriminated union from §4 of the design. Pure function; no logging, no side effects beyond reading the Page and Response.
  - Implement the four signal checks in order: login redirect (final URL pattern match), auth status (401/403 from the main response), login form (single `page.evaluate` returning selector or null), captcha (single `page.evaluate` matching iframe src patterns plus a Cloudflare title check). The Cloudflare interstitial check also reads `cf-mitigated` from the response headers.
  - Rationale: keeping detection in one pure module makes it independently unit-testable against fake Pages and Responses, and keeps the wiring change in `PageIterator` minimal.
  - Affects: `services/scanner-runner/src/core/auth-wall.ts`.
  - Dependencies: none, can land in parallel with task 1.

- [ ] 3. Surface the response from BrowserManager.navigateToPage
  - Change `navigateToPage` to return `Promise<Response | null>` instead of `Promise<void>`, returning whatever `page.goto` returned. SSRF validation, final-URL validation, and the wait-strategy handling all stay unchanged.
  - Rationale: the detector needs the response to read the status code; this is the smallest change to the existing seam that exposes it.
  - Affects: `services/scanner-runner/src/core/browser-manager.ts`, the call sites in `PageIterator.processPage` and the auth-hydrator's form-mode login navigation.
  - Dependencies: none, but task 4 depends on it.

- [ ] 4. Wire the detector into PageIterator
  - In `PageIterator.processPage`, after `navigateToPage` returns and before `executePreScanActions` runs, call `detectAuthWall` with the response and the page entry's requested URL. Track auth configuration on the iterator (it already knows from `provenance.auth?.mode`) and whether hydration succeeded (track via the existing `auth_hydrated` audit event).
  - Append any emitted Issue to the page's results next to scan-callback issues. Severity per the design's §3: `info` when auth is not configured, `serious` when auth is configured and hydration succeeded, no issue at all when hydration failed (the `auth-hydration-failed` issue covers it).
  - Rationale: per-page detection in the iterator means every scanner inherits it without per-scanner code changes, mirroring the auth-scanning seam.
  - Affects: `services/scanner-runner/src/core/page-iterator.ts`.
  - Dependencies: tasks 1, 2, 3.

- [ ] 5. Unit-test each signal in isolation
  - Add `services/scanner-runner/tests/core/auth-wall.test.ts` covering: login redirect for each pattern in the design's §2.1, 401 and 403 responses, login-form heuristic with one positive and two negative cases (form without password input; password input outside a form), each captcha provider, and combinations of multiple signals on the same page.
  - Rationale: the detector is the kind of code that grows false positives over time; locking each pattern down with a unit test makes future tuning safe.
  - Affects: `services/scanner-runner/tests/core/auth-wall.test.ts`.
  - Dependencies: task 2.

- [ ] 6. Integration test against an unauthenticated scan of a login-redirecting fixture
  - Extend the existing real-browser integration test infrastructure under `services/scanner-runner/tests/integration/` with a new test that points an unauthenticated scan at a fixture homepage that 302-redirects to `/login`, asserts the page result contains an `auth-wall-detected` issue at severity `info`, and asserts the metadata includes `requested_url`, `final_url`, and a `login_redirect` signal.
  - Add a second case in the same test that points the scan at a fixture rendering a login form without a redirect, and asserts a `login_form` signal fires.
  - Rationale: the unit tests cover the detector in isolation; the integration test proves the wiring through `PageIterator` works end to end.
  - Affects: `services/scanner-runner/tests/integration/auth-wall.test.ts`, the existing fixture login app.
  - Dependencies: tasks 4, 5.

- [ ] 7. Document the detector in the architecture doc
  - Add a short section to `docs/architecture/system.md#authenticated-scanning` describing the detector's signals, where it runs, and how it relates to `Provenance.auth`. Cross-link from `clients/cli/README.md#authenticated-scans` so a developer reading "I see auth-wall-detected in my report" finds the right next step.
  - Rationale: the issue surfaces in real reports; the doc surface is the place developers go when they encounter it.
  - Affects: `docs/architecture/system.md`, `clients/cli/README.md`.
  - Dependencies: tasks 1 through 6.

## Verification Criteria

- An unauthenticated scan submitted against a target whose homepage 302-redirects to `/login` produces a page result with an `auth-wall-detected` issue at severity `info`, metadata includes a `login_redirect` signal, and the scan otherwise completes normally with whatever findings the login page produced.
- An unauthenticated scan submitted against a target rendering a login form on `/` (no redirect) produces an `auth-wall-detected` issue with a `login_form` signal and the form's selector in the metadata, and never reads any input values from the form.
- A scan submitted with `--auth-recipe` against a target whose stale recipe successfully hydrates but where the post-login pages still redirect to `/login` produces an `auth-wall-detected` issue at severity `serious` with `auth_configured: true` and `auth_mode: 'form'` in the metadata.
- A scan submitted with `--auth-recipe` against a target where hydration fails (recipe success selector never appears) emits the existing `auth-hydration-failed` issue and no `auth-wall-detected` issue. Two issues for the same root cause is forbidden.
- An unauthenticated scan submitted against a target that returns a 403 from the homepage produces an `auth-wall-detected` issue with an `auth_status` signal and the status code in metadata.
- The unauthenticated path against a fully public target produces zero `auth-wall-detected` issues, and no new fields appear in the unified report for those runs.
- `bun run ci` passes in `services/scanner-runner`, including the new unit and integration tests.
- A grep across the unified report, scan stage log, and any NATS event payload finds zero matches for any input value on a login-form page (the detector reads only selectors and presence, not values).

## Potential Risks and Mitigations

1. **False positives on legitimate login pages**
   Mitigation: a redirect-away-from-the-requested-URL is required for the login-redirect signal. If the developer pointed the scan at the login page directly, the requested and final URLs match and the signal does not fire. Document this in the architecture section so a developer scanning their own login page knows the detector will stay quiet.

2. **Captcha alone produces noisy issues on real authenticated apps**
   Mitigation: captcha alone never produces an issue. The captcha signal accompanies other signals when they fire; on its own it is informational only and is recorded in the metadata of an existing scan-callback issue (or omitted entirely when no other signal fires). Reflected in the unit tests for the captcha-only case.

3. **The signal regexes drift from real frameworks**
   Mitigation: each pattern in §2.1 of the design has a citation in the unit test (Devise = Rails, Allauth = Django, ASP.NET Identity = .NET). Adding a new pattern is a one-line change with one new test case. If a major framework is missing, a follow-up issue captures it; the detector is not a global URL classifier.

4. **`page.evaluate` for the form heuristic adds latency to every page navigation**
   Mitigation: the evaluate call is a single DOM query selecting `form input[type=password]` and reading three attributes. Measure once on the existing fixture pages and document the cost in the architecture section. If it exceeds a few milliseconds per page, gate the heuristic behind a Provenance flag with default-on; do not let it grow into a feature toggle without justification.

5. **The detector hides behind hydration-failure on configured auth scans**
   Mitigation: the design is explicit that when hydration fails the detector stays silent. The integration test in task 6 covers this case so a refactor that accidentally double-emits the issue fails the build.

6. **A future scanner-specific detector wants to override severity**
   Mitigation: defer. The detector emits one Issue with one severity policy. If a real consumer surfaces a need to suppress or upgrade the issue per scanner, that lands as a separate proposal with a concrete justification.

## Alternative Approaches

1. **Per-scanner detection**: have each scanner (axe, Lighthouse, etc.) detect its own auth wall and refuse to run. Rejected because every scanner reinvents the heuristic and falls out of sync. The shared `PageIterator` is the right seam.

2. **Header-based detection only**: surface `auth-wall-detected` solely from the response status. Rejected because single-page apps render login forms client-side after a 200 response; status codes catch only the API-style auth walls.

3. **Treat auth-wall as a critical issue that fails the scan**: rejected because the scan still produces useful data on the wall page (the page itself may have axe findings worth fixing), and a fail-fast detector is impossible to opt out of for the legitimate case where someone scans their login page on purpose.

4. **Build a learned classifier**: train a model on labeled login pages. Rejected; the failure mode is too narrow to justify ML, the regex patterns cover the framework-shared shape, and a deterministic heuristic is debuggable in a way a classifier never is.

## Assumptions

- The unified report consumer accepts a new issue id by reading the catalog entry; no consumer-side code change is required to render the issue.
- `Page.evaluate` calls add negligible latency on a page that has already settled into its load state. Measured on the existing fixture; revisit if real targets show otherwise.
- The set of captcha providers in §2.4 covers the deployments StageFlow will encounter for the lifetime of this design. Adding a new provider is a single-line patch.
- The scanner runner already emits the `auth_hydrated` audit event from PR #111, available to `PageIterator.processPage` for the severity calibration in task 4.

## Dependencies

- The deterministic authenticated-scanning runtime in `services/scanner-runner/src/core/{auth-hydrator,page-iterator,browser-manager}.ts`, shipped in PRs #111–#113.
- The existing fixture login app under `services/scanner-runner/tests/integration/` from PR #113.
- The unified report's issue catalog under `libs/contracts/report/issue-catalog/`.

## Notes

- The detector is the follow-up flagged in `docs/design/authenticated-scanning.md` §3 ("a generic version is worth doing eventually") and in the auth-scanning plan's Notes section ("a follow-up plan should cover a generic auth-wall detector").
- Out of scope: the LinkedIn-specific and Indeed-specific patterns in `agent-harness-legacy/runtimes/job-automation/patchright-companion/src/auth-wall.ts`. Those targets are AlchemizeCV's domain and the detector here is intentionally framework-shape rather than site-specific.
- Out of scope: detection of cookie banners, geo-gates, A/B-test variants, and modal dialogs. Those are real failure modes for completeness audits but they are not auth walls and they belong to a separate proposal if they ever need one.
- The detector is the last item in the auth-scanning workstream as scoped at the start of this effort. After it lands, the workstream is done; future work on authentication should branch off a fresh design rather than extending this one.
