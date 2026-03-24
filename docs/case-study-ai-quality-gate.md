# Case Study: AI Coding Agent Quality Gates

This case study evaluates StageFlow as an automated quality gate for AI-assisted development workflows. It captures the tradeoffs between StageFlow, direct `axe-core` checks, and Lighthouse-style score gates when the goal is to catch regressions before they reach production.

> [!NOTE]
> **Status note:** This is an evaluation and roadmap-oriented case study. StageFlow already supports multi-scanner scans, remote project records, baseline promotion, diffing, and CLI quality gates today. Auto-promotion and tighter deploy-hook workflows described below are future polish, not shipped guarantees.

## Use-case summary

A solo developer using AI coding agents wants an automated way to catch accessibility, SEO, and link regressions introduced by fast-moving code changes before they reach production.

## Goal

I want my AI coding agent to know — within seconds of making a change — whether it broke something on the live site: a missing landmark, a dropped meta tag, a dead link.

So that the agent can self-correct before committing, and I stop discovering regressions from users or manual Lighthouse runs days after the damage is done.

## Current Situation

**What I'm doing now:** Nothing automated. I run Lighthouse in DevTools when I remember, occasionally check axe. When I ask an AI agent to "refactor the nav component," it has no idea it dropped `aria-label` on the hamburger button or removed the canonical URL. I only find out when I manually check — which I don't always do.

**The problem:** AI agents are fast but blind to quality signals outside of linting and type checking. Each AI-assisted sprint creates more surface area I can't manually verify. I've already shipped accessibility regressions that took longer to find than they would have taken to catch.

**Pain level: High.** Not because my site is broken today — high because the feedback loop is broken. The cost compounds with every sprint.

## Options

**Option 1: StageFlow with remote projects and baseline tracking.**
Register a project on a running StageFlow API, promote a baseline, and use `stageflow scan --project ...` to diff against it. Wire the CLI into a post-deploy hook or agent loop. The diff output already highlights new, fixed, and unchanged issues.

**Option 2: axe-core in Playwright test suite.**
Add `@axe-core/playwright` to existing E2E tests. Catches WCAG violations on routes the test suite already visits. No external service, runs in-process.

**Option 3: Lighthouse CI + custom scripts.**
`lhci autorun` in GitHub Actions. Score budgets fail the build on performance or accessibility drops. Add separate scripts for link checking, OG validation, security headers.

## Evaluation Criteria (1-5)

| Criterion | StageFlow + Projects | axe-core in tests | LHCI + scripts |
|---|---|---|---|
| Ease of setup | 4 | 3 | 2 |
| Daily use friction | 5 (automatic) | 4 | 3 |
| Scanner coverage | 5 (7 scanners) | 2 (WCAG only) | 3 |
| Regression detection | 5 (stable ID diff) | 1 (no diffing) | 2 (score-based) |
| Agent-actionable output | 5 (JSON + selectors) | 3 | 2 |
| Cost | 3 | 5 | 4 |
| Maintenance burden | 4 (hosted) | 3 | 1 |
| Speed | 3 (~15-30s) | 5 (~2s) | 3 |

## Pros and Cons

### Option 1: StageFlow with project baseline tracking

**Pros:**
- Single invocation covers axe, SEO, link-checker, security-headers, open-graph, Lighthouse — all the signals that AI agents are blind to.
- Stable content-based issue IDs let the diff tell me exactly which violations are new vs. pre-existing. This is the critical primitive — "did this change make things worse?" requires it.
- JSON output with CSS selectors, HTML snippets, WCAG references, and fix guidance is directly consumable by an AI agent for self-correction.
- `--fail-on` exit codes work as a CI gate with zero output parsing.
- Project baseline on the server means I don't manage local JSON files — the API already knows what "normal" looks like for my site once I promote a baseline.

**Cons:**
- External service dependency. If stageflow.org is down, the gate is broken.
- Scan latency (~15-30 seconds) is acceptable for CI but too slow for a tight edit-scan-fix loop inside an agent session.
- New platform (v0.1.0). Limited track record.

### Option 2: axe-core in tests

**Pros:**
- Zero external dependency. Fastest feedback (~2 seconds).
- Well-established library with years of production use.

**Cons:**
- WCAG only. Misses broken links, SEO metadata, security headers, OG tags — exactly the things AI agents silently break.
- No diffing. Can't distinguish new violations from pre-existing ones.
- Coverage limited to routes the test suite visits.

### Option 3: LHCI + scripts

**Pros:**
- Score budgets catch performance regressions.
- No external service dependency.

**Cons:**
- Score-based diffing tells me something dropped, not what. I'm back to manual investigation.
- Assembling link-checker + OG validator + security-headers checker is a maintenance project.
- Lighthouse scores are noisy in CI due to CPU throttling on GitHub runners.

## Tradeoffs

StageFlow wins on coverage breadth and diff quality. axe-in-tests wins on speed and zero-dependency reliability. The real question is: do I only care about WCAG, or do I care about the full quality surface?

If an AI agent removes `og:image`, drops the canonical URL, and introduces a broken link in the same refactor — axe-in-tests catches none of it. StageFlow catches all three.

The practical answer: axe-in-tests as the fast inner loop (catches WCAG in seconds during development), StageFlow as the comprehensive outer loop (catches everything else on deploy).

## User Impact

- **Pre-ship confidence:** Replaces my mental checklist with an automated gate. I stop asking "did I check accessibility?" because the answer is always yes.
- **Agent feedback loops:** The JSON diff report is directly pasteable into an agent context. "Fix these 2 new issues" closes the loop from "agent introduced a bug" to "agent fixed the bug" without manual translation.
- **Risk if unreliable:** If scans are slow (>2 min) or noisy (false positives), I'll skip them. The gate becomes a checkbox exercise.

## Reversibility

High. The integration is a CLI call in CI config. Removing it means deleting one workflow step. No SDK dependency, no test changes, no database migration. Local JSON baselines are portable.

## Recommendation

**Use the current remote-project flow now, then tighten the automation around it.** The scanning, baseline promotion, and diff primitives already exist. The highest-value next step is a smoother preview/deploy loop around those capabilities, not inventing the project layer from scratch.

The ideal agent workflow:
1. Agent makes code changes
2. Deploy to preview URL
3. `stageflow scan --project mysite` → API runs scan, diffs against the promoted baseline, returns the result
4. If regressions: agent reads the diff (selectors + fix guidance), self-corrects, redeploys
5. If clean: baseline auto-promotes, agent commits

## Next Step

Make the current flow easier to adopt in CI and agent loops: clearer project bootstrapping, easier baseline promotion in deploy workflows, and stronger examples for feeding diffs back into automated remediation.
