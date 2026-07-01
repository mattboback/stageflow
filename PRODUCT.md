# Product

StageFlow is a self-hostable frontend quality platform for teams that want one repeatable
answer to a practical question: **did this change make the frontend worse?**

It runs accessibility, performance, SEO, link-health, security-header, social-metadata,
content-quality, and guided-navigation scanners behind one API. Results are normalized into a
single report contract and can be compared against a promoted baseline per project.

## Primary users

- **First-time evaluators** who visit `stageflow.org`, scan a URL, and need to trust the report quickly.
- **Frontend engineers** who triage evidence, selectors, screenshots, severity, and remediation.
- **CI/self-hosting users** who use the CLI to gate builds on new issues or severity thresholds.

## Product surfaces

| Surface | Role |
| --- | --- |
| Web UI (`clients/web`) | React Router app for scan submission, live status, report triage, and artifact access |
| CLI (`clients/cli`) | Terminal-first scans, JSON/Markdown output, `--fail-on` quality gates, project workflows |
| Platform API | URL/ZIP intake, project CRUD, report/diff retrieval, SSE progress stream |
| Self-host stack | Podman compose stack with NATS, MinIO, PostgreSQL, Grafana, API, orchestrator, scanner runner |

## Product principles

1. **Show the report, do not pitch it.** The fastest path is a real scan and a real report.
2. **Trust through evidence.** Findings should show exact rules, selectors, occurrences, screenshots, and fixes.
3. **Calm under criticality.** Serious issues are presented clearly without alarmist language.
4. **One contract, many scanners.** Consumers should not need scanner-specific branches to use results.
5. **Useful in CI.** Baselines and deterministic exit codes make the tool automation-friendly.

## Success criteria

A reviewer should be able to understand the project in minutes:

1. run or inspect a scan;
2. see why the architecture is safe around untrusted inputs;
3. understand the report contract and baseline diff model;
4. identify the strongest implementation surfaces in the codebase.
