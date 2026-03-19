package events

import (
	"errors"
	"fmt"

	"github.com/mattboback/stageflow/libs/go/models"
)

const (
	EventJobCreated   = "job.created"
	EventJobCompleted = "job.completed"
	EventJobFailed    = "job.failed"

	EventExtractionReady  = "extraction.ready"
	EventExtractionFailed = "extraction.failed"

	EventScanPageCompleted = "scan.page.completed"
	EventScanCompleted     = "scan.completed"
	EventScanFailed        = "scan.failed"
)

// Common string constants used inside payloads.
const (
	InputTypeZip  = "zip"
	InputTypeURLs = "urls"

	JobStatusSuccess = "success"
	JobStatusFailed  = "failed"

	JobFailStageExtraction = "extraction"
	JobFailStageScanning   = "scanning"
	JobFailStageReporting  = "reporting"
)

type JobCreatedPayload struct {
	JobID     string           `json:"job_id"`
	InputType string           `json:"input_type"` // "zip" or "urls"
	InputPath string           `json:"input_path"` // MinIO path for zip
	URLs      []string         `json:"urls,omitempty"`
	Config    models.JobConfig `json:"config"`
}

func (p *JobCreatedPayload) Validate() error {
	if p == nil {
		return errors.New("events: JobCreatedPayload is nil")
	}

	if p.JobID == "" {
		return errors.New("events: JobCreatedPayload.job_id is required")
	}

	switch p.InputType {
	case InputTypeZip:
		if p.InputPath == "" {
			return fmt.Errorf("events: JobCreatedPayload.input_path is required when input_type=%q", InputTypeZip)
		}
	case InputTypeURLs:
		if len(p.URLs) == 0 {
			return fmt.Errorf("events: JobCreatedPayload.urls is required when input_type=%q", InputTypeURLs)
		}
	default:
		return fmt.Errorf(
			"events: JobCreatedPayload.input_type must be %q or %q (got %q)",
			InputTypeZip,
			InputTypeURLs,
			p.InputType,
		)
	}

	if len(p.Config.Modules) == 0 {
		return errors.New("events: JobCreatedPayload.config.modules must not be empty")
	}

	return nil
}

type ExtractionReadyPayload struct {
	JobID                  string `json:"job_id"`
	ProvenancePath         string `json:"provenance_path"` // /workspace/provenance.json
	BaseURL                string `json:"base_url"`        // http://localhost:8080
	TotalPages             int    `json:"total_pages"`
	StageLogPath           string `json:"stage_log_path,omitempty"`
	RecipePath             string `json:"recipe_path,omitempty"`
	ProvenanceArtifactPath string `json:"provenance_artifact_path,omitempty"` // MinIO path for provenance.json
}

func (p *ExtractionReadyPayload) Validate() error {
	if p == nil {
		return errors.New("events: ExtractionReadyPayload is nil")
	}

	if p.JobID == "" {
		return errors.New("events: ExtractionReadyPayload.job_id is required")
	}

	if p.ProvenancePath == "" {
		return errors.New("events: ExtractionReadyPayload.provenance_path is required")
	}

	if p.BaseURL == "" {
		return errors.New("events: ExtractionReadyPayload.base_url is required")
	}

	if p.TotalPages <= 0 {
		return fmt.Errorf("events: ExtractionReadyPayload.total_pages must be > 0 (got %d)", p.TotalPages)
	}

	return nil
}

type ExtractionFailedPayload struct {
	JobID        string `json:"job_id"`
	Error        string `json:"error"`
	ErrorDetails string `json:"error_details,omitempty"`
	StageLogPath string `json:"stage_log_path,omitempty"`
	RecipePath   string `json:"recipe_path,omitempty"`
}

func (p *ExtractionFailedPayload) Validate() error {
	if p == nil {
		return errors.New("events: ExtractionFailedPayload is nil")
	}

	if p.JobID == "" {
		return errors.New("events: ExtractionFailedPayload.job_id is required")
	}

	if p.Error == "" {
		return errors.New("events: ExtractionFailedPayload.error is required")
	}

	return nil
}

type ScanPageCompletedPayload struct {
	JobID       string `json:"job_id"`
	ScannerType string `json:"scanner_type,omitempty"` // e.g., "axe", "lighthouse"
	PageID      string `json:"page_id"`
	PageIndex   int    `json:"page_index"` // 1-based
	TotalPages  int    `json:"total_pages"`
}

func (p *ScanPageCompletedPayload) Validate() error {
	if p == nil {
		return errors.New("events: ScanPageCompletedPayload is nil")
	}

	if p.JobID == "" {
		return errors.New("events: ScanPageCompletedPayload.job_id is required")
	}

	if p.PageID == "" {
		return errors.New("events: ScanPageCompletedPayload.page_id is required")
	}

	if p.PageIndex < 1 {
		return fmt.Errorf("events: ScanPageCompletedPayload.page_index must be >= 1 (got %d)", p.PageIndex)
	}

	if p.TotalPages < 1 {
		return fmt.Errorf("events: ScanPageCompletedPayload.total_pages must be >= 1 (got %d)", p.TotalPages)
	}

	if p.PageIndex > p.TotalPages {
		return fmt.Errorf(
			"events: ScanPageCompletedPayload.page_index (%d) cannot exceed total_pages (%d)",
			p.PageIndex,
			p.TotalPages,
		)
	}

	return nil
}

