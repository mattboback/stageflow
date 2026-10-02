package orchestrator

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	artifacts "github.com/mattboback/stageflow/internal/artifacts"
	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
	"github.com/mattboback/stageflow/libs/go/models"
	podman "github.com/mattboback/stageflow/services/orchestrator/internal/adapters/runtime"
)

const maxOutcomeBytes = 1 << 20
const maxProgressBytes = 8 << 20

type scannerOutcome struct {
	Status       string `json:"status"`
	Error        string `json:"error"`
	PagesScanned int    `json:"pages_scanned"`
	Summary      struct {
		Total    int `json:"total_violations"`
		Critical int `json:"critical_issues"`
		Serious  int `json:"serious_issues"`
		Moderate int `json:"moderate_issues"`
		Minor    int `json:"minor_issues"`
	} `json:"summary"`
	Timing struct {
		Total int64 `json:"total_ms"`
	} `json:"timing"`
}

func (out scannerOutcome) Result(name, id string) *models.ScannerResult {
	return &models.ScannerResult{
		ScannerType:    name,
		ResultsPath:    id + "/" + name + "/results.json",
		Success:        true,
		Error:          out.Error,
		PagesScanned:   out.PagesScanned,
		TotalIssues:    out.Summary.Total,
		CriticalIssues: out.Summary.Critical,
		SeriousIssues:  out.Summary.Serious,
		ModerateIssues: out.Summary.Moderate,
		MinorIssues:    out.Summary.Minor,
		TotalMs:        out.Timing.Total,
	}
}
func readBounded(root *os.Root, name string, limit int64) ([]byte, error) {
	file, err := root.OpenFile(name, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	if !info.Mode().IsRegular() || info.Size() > limit {
		return nil, fmt.Errorf("invalid or oversized file %s", name)
	}

	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, err
	}

	if int64(len(data)) > limit {
		return nil, errors.New("file exceeds limit")
	}

	return data, nil
}
func (o *Orchestrator) readOutcome(id, name string) (*scannerOutcome, error) {
	root, err := os.OpenRoot(filepath.Join(o.config.DataDir, "jobs", id))
	if err != nil {
		return nil, err
	}
	defer root.Close()

	data, err := readBounded(root, name+"/outcome.json", maxOutcomeBytes)
	if err != nil {
		return nil, err
	}

	if checkErr1 := validateOutcomeDocument(data); checkErr1 != nil {
		return nil, checkErr1
	}

	var out scannerOutcome
	if checkErr2 := json.Unmarshal(data, &out); checkErr2 != nil {
		return nil, checkErr2
	}

	if out.Status != "success" && out.Status != "partial" && out.Status != "failed" {
		return nil, errors.New("invalid outcome status")
	}

	if len(out.Error) > 4096 || out.PagesScanned < 0 || out.PagesScanned > 100 {
		return nil, errors.New("invalid outcome counts")
	}

	if out.Status != "failed" && out.PagesScanned == 0 {
		return nil, errors.New("successful scanner has no pages")
	}

	return &out, nil
}
func (o *Orchestrator) readProgress(ctx context.Context, job *models.Job, name string) error {
	root, err := os.OpenRoot(filepath.Join(o.config.DataDir, "jobs", job.ID))
	if err != nil {
		return err
	}
	defer root.Close()

	data, err := readBounded(root, name+"/progress.jsonl", maxProgressBytes)
	if err != nil {
		return err
	}
	// A crash can leave an incomplete final line. Complete lines remain useful.
	last := bytes.LastIndexByte(data, '\n')
	if last < 0 {
		return nil
	}

	lines := bufio.NewScanner(bytes.NewReader(data[:last+1]))
	lines.Buffer(make([]byte, 4096), 16<<10)

	pages, total := 0, 0

	for lines.Scan() {
		var entry struct {
			Pages int `json:"pages_scanned"`
			Total int `json:"total_pages"`
		}
		if json.Unmarshal(lines.Bytes(), &entry) != nil {
			continue
		}

		if entry.Pages < 0 || entry.Total < 1 || entry.Total > 100 || entry.Pages > entry.Total {
			continue
		}

		if entry.Pages > pages {
			pages = entry.Pages
		}

		if entry.Total > total {
			total = entry.Total
		}
	}

	if checkErr3 := lines.Err(); checkErr3 != nil {
		return checkErr3
	}

	if total == 0 {
		return nil
	}

	return o.config.Database.UpdateScannerProgress(ctx, job.ID, name, pages, total)
}
func (o *Orchestrator) cleanupJob(ctx context.Context, job *models.Job) error {
	podID := job.PodID
	if podID == "" {
		podID = podman.JobPodName(job.ID)
	}

	if checkErr4 := o.config.PodmanClient.RemovePod(ctx, podID, true); checkErr4 != nil {
		return checkErr4
	}

	root, err := os.OpenRoot(filepath.Join(o.config.DataDir, "jobs", job.ID))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	if err != nil {
		return err
	}

	defer root.Close()

	for _, name := range []string{"input.zip", "site", "auth.json", "auth", ".provenance"} {
		if checkErr5 := root.RemoveAll(name); checkErr5 != nil {
			return checkErr5
		}
	}

	for _, scanner := range job.ExpectedScanners {
		for _, file := range []string{"outcome.json", "progress.jsonl"} {
			if removeErr := root.RemoveAll(scanner + "/" + file); removeErr != nil {
				return removeErr
			}
		}
	}

	return nil
}
func (o *Orchestrator) retain(ctx context.Context, now time.Time) error {
	jobs, err := o.config.Database.TerminalJobsBefore(ctx, now.Add(-24*time.Hour))
	if err != nil {
		return err
	}

	for _, job := range jobs {
		if job.CleanupPending {
			continue
		}

		if checkErr6 := o.config.Database.DeleteJob(ctx, job.ID); checkErr6 != nil {
			return checkErr6
		}

		if checkErr7 := os.RemoveAll(filepath.Join(o.config.DataDir, "jobs", job.ID)); checkErr7 != nil {
			return checkErr7
		}
	}

	return o.retainOrphans(ctx, now)
}

