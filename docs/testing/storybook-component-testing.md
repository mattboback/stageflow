# Storybook Component Testing

This document defines the monorepo standard for Storybook-driven component testing.

## Current Scope

- Foundation: monorepo conventions and CI pattern.
- Active implementation: `frontend/` only.
- Storybook version baseline: `8.6.x`.
- Story docs landing page: `frontend/src/stories/Introduction.mdx`.
- Coverage baseline:
  - UI primitives (`src/lib/components/ui/*.stories.ts`).
  - Workflow components for playground, scan status, and report surfaces.
- Test levels in active integration:
  - Interaction tests via `play` functions.
  - Accessibility checks via axe in Storybook test runner.
- Deferred to Phase 2:
  - Visual regression snapshots with Playwright.

## Conventions

1. Story file placement
- Co-locate stories with components as `*.stories.ts`.
- Use `src/stories/*.mdx` for package-level Storybook documentation pages.
- For components that need local story-only state or `Snippet` props, use story harness
  components in `src/lib/components/**/story-harnesses/`.

2. Selector strategy
- Prefer accessibility-first queries in play functions:
  - `getByRole`
  - `getByLabelText`
  - `getByText`
- Use `data-testid` only for cases without stable semantic selectors.

3. Testing focus
- Test user-visible behavior rather than implementation details.
- Keep each story test deterministic and independent.
- Put interaction assertions in the story that demonstrates the behavior.

4. Accessibility baseline
- Storybook a11y addon is enabled globally.
- Test runner runs axe checks against `#storybook-root` for each visited story.
- WCAG scope is `wcag2a` + `wcag2aa`.
- Interaction/a11y runner targets the built static output (`storybook-static`) for stable CI runs.

## Frontend Commands

Run from `frontend/`:

```bash
bun run storybook
bun run build-storybook
bun run test-storybook
bun run test-storybook:ci
bun run test-storybook:watch
```

## CI Pattern

- Storybook runs in a dedicated `frontend-storybook` CI job.
- The Storybook job is a required gate for pull requests to `main`.
- Failed runs upload `junit.xml`, `test-results`, and `storybook-static` artifacts.

## Migration Strategy

1. Keep existing Vitest component/unit tests in place during migration.
2. Add Storybook stories/tests incrementally for critical components first.
3. Expand coverage to new feature components alongside product development.

## Planned Phase 2

- Add Playwright screenshot baselines for targeted stories.
- Run visual diffs in a separate CI job initially.