type ScanCompletedPayload struct {
	JobID             string      `json:"job_id"`
	ScannerType       string      `json:"scanner_type,omitempty"` // e.g., "axe", "lighthouse"
	ResultsPath       string      `json:"results_path"`           // MinIO object key for JSON results
	ReportPath        string      `json:"report_path"`            // MinIO object key for HTML report
	StageLogPath      string      `json:"stage_log_path,omitempty"`
	RecipePath        string      `json:"recipe_path,omitempty"`
	TotalPagesScanned int         `json:"total_pages_scanned"`
	Summary           ScanSummary `json:"summary"`
	Timing            *ScanTiming `json:"timing,omitempty"`
}

func (p *ScanCompletedPayload) Validate() error {
	if p == nil {
		return errors.New("events: ScanCompletedPayload is nil")
	}

	if p.JobID == "" {
		return errors.New("events: ScanCompletedPayload.job_id is required")
	}

	if p.ResultsPath == "" {
		return errors.New("events: ScanCompletedPayload.results_path is required")
	}

	if p.ReportPath == "" {
		return errors.New("events: ScanCompletedPayload.report_path is required")
	}

	if p.TotalPagesScanned < 0 {
		return fmt.Errorf("events: ScanCompletedPayload.total_pages_scanned must be >= 0 (got %d)", p.TotalPagesScanned)
	}

	if err := p.Summary.Validate(); err != nil {
		return fmt.Errorf("events: ScanCompletedPayload.summary invalid: %w", err)
	}

	if p.Timing != nil {
		if err := p.Timing.Validate(); err != nil {
			return fmt.Errorf("events: ScanCompletedPayload.timing invalid: %w", err)
		}
	}

	return nil
}

type ScanSummary struct {
	TotalViolations int            `json:"total_violations"`
	BySeverity      map[string]int `json:"by_severity"`
	PagesScanned    int            `json:"pages_scanned,omitempty"`
	CriticalIssues  int            `json:"critical_issues,omitempty"`
	SeriousIssues   int            `json:"serious_issues,omitempty"`
	ModerateIssues  int            `json:"moderate_issues,omitempty"`
	MinorIssues     int            `json:"minor_issues,omitempty"`
}

func (s *ScanSummary) Validate() error {
	if s == nil {
		return errors.New("events: ScanSummary is nil")
	}

	if s.TotalViolations < 0 {
		return fmt.Errorf("events: ScanSummary.total_violations must be >= 0 (got %d)", s.TotalViolations)
	}

	// Not omitempty in JSON contract; require it to be present (non-nil).
	if s.BySeverity == nil {
		return errors.New("events: ScanSummary.by_severity must not be nil")
	}

	for k, v := range s.BySeverity {
		if k == "" {
			return errors.New("events: ScanSummary.by_severity has empty key")
		}

		if v < 0 {
			return fmt.Errorf("events: ScanSummary.by_severity[%q] must be >= 0 (got %d)", k, v)
		}
	}

	return nil
}

type ScanTiming struct {
	TotalMs            int64 `json:"total_ms"`
	PageIterationMs    int64 `json:"page_iteration_ms"`
	WriteResultsMs     int64 `json:"write_results_ms"`
	UploadArtifactsMs  int64 `json:"upload_artifacts_ms"`
	PublishCompletedMs int64 `json:"publish_completed_ms"`
	FinalizationMs     int64 `json:"finalization_ms"`
}

func (t *ScanTiming) Validate() error {
	if t == nil {
		return errors.New("events: ScanTiming is nil")
	}

	fields := []struct {
		name string
		val  int64
	}{
		{"total_ms", t.TotalMs},
		{"page_iteration_ms", t.PageIterationMs},
		{"write_results_ms", t.WriteResultsMs},
		{"upload_artifacts_ms", t.UploadArtifactsMs},
		{"publish_completed_ms", t.PublishCompletedMs},
		{"finalization_ms", t.FinalizationMs},
	}

	for _, f := range fields {
		if f.val < 0 {
			return fmt.Errorf("events: ScanTiming.%s must be >= 0 (got %d)", f.name, f.val)
		}
	}

	componentSum := t.PageIterationMs + t.WriteResultsMs + t.UploadArtifactsMs + t.PublishCompletedMs + t.FinalizationMs
	if t.TotalMs > 0 && componentSum > t.TotalMs {
		return fmt.Errorf("events: ScanTiming component sum (%d) exceeds total_ms (%d)", componentSum, t.TotalMs)
	}

	return nil
}

