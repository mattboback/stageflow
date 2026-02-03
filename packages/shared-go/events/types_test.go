package events

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/mattboback/stageflow/packages/shared-go/models"
)

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

func mustUnmarshalMap(t *testing.T, b []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal map: %v", err)
	}
	return m
}

func TestEventConstants_NonEmptyUniqueAndWellFormed(t *testing.T) {
	t.Parallel()

	events := []string{
		EventJobCreated,
		EventJobCompleted,
		EventJobFailed,
		EventExtractionReady,
		EventExtractionFailed,
		EventScanPageCompleted,
		EventScanCompleted,
		EventScanFailed,
	}

	seen := map[string]struct{}{}
	for _, e := range events {
		if e == "" {
			t.Fatalf("event constant should not be empty")
		}
		if _, ok := seen[e]; ok {
			t.Fatalf("duplicate event constant: %q", e)
		}
		seen[e] = struct{}{}

		if strings.ToLower(e) != e {
			t.Fatalf("event constant should be lowercase: %q", e)
		}
		if !strings.Contains(e, ".") {
			t.Fatalf("event constant should be dot-delimited: %q", e)
		}
		if strings.ContainsAny(e, " \t\n") {
			t.Fatalf("event constant should not contain whitespace: %q", e)
		}
	}

	// Stronger convention checks: prefix by domain.
	prefixCases := []struct {
		event  string
		prefix string
	}{
		{EventJobCreated, "job."},
		{EventJobCompleted, "job."},
		{EventJobFailed, "job."},
		{EventExtractionReady, "extraction."},
		{EventExtractionFailed, "extraction."},
		{EventScanPageCompleted, "scan."},
		{EventScanCompleted, "scan."},
		{EventScanFailed, "scan."},
	}
	for _, tc := range prefixCases {
		if !strings.HasPrefix(tc.event, tc.prefix) {
			t.Fatalf("expected %q to have prefix %q", tc.event, tc.prefix)
		}
	}
}

func TestJobCreatedPayload_Validate_AndJSONShape(t *testing.T) {
	t.Parallel()

	ok := &JobCreatedPayload{
		JobID:     "job-123",
		InputType: InputTypeZip,
		InputPath: "staging/job-123/site.zip",
		Config: models.JobConfig{
			Modules:    []string{"axe"},
			Screenshot: true,
		},
	}
	if err := ok.Validate(); err != nil {
		t.Fatalf("expected valid payload: %v", err)
	}

	b := mustMarshal(t, ok)
	m := mustUnmarshalMap(t, b)

	for _, k := range []string{"job_id", "input_type", "input_path", "config"} {
		if _, exists := m[k]; !exists {
			t.Fatalf("expected key %q in JSON: %s", k, string(b))
		}
	}
	if _, exists := m["urls"]; exists {
		t.Fatalf("expected urls to be omitted for zip input: %s", string(b))
	}

	// Invalid: missing modules
	bad := &JobCreatedPayload{
		JobID:     "job-123",
		InputType: InputTypeZip,
		InputPath: "staging/job-123/site.zip",
	}
	if err := bad.Validate(); err == nil {
		t.Fatalf("expected Validate() to fail")
	}
}

func TestExtractionReadyPayload_Validate_AndJSONOmitEmpty(t *testing.T) {
	t.Parallel()

	p := &ExtractionReadyPayload{
		JobID:          "job-123",
		ProvenancePath: "/workspace/provenance.json",
		BaseURL:        "http://localhost:8080",
		TotalPages:     5,
	}
	if err := p.Validate(); err != nil {
		t.Fatalf("expected valid payload: %v", err)
	}

	b := mustMarshal(t, p)
	m := mustUnmarshalMap(t, b)

	for _, k := range []string{"job_id", "provenance_path", "base_url", "total_pages"} {
		if _, exists := m[k]; !exists {
			t.Fatalf("expected key %q in JSON: %s", k, string(b))
		}
	}

	if _, exists := m["stage_log_path"]; exists {
		t.Fatalf("expected stage_log_path to be omitted when empty: %s", string(b))
	}
	if _, exists := m["recipe_path"]; exists {
		t.Fatalf("expected recipe_path to be omitted when empty: %s", string(b))
	}
	if _, exists := m["provenance_artifact_path"]; exists {
		t.Fatalf("expected provenance_artifact_path to be omitted when empty: %s", string(b))
	}
}

func TestScanTiming_Validate_ComponentSum(t *testing.T) {
	t.Parallel()

	timing := &ScanTiming{
		TotalMs:            5000,
		PageIterationMs:    3000,
		WriteResultsMs:     500,
		UploadArtifactsMs:  1000,
		PublishCompletedMs: 100,
		FinalizationMs:     400,
	}
	if err := timing.Validate(); err != nil {
		t.Fatalf("expected valid timing: %v", err)
	}

	bad := &ScanTiming{
		TotalMs:            10,
		PageIterationMs:    10,
		WriteResultsMs:     10,
		UploadArtifactsMs:  10,
		PublishCompletedMs: 10,
		FinalizationMs:     10,
	}
	if err := bad.Validate(); err == nil {
		t.Fatalf("expected invalid timing due to component sum > total")
	}
}

