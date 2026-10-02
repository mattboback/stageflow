// Package models defines shared data structures for scan jobs.
package models

import (
	"encoding/json"
	"time"
)

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
	ClaimedAt           *time.Time                `json:"-"`
	ExtractionStartedAt *time.Time                `json:"-"`
	ScanStartedAt       *time.Time                `json:"-"`
	CleanupPending      bool                      `json:"-"`
	ScannerAttempts     map[string]int            `json:"-"`
	ScannerStates       map[string]string         `json:"-"`
	ScannerContainerIDs map[string]string         `json:"-"`
	ID                  string                    `json:"id"`
	State               JobState                  `json:"state"`
	InputType           string                    `json:"input_type"`     // "zip" or "urls"
	InputPath           string                    `json:"input_path"`     // Job-scoped file path (for zip)
	URLs                []string                  `json:"urls,omitempty"` // URLs to scan (for urls type)
	PodID               string                    `json:"pod_id,omitempty"`
	Config              JobConfig                 `json:"config"`
	CreatedAt           time.Time                 `json:"created_at"`
	UpdatedAt           time.Time                 `json:"updated_at"`
	CompletedAt         *time.Time                `json:"completed_at,omitempty"`
	Error               string                    `json:"error,omitempty"`
	ErrorDetails        string                    `json:"error_details,omitempty"`
	LastStage           string                    `json:"last_stage,omitempty"`
	TotalPages          int                       `json:"total_pages,omitempty"`
	CurrentPage         int                       `json:"current_page,omitempty"`
	TotalViolations     int                       `json:"violations,omitempty"`
	ReportJSONKey       string                    `json:"report_json_key,omitempty"`
	ProvenanceKey       string                    `json:"provenance_key,omitempty"`
	ExpectedScanners    []string                  `json:"expected_scanners,omitempty"`
	CompletedScanners   []string                  `json:"completed_scanners,omitempty"`
	ScannerProgress     map[string]int            `json:"scanner_progress,omitempty"`
	ScannerResults      map[string]*ScannerResult `json:"scanner_results,omitempty"`
}

// ScannerResult captures per-scanner artifacts and metrics for a job.
type ScannerResult struct {
	ScannerType    string `json:"scanner_type"`
	ResultsPath    string `json:"results_path"`
	Success        bool   `json:"success"`
	Error          string `json:"error,omitempty"`
	PagesScanned   int    `json:"pages_scanned,omitempty"`
	TotalIssues    int    `json:"total_issues,omitempty"`
	CriticalIssues int    `json:"critical_issues,omitempty"`
	SeriousIssues  int    `json:"serious_issues,omitempty"`
	ModerateIssues int    `json:"moderate_issues,omitempty"`
	MinorIssues    int    `json:"minor_issues,omitempty"`
	TotalMs        int64  `json:"total_ms,omitempty"`
}

// JobConfig contains configuration for a scan job.
// VerificationConfig identifies an original finding without carrying credentials.
type VerificationConfig struct {
	SourceJobID string `json:"source_job_id"`
	PageID      string `json:"page_id"`
	RuleID      string `json:"rule_id"`
	Scanner     string `json:"scanner"`
	SubjectID   string `json:"subject_id,omitempty"`
}

type JobConfig struct {
	Profile             string                    `json:"profile,omitempty"`
	IdentityScope       string                    `json:"identity_scope,omitempty"`
	Verification        *VerificationConfig       `json:"verification,omitempty"`
	PagePaths           []string                  `json:"page_paths,omitempty"`
	Modules             []string                  `json:"modules"` // e.g., ["axe", "keyboard"]
	ScannerConfigs      map[string]map[string]any `json:"scanner_configs,omitempty"`
	Screenshot          bool                      `json:"screenshot"`
	AllowPrivateTargets bool                      `json:"allow_private_targets,omitempty"`
	// Auth carries the optional Provenance.auth block; it is omitted for
	// unauthenticated scans. Form recipes carry literal values only: the
	// Platform API and the orchestrator both reject {from_env: NAME}
	// references. For storage_state the Platform API writes the client's state
	// to a private job file before admitting the job, so only the mode is
	// stored here, and the orchestrator refuses to launch a scanner while
	// inline content is still present.
	Auth json.RawMessage `json:"auth,omitempty"`
}

// JobStatus is the response for job status queries.
type JobStatus struct {
	ID                string                   `json:"id"`
	State             JobState                 `json:"state"`
	Progress          *JobProgress             `json:"progress,omitempty"`
	Artifacts         *ArtifactLocation        `json:"artifacts,omitempty"`
	Error             string                   `json:"error,omitempty"`
	ErrorDetails      string                   `json:"error_details,omitempty"`
	LastStage         string                   `json:"last_stage,omitempty"`
	TotalViolations   int                      `json:"violations,omitempty"`
	ExpectedScanners  []string                 `json:"expected_scanners,omitempty"`
	CompletedScanners []string                 `json:"completed_scanners,omitempty"`
	RemainingScanners []string                 `json:"remaining_scanners,omitempty"`
	ScannerStatuses   map[string]ScannerStatus `json:"scanner_statuses,omitempty"`
	CreatedAt         time.Time                `json:"created_at"`
	UpdatedAt         time.Time                `json:"updated_at"`
}

// JobProgress tracks scan progress.
type JobProgress struct {
	CurrentPage int `json:"current_page"`
	TotalPages  int `json:"total_pages"`
	Percentage  int `json:"percentage"`
}

// ArtifactLocation contains the API file routes for a job's public artifacts.
type ArtifactLocation struct {
	ReportJSON       string                       `json:"report_json"`
	ProvenanceJSON   string                       `json:"provenance_json,omitempty"`
	Screenshots      []ScreenshotArtifact         `json:"screenshots,omitempty"`
	ScannerArtifacts map[string]*ScannerArtifacts `json:"scanner_artifacts,omitempty"`
}

// ScannerArtifacts contains URLs for a specific scanner's artifacts.
type ScannerArtifacts struct {
	ScannerType string `json:"scanner_type"`
	ResultsJSON string `json:"results_json"`
}

type ScreenshotKind string

const ScreenshotKindPageOverview ScreenshotKind = "page_overview"

// ScreenshotArtifact contains metadata and URLs for a screenshot.
type ScreenshotArtifact struct {
	Kind       ScreenshotKind `json:"kind"`
	ArtifactID string         `json:"artifact_id"`
	ScannerID  string         `json:"scanner_id"`
	PageID     string         `json:"page_id"`
	PageURL    string         `json:"page_url,omitempty"`
	URL        string         `json:"url"` // API file route for the full-size screenshot
}

// ScannerStatus is a durable per-scanner snapshot, including unsuccessful outcomes.
type ScannerStatus struct {
	State        string `json:"state"`
	PagesScanned int    `json:"pages_scanned"`
	TotalPages   int    `json:"total_pages"`
	Violations   int    `json:"violations"`
	TotalMs      int64  `json:"total_ms,omitempty"`
	Error        string `json:"error,omitempty"`
}
