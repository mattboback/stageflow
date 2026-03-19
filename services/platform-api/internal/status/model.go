// Package status defines the projection store models.
package status

import (
	"time"

	"github.com/mattboback/stageflow/libs/go/models"
)

// JobRecord mirrors the persisted projection row.
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
	ReportKey             string
	ScanStageLogKey       string
	ScanRecipeKey         string
	ExtractionStageLogKey string
	ExtractionRecipeKey   string
	ProvenanceKey         string
	LastStage             string
	LastErrorDetails      string
	ExpectedScanners      []string
	CompletedScanners     []string
	ScannerArtifacts      map[string]*ScannerArtifactRecord // Per-scanner artifact keys
}

// ScannerArtifactRecord holds MinIO keys for a single scanner's artifacts.
type ScannerArtifactRecord struct {
	ScannerType string `json:"scanner_type"`
	ResultsKey  string `json:"results_key"`
	ReportKey   string `json:"report_key"`
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
		ExpectedScanners:  cloneStringSlice(r.ExpectedScanners),
		CompletedScanners: cloneStringSlice(r.CompletedScanners),
		RemainingScanners: remainingScanners(r.ExpectedScanners, r.CompletedScanners),
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

func cloneStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	cloned := make([]string, len(values))
	copy(cloned, values)

	return cloned
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
