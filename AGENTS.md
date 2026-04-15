# Agent Rules and Guidelines

## Who We Are

This repository is maintained by a solo developer it has no users and no contributors. Optimize for clarity, speed, and coherence over compatibility.

Because of this:

1. Do not preserve legacy patterns, compatibility layers, shimsEVER.

# AGENTS.md

Behavioral guidelines to reduce common LLM coding mistakes. Merge with project-specific instructions as needed.

**Tradeoff:** These guidelines bias toward caution over speed. For trivial tasks, use judgment.

## 1. Think Before Coding

**Don't assume. Don't hide confusion. Surface tradeoffs.**

Before implementing:

- State your assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them - don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

## 2. Simplicity First

**Minimum code that solves the problem. Nothing speculative.**

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.

Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

## 3. Surgical Changes

**Touch only what you must. Clean up only your own mess.**

When editing existing code:

- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it - don't delete it.

When your changes create orphans:

- Remove imports/variables/functions that YOUR changes made unused.

The test: Every changed line should trace directly to the user's request.

## 4. Goal-Driven Execution

**Define success criteria. Loop until verified.**

Transform tasks into verifiable goals:

- "Add validation" → "Write tests for invalid inputs, then make them pass"
- "Fix the bug" → "Write a test that reproduces it, then make it pass"
- "Refactor X" → "Ensure tests pass before and after"

For multi-step tasks, state a brief plan:

```
1. [Step] → verify: [check]
2. [Step] → verify: [check]
3. [Step] → verify: [check]
```

Strong success criteria let you loop independently. Weak criteria ("make it work") require constant clarification.

---

**These guidelines are working if:** fewer unnecessary changes in diffs, fewer rewrites due to overcomplication, and clarifying questions come before implementation rather than after mistakes.

# StageFlow repo guidance

This repo owns the StageFlow application code, local compose workflows, and self-hosting assets.

## Shared VPS production deployment

Production for `stageflow.org` on the shared VPS is **not** managed from this repo.
It is managed from the root deployment workspace at `/home/matt/Deployment`.

Read these files before changing production routing, topology, or restart behavior:

1. `/home/matt/Deployment/DEPLOYMENT_STRATEGY.md`
2. `/home/matt/Deployment/justfile`
3. `/home/matt/Deployment/gateway/Caddyfile`

Canonical production command:

```bash
cd /home/matt/Deployment
just deploy stageflow
```

## Ownership boundary

- This repo owns local and staging compose flows under `infra/compose/`.
- This repo does **not** own the shared VPS systemd or quadlet install path.
- The root deployment workspace owns `stageflow-quadlets/`, the gateway routing, and the authoritative production restart flow.

## Anti-drift rules

- Do not add standalone VPS production deploy scripts here.
- Do not add repo-local systemd or quadlet files for the shared VPS here.
- Keep production guardrails pointing back to `/home/matt/Deployment`.
- If local compose and shared VPS production differ, document the difference explicitly instead of blurring the workflows.
