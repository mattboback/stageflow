package manifesttmpl

import (
	"strings"
	"testing"
)

func TestConfigYAML_HappyPath(t *testing.T) {
	out := ConfigYAML(ConfigParams{
		APIURL:   "http://localhost:8080",
		Scanners: "axe,lighthouse",
		Suggestion: Suggestion{
			Command:       "bun run dev",
			Cwd:           "clients/web",
			CommandSource: "detected package.json script",
			URL:           "http://127.0.0.1:5173",
		},
	})

	mustContain(t, out, `api_url: "http://localhost:8080"`)
	mustContain(t, out, `- http://127.0.0.1:5173`)
	mustContain(t, out, `scanners: axe,lighthouse`)
	mustContain(t, out, `cmd: ["bun", "run", "dev"]`)
	mustContain(t, out, `cwd: clients/web`)
	mustContain(t, out, `url: http://127.0.0.1:5173`)
	mustContain(t, out, "# detected package.json script")

	if !strings.HasSuffix(out, "\n") {
		t.Fatalf("expected trailing newline")
	}
}

func TestConfigYAML_EmptyDefaultsAndPlaceholder(t *testing.T) {
	out := ConfigYAML(ConfigParams{})

	mustContain(t, out, `api_url: "http://localhost:8080"`)
	mustContain(t, out, `- http://127.0.0.1:3000`)
	mustContain(t, out, `url: http://127.0.0.1:3000`)
	mustContain(t, out, DevStartCommandPlaceholder)
	mustContain(t, out, "# Replace this placeholder")
	mustContain(t, out, `cwd: .`)
}

func TestGuideMarkdown_HasKeySections(t *testing.T) {
	out := GuideMarkdown()
	mustContain(t, out, "# StageFlow project setup")
	mustContain(t, out, "## Quick setup")
	mustContain(t, out, "## Localhost/private scans")
}

func mustContain(t *testing.T, out, sub string) {
	t.Helper()

	if !strings.Contains(out, sub) {
		t.Fatalf("output missing %q:\n%s", sub, out)
	}
}
