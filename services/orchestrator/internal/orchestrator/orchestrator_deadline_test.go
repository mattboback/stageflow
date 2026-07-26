package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/mattboback/stageflow/libs/go/models"
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

// sweepJobInState inserts a job in the given state with the given age and runs one
// sweep, returning the job as stored afterwards.
func sweepJobInState(t *testing.T, state models.JobState, age time.Duration) (*models.Job, *mockPublisher) {
	t.Helper()

	db := newInMemoryDB(t)
	publisher := &mockPublisher{}
	orch := NewOrchestrator(&Config{
		PodmanClient:         &mockPodmanClient{},
		Database:             db,
		Publisher:            publisher,
		DeadlinePollInterval: 5 * time.Millisecond,
		SetupTimeout:         15 * time.Millisecond,
	})

	job := &models.Job{
		ID:        "job-" + string(state),
		State:     state,
		InputType: "zip",
		InputPath: "path.zip",
		Config:    models.JobConfig{Modules: []string{"axe"}},
		CreatedAt: time.Now().Add(-age),
		UpdatedAt: time.Now().Add(-age),
	}
	insertJob(t, db, job)

	if err := orch.runDeadlineSweep(context.Background()); err != nil {
		t.Fatalf("runDeadlineSweep failed: %v", err)
	}

	stored, err := db.GetJob(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("failed to fetch job: %v", err)
	}

	return stored, publisher
}

// A job whose setup never completed used to have no timer at all: PENDING and
// READY_TO_SCAN precede both stage deadlines, so nothing reaped them and the
// client saw "Scan in progress" indefinitely.

func TestRunDeadlineSweepFailsStuckPendingJob(t *testing.T) {
	stored, publisher := sweepJobInState(t, models.JobStatePending, time.Minute)

	if stored.State != models.JobStateFailed {
		t.Fatalf("state = %s, want FAILED: a job stuck in setup must reach a terminal state", stored.State)
	}

	if stored.CompletedAt == nil {
		t.Fatal("expected CompletedAt to be set")
	}

	if publisher.failedCount() == 0 {
		t.Fatal("expected a job.failed event so the client stops waiting")
	}
}

func TestRunDeadlineSweepFailsStuckReadyJob(t *testing.T) {
	// StartScanning returns bare on every failure path, so READY_TO_SCAN wedges the
	// same way PENDING does.
	stored, publisher := sweepJobInState(t, models.JobStateReady, time.Minute)

	if stored.State != models.JobStateFailed {
		t.Fatalf("state = %s, want FAILED", stored.State)
	}

	if publisher.failedCount() == 0 {
		t.Fatal("expected a job.failed event")
	}
}

func TestRunDeadlineSweepLeavesRecentPendingJobAlone(t *testing.T) {
	// The guard that matters: the sweep must not pre-empt a handler that is still
	// working through its delivery. Its later state write would silently affect no
	// rows against the terminal-state guard.
	stored, publisher := sweepJobInState(t, models.JobStatePending, 0)

	if stored.State != models.JobStatePending {
		t.Fatalf("state = %s, want PENDING: a job still within its setup budget must be left alone", stored.State)
	}

	if publisher.failedCount() != 0 {
		t.Fatalf("failedCount = %d, want 0", publisher.failedCount())
	}
}

func TestRunDeadlineSweepIgnoresCompletingJob(t *testing.T) {
	// Deliberately excluded. A COMPLETING job has all its scan data; failing it
	// would destroy a finished result. Overdue completion wants retrying, which is
	// what ClaimJobCompletion exists for.
	stored, publisher := sweepJobInState(t, models.JobStateCompleting, time.Hour)

	if stored.State != models.JobStateCompleting {
		t.Fatalf("state = %s, want COMPLETING left untouched", stored.State)
	}

	if publisher.failedCount() != 0 {
		t.Fatalf("failedCount = %d, want 0", publisher.failedCount())
	}
}
