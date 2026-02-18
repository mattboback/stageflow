# Standing Rules

## Stale Policy

| Condition | Days | Action |
|-----------|------|--------|
| Issue waiting on reporter | 30 | Comment asking for update |
| Issue waiting on reporter | 60 | Close as stale |
| PR waiting on author | 30 | Close as stale |

## External PR Handling

- Never merge external PRs directly
- Extract intent, implement the fix, thank contributor, close their PR with explanation
- Credit the contributor in commit message or CHANGELOG

## Commit Quality

- Squash "wip" and "Update" commits before merging to main
- Use conventional commits: `feat:`, `fix:`, `chore:`, `refactor:`, `docs:`, `test:`

## Artifact Policy

- Never commit screenshots, build outputs, coverage reports, or tool caches
- If an artifact slips through, remove with `git rm` and update `.gitignore` same PR
