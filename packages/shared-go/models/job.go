// Package models defines shared data structures for scan jobs.
package models

import "time"

// JobState represents the current state of a job.
type JobState string

// JobState constants define the stable lifecycle values stored in the DB and events.
const (
	JobStatePending    JobState = "PENDING"
	JobStateExtracting JobState = "EXTRACTING"
	JobStateReady      JobState = "READY_TO_SCAN"
	JobStateScanning   JobState = "SCANNING"
	JobStateCompleting JobState = "COMPLETING"
	JobStateDone       JobState = "DONE"
	JobStateFailed     JobState = "FAILED"
)

// Common job input types.
const (
	JobInputTypeZip  = "zip"
	JobInputTypeURLs = "urls"
)

// AllJobStates lists every known job state (useful for validation / UIs).
var AllJobStates = []JobState{
	JobStatePending,
	JobStateExtracting,
	JobStateReady,
	JobStateScanning,
	JobStateCompleting,
	JobStateDone,
	JobStateFailed,
}

func (s JobState) IsValid() bool {
	switch s {
	case JobStatePending,
		JobStateExtracting,
		JobStateReady,
		JobStateScanning,
		JobStateCompleting,
		JobStateDone,
		JobStateFailed:
		return true
	default:
		return false
	}
}

func (s JobState) IsTerminal() bool {
	return s == JobStateDone || s == JobStateFailed
}

// Job is the persisted scan job record shared across services and APIs.
type Job struct {
	ID                    string                    `json:"id"`
	State                 JobState                  `json:"state"`
	InputType             string                    `json:"input_type"`     // "zip" or "urls"
	InputPath             string                    `json:"input_path"`     // MinIO path (for zip)
	URLs                  []string                  `json:"urls,omitempty"` // URLs to scan (for urls type)
	PodID                 string                    `json:"pod_id,omitempty"`
	Config                JobConfig                 `json:"config"`
	CreatedAt             time.Time                 `json:"created_at"`
	UpdatedAt             time.Time                 `json:"updated_at"`
	CompletedAt           *time.Time                `json:"completed_at,omitempty"`
	Error                 string                    `json:"error,omitempty"`
	ErrorDetails          string                    `json:"error_details,omitempty"`
	LastStage             string                    `json:"last_stage,omitempty"`
	TotalPages            int                       `json:"total_pages,omitempty"`
	CurrentPage           int                       `json:"current_page,omitempty"`
	TotalViolations       int                       `json:"violations,omitempty"`
	ReportJSONKey         string                    `json:"report_json_key,omitempty"`
	ReportKey             string                    `json:"report_key,omitempty"`
	ScanStageLogKey       string                    `json:"scan_stage_log_key,omitempty"`
	ScanRecipeKey         string                    `json:"scan_recipe_key,omitempty"`
	ExtractionStageLogKey string                    `json:"extraction_stage_log_key,omitempty"`
	ExtractionRecipeKey   string                    `json:"extraction_recipe_key,omitempty"`
	ProvenancePath        string                    `json:"provenance_path,omitempty"`
	ProvenanceKey         string                    `json:"provenance_key,omitempty"`
	ExpectedScanners      []string                  `json:"expected_scanners,omitempty"`
	CompletedScanners     []string                  `json:"completed_scanners,omitempty"`
	ScannerResults        map[string]*ScannerResult `json:"scanner_results,omitempty"`
}

// ScannerResult captures per-scanner artifacts and metrics for a job.
type ScannerResult struct {
	ScannerType    string `json:"scanner_type"`
	ResultsPath    string `json:"results_path"`
	ReportPath     string `json:"report_path"`
	StageLogPath   string `json:"stage_log_path,omitempty"`
	RecipePath     string `json:"recipe_path,omitempty"`
	Success        bool   `json:"success"`
	Error          string `json:"error,omitempty"`
	PagesScanned   int    `json:"pages_scanned,omitempty"`
	TotalIssues    int    `json:"total_issues,omitempty"`
	CriticalIssues int    `json:"critical_issues,omitempty"`
	SeriousIssues  int    `json:"serious_issues,omitempty"`
	ModerateIssues int    `json:"moderate_issues,omitempty"`
	MinorIssues    int    `json:"minor_issues,omitempty"`
}

