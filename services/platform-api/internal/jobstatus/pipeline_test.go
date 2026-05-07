package jobstatus

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/models"
	"github.com/mattboback/stageflow/services/platform-api/internal/status"
)

type fakeReader struct {
	records map[string]*status.JobRecord
	err     error
}

func (f *fakeReader) GetJob(_ context.Context, jobID string) (*status.JobRecord, error) {
	if f.err != nil {
		return nil, f.err
	}

	rec, ok := f.records[jobID]
	if !ok {
		return nil, status.ErrJobNotFound
	}

	return cloneJobRecord(rec), nil
}

func TestPipelineBeginSeedsImmediateStatus(t *testing.T) {
	t.Parallel()

	pipeline := New(&Config{})
	now := time.Now().UTC()

	payload := &events.JobCreatedPayload{
		JobID:     "job-begin",
		InputType: models.JobInputTypeURLs,
		URLs:      []string{"https://example.com", "https://example.com/about"},
		Config: models.JobConfig{
			Modules: []string{"axe"},
		},
	}

	rec, err := pipeline.Begin(context.Background(), BeginJob{
		ObservedAt: now,
		Payload:    payload,
	})
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	// Begin seeds a PENDING record (state advancement is deferred to Apply).
	if rec.State != models.JobStatePending {
		t.Fatalf("State = %q, want %q", rec.State, models.JobStatePending)
	}

	if rec.TotalPages != 2 || rec.CurrentPage != 0 {
		t.Fatalf("unexpected progress: %+v", rec)
	}

	if len(rec.ExpectedScanners) != 1 || rec.ExpectedScanners[0] != "axe" {
		t.Fatalf("unexpected scanners: %+v", rec.ExpectedScanners)
	}

	// Apply with SignalJobCreated advances the state to SCANNING.
	applied, err := pipeline.Apply(context.Background(), Signal{
		Kind:       SignalJobCreated,
		ObservedAt: now,
		JobCreated: payload,
	})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if applied.State != models.JobStateScanning {
		t.Fatalf("State after Apply = %q, want %q", applied.State, models.JobStateScanning)
	}

	loaded, err := pipeline.Current(context.Background(), "job-begin")
	if err != nil {
		t.Fatalf("Current() error = %v", err)
	}

	if loaded.State != models.JobStateScanning {
		t.Fatalf("Current().State = %q, want %q", loaded.State, models.JobStateScanning)
	}
}

func TestPipelineApplyPublishesJobCreatedWatchUpdate(t *testing.T) {
	t.Parallel()

	pipeline, sub, now := newWatchedPipeline(t)
	defer sub.Close()

	// Apply with SignalJobCreated must publish to watchers (advances PENDING → SCANNING).
	applyJobCreatedToPipeline(t, pipeline, now)

	change := waitWatchUpdate(t, sub, "job.created")
	if change.Signal.Kind != SignalJobCreated {
		t.Fatalf("Signal.Kind = %q, want %q", change.Signal.Kind, SignalJobCreated)
	}

	if change.Snapshot.State != models.JobStateScanning {
		t.Fatalf("State = %q, want %q", change.Snapshot.State, models.JobStateScanning)
	}
}

