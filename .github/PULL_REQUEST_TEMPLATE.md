## Summary

Describe the change, the motivation behind it, and any context a reviewer should know.

Fixes # (issue)

## Areas touched

- [ ] `clients/web`
- [ ] `clients/cli`
- [ ] `services/platform-api`
- [ ] `services/scanner-runner`
- [ ] Docs / repo metadata
- [ ] Devtools / infrastructure

## Validation

List the commands you ran and any manual verification performed.

- [ ] `just ci`
- [ ] `just storybook-test` (if UI or component behavior changed)
- [ ] `just shell-tests` (if CLI or setup behavior changed)
- [ ] Local manual verification (UI or CLI)
- [ ] Deployment or runtime follow-up noted below

## Screenshots / docs

Attach screenshots, terminal output snippets, or note why they were not needed.

## Checklist

- [ ] Docs, screenshots, or configuration references were updated when needed
- [ ] New env vars, operational steps, or migration notes were documented
- [ ] I verified the change at the surface it affects