func TestScanCompletedPayload_Validate_AndJSONOmitEmpty(t *testing.T) {
	t.Parallel()

	p := &ScanCompletedPayload{
		JobID:             "job-123",
		ScannerType:       "axe",
		ResultsPath:       "job-123/axe/results.json",
		ReportPath:        "job-123/axe/report.html",
		TotalPagesScanned: 10,
		Summary: ScanSummary{
			TotalViolations: 25,
			BySeverity: map[string]int{
				"critical": 2,
				"serious":  5,
				"moderate": 10,
				"minor":    8,
			},
		},
		// Timing intentionally nil for omit-empty verification below.
	}
	if err := p.Validate(); err != nil {
		t.Fatalf("expected valid payload: %v", err)
	}

	b := mustMarshal(t, p)
	m := mustUnmarshalMap(t, b)

	for _, k := range []string{"job_id", "results_path", "report_path", "total_pages_scanned", "summary"} {
		if _, exists := m[k]; !exists {
			t.Fatalf("expected key %q in JSON: %s", k, string(b))
		}
	}
	if _, exists := m["timing"]; exists {
		t.Fatalf("expected timing to be omitted when nil: %s", string(b))
	}

	// Now include timing and ensure it shows up
	p.Timing = &ScanTiming{
		TotalMs:            100,
		PageIterationMs:    10,
		WriteResultsMs:     10,
		UploadArtifactsMs:  10,
		PublishCompletedMs: 10,
		FinalizationMs:     10,
	}
	if err := p.Validate(); err != nil {
		t.Fatalf("expected valid payload with timing: %v", err)
	}
	b2 := mustMarshal(t, p)
	m2 := mustUnmarshalMap(t, b2)
	if _, exists := m2["timing"]; !exists {
		t.Fatalf("expected timing to be present when non-nil: %s", string(b2))
	}
}

func TestScanFailedPayload_Validate(t *testing.T) {
	t.Parallel()

	ok := &ScanFailedPayload{
		JobID:       "job-123",
		ScannerType: "lighthouse",
		Error:       "Page load failed",
		PageID:      "page-5",
	}
	if err := ok.Validate(); err != nil {
		t.Fatalf("expected valid payload: %v", err)
	}

	bad := &ScanFailedPayload{JobID: "job-123"}
	if err := bad.Validate(); err == nil {
		t.Fatalf("expected invalid payload")
	}
}

func TestJobCompletedPayload_Validate(t *testing.T) {
	t.Parallel()

	ok := &JobCompletedPayload{
		JobID:  "job-123",
		Status: JobStatusSuccess,
		Artifacts: ArtifactLocations{
			ReportJSON: "/results/report.json",
			ReportHTML: "/results/report.html",
		},
		ScannerArtifacts: map[string]ScannerArtifacts{
			"axe": {
				ScannerType: "axe",
				ResultsPath: "/axe/results.json",
				ReportPath:  "/axe/report.html",
			},
		},
	}
	if err := ok.Validate(); err != nil {
		t.Fatalf("expected valid payload: %v", err)
	}

	bad := &JobCompletedPayload{
		JobID:  "job-123",
		Status: "nope",
	}
	if err := bad.Validate(); err == nil {
		t.Fatalf("expected invalid payload")
	}
}

func TestJobFailedPayload_Validate(t *testing.T) {
	t.Parallel()

	ok := &JobFailedPayload{
		JobID:  "job-123",
		Status: JobStatusFailed,
		Stage:  JobFailStageScanning,
		Error:  "Timeout exceeded",
	}
	if err := ok.Validate(); err != nil {
		t.Fatalf("expected valid payload: %v", err)
	}

	bad := &JobFailedPayload{
		JobID:  "job-123",
		Status: JobStatusFailed,
		Stage:  "unknown",
		Error:  "nope",
	}
	if err := bad.Validate(); err == nil {
		t.Fatalf("expected invalid payload")
	}
}

func TestScanPageCompletedPayload_Validate(t *testing.T) {
	t.Parallel()

	ok := &ScanPageCompletedPayload{
		JobID:      "job-123",
		PageID:     "page-3",
		PageIndex:  3,
		TotalPages: 10,
	}
	if err := ok.Validate(); err != nil {
		t.Fatalf("expected valid payload: %v", err)
	}

	bad := &ScanPageCompletedPayload{
		JobID:      "job-123",
		PageID:     "page-3",
		PageIndex:  11,
		TotalPages: 10,
	}
	if err := bad.Validate(); err == nil {
		t.Fatalf("expected invalid payload")
	}
}

func TestExtractionFailedPayload_Validate(t *testing.T) {
	t.Parallel()

	ok := &ExtractionFailedPayload{
		JobID: "job-123",
		Error: "Invalid zip file",
	}
	if err := ok.Validate(); err != nil {
		t.Fatalf("expected valid payload: %v", err)
	}

	bad := &ExtractionFailedPayload{JobID: "job-123"}
	if err := bad.Validate(); err == nil {
		t.Fatalf("expected invalid payload")
	}
}