type ScanFailedPayload struct {
	JobID        string `json:"job_id"`
	ScannerType  string `json:"scanner_type,omitempty"` // e.g., "axe", "lighthouse"
	Error        string `json:"error"`
	ErrorDetails string `json:"error_details,omitempty"`
	StageLogPath string `json:"stage_log_path,omitempty"`
	RecipePath   string `json:"recipe_path,omitempty"`
	PageID       string `json:"page_id,omitempty"` // If failed on specific page
}

func (p *ScanFailedPayload) Validate() error {
	if p == nil {
		return errors.New("events: ScanFailedPayload is nil")
	}

	if p.JobID == "" {
		return errors.New("events: ScanFailedPayload.job_id is required")
	}

	if p.Error == "" {
		return errors.New("events: ScanFailedPayload.error is required")
	}

	return nil
}

type JobCompletedPayload struct {
	JobID            string                      `json:"job_id"`
	Status           string                      `json:"status"` // "success"
	Artifacts        ArtifactLocations           `json:"artifacts"`
	ScannerArtifacts map[string]ScannerArtifacts `json:"scanner_artifacts,omitempty"` // keyed by scanner type
}

func (p *JobCompletedPayload) Validate() error {
	if p == nil {
		return errors.New("events: JobCompletedPayload is nil")
	}

	if p.JobID == "" {
		return errors.New("events: JobCompletedPayload.job_id is required")
	}

	if p.Status != JobStatusSuccess {
		return fmt.Errorf("events: JobCompletedPayload.status must be %q (got %q)", JobStatusSuccess, p.Status)
	}

	if err := p.Artifacts.Validate(); err != nil {
		return fmt.Errorf("events: JobCompletedPayload.artifacts invalid: %w", err)
	}

	for k, v := range p.ScannerArtifacts {
		if k == "" {
			return errors.New("events: JobCompletedPayload.scanner_artifacts has empty key")
		}

		if err := v.Validate(); err != nil {
			return fmt.Errorf("events: JobCompletedPayload.scanner_artifacts[%q] invalid: %w", k, err)
		}
	}

	return nil
}

type ArtifactLocations struct {
	ReportJSON     string `json:"report_json"` // MinIO path or presigned URL
	ReportHTML     string `json:"report_html"` // MinIO path or presigned URL
	ScanStageLog   string `json:"scan_stage_log,omitempty"`
	ScanRecipe     string `json:"scan_recipe,omitempty"`
	ProvenanceJSON string `json:"provenance_json,omitempty"`
}

func (a *ArtifactLocations) Validate() error {
	if a == nil {
		return errors.New("events: ArtifactLocations is nil")
	}

	if a.ReportJSON == "" {
		return errors.New("events: ArtifactLocations.report_json is required")
	}

	if a.ReportHTML == "" {
		return errors.New("events: ArtifactLocations.report_html is required")
	}

	return nil
}

type ScannerArtifacts struct {
	ScannerType  string `json:"scanner_type"`
	ResultsPath  string `json:"results_path"`
	ReportPath   string `json:"report_path"`
	StageLogPath string `json:"stage_log_path,omitempty"`
	RecipePath   string `json:"recipe_path,omitempty"`
}

func (s *ScannerArtifacts) Validate() error {
	if s == nil {
		return errors.New("events: ScannerArtifacts is nil")
	}

	if s.ScannerType == "" {
		return errors.New("events: ScannerArtifacts.scanner_type is required")
	}

	if s.ResultsPath == "" {
		return errors.New("events: ScannerArtifacts.results_path is required")
	}

	if s.ReportPath == "" {
		return errors.New("events: ScannerArtifacts.report_path is required")
	}

	return nil
}

type JobFailedPayload struct {
	JobID        string `json:"job_id"`
	Status       string `json:"status"` // "failed"
	Stage        string `json:"stage"`  // "extraction", "scanning", "reporting"
	Error        string `json:"error"`
	ErrorDetails string `json:"error_details,omitempty"`
}

func (p *JobFailedPayload) Validate() error {
	if p == nil {
		return errors.New("events: JobFailedPayload is nil")
	}

	if p.JobID == "" {
		return errors.New("events: JobFailedPayload.job_id is required")
	}

	if p.Status != JobStatusFailed {
		return fmt.Errorf("events: JobFailedPayload.status must be %q (got %q)", JobStatusFailed, p.Status)
	}

	if p.Stage == "" {
		return errors.New("events: JobFailedPayload.stage is required")
	}

	switch p.Stage {
	case JobFailStageExtraction, JobFailStageScanning, JobFailStageReporting:
		// ok
	default:
		return fmt.Errorf("events: JobFailedPayload.stage must be one of %q|%q|%q (got %q)",
			JobFailStageExtraction, JobFailStageScanning, JobFailStageReporting, p.Stage)
	}

	if p.Error == "" {
		return errors.New("events: JobFailedPayload.error is required")
	}

	return nil
}
