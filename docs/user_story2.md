# User Story 2: Multi-Site Quality Monitoring for Freelancers and Small Teams

## Decision Summary

As a developer managing multiple client or project sites, I am evaluating whether to use StageFlow as a continuous quality monitoring service that watches all my sites and tells me when something regresses — without it becoming a second job.

## Goal

I want a single system that scans my 5-10 sites on a schedule, compares each scan against a known-good baseline, and alerts me when something breaks: a WordPress plugin update that removes alt attributes, a hosting migration that drops security headers, a CMS change that kills OG tags.

So that I catch problems within hours instead of discovering them weeks later from a client call, a Google ranking drop, or a compliance audit.

## Current Situation

**What I'm doing now:** Running Lighthouse manually from DevTools a few times a year per site. Checking securityheaders.com when I remember. Relying on Google Search Console to tell me when SEO falls off — which means I find out weeks after the damage. When a client asks "did the update break anything?" I spend an hour clicking through the site manually.

**The problem:** Client sites get touched constantly — plugin auto-updates, CMS upgrades, hosting migrations, content editor mistakes. Each can silently break something real. I have no systematic, low-effort way to detect these across all my sites.

**Pain level: High.** Not because the tools don't exist — because combining them (axe + Lighthouse + header checker + link checker + OG validator) and running them consistently across 8 domains is a multi-tool, multi-login, copy-paste-into-a-spreadsheet workflow I will simply not maintain.

## Options

**Option 1: StageFlow with project registry and scheduled scans.**
Each site is a registered project with a baseline. The platform scans on a schedule (daily/weekly), diffs against baseline, and sends a webhook or notification on regression. One dashboard shows all projects and their status.

**Option 2: Ahrefs Site Audit + manual Lighthouse.**
Ahrefs covers SEO and broken links with scheduling built in. Lighthouse covers performance and basic accessibility manually. Security headers, OG tags need separate tools. Each costs separately, results live in different places.

**Option 3: DIY toolchain (axe-cli + lighthouse-cli + broken-link-checker + http-observatory).**
Open source, run from a VPS cron job or GitHub Actions. Full control. The integration glue — normalizing outputs, diffing, alerting — is all DIY.

## Evaluation Criteria (1-5)

| Criterion | StageFlow + Projects | Ahrefs + manual | DIY toolchain |
|---|---|---|---|
| Ease of setup | 4 | 3 | 2 |
| Daily use friction | 5 (automatic) | 2 (multi-tool) | 3 (scripts) |
| Scanner coverage | 5 (7 scanners, unified) | 3 (gaps in a11y, headers) | 4 (if maintained) |
| Multi-site management | 4 (project registry) | 3 (Ahrefs dashboard) | 2 (manual) |
| Regression detection | 5 (stable ID diff) | 2 (no issue-level diff) | 2 (DIY) |
| Alerting | 4 (webhook) | 3 (Ahrefs email) | 2 (DIY) |
| Cost | 4 | 1 ($129+/mo) | 5 |
| Maintenance burden | 4 | 3 | 1 |

## Pros and Cons

### Option 1: StageFlow with project registry and scheduled scans

**Pros:**
- One platform covers axe, SEO, security-headers, link-checker, open-graph, Lighthouse, spelling-grammar. No stitching tools together.
- Stable issue IDs make weekly diffs meaningful: I can tell exactly which violations appeared since last scan, not just that a score changed.
- Project-level baseline means I don't manage JSON files per client. The server tracks what "normal" looks like for each site.
- Scheduled scanning means I do nothing between check-ins. The platform watches, I react.
- `--format markdown` output can go directly into a client email, Slack message, or GitHub issue without reformatting.
- Webhook on regression: I find out when it happens, not when I remember to check.

**Cons:**
- The project registry, scheduling, and webhook features don't exist yet. They're the logical next step but require development.
- Platform is v0.1.0. No uptime SLA, no public status page.
- No multi-site dashboard yet. Today each scan is independent.
- Cost model undocumented. Free access may be temporary.

### Option 2: Ahrefs + manual Lighthouse

