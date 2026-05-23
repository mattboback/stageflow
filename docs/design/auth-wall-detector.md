# Generic Auth-Wall Detector

Status: design note. Defines a diagnostic detector that flags scans which
landed on an authentication wall instead of the requested target. The
execution plan lives at
`docs/plans/2026-05-23-auth-wall-detector-v1.md`.

> **See also:** the deterministic authenticated-scanning design at
> [`docs/design/authenticated-scanning.md`](./authenticated-scanning.md),
> which §3 explicitly defers a generic detector to a follow-up. This is
> that follow-up.

This document does two things:

1. Describes the problem the detector solves and the problems it
   deliberately leaves alone.
2. Gives a code-level design for where the detector lives, what signals
   it raises, and how it surfaces in the unified report.

---

## 1. The problem

The deterministic auth path (storage state and form recipes, shipped in
PR #111–#113) lets a scan pass through a login when the developer
configures it. It does nothing for the much more common case where the
scan is run with no auth configured against a target whose homepage
redirects unauthenticated visitors to a login page.

Today that scan looks completely successful. axe and Lighthouse run
against the login page, surface a handful of contrast and missing-alt
findings, and the unified report says the target is in good shape. The
developer reads "no critical issues" and ships. Their actual app is
unscanned.

The detector's job is to make this failure mode loud, not to fix it.
When a scan lands on an auth wall, the unified report should say so in
the same place every other finding lives, with enough context that the
developer knows the next move (capture storage state, write a recipe,
or skip the page).

The detector is intentionally diagnostic-only. Fixing the problem is
what `Provenance.auth` is for.

## 2. What gets detected

Four signals, each independently sufficient to mark a page as
auth-walled. Multiple signals on the same page reinforce confidence and
all surface in the issue metadata.

### 2.1 Login redirect

The page navigated away from the requested URL to a path that matches a
login pattern. The pattern is the union of these regexes against the
final URL's pathname (case-insensitive):

- `^/login/?$`, `^/log-in/?$`, `^/sign[_-]?in/?$`
- `^/auth(/|$)`, `^/sso(/|$)`, `^/oauth(/|$)`, `^/openid(/|$)`
- `^/accounts?/login/?$` (Django, Allauth)
- `^/users/sign_in/?$` (Devise)
- `^/identity/account/login` (ASP.NET Identity)
- `^/admin/login/?$`

A redirect away from the requested URL is required; landing on a path
that already matched a login pattern (the developer pointed the scan at
the login page directly) is not a wall, it is the requested target.

### 2.2 Auth status code

The main-document response carried `401 Unauthorized` or `403 Forbidden`.
This catches API-style auth walls and SSO interstitials that respond
with a status code rather than a redirect. Implementation captures the
return value of `page.goto()` and reads `response.status()`.

### 2.3 Login form heuristic

The DOM after navigation contains a `<form>` with at least one
`<input type="password">` and at least one `<input>` whose name, id, or
autocomplete attribute matches `email|user|login|account|username`. This
catches single-page apps that render a login form without changing the
URL. The check is a single `page.evaluate()` call that returns a boolean
plus the form's selector for the metadata.

### 2.4 Captcha presence

The DOM contains an iframe whose `src` attribute matches one of the
common managed-challenge providers:

- Google reCAPTCHA: `^https://www\.google\.com/recaptcha/`
- hCaptcha: `^https://(newassets\.)?hcaptcha\.com/`
- Cloudflare Turnstile: `^https://challenges\.cloudflare\.com/`
- Cloudflare interstitial: `<title>Just a moment...</title>` AND a
  `cf-mitigated` HTTP response header

A captcha alone is not necessarily an auth wall, but in the path of an
unauthenticated scan it almost always means the scan did not see real
content. The signal is reported alongside any others; it does not
single-handedly produce a finding without at least one of (2.1, 2.2,
2.3) also firing.

## 3. When the detector runs

The detector runs on every page navigation in `PageIterator`, after
`BrowserManager.navigateToPage` returns the response and before the
scan callback executes.

It runs whether or not `Provenance.auth` is configured. Two modes:

- **Auth not configured.** Any signal produces an Issue with id
  `auth-wall-detected`, severity `info`, category `auth`. The scan
  proceeds; the page still gets audited for whatever findings exist
  on the wall page.
- **Auth configured and `auth_hydrated` succeeded.** A signal here
  means session hydration worked but the cookies did not carry to the
  requested page (stale storage state, scoped cookie, or the
  application invalidated the session mid-scan). The Issue is raised
  at severity `serious` and includes the configured auth mode in
  metadata so the developer can correlate.
- **Auth configured and hydration failed.** The detector defers to
  the existing `auth-hydration-failed` issue from
  PR #111 and stays quiet. Two issues for the same root cause is
  noise.

## 4. Where the code lives

A single new module: `services/scanner-runner/src/core/auth-wall.ts`.

```
detectAuthWall(page: Page, response: Response | null, request: AuthWallRequest): AuthWallSignals
```

`AuthWallRequest` is `{ requestedUrl: string; authConfigured: boolean;
authHydrated: boolean }`. `AuthWallSignals` is an array of typed signals
with enough metadata for the Issue:

```ts
type AuthWallSignal =
  | { type: 'login_redirect'; finalUrl: string; matchedPattern: string }
  | { type: 'auth_status'; status: 401 | 403 }
  | { type: 'login_form'; selector: string }
  | { type: 'captcha'; provider: 'recaptcha' | 'hcaptcha' | 'turnstile' | 'cf_interstitial' };
```

The detector is deterministic, side-effect free, and runs entirely off
data already in hand (the `Page` instance, the `Response` from
`page.goto`, and a small string comparison). No new network calls, no
new browser navigations.

Wiring in `PageIterator.processPage`:

1. `navigateToPage` is updated to return the `Response | null` from
   `page.goto` instead of `void`. The existing SSRF and final-URL
   validation continue to run.
2. `processPage` calls `detectAuthWall` immediately after navigation.
3. Any signals emitted are appended to the page's issues alongside
   whatever the scan callback returns. The scan callback still runs;
   the detector never short-circuits a scan.

The `executePreScanActions` step runs after the detector. If pre-scan
actions navigate to the real content (a soft-paywall dismissal, for
example), the detector's verdict on the initial landing still stands —
that is the point.

## 5. The Issue surface

Single issue id: `auth-wall-detected`. Category `auth`. Severity is the
function of the auth configuration described in §3.

Issue metadata always includes:

- `signals`: the array of `AuthWallSignal` values that fired.
- `requested_url`: the URL the scan tried to reach.
- `final_url`: the URL the page settled on after redirects.
- `auth_configured`: `true | false`.
- `auth_mode`: present only when `auth_configured` is true; one of
  `'storage_state' | 'form'`.

Description text is produced from the signals: a one-line summary
("Scan landed on a login page; configure `Provenance.auth` to scan
post-login content") followed by a per-signal explanation. The
description never includes any DOM text from the wall page beyond the
matched form selector — the wall page may be a tenant-specific SSO
portal, and we do not want to put tenant identifiers in someone else's
report.

A new fixture file under
`libs/contracts/report/issue-catalog/auth-wall-detected.json` documents
the issue for the unified report consumer, mirroring the pattern used
by `auth-hydration-failed`.

## 6. Trust and privacy

The detector reads only:

- The URL (already public to the scanner).
- The HTTP response status (already public to the scanner).
- A boolean from a one-shot `page.evaluate` (returns only the matched
  form's selector and the iframe provider name; no input values, no
  cookies, no localStorage).

Nothing the detector emits leaves the scanner runner with more
sensitivity than what already flows through page-completed events.

## 7. What this detector explicitly does not do

- **It does not attempt to log in.** That is `Provenance.auth`.
- **It does not classify the auth wall as "fixable."** A 403 may be
  intended (admin route the scan was never supposed to see); the
  developer decides.
- **It does not detect arbitrary "interesting state" the scan might
  miss.** Modal dialogs, cookie banners, geo-gates, and A/B-test
  variants are out of scope. The detector targets one specific
  failure mode: scans that quietly succeeded against a login wall.
- **It does not learn site-specific patterns.** No regex grew here for
  LinkedIn, Indeed, or any other named target. The patterns are the
  ones every web framework's login flow shares. Site-specific
  detectors live in their own scanner if they ever need to exist.
- **It does not look at the response body.** Status code and DOM are
  enough; parsing arbitrary HTML for substrings is a path toward
  false positives we do not need.
