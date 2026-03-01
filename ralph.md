This ralph plan converts the existing Phase 1 portion of `plan.md` into a focused, single-run command. It is intentionally limited to API-side groundwork for opt-in private/local target scanning: request contract changes, scoped SSRF validation, config plumbing, and tests. It does not include the new CLI module, docs updates, compose wiring, or any transport changes.

This phase changes API acceptance rules only.

- It does not alter orchestrator or scanner networking.
- Accepting a loopback or RFC1918 URL at the API layer does not guarantee end-to-end reachability from scanner containers to the CLI caller's workstation or LAN.
- Any true developer-workstation `localhost` or broader private-network workflow requires a later transport phase (daemon/proxy/host-bridge) or artifact mode.

## Public APIs, Interfaces, and Types

- Extend `POST /api/v1/jobs/urls` to accept an optional `allow_private_targets` boolean in the JSON body.
- Add a new API runtime toggle `PLATFORM_API_ALLOW_PRIVATE_TARGETS`, default `false`.
- Thread that toggle through API startup config into `api.ServerConfig` so the handler can enforce it.
- Preserve existing behavior when the new request field is omitted or `false`.
- Add one new validation failure path:
  if `allow_private_targets=true` is requested while the server toggle is `false`, return `400 Bad Request` with a structured validation-style error on `allow_private_targets`.
- Do not change endpoint paths, success response shape, event publishing, or existing public-target behavior.

## Test Cases and Scenarios

- Public URL submission still succeeds unchanged.
- Loopback and RFC1918 targets still fail when `allow_private_targets` is omitted.
- Loopback and RFC1918 targets still fail when `allow_private_targets=true` but the server env/config toggle is disabled, with the new structured `400` response.
- `localhost`, `127.0.0.1`, and `::1` succeed at the API validation layer when both the request flag and server toggle are enabled.
- RFC1918 IPv4 ranges succeed at the API validation layer in private-target mode.
- Hostnames that resolve to public addresses and the explicitly allowed private/local addresses succeed in private-target mode.
- Metadata `169.254.169.254` still fails even in private-target mode.
- Link-local, unspecified, multicast, invalid URLs, and non-HTTP(S) schemes still fail in all modes.
- Existing module normalization and scanner config validation remain unchanged.

## Assumptions and Defaults

- The source of truth is the existing Phase 1 scope in `plan.md`.
- This is a single focused ralph command for Phase 1 only.
- No repo files are to be edited by me in this planning turn.
- No CLI scaffolding, README changes, compose wiring, or transport changes are included in this command.
- Verification should use the repo-standard commands, ending with `just ci`.

  <background>
  Use the existing Phase 1 plan in `plan.md` as the product spec, but ground the implementation in the current API code. You are working in the StageFlow repo. Focus only on API-side groundwork for opt-in private/local target scanning so a later CLI phase can build on it without weakening the default public SSRF posture.

Relevant code paths already exist:

- `platform/api/internal/api/handlers_jobs_url_submit.go` handles `POST /api/v1/jobs/urls` and currently calls `validateTargetURLsWithResolver` directly
- `platform/api/internal/api/security.go` contains URL parsing and SSRF checks
- `platform/api/cmd/server/config.go` and `platform/api/cmd/server/main.go` load runtime config and construct `api.ServerConfig`
- `platform/api/internal/api/server.go` defines `ServerConfig`
- `platform/api/internal/api/security_test.go`, `security_dns_test.go`, and `handlers_test.go` contain the existing test patterns

Maintain the current product boundary:

