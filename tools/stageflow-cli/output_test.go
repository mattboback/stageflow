package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestRenderReportSummaryQuietAndJSON(t *testing.T) {
	status := sampleJobStatus("job-output", "DONE")
	doc := sampleReport(status.ID)

	t.Run("summary", func(t *testing.T) {
		var out bytes.Buffer

		err := renderReport(&out, status, doc, renderOptions{
			format:      outputFormatSummary,
			maxIssues:   10,
			minSeverity: severityInfo,
			threshold: thresholdResult{
				Evaluated: true,
				Passed:    true,
				Detail:    "critical<=1",
			},
		})
		if err != nil {
			t.Fatalf("renderReport returned error: %v", err)
		}

		got := out.String()
		if !strings.Contains(got, "Severity Totals: critical=1") || !strings.Contains(got, "Thresholds: PASS") {
			t.Fatalf("unexpected summary output: %s", got)
		}
	})

	t.Run("quiet", func(t *testing.T) {
		var out bytes.Buffer

		err := renderReport(&out, status, doc, renderOptions{
			format:      outputFormatQuiet,
			maxIssues:   10,
			minSeverity: severityInfo,
		})
		if err != nil {
			t.Fatalf("renderReport returned error: %v", err)
		}

		got := strings.TrimSpace(out.String())
		if got != "PASS: critical=1 serious=0 total=2" {
			t.Fatalf("quiet output = %q", got)
		}
	})

	t.Run("json", func(t *testing.T) {
		var out bytes.Buffer

		err := renderReport(&out, status, doc, renderOptions{
			format:      outputFormatJSON,
			maxIssues:   1,
			minSeverity: severityInfo,
		})
		if err != nil {
			t.Fatalf("renderReport returned error: %v", err)
		}

		var payload struct {
			JobID string `json:"job_id"`
			State string `json:"state"`
		}

		unmarshalErr := json.Unmarshal(out.Bytes(), &payload)
		if unmarshalErr != nil {
			t.Fatalf("unmarshal json output: %v\n%s", unmarshalErr, out.String())
		}

		if payload.JobID != status.ID || payload.State != status.State {
			t.Fatalf("payload = %#v", payload)
		}
	})
}
