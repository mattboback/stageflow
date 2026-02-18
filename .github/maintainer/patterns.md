# Observed Patterns

## Artifact Commits

### Screenshot blobs in `.claude/` and `output/`
- **First seen:** 2026-02-18 (initial triage)
- **Frequency:** 29 PNG files currently tracked
- **Root cause:** AI agent sessions (dev-browser skill, playwright) produce screenshots that were committed without gitignore coverage
- **Resolution:** Remove files, add to `.gitignore`
- **Prevention:** Add `output/`, `.claude/skills/*/tmp/` to `.gitignore`

## Commit Message Quality

### Low-signal commit messages
- **Observed:** "wip", "wip", "Update", "Update" in recent history
- **Impact:** Harms changelog generation and contributor understanding
- **Note:** These are local development commits; public-facing commits are better (feat/fix/chore convention used)

## Codebase Patterns

- Polyglot monorepo: Go + TypeScript/Bun + SvelteKit. `go.work` for workspace.
- Consistent conventional commits on meaningful changes (`feat:`, `fix:`, `chore:`, `refactor:`)
- Strong SSRF guardrails and ZIP safety limits
- Test coverage enforced in CI (~50% threshold via vitest)
- Dependabot active for Go modules and GitHub Actions