func TestPipelineApplyAccumulatesScannerCompletionViolations(t *testing.T) {
	t.Parallel()

	pipeline, sub, now := newWatchedPipeline(t)
	defer sub.Close()

	applyJobCreatedToPipeline(t, pipeline, now)
	_ = waitWatchUpdate(t, sub, "job.created")

	if _, applyErr := pipeline.Apply(context.Background(), Signal{
		Kind:       SignalScanCompleted,
		ObservedAt: now.Add(time.Second),
		ScanCompleted: &events.ScanCompletedPayload{
			JobID:             "job-watch",
			ScannerType:       "axe",
			ResultsPath:       "job-watch/axe/results.json",
			ReportPath:        "job-watch/axe/report.html",
			TotalPagesScanned: 1,
			Summary: events.ScanSummary{
				TotalViolations: 3,
				BySeverity:      map[string]int{"critical": 1},
			},
		},
	}); applyErr != nil {
		t.Fatalf("Apply() error = %v", applyErr)
	}

	change := waitWatchUpdate(t, sub, "first scanner completion")
	if change.Signal.Kind != SignalScanCompleted {
		t.Fatalf("Signal.Kind = %q, want %q", change.Signal.Kind, SignalScanCompleted)
	}

	if change.Snapshot.TotalViolations != 3 {
		t.Fatalf("TotalViolations = %d, want 3", change.Snapshot.TotalViolations)
	}

	if len(change.Snapshot.CompletedScanners) != 1 || change.Snapshot.CompletedScanners[0] != "axe" {
		t.Fatalf("unexpected completed scanners: %+v", change.Snapshot.CompletedScanners)
	}

	if _, applyErr := pipeline.Apply(context.Background(), Signal{
		Kind:       SignalScanCompleted,
		ObservedAt: now.Add(2 * time.Second),
		ScanCompleted: &events.ScanCompletedPayload{
			JobID:             "job-watch",
			ScannerType:       "lighthouse",
			ResultsPath:       "job-watch/lighthouse/results.json",
			ReportPath:        "job-watch/lighthouse/report.html",
			TotalPagesScanned: 1,
			Summary: events.ScanSummary{
				TotalViolations: 2,
				BySeverity:      map[string]int{"serious": 2},
			},
		},
	}); applyErr != nil {
		t.Fatalf("Apply() error = %v", applyErr)
	}

	change = waitWatchUpdate(t, sub, "second scanner completion")
	if change.Snapshot.TotalViolations != 5 {
		t.Fatalf("TotalViolations = %d, want 5", change.Snapshot.TotalViolations)
	}

	if len(change.Snapshot.CompletedScanners) != 2 ||
		change.Snapshot.CompletedScanners[0] != "axe" ||
		change.Snapshot.CompletedScanners[1] != "lighthouse" {
		t.Fatalf("unexpected completed scanners: %+v", change.Snapshot.CompletedScanners)
	}

	if _, applyErr := pipeline.Apply(context.Background(), Signal{
		Kind:       SignalScanCompleted,
		ObservedAt: now.Add(3 * time.Second),
		ScanCompleted: &events.ScanCompletedPayload{
			JobID:             "job-watch",
			ScannerType:       "axe",
			ResultsPath:       "job-watch/axe/results.json",
			ReportPath:        "job-watch/axe/report.html",
			TotalPagesScanned: 1,
			Summary: events.ScanSummary{
				TotalViolations: 3,
			},
		},
	}); applyErr != nil {
		t.Fatalf("Apply() error = %v", applyErr)
	}

	change = waitWatchUpdate(t, sub, "duplicate scanner completion")
	if change.Snapshot.TotalViolations != 5 {
		t.Fatalf(
			"TotalViolations = %d, want duplicate scanner completion to stay 5",
			change.Snapshot.TotalViolations,
		)
	}
}

func newWatchedPipeline(t *testing.T) (*Pipeline, Subscription, time.Time) {
	t.Helper()

	pipeline := New(&Config{})
	now := time.Now().UTC()

	if _, err := pipeline.Begin(context.Background(), BeginJob{
		ObservedAt: now,
		Payload: &events.JobCreatedPayload{
			JobID:     "job-watch",
			InputType: models.JobInputTypeURLs,
			URLs:      []string{"https://example.com"},
			Config:    models.JobConfig{Modules: []string{"axe"}},
		},
	}); err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	_, sub, err := pipeline.Watch(context.Background(), "job-watch", WatchOptions{})
	if err != nil {
		t.Fatalf("Watch() error = %v", err)
	}

	return pipeline, sub, now
}

func applyJobCreatedToPipeline(t *testing.T, pipeline *Pipeline, observedAt time.Time) {
	t.Helper()

	if _, applyErr := pipeline.Apply(context.Background(), Signal{
		Kind:       SignalJobCreated,
		ObservedAt: observedAt,
		JobCreated: &events.JobCreatedPayload{
			JobID:     "job-watch",
			InputType: models.JobInputTypeURLs,
			URLs:      []string{"https://example.com"},
			Config:    models.JobConfig{Modules: []string{"axe"}},
		},
	}); applyErr != nil {
		t.Fatalf("Apply(JobCreated) error = %v", applyErr)
	}
}

func waitWatchUpdate(t *testing.T, sub Subscription, label string) Change {
	t.Helper()

	select {
	case change := <-sub.Updates():
		return change
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for watch update from %s", label)
	}

	return Change{}
}

func TestPipelineApplyKeepsFailureStickyAgainstLateSuccess(t *testing.T) {
	t.Parallel()

	pipeline := New(&Config{
		CurrentReader: &fakeReader{
			records: map[string]*status.JobRecord{
				"job-sticky": {
					JobID:     "job-sticky",
					State:     models.JobStateFailed,
					CreatedAt: time.Now().UTC().Add(-time.Minute),
					UpdatedAt: time.Now().UTC().Add(-time.Second),
					CompletedAt: func() *time.Time {
						ts := time.Now().UTC().Add(-time.Second)
						return &ts
					}(),
					Error:            "browser crashed",
					LastStage:        events.JobFailStageScanning,
					LastErrorDetails: "playwright lost connection",
				},
			},
		},
	})

	_, sub, err := pipeline.Watch(context.Background(), "job-sticky", WatchOptions{})
	if err != nil {
		t.Fatalf("Watch() error = %v", err)
	}
	defer sub.Close()

	rec, err := pipeline.Apply(context.Background(), Signal{
		Kind:       SignalJobCompleted,
		ObservedAt: time.Now().UTC(),
		JobCompleted: &events.JobCompletedPayload{
			JobID:  "job-sticky",
			Status: events.JobStatusSuccess,
			Artifacts: events.ArtifactLocations{
				ReportJSON: "job-sticky/report.json",
				ReportHTML: "job-sticky/report.html",
			},
		},
	})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if rec.State != models.JobStateFailed {
		t.Fatalf("State = %q, want %q", rec.State, models.JobStateFailed)
	}

	select {
	case change := <-sub.Updates():
		t.Fatalf("unexpected watch update: %+v", change)
	case <-time.After(250 * time.Millisecond):
	}
}

