# Roadmap

StageFlow's `0.1.0` release covers the full scan loop: URL/ZIP intake, eight scanners,
contract-driven reports, baseline diffing, a SvelteKit web app, and a Go CLI with Project
Mode. This roadmap sketches where it goes next. It is intentionally a direction, not a
commitment — issues and discussion are welcome.

## Near term

- **Scanner plugin SDK** — document and stabilize the manifest contract so third parties can
  publish scanners without touching core, including a template repo and conformance test.
- **CI integrations** — first-class GitHub Action and a reusable workflow wrapping the CLI's
  severity gate, with PR-comment summaries of new/fixed issues vs. the promoted baseline.
- **Report diffing UX** — surface baseline-vs-current diffs directly in the web report, not
  just the CLI exit code.

## Medium term

- **Metrics depth** — build on the in-process `/metrics` collector with per-scanner timing
  histograms and queue-depth gauges; ship an updated Grafana dashboard.
- **Horizontal orchestrator** — validate multi-replica orchestration and move admin-API rate
  limiting to a shared/edge limiter (see `services/orchestrator/AUDIT.md`).
- **Scheduled scans** — recurring scans against a saved project with trend history.

## Exploratory

- **More scanners** — broken-image/asset budget, mixed-content, cookie/consent, and
  Core Web Vitals field-data correlation.
- **Hosted multi-tenant mode** — auth, per-tenant isolation, and quotas beyond the current
  single-tenant self-hosted model.

## Non-goals

- Becoming a general-purpose APM or uptime monitor — StageFlow is a *frontend quality gate*.
- Replacing edge security controls (WAF, perimeter rate limiting); see the threat model in
  [docs/architecture/system.md](docs/architecture/system.md).

Have an idea or need? Open an issue — see [CONTRIBUTING.md](CONTRIBUTING.md).