// JobConfig contains configuration for a scan job.
type JobConfig struct {
	Modules             []string                  `json:"modules"` // e.g., ["axe", "keyboard"]
	ScannerConfigs      map[string]map[string]any `json:"scanner_configs,omitempty"`
	Screenshot          bool                      `json:"screenshot"`
	HighlightStyle      string                    `json:"highlight_style,omitempty"`
	AllowPrivateTargets bool                      `json:"allow_private_targets,omitempty"`
}

// JobStatus is the response for job status queries.
type JobStatus struct {
	ID                string            `json:"id"`
	State             JobState          `json:"state"`
	Progress          *JobProgress      `json:"progress,omitempty"`
	Artifacts         *ArtifactLocation `json:"artifacts,omitempty"`
	Error             string            `json:"error,omitempty"`
	ErrorDetails      string            `json:"error_details,omitempty"`
	LastStage         string            `json:"last_stage,omitempty"`
	TotalViolations   int               `json:"violations,omitempty"`
	ExpectedScanners  []string          `json:"expected_scanners,omitempty"`
	CompletedScanners []string          `json:"completed_scanners,omitempty"`
	RemainingScanners []string          `json:"remaining_scanners,omitempty"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
}

// JobProgress tracks scan progress.
type JobProgress struct {
	CurrentPage int `json:"current_page"`
	TotalPages  int `json:"total_pages"`
	Percentage  int `json:"percentage"`
}

// ArtifactLocation contains URLs to download artifacts.
type ArtifactLocation struct {
	ReportJSON         string                       `json:"report_json"` // Presigned URL
	ReportHTML         string                       `json:"report_html"` // Presigned URL
	ScanStageLog       string                       `json:"scan_stage_log,omitempty"`
	ScanRecipe         string                       `json:"scan_recipe,omitempty"`
	ExtractionStageLog string                       `json:"extraction_stage_log,omitempty"`
	ExtractionRecipe   string                       `json:"extraction_recipe,omitempty"`
	ProvenanceJSON     string                       `json:"provenance_json,omitempty"` // Presigned URL
	Screenshots        []ScreenshotArtifact         `json:"screenshots,omitempty"`     // Screenshot presigned URLs
	ScannerArtifacts   map[string]*ScannerArtifacts `json:"scanner_artifacts,omitempty"`
}

// ScannerArtifacts contains URLs for a specific scanner's artifacts.
type ScannerArtifacts struct {
	ScannerType  string `json:"scanner_type"`
	ResultsJSON  string `json:"results_json"`
	ReportHTML   string `json:"report_html"`
	ScanStageLog string `json:"scan_stage_log,omitempty"`
	ScanRecipe   string `json:"scan_recipe,omitempty"`
}

type ScreenshotKind string

const (
	ScreenshotKindViolation    ScreenshotKind = "violation"
	ScreenshotKindPageOverview ScreenshotKind = "page_overview"
)

// ScreenshotArtifact contains metadata and URLs for a screenshot.
type ScreenshotArtifact struct {
	Kind            ScreenshotKind `json:"kind"`
	IssueID         string         `json:"issue_id,omitempty"`
	OccurrenceIndex *int           `json:"occurrence_index,omitempty"`
	ArtifactID      string         `json:"artifact_id"`
	ScannerID       string         `json:"scanner_id"`
	PageID          string         `json:"page_id"`
	PageURL         string         `json:"page_url,omitempty"`
	URL             string         `json:"url"`                     // Presigned URL for full-size screenshot
	ThumbnailURL    string         `json:"thumbnail_url,omitempty"` // Presigned URL for thumbnail
}