func TestPipelineCurrentFallsBackToReader(t *testing.T) {
	t.Parallel()

	pipeline := New(&Config{
		CurrentReader: &fakeReader{
			records: map[string]*status.JobRecord{
				"job-reader": {
					JobID:     "job-reader",
					State:     models.JobStateDone,
					CreatedAt: time.Now().UTC().Add(-time.Minute),
					UpdatedAt: time.Now().UTC(),
				},
			},
		},
	})

	rec, err := pipeline.Current(context.Background(), "job-reader")
	if err != nil {
		t.Fatalf("Current() error = %v", err)
	}

	if rec.State != models.JobStateDone {
		t.Fatalf("State = %q, want %q", rec.State, models.JobStateDone)
	}
}

func TestPipelineCurrentPrefersCacheOverReader(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	pipeline := New(&Config{
		CurrentReader: &fakeReader{
			records: map[string]*status.JobRecord{
				"job-cache-wins": {
					JobID:     "job-cache-wins",
					State:     models.JobStatePending, // Orchestrator still shows PENDING
					CreatedAt: now.Add(-time.Minute),
					UpdatedAt: now,
				},
			},
		},
	})

	// Seed cache via Begin + Apply (simulates NATS consumer running first).
	if _, err := pipeline.Begin(context.Background(), BeginJob{
		ObservedAt: now,
		Payload: &events.JobCreatedPayload{
			JobID:     "job-cache-wins",
			InputType: models.JobInputTypeURLs,
			URLs:      []string{"https://example.com"},
			Config:    models.JobConfig{Modules: []string{"axe"}},
		},
	}); err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	if _, err := pipeline.Apply(context.Background(), Signal{
		Kind:       SignalJobCreated,
		ObservedAt: now,
		JobCreated: &events.JobCreatedPayload{
			JobID:     "job-cache-wins",
			InputType: models.JobInputTypeURLs,
			URLs:      []string{"https://example.com"},
			Config:    models.JobConfig{Modules: []string{"axe"}},
		},
	}); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	// Current must return the cache's SCANNING state, not the reader's PENDING.
	rec, err := pipeline.Current(context.Background(), "job-cache-wins")
	if err != nil {
		t.Fatalf("Current() error = %v", err)
	}

	if rec.State != models.JobStateScanning {
		t.Fatalf("State = %q, want %q (cache should take priority over reader)", rec.State, models.JobStateScanning)
	}
}

func TestPipelineCurrentPropagatesReaderErrors(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("reader boom")
	pipeline := New(&Config{CurrentReader: &fakeReader{err: wantErr}})

	_, err := pipeline.Current(context.Background(), "job-reader")
	if !errors.Is(err, wantErr) {
		t.Fatalf("Current() error = %v, want %v", err, wantErr)
	}
}

func TestPipelineWatchDeliversInitialReaderSnapshot(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	pipeline := New(&Config{
		CurrentReader: &fakeReader{
			records: map[string]*status.JobRecord{
				"job-watch-reader": {
					JobID:      "job-watch-reader",
					State:      models.JobStateReady,
					TotalPages: 4,
					CreatedAt:  now.Add(-time.Minute),
					UpdatedAt:  now,
				},
			},
		},
	})

	rec, sub, err := pipeline.Watch(context.Background(), "job-watch-reader", WatchOptions{})
	if err != nil {
		t.Fatalf("Watch() error = %v", err)
	}
	defer sub.Close()

	if rec.State != models.JobStateReady {
		t.Fatalf("initial snapshot state = %q, want %q", rec.State, models.JobStateReady)
	}

	select {
	case <-sub.Updates():
		t.Fatal("did not expect immediate update event for initial snapshot")
	case <-time.After(150 * time.Millisecond):
	}
}

func TestPipelineWatchClosesWhenContextCanceled(t *testing.T) {
	t.Parallel()

	pipeline := New(&Config{})
	if _, err := pipeline.Begin(context.Background(), BeginJob{
		ObservedAt: time.Now().UTC(),
		Payload: &events.JobCreatedPayload{
			JobID:     "job-watch-cancel",
			InputType: models.JobInputTypeURLs,
			URLs:      []string{"https://example.com"},
		},
	}); err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	_, sub, err := pipeline.Watch(ctx, "job-watch-cancel", WatchOptions{})
	if err != nil {
		t.Fatalf("Watch() error = %v", err)
	}

	cancel()

	select {
	case _, ok := <-sub.Updates():
		if ok {
			t.Fatal("expected closed updates channel after cancel")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for watcher close")
	}
}
