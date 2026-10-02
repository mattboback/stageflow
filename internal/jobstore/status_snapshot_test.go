package db

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/mattboback/stageflow/libs/go/models"
)

func TestDurableSnapshotKeepsFailureAcrossReaders(t *testing.T) {
	database := setupTestDB(t)
	job := &models.Job{
		ID:        "snapshot",
		State:     models.JobStateScanning,
		InputType: models.JobInputTypeURLs,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mustCreateJob(t, database, job)

	ctx := context.Background()
	if err := database.SetExpectedScanners(ctx, job.ID, []string{"seo", "axe", "lighthouse"}); err != nil {
		t.Fatal(err)
	}

	if err := database.UpdateScannerProgress(ctx, job.ID, "lighthouse", 28, 30); err != nil {
		t.Fatal(err)
	}

	if err := database.UpdateScannerProgress(ctx, job.ID, "axe", 2, 30); err != nil {
		t.Fatal(err)
	}

	if _, err := database.RecordScannerCompletion(
		ctx,
		job.ID,
		&models.ScannerResult{ScannerType: "axe", Success: true, PagesScanned: 30, TotalIssues: 9, TotalMs: 50},
	); err != nil {
		t.Fatal(err)
	}

	if _, err := database.RecordScannerFailure(ctx, job.ID, "seo", "browser closed"); err != nil {
		t.Fatal(err)
	}

	for range 2 {
		rec, err := (StatusReader{Database: database}).GetJob(ctx, job.ID)
		if err != nil {
			t.Fatal(err)
		}

		wire := rec.ToModel()
		if !reflect.DeepEqual(wire.CompletedScanners, []string{"axe"}) ||
			!reflect.DeepEqual(wire.RemainingScanners, []string{"lighthouse"}) {
			t.Fatalf("misleading settled scanners: %#v", wire)
		}

		if rec.ScannerStatuses["seo"].State != "failed" || rec.ScannerStatuses["lighthouse"].PagesScanned != 28 ||
			rec.CurrentPage != 30 ||
			rec.TotalViolations != 9 {
			t.Fatalf("snapshot: %#v", rec)
		}
	}
}
func TestSnapshotRedactsLiveScannerFailure(t *testing.T) {
	job := &models.Job{
		State:            models.JobStateScanning,
		ExpectedScanners: []string{"seo"},
		Config: models.JobConfig{
			Auth: json.RawMessage(`{"type":"form","steps":[{"type":"fill","value":"secret password"}]}`),
		},
		ScannerResults: map[string]*models.ScannerResult{"seo": {Error: "failed at ?password=secret+password"}},
	}

	data, err := json.Marshal(Snapshot(job).ToModel())
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(data), "secret") {
		t.Fatalf("secret leaked: %s", data)
	}
}
