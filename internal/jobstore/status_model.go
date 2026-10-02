// Package status defines job status projection models.
package db

import (
	"errors"
	"time"

	"github.com/mattboback/stageflow/libs/go/models"
)

// ErrJobNotFound is returned when a job is missing from a status source.
var ErrJobNotFound = errors.New("job not found")

// JobRecord represents a job status projection.
type JobRecord struct {
	JobID                 string
	State                 models.JobState
	InputType             string
	CreatedAt             time.Time
	UpdatedAt             time.Time
	CompletedAt           *time.Time
	Error                 string
	TotalPages            int
	CurrentPage           int
	TotalViolations       int
	ReportJSONKey         string
	ScanStageLogKey       string
	ScanRecipeKey         string
	ExtractionStageLogKey string
	ExtractionRecipeKey   string
	ProvenanceKey         string
	LastStage             string
	LastErrorDetails      string
	ExpectedScanners      []string
	CompletedScanners     []string
	ScannerStatuses       map[string]models.ScannerStatus
	ScannerArtifacts      map[string]*ScannerArtifactRecord // Per-scanner artifact keys
}

// ScannerArtifactRecord holds Job-scoped file paths for a single scanner's artifacts.
type ScannerArtifactRecord struct {
	ScannerType string `json:"scanner_type"`
	ResultsKey  string `json:"results_key"`
	StageLogKey string `json:"stage_log_key,omitempty"`
	RecipeKey   string `json:"recipe_key,omitempty"`
}

// ToModel converts the record into the API-friendly JobStatus payload.
func (r *JobRecord) ToModel() *models.JobStatus {
	job := &models.JobStatus{
		ID:                r.JobID,
		State:             r.State,
		Error:             r.Error,
		ErrorDetails:      r.LastErrorDetails,
		LastStage:         r.LastStage,
		TotalViolations:   r.TotalViolations,
		ExpectedScanners:  CloneStrings(r.ExpectedScanners),
		CompletedScanners: CloneStrings(r.CompletedScanners),
		RemainingScanners: remainingScanners(r.ExpectedScanners, r.settledScanners()),
		ScannerStatuses:   r.ScannerStatuses,
		CreatedAt:         r.CreatedAt,
		UpdatedAt:         r.UpdatedAt,
	}

	if r.TotalPages > 0 {
		progress := &models.JobProgress{
			CurrentPage: r.CurrentPage,
			TotalPages:  r.TotalPages,
		}
		if r.TotalPages > 0 && r.CurrentPage >= 0 {
			progress.Percentage = clampPercentage(r.CurrentPage, r.TotalPages)
		}

		job.Progress = progress
	}

	return job
}

func remainingScanners(expected, completed []string) []string {
	if len(expected) == 0 {
		return nil
	}

	completedSet := make(map[string]struct{}, len(completed))
	for _, scannerType := range completed {
		if scannerType == "" {
			continue
		}

		completedSet[scannerType] = struct{}{}
	}

	remaining := make([]string, 0, len(expected))
	for _, scannerType := range expected {
		if scannerType == "" {
			continue
		}

		if _, done := completedSet[scannerType]; done {
			continue
		}

		remaining = append(remaining, scannerType)
	}

	if len(remaining) == 0 {
		return nil
	}

	return remaining
}

func clampPercentage(current, total int) int {
	if total <= 0 {
		return 0
	}

	value := (current * 100) / total
	switch {
	case value < 0:
		return 0
	case value > 100:
		return 100
	default:
		return value
	}
}

func (r *JobRecord) settledScanners() []string {
	settled := CloneStrings(r.CompletedScanners)
	for name, scanner := range r.ScannerStatuses {
		if scanner.State == "failed" {
			settled = append(settled, name)
		}
	}

	return settled
}
