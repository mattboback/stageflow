// Package manifesttmpl renders the YAML and Markdown templates that
// `stageflow project init` scaffolds inside a project's .stageflow directory.
package manifesttmpl

import (
	"fmt"
	"strconv"
	"strings"
)

// DevStartCommandPlaceholder is written to `dev.start.cmd` when the CLI could
// not detect a reasonable dev command for the project.
const DevStartCommandPlaceholder = "__STAGEFLOW_SET_DEV_START_CMD__"

// Suggestion captures the best-effort detection results the CLI uses when
// scaffolding a project config.
type Suggestion struct {
	Command       string
	Cwd           string
	CommandSource string
	URL           string
}

// ConfigParams carries the inputs that drive ConfigYAML rendering.
type ConfigParams struct {
	APIURL     string
	Scanners   string
	Suggestion Suggestion
}

// ConfigYAML returns the contents of .stageflow/config.yaml for a freshly
// scaffolded project.
func ConfigYAML(params ConfigParams) string {
	baseAPI := strings.TrimSpace(params.APIURL)
	if baseAPI == "" {
		baseAPI = "http://localhost:8080"
	}

	devURL := strings.TrimSpace(params.Suggestion.URL)
	if devURL == "" {
		devURL = "http://127.0.0.1:3000"
	}

	devCommand := strings.TrimSpace(params.Suggestion.Command)
	if devCommand == "" {
		devCommand = DevStartCommandPlaceholder
	}

	parts := strings.Fields(devCommand)

	var quotedParts []string
	for _, part := range parts {
		quotedParts = append(quotedParts, strconv.Quote(part))
	}

	devCommandFormatted := strings.Join(quotedParts, ", ")

	devCwd := strings.TrimSpace(params.Suggestion.Cwd)
	if devCwd == "" {
		devCwd = "."
	}

	commandComment := "Replace this placeholder with your real dev command."
	if source := strings.TrimSpace(params.Suggestion.CommandSource); source != "" {
		commandComment = source
	}

	return strings.TrimSpace(fmt.Sprintf(`
version: 1

stageflow:
  api_url: %s
  # Optional: link this repo to a hosted StageFlow project for regression memory.
  # remote_project: your-hosted-project-slug
  # remote_api_url: https://api.stageflow.example

scan:
  # Set this to the page URL your dev server serves.
  urls:
    - %s
  scanners: %s
  allow_private_targets: true

dev:
  start:
    # %s
    cmd: [%s]
    cwd: %s
  ready:
    # Match this to your dev server URL.
    url: %s
`, strconv.Quote(baseAPI), devURL, params.Scanners, commandComment, devCommandFormatted, devCwd, devURL)) + "\n"
}

// GuideMarkdown returns the contents of .stageflow/README.md that walks users
// through setting up Project Mode.
func GuideMarkdown() string {
	return strings.TrimSpace(`
# StageFlow project setup

This folder configures `+"`stageflow project`"+` for this repository.

## Quick setup

1. Open `+"`config.yaml`"+` in this folder.
2. Set `+"`dev.start.cmd`"+` to the command that starts your app.
3. Set `+"`dev.ready.url`"+` to the URL that returns HTTP 2xx or 3xx when your app is ready.
4. Set `+"`scan.urls`"+` to the page URLs you want scanned.
5. Optional: set `+"`stageflow.remote_project`"+` (and `+"`stageflow.remote_api_url`"+` if needed) to link hosted regression memory.
6. Run `+"`stageflow project`"+` again.

## Example dev commands

- npm: `+"`cmd: [\"npm\", \"run\", \"dev\"]`"+`
- bun: `+"`cmd: [\"bun\", \"run\", \"dev\"]`"+`
- pnpm: `+"`cmd: [\"pnpm\", \"dev\"]`"+`
- yarn: `+"`cmd: [\"yarn\", \"dev\"]`"+`

## Localhost/private scans

For local targets like `+"`localhost`"+` and `+"`127.0.0.1`"+`:

1. Start the StageFlow local overlay:
   - `+"`just dev up local`"+`
   - `+"`just dev init local`"+`
2. Re-run `+"`stageflow project`"+`.

## Hosted regression memory (optional)

After your local localhost loop passes, you can follow up against a hosted project baseline:

- Set `+"`stageflow.remote_project`"+` to the hosted project slug.
- Set `+"`stageflow.remote_api_url`"+` if the hosted project uses a different API base URL.
- Run `+"`stageflow project hosted --format json`"+` to scan the hosted project without starting local dev.
- Run `+"`stageflow project doctor --format json`"+` when agents need structured setup/readiness details.

## Troubleshooting

- If you see "ENOENT" for your dev command, verify `+"`dev.start.cmd`"+` and `+"`dev.start.cwd`"+`.
- If readiness times out, verify `+"`dev.ready.url`"+` responds while your app is running.

For full documentation, see the [Project Mode guide](https://github.com/mattboback/stageflow/blob/main/docs/PROJECT_MODE.md).
`) + "\n"
}
