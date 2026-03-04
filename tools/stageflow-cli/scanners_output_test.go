package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderScanners_DefaultTextOutput(t *testing.T) {
	response := ScannersResponse{
		Scanners: []ScannerInfo{
			{ID: "seo", Name: "SEO Scanner", Enabled: true, Categories: []string{"seo"}, Version: "1.0.0"},
		},
		Total:   1,
		Enabled: 1,
	}

	var buf bytes.Buffer

	err := renderScanners(&buf, response, outputFormatText)
	requireNoErr(t, err)

	output := buf.String()
	if !strings.Contains(output, "Scanners (enabled 1/1)") {
		t.Fatalf("text output missing summary header: %q", output)
	}

	if !strings.Contains(output, "seo") {
		t.Fatalf("text output missing scanner id: %q", output)
	}
}

func TestRenderScanners_JSONOutput(t *testing.T) {
	response := ScannersResponse{
		Scanners: []ScannerInfo{
			{ID: "axe", Name: "Axe Accessibility Scanner", Enabled: true},
		},
		Total:   1,
		Enabled: 1,
	}

	var buf bytes.Buffer

	err := renderScanners(&buf, response, outputFormatJSON)
	requireNoErr(t, err)

	output := buf.String()
	if !strings.Contains(output, "\"scanners\"") {
		t.Fatalf("json output missing scanners field: %q", output)
	}

	if !strings.Contains(output, "\"axe\"") {
		t.Fatalf("json output missing scanner id: %q", output)
	}
}

func TestRenderScanners_MarkdownOutput(t *testing.T) {
	response := ScannersResponse{
		Scanners: []ScannerInfo{
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

	err := renderScanners(&buf, response, outputFormatMarkdown)
	requireNoErr(t, err)

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