**Pros:**
- Ahrefs has scheduling, a dashboard, and email alerts built in.
- Established company with real uptime history.
- Lighthouse data matches Google's ranking signals.

**Cons:**
- Ahrefs does not do WCAG-level accessibility scanning. Won't catch a missing `aria-label` or broken landmark structure.
- No unified report. Correlating findings across 3-4 tools every time.
- Ahrefs Lite is $129/month. Hard to justify for a freelancer's monitoring overhead.
- Security headers and OG validation require separate tools with separate logins.
- No issue-level diffing — just "score went up/down."

### Option 3: DIY toolchain

**Pros:**
- Complete control over everything.
- As cheap as a cron job on a $5 VPS.
- No external platform dependency.

**Cons:**
- Each scanner has a different output format. Normalization is a project, not a feature.
- No stable issue IDs without building that logic myself.
- Maintenance surface is real: scanner version upgrades, Node compatibility, broken dependencies. This is a second job.
- Time to first useful result: days of setup, not minutes.
- Zero built-in UI — I'm reading JSON files or building my own report renderer.

## Tradeoffs

The core tradeoff is **capability versus infrastructure maturity**.

StageFlow's scanner combination and stable ID diffing is the right primitive for multi-site monitoring. Nothing else in this evaluation gives me "scan 8 sites weekly, tell me exactly what changed" without building it myself.

But the operational layer — scheduling, project registry, dashboard, alerting — is the gap. Today StageFlow is a scanner I have to operate. The product I want is a monitor that operates itself.

Ahrefs fills the operational gap (scheduling, dashboard, alerts) but drops scanner quality and costs $1,500+/year. The DIY approach fills every gap in theory but creates a maintenance burden that compounds.

The practical question: is the project + scheduling + webhook layer a small enough addition to StageFlow to be worth building, or should I use Ahrefs now and switch later?

## User Impact

**If the project layer exists:**
- I register 8 client sites in 30 minutes. Each gets a baseline scan.
- Weekly scans run automatically. I get a Slack notification only when something regresses.
- When a client asks "did the update break anything?" I pull up their project's diff history and answer in 60 seconds.
- I can proactively reach out to clients about regressions I caught — this is billable value.

**Without the project layer (today):**
- I write a shell script that loops over client domains, runs `stageflow scan`, and diffs against saved JSON files. 2-3 hours of setup.
- I set up cron and a Slack webhook myself. Another hour.
- I manage 8 pairs of baseline/current JSON files. Ongoing friction.
- It works, but it's fragile — the kind of tooling that rots the moment I stop actively maintaining it.

## Reversibility

High. The integration is CLI calls and JSON files. No SDK dependency, no database migration. If the platform disappears, I keep my report archives and can switch to the DIY approach with the same data format.

## Recommendation

**Build the project + scheduled scan + webhook layer in StageFlow.** This is the minimum viable product that turns StageFlow from a scanner into a monitoring service.

The four primitives needed:

1. **Project entity** — id, name, URLs, scanner config, current baseline job ID.
2. **Baseline management** — server stores which scan is the baseline. Auto-promotes on clean scans or manual promotion via API.
3. **Scheduled scanning** — cron-style triggers per project. Daily or weekly.
4. **Webhook on regression** — POST to a configured URL when a diff shows new issues above a severity threshold. Payload includes the diff summary and new issue details.

Everything else — the scanning pipeline, report aggregation, issue ID stability, diff logic — already exists. The product gap is the stateful wrapper and the trigger mechanism.

## Next Step

1. Design the `projects` table schema: id, name, urls, scanners, baseline_job_id, scan_schedule, webhook_url, created_at.
2. Add `POST /api/v1/projects`, `GET /api/v1/projects/{id}`, `POST /api/v1/projects/{id}/scan` endpoints.
3. Wire the scan endpoint to run a normal job, then diff against the stored baseline and include the diff in the response.
4. Add a cron runner that triggers project scans on schedule.
5. Add webhook dispatch on regression detection.

The CLI grows one command: `stageflow project scan --remote mysite` (or the existing `stageflow scan` gains a `--project` flag). The rest is server-side.
