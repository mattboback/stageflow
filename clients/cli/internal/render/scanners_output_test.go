package render

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
	"github.com/mattboback/stageflow/clients/cli/internal/testsupport"
)

func TestRenderScanners_DefaultTextOutput(t *testing.T) {
	response := apiclient.ScannersResponse{
		Scanners: []apiclient.ScannerInfo{
			{ID: "seo", Name: "SEO Scanner", Enabled: true, Categories: []string{"seo"}, Version: "1.0.0"},
		},
		Total:   1,
		Enabled: 1,
	}

	var buf bytes.Buffer

	err := Scanners(&buf, response, FormatText)
	testsupport.RequireNoErr(t, err)

	output := buf.String()
	if !strings.Contains(output, "Scanners (enabled 1/1)") {
		t.Fatalf("text output missing summary header: %q", output)
	}

	if !strings.Contains(output, "seo") {
		t.Fatalf("text output missing scanner id: %q", output)
	}
}

func TestRenderScanners_JSONOutput(t *testing.T) {
	response := apiclient.ScannersResponse{
		Scanners: []apiclient.ScannerInfo{
			{ID: "axe", Name: "Axe Accessibility Scanner", Enabled: true},
		},
		Total:   1,
		Enabled: 1,
	}

	var buf bytes.Buffer

	err := Scanners(&buf, response, FormatJSON)
	testsupport.RequireNoErr(t, err)

	output := buf.String()
	if !strings.Contains(output, "\"scanners\"") {
		t.Fatalf("json output missing scanners field: %q", output)
	}

	if !strings.Contains(output, "\"axe\"") {
		t.Fatalf("json output missing scanner id: %q", output)
	}
}

func TestRenderScanners_MarkdownOutput(t *testing.T) {
	response := apiclient.ScannersResponse{
		Scanners: []apiclient.ScannerInfo{
			{
				ID:         "link-checker",
				Name:       "Link Checker",
				Enabled:    true,
				Categories: []string{"quality"},
				Version:    "1.0.0",
			},
		},
		Total:   1,
		Enabled: 1,
	}

	var buf bytes.Buffer

	err := Scanners(&buf, response, FormatMarkdown)
	testsupport.RequireNoErr(t, err)

	output := buf.String()
	for _, want := range []string{
		"# Scanners",
		"| ID | Name | Enabled | Categories | Version | Built-In |",
		"| link-checker | Link Checker | true | quality | 1.0.0 | false |",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("markdown output missing %q: %q", want, output)
		}
	}
}
