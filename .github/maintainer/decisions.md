# Decision Log

## 2026-02

### [HYGIENE] Remove tracked screenshot artifacts
**Date:** 2026-02-18
**Decision:** Remove `output/playwright/*.png` and `.claude/skills/dev-browser/tmp/*.png` from git. Add both paths to `.gitignore`. These are agent/session working artifacts with no value in version history.
**Reasoning:** 29 PNG files added during dev/AI sessions — bloat, no historical value, confuse contributors.

### [HYGIENE] Add .gitignore entries for tool caches
**Date:** 2026-02-18
**Decision:** Add `.ruff_cache/`, `.sisyphus/`, `output/`, `.claude/skills/*/tmp/` to `.gitignore`.
**Reasoning:** These paths exist locally but were not covered by `.gitignore`. Risk of future accidental commits.

### [REPO-META] Add GitHub topics and description
**Date:** 2026-02-18
**Decision:** Add topics and a one-line description to the GitHub repo metadata.
**Reasoning:** Repo has no description or topics — hurts discoverability.
