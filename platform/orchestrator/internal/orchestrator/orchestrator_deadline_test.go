package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/mattboback/stageflow/packages/shared-go/models"
)

func TestRunDeadlineSweepFailsStuckJob(t *testing.T) {
	db := newInMemoryDB(t)
	publisher := &mockPublisher{}
	orch := NewOrchestrator(&Config{
		PodmanClient:         &mockPodmanClient{},
		Database:             db,
		Publisher:            publisher,
		DeadlinePollInterval: 5 * time.Millisecond,
		ExtractionTimeout:    15 * time.Millisecond,
	})

	job := &models.Job{
		ID:        "stuck-job",
		State:     models.JobStateExtracting,
		InputType: "zip",
		InputPath: "path.zip",
		Config:    models.JobConfig{Modules: []string{"axe"}},
		CreatedAt: time.Now().Add(-time.Minute),
		UpdatedAt: time.Now().Add(-time.Minute),
	}
	insertJob(t, db, job)

	if err := orch.runDeadlineSweep(context.Background()); err != nil {
		t.Fatalf("runDeadlineSweep failed: %v", err)
	}

	stored, err := db.GetJob(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("failed to fetch job: %v", err)
	}

	if stored.State != models.JobStateFailed {
		t.Fatalf("expected job to fail due to timeout, got %s", stored.State)
	}

	if stored.CompletedAt == nil {
		t.Fatalf("expected CompletedAt to be set after failure")
	}

	if publisher.failedCount() == 0 {
		t.Fatalf("expected a job.failed event to be published")
	}
}
