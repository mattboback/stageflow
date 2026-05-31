# AlchemizeCV Dogfood QA

Runs the live StageFlow web Playground against the private AlchemizeCV demo
workspace with form authentication. The runner starts system Chrome with a
Chrome DevTools Protocol port, attaches via Playwright `connectOverCDP`, and
wraps itself in Xvfb when no display is available.

This is intentionally opt-in. Do not add it to default CI because it depends on
live services and demo-account credentials.

## Usage

```bash
export QA_LOGIN_EMAIL='demo@example.com'
export QA_LOGIN_PASS='...'

just dogfood-alchemizecv
```

Optional environment variables:

| Variable | Default | Description |
| --- | --- | --- |
| `QA_SITE` | `https://alchemizecv.com` | Target app origin |
| `STAGEFLOW_QA_SITE` | `https://stageflow.org` | StageFlow origin |
| `STAGEFLOW_QA_CHROME` | `/usr/bin/google-chrome` | Chrome executable |
| `STAGEFLOW_QA_CDP_PORT` | `9223` | Local CDP port |
| `STAGEFLOW_QA_TIMEOUT_MS` | `720000` | Max wait for scan completion |
| `ARTIFACT_ROOT` | `output/alchemizecv-prod-qa/<UTC timestamp>` | Output directory |

## Output

The runner writes:

- `report.md`
- `scan-verification.json`
- `report-results-fetch.json`
- `latest-status.json`
- `console.json`
- `network.json`
- `screenshots/*.png`

The verification file must show:

- `state: "DONE"`
- `reportStatus: 200`
- no missing `/dashboard`, `/applications`, or `/profile` coverage
- `authIssue: false`
- all standard scanners present with `status: "success"`

## Local Checks

```bash
node --test devtools/qa/alchemizecv-dogfood/lib.test.mjs
```

Before copying screenshots into `docs/images`, also run a secret check against
the artifact directory:

```bash
rg --fixed-strings "$QA_LOGIN_PASS" output/alchemizecv-prod-qa README.md docs/images
```

That command should return no matches.