- public URL scanning behavior must stay unchanged by default
- the new request flag is optional and defaults false
- private/local target scans must only be allowed when both the request flag and server config opt in
- this phase is API groundwork only and must not be treated as end-to-end localhost or private-network support
- do not add CLI code, docs changes, infra wiring, or transport changes in this run
  </background>

  <setup>
  1. Read `plan.md` for the Phase 1 intent, then inspect the current implementations in `platform/api/internal/api/handlers_jobs_url_submit.go`, `platform/api/internal/api/security.go`, `platform/api/cmd/server/config.go`, `platform/api/cmd/server/main.go`, and `platform/api/internal/api/server.go`.
  2. Review the existing tests in `platform/api/internal/api/security_test.go`, `platform/api/internal/api/security_dns_test.go`, and `platform/api/internal/api/handlers_test.go` so the new coverage follows the current style and helper patterns.
  3. Keep the change set minimal and local to the API module. Reuse existing `httputil` structured error helpers and current validation flow instead of inventing a new abstraction.
  </setup>

  <tasks>
  1. Add a new runtime config flag for the API process.
     - In `platform/api/cmd/server/config.go`, add a boolean field for private target allowance and load it from `PLATFORM_API_ALLOW_PRIVATE_TARGETS` using the shared config bool helper, defaulting to `false`.
     - In `platform/api/cmd/server/main.go`, pass that value into `api.NewServer`.
     - In `platform/api/internal/api/server.go`, extend `ServerConfig` with a boolean field that exposes this toggle to handlers.
     - Do not change validation requirements in `Config.Validate`; this setting is optional.

  2. Extend the URL submission request contract without changing current defaults.
     - In `platform/api/internal/api/handlers_jobs_url_submit.go`, add `AllowPrivateTargets bool` to the inline request struct with the JSON key `allow_private_targets`.
     - Keep all existing body size, URL count, URL length, module normalization, scanner config, and highlight style behavior unchanged.
     - Preserve existing success responses and event publishing.

  3. Add the new request-level gate before URL validation is relaxed.
     - In `handleJobURLSubmit`, if `req.AllowPrivateTargets` is `true` while the server config toggle is `false`, return `400` using `httputil.RespondStructuredError` plus `httputil.NewValidationError`.
     - Set the field to `allow_private_targets`.
     - Use a clear message that this API instance does not permit private target scans.
     - Return before job creation or publish logic.
     - Do not convert other existing URL validation failures to structured errors in this pass; keep their current response style unchanged.

  4. Refactor SSRF validation in `platform/api/internal/api/security.go` into an explicit mode instead of a blanket bypass.
     - Keep `parseTargetURL` strict and unchanged in spirit: still require non-empty input, parseable URLs, `http` or `https` only, and a host.
     - Introduce a small validation mode or options type that makes the decision explicit at the call site.
     - Update `validateTargetURLsWithResolver` so it accepts that mode and routes each target through mode-aware host validation.
     - Update the handler call in `handleJobURLSubmit` so the active path uses the correct mode based on `req.AllowPrivateTargets` and the server config toggle.

  5. Make private-target mode permissive enough for broader local workflows without becoming fully unrestricted.
     - Public mode must preserve the current effective behavior.
     - Private-target mode must still reject:
       - metadata `169.254.169.254`
       - link-local addresses
       - unspecified addresses
       - multicast addresses
       - invalid or unparseable URLs
       - non-`http` and non-`https` schemes
       - IPv6 unique local ranges
     - Private-target mode may additionally allow:
       - IPv4 loopback `127.0.0.0/8`
       - IPv6 loopback `::1`
       - RFC1918 IPv4 ranges `10.0.0.0/8`, `172.16.0.0/12`, and `192.168.0.0/16`
       - hostnames that resolve to public addresses and those explicitly allowed private/local addresses
     - Keep blocking broader reserved ranges that are not part of that allowlist, including metadata and other already-blocked non-public ranges.

  6. Make hostname resolution rules decision-complete.
     - For literal IPs, evaluate the parsed address directly against the mode-specific rules.
     - For hostnames, resolve all returned IPs.
     - In public mode, reject if any resolved address is non-public by current rules.
     - In private-target mode, allow only when every resolved address is either public or explicitly allowed private/local.
     - In private-target mode, reject if any resolution includes metadata, link-local, unspecified, multicast, unique-local, or any other still-blocked range.
     - Preserve nil-resolver fallback to `net.DefaultResolver`.

  7. Update and add tests to prove the behavior.
     - In `platform/api/internal/api/security_test.go`, add or adjust coverage for:
       - allowed public URLs in public mode
       - blocked loopback and RFC1918 targets in public mode
       - allowed `localhost`, `127.0.0.1`, `::1`, and RFC1918 IPv4 addresses in private-target mode
       - blocked metadata, link-local, unspecified, multicast, invalid schemes, and invalid URLs in private-target mode
     - In `platform/api/internal/api/security_dns_test.go`, cover hostname resolution behavior for localhost-style names and ensure metadata or other still-blocked resolutions are rejected in private-target mode.
     - In `platform/api/internal/api/handlers_test.go`, add handler tests that verify:
       - `allow_private_targets` omitted keeps the current behavior
       - `allow_private_targets=true` with server toggle off returns the new structured `400` validation error
       - `allow_private_targets=true` with server toggle on permits an otherwise-blocked private/local target and still reaches normal job creation flow
     - Keep test helpers and style consistent with the existing API tests.

  8. Keep the implementation tight.
     - Do not add new endpoints, new packages, or broad refactors.
     - Do not touch frontend, tools, infra, docs, or transport code in this run.
     - If you need to rename helpers in `security.go` for clarity, keep the surface area small and local to the file.
  </tasks>

  <testing>
  1. Run targeted API tests first:
     `cd platform/api && go test ./...`
  2. Run the full repo-standard verification after the API tests pass:
     `just ci`
  3. Successful completion means:
     - all new and existing relevant API tests pass
     - default public-target behavior is unchanged
     - private/local targets are only permitted when both the request flag and server config toggle are enabled
     - metadata and other still-blocked targets remain rejected in all modes
     - the phase is clearly limited to API groundwork, not end-to-end localhost or private-network support
  4. If full verification fails for reasons unrelated to this change, clearly report the exact failing area and why it is unrelated after confirming the API-specific tests still pass.
  </testing>

Output `<promise>COMPLETE</promise>` when all tasks are done.
