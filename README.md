# StageFlow

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

StageFlow audits a website at [stageflow.org](https://stageflow.org) and diffs each deployment against the last good one, so you see what a release broke, what it fixed and how the score moved.

![The /demo comparison of two versions of a sample site: the score down 3, 3 new findings and 1 fixed, then the report's prioritized fixes](docs/images/report-review.png)

Five scanners check every page: axe for accessibility, Lighthouse for accessibility, best practices and SEO, plus StageFlow's own SEO, security-header and link checks. Each scan runs in its own sandboxed containers, and the report marks findings on screenshots of each page and says how to fix them. Scan any public URL at [stageflow.org](https://stageflow.org), no account needed; [/demo](https://stageflow.org/demo) shows the comparison above. Scan records and reports expire 24 hours after completion ([privacy](docs/privacy.md)).

## In CI

Scan a reviewed build once and keep the JSON as the baseline. On every change, scan again and diff:

```bash
stageflow scan https://preview.example.com --profile staging --format json --max-issues 0 > current.json
stageflow diff baseline.json current.json --fail-on-new=any \
  --format markdown >> "$GITHUB_STEP_SUMMARY"
```

Deploy a preview first, then scan its public URL. Hosted scans accept up to 25 pages; ZIP and build-directory uploads are available on self-hosted deployments. The diff exits 1 when a release adds a finding and writes this to the step summary:

![The diff's markdown digest in a GitHub step summary: score 75 to 70, 3 new issues and 1 fixed, each new one with its page and selector](docs/images/step-summary.png)

Download the CLI from [GitHub Releases](https://github.com/mattboback/stageflow/releases) or run `go install github.com/mattboback/stageflow/clients/cli/cmd/stageflow@latest`. The [CLI guide](docs/cli.md) covers URL and authenticated scans, gates, the composite action and exit codes. In the browser, [projects](https://stageflow.org/projects) keep runs and baselines on your device, with no account.

## More

- [Case study](docs/case-study.md): the design decisions, what I removed and why, and a short code tour.
- [Architecture](docs/architecture.md): data flow, the job state machine and the trust boundaries.
- [Self-hosting](docs/self-hosting.md): one Compose file and rootless Podman.
- [Security policy](SECURITY.md) and [contributing](CONTRIBUTING.md).

## License

StageFlow is released under the [MIT License](LICENSE).