func validateOutcomeDocument(data []byte) error {
	var doc map[string]json.RawMessage
	if checkErr8 := json.Unmarshal(data, &doc); checkErr8 != nil {
		return checkErr8
	}

	for _, field := range []string{"status", "error", "pages_scanned", "summary", "timing"} {
		if value, ok := doc[field]; !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return fmt.Errorf("outcome is missing %s", field)
		}
	}

	var (
		summary    map[string]json.RawMessage
		timing     map[string]json.RawMessage
		severities map[string]json.RawMessage
	)
	if json.Unmarshal(doc["summary"], &summary) != nil || json.Unmarshal(doc["timing"], &timing) != nil ||
		json.Unmarshal(summary["by_severity"], &severities) != nil {
		return errors.New("invalid outcome summary or timing")
	}

	check := func(values map[string]json.RawMessage, fields []string) error {
		for _, field := range fields {
			raw, ok := values[field]

			var value int64
			if !ok || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) || json.Unmarshal(raw, &value) != nil ||
				value < 0 {
				return fmt.Errorf("invalid outcome %s", field)
			}
		}

		return nil
	}
	if checkErr9 := check(
		summary,
		[]string{
			"total_violations",
			"pages_scanned",
			"critical_issues",
			"serious_issues",
			"moderate_issues",
			"minor_issues",
		},
	); checkErr9 != nil {
		return checkErr9
	}

	if checkErr10 := check(
		severities,
		[]string{"critical", "serious", "moderate", "minor", "info"},
	); checkErr10 != nil {
		return checkErr10
	}

	return check(
		timing,
		[]string{"total_ms", "page_iteration_ms", "write_results_ms", "publish_completed_ms", "finalization_ms"},
	)
}

// Rejected uploads and interrupted DELETEs have no row. Their directories are
// retained briefly so in-flight intake can finish, then removed independently.
func (o *Orchestrator) retainOrphans(ctx context.Context, now time.Time) error {
	root, err := os.OpenRoot(filepath.Join(o.config.DataDir, "jobs"))
	if err != nil {
		return err
	}
	defer root.Close()

	directory, err := root.Open(".")
	if err != nil {
		return err
	}
	defer directory.Close()

	entries, err := directory.ReadDir(-1)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		info, infoErr := entry.Info()
		if infoErr != nil {
			return infoErr
		}

		if now.Sub(info.ModTime()) < time.Hour {
			continue
		}

		exists, existsErr := o.config.Database.JobExists(ctx, entry.Name())
		if existsErr != nil {
			return existsErr
		}

		if exists {
			continue
		}

		if checkErr11 := root.RemoveAll(entry.Name()); checkErr11 != nil {
			return checkErr11
		}
	}

	return nil
}

func (o *Orchestrator) validateResults(id, name string, outcome *scannerOutcome) error {
	root, err := os.OpenRoot(filepath.Join(o.config.DataDir, "jobs", id))
	if err != nil {
		return err
	}
	defer root.Close()

	data, err := readBounded(root, name+"/results.json", artifacts.MaxArtifactBytes)
	if err != nil {
		return err
	}

	var doc report.UnifiedReportV2
	if decodeErr := json.Unmarshal(data, &doc); decodeErr != nil {
		return decodeErr
	}

	if doc.Version != "2.2.0" || doc.Meta.JobId != id || len(doc.Pages) != outcome.PagesScanned ||
		doc.Summary.TotalIssues != outcome.Summary.Total {
		return errors.New("results disagree with terminal outcome")
	}

	return nil
}
