package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/mattboback/stageflow/libs/go/models"
)

// StatusReader exposes only public job data; credentials and input configuration
// never escape through polling or SSE. Both use this same durable snapshot.
type StatusReader struct{ Database *Database }

func (r StatusReader) GetJob(ctx context.Context, id string) (*JobRecord, error) {
	job, err := r.Database.GetJob(ctx, id)
	if err != nil {
		return nil, err
	}

	return Snapshot(job), nil
}

// Snapshot is pure: each reader sees the same states from one jobs row.
func Snapshot(job *models.Job) *JobRecord {
	redact := snapshotRedactor(job.Config)

	r := &JobRecord{
		JobID: job.ID, State: job.State, InputType: job.InputType, CreatedAt: job.CreatedAt,
		UpdatedAt: job.UpdatedAt, CompletedAt: job.CompletedAt, Error: redact(job.Error),
		LastStage: job.LastStage, LastErrorDetails: redact(job.ErrorDetails), TotalPages: job.TotalPages,
		ReportJSONKey: job.ReportJSONKey, ScanStageLogKey: job.ScanStageLogKey,
		ScanRecipeKey: job.ScanRecipeKey, ExtractionStageLogKey: job.ExtractionStageLogKey,
		ExtractionRecipeKey: job.ExtractionRecipeKey, ProvenanceKey: job.ProvenanceKey,
		ExpectedScanners: CloneStrings(job.ExpectedScanners),
		ScannerStatuses:  make(map[string]models.ScannerStatus),
		ScannerArtifacts: make(map[string]*ScannerArtifactRecord),
	}
	for _, name := range job.ExpectedScanners {
		state := "pending"
		if job.State == models.JobStateScanning || job.State == models.JobStateCompleting {
			state = "running"
		}

		scanner := models.ScannerStatus{
			State:        state,
			PagesScanned: job.ScannerProgress[name],
			TotalPages:   job.TotalPages,
		}
		if result := job.ScannerResults[name]; result != nil {
			scanner.State = "failed"
			if result.Success {
				scanner.State = "completed"

				r.CompletedScanners = append(r.CompletedScanners, name)
			}

			if result.PagesScanned > scanner.PagesScanned {
				scanner.PagesScanned = result.PagesScanned
			}

			scanner.Error, scanner.Violations, scanner.TotalMs = redact(
				result.Error,
			), result.TotalIssues, result.TotalMs
			r.TotalViolations += result.TotalIssues
			r.ScannerArtifacts[name] = &ScannerArtifactRecord{
				ScannerType: name,
				ResultsKey:  result.ResultsPath,
				StageLogKey: result.StageLogPath,
				RecipeKey:   result.RecipePath,
			}
		}

		r.ScannerStatuses[name] = scanner
		if scanner.PagesScanned > r.CurrentPage {
			r.CurrentPage = scanner.PagesScanned
		}
	}

	if len(job.ScannerProgress) == 0 && r.CurrentPage < job.CurrentPage {
		r.CurrentPage = job.CurrentPage
	}

	if job.State.IsTerminal() {
		r.TotalViolations = job.TotalViolations
	}

	return r
}

// UpdateScannerProgress merges independent scanner progress atomically and keeps
// the legacy aggregate as the maximum, never whichever scanner reported last.
func (d *Database) UpdateScannerProgress(ctx context.Context, id, scanner string, page, total int) error {
	return d.execJobUpdate(ctx, id, `UPDATE jobs SET scanner_progress = jsonb_set(
 COALESCE(scanner_progress, '{}')::jsonb, ARRAY[?]::text[],
 to_jsonb(GREATEST(COALESCE((scanner_progress::jsonb ->> ?)::int, 0), ?)))::text,
 current_page = GREATEST(COALESCE(current_page, 0), ?),
 total_pages = GREATEST(COALESCE(total_pages, 0), ?), updated_at = ? WHERE id = ?`,
		scanner, scanner, page, page, total, time.Now(), id)
}

func snapshotRedactor(config models.JobConfig) func(string) string {
	raw, marshalErr := json.Marshal(config)
	if marshalErr != nil {
		return redactUnreadableConfig
	}

	_, secrets, sanitizeErr := sanitizeTerminalJobConfig(string(raw))
	if sanitizeErr != nil {
		return redactUnreadableConfig
	}

	return func(value string) string { return redactKnownConfigSecrets(value, secrets) }
}
func redactUnreadableConfig(value string) string {
	if value == "" {
		return ""
	}

	return redactedConfigValue
}
