package integration

import (
	"context"
	"errors"
	"net"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"

	"github.com/mattboback/stageflow/libs/go/events"
	sharedmsg "github.com/mattboback/stageflow/libs/go/messaging"
	"github.com/mattboback/stageflow/libs/go/models"
	platformmsg "github.com/mattboback/stageflow/services/platform-api/internal/messaging"
	"github.com/mattboback/stageflow/services/platform-api/internal/status"
	"github.com/mattboback/stageflow/services/platform-api/internal/jobstatus"
)

type integrationFixture struct {
	ctx    context.Context
	client *sharedmsg.Client
	pipe   jobstatus.JobStatusPipeline
}

func TestServiceIntegrationWithNATS(t *testing.T) {
	fixture := newIntegrationFixture(t)

	jobID := "nats-integration-job"

	createdEnv := events.NewEnvelope(events.EventJobCreated, jobID, "integration-test", buildJobCreated(jobID))
	svc := platformmsg.NewService(fixture.client)

	if publishErr := svc.PublishJobCreated(fixture.ctx, createdEnv); publishErr != nil {
		t.Fatalf("failed to publish job.created: %v", publishErr)
	}

	assertExtractionLifecycle(fixture.ctx, t, fixture.client, fixture.pipe, jobID)
	assertScanLifecycle(fixture.ctx, t, fixture.client, fixture.pipe, jobID)
	publishEnvelope(
		fixture.ctx,
		t,
		fixture.client,
		sharedmsg.SubjectJobCompleted,
		events.NewEnvelope(events.EventJobCompleted, jobID, "integration-test", &events.JobCompletedPayload{
			JobID:  jobID,
			Status: "success",
			Artifacts: events.ArtifactLocations{
				ReportJSON: jobID + "/report.json",
				ReportHTML: jobID + "/report.html",
			},
		}),
	)

	final := waitForRecord(t, fixture.pipe, jobID, func(rec *status.JobRecord) bool {
		return rec.State == "DONE" && rec.ReportJSONKey != "" && rec.ReportKey != ""
	})

	if final.ReportJSONKey == "" || final.ReportKey == "" {
		t.Fatalf("expected artifacts to be set, got %+v", final)
	}
}

func TestServiceIntegrationWithNATSURLJobs(t *testing.T) {
	fixture := newIntegrationFixture(t)

	jobID := "nats-url-job"

	publishEnvelope(
		fixture.ctx,
		t,
		fixture.client,
		sharedmsg.SubjectJobCreated,
		events.NewEnvelope(events.EventJobCreated, jobID, "integration-test", buildURLJobCreated(jobID)),
	)

	waitForRecord(t, fixture.pipe, jobID, func(rec *status.JobRecord) bool {
		return rec.State == models.JobStateScanning &&
			rec.InputType == events.InputTypeURLs &&
			rec.TotalPages == 2 &&
			rec.CurrentPage == 0 &&
			len(rec.ExpectedScanners) == 1 &&
			rec.ExpectedScanners[0] == "axe"
	})

	publishEnvelope(
		fixture.ctx,
		t,
		fixture.client,
		sharedmsg.SubjectScanPageCompleted,
		events.NewEnvelope(events.EventScanPageCompleted, jobID, "integration-test", &events.ScanPageCompletedPayload{
			JobID:       jobID,
			ScannerType: "axe",
			PageID:      "page-1",
			PageIndex:   1,
			TotalPages:  2,
		}),
	)

	waitForRecord(t, fixture.pipe, jobID, func(rec *status.JobRecord) bool {
		return rec.State == models.JobStateScanning && rec.CurrentPage >= 1 && rec.TotalPages == 2
	})

	publishEnvelope(
		fixture.ctx,
		t,
		fixture.client,
		sharedmsg.SubjectScanCompleted,
		events.NewEnvelope(events.EventScanCompleted, jobID, "integration-test", &events.ScanCompletedPayload{
			JobID:             jobID,
			ScannerType:       "axe",
			ResultsPath:       jobID + "/axe/results.json",
			ReportPath:        jobID + "/axe/report.html",
			TotalPagesScanned: 2,
			Summary: events.ScanSummary{
				TotalViolations: 0,
				BySeverity:      map[string]int{"critical": 0},
			},
		}),
	)

	publishEnvelope(
		fixture.ctx,
		t,
		fixture.client,
		sharedmsg.SubjectJobCompleted,
		events.NewEnvelope(events.EventJobCompleted, jobID, "integration-test", &events.JobCompletedPayload{
			JobID:  jobID,
			Status: "success",
			Artifacts: events.ArtifactLocations{
				ReportJSON: jobID + "/report.json",
				ReportHTML: jobID + "/report.html",
			},
			ScannerArtifacts: map[string]events.ScannerArtifacts{
				"axe": {
					ScannerType: "axe",
					ResultsPath: jobID + "/axe/results.json",
					ReportPath:  jobID + "/axe/report.html",
				},
			},
		}),
	)

	final := waitForRecord(t, fixture.pipe, jobID, func(rec *status.JobRecord) bool {
		return rec.State == models.JobStateDone &&
			rec.ReportJSONKey != "" &&
			len(rec.CompletedScanners) == 1 &&
			rec.CompletedScanners[0] == "axe"
	})

	artifact := final.ScannerArtifacts["axe"]
	if artifact == nil {
		t.Fatalf("expected axe scanner artifacts, got %#v", final.ScannerArtifacts)
	}

	if artifact.ResultsKey != jobID+"/axe/results.json" || artifact.ReportKey != jobID+"/axe/report.html" {
		t.Fatalf("unexpected axe scanner artifact keys: %#v", artifact)
	}
}

func TestServiceIntegrationFailureRemainsStickyAfterLateSuccess(t *testing.T) {
	fixture := newIntegrationFixture(t)

	jobID := "nats-failed-job"

	publishEnvelope(
		fixture.ctx,
		t,
		fixture.client,
		sharedmsg.SubjectJobCreated,
		events.NewEnvelope(events.EventJobCreated, jobID, "integration-test", buildURLJobCreated(jobID)),
	)

	waitForRecord(t, fixture.pipe, jobID, func(rec *status.JobRecord) bool {
		return rec.State == models.JobStateScanning
	})

	publishEnvelope(
		fixture.ctx,
		t,
		fixture.client,
		sharedmsg.SubjectScanFailed,
		events.NewEnvelope(events.EventScanFailed, jobID, "integration-test", &events.ScanFailedPayload{
			JobID:        jobID,
			ScannerType:  "axe",
			Error:        "browser crashed",
			ErrorDetails: "playwright lost connection",
			StageLogPath: jobID + "/axe/stage.log",
			RecipePath:   jobID + "/axe/recipe.json",
		}),
	)

	failed := waitForRecord(t, fixture.pipe, jobID, func(rec *status.JobRecord) bool {
		return rec.State == models.JobStateFailed &&
			rec.Error == "browser crashed" &&
			rec.LastStage == "scanning"
	})

	if failed.CompletedAt == nil {
		t.Fatalf("expected completed_at to be set for failed job")
	}

	publishEnvelope(
		fixture.ctx,
		t,
		fixture.client,
		sharedmsg.SubjectJobCompleted,
		events.NewEnvelope(events.EventJobCompleted, jobID, "integration-test", &events.JobCompletedPayload{
			JobID:  jobID,
			Status: "success",
			Artifacts: events.ArtifactLocations{
				ReportJSON: jobID + "/late/report.json",
				ReportHTML: jobID + "/late/report.html",
			},
		}),
	)

	time.Sleep(250 * time.Millisecond)

	final, err := fixture.pipe.Current(context.Background(), jobID)
	if err != nil {
		t.Fatalf("failed to reload failed record: %v", err)
	}

	if final.State != models.JobStateFailed {
		t.Fatalf("late success should not override failure state, got %+v", final)
	}

	if final.ReportJSONKey != "" || final.ReportKey != "" {
		t.Fatalf("late success should not backfill success artifacts onto failed job, got %+v", final)
	}

	if final.Error != "browser crashed" || final.LastErrorDetails != "playwright lost connection" {
		t.Fatalf("expected failure details to remain intact, got %+v", final)
	}
}

func buildJobCreated(jobID string) *events.JobCreatedPayload {
	return &events.JobCreatedPayload{
		JobID:     jobID,
		InputType: "zip",
		InputPath: "staging/" + jobID + "/input.zip",
		Config:    models.JobConfig{Modules: []string{"axe", "lighthouse"}},
	}
}

func buildURLJobCreated(jobID string) *events.JobCreatedPayload {
	return &events.JobCreatedPayload{
		JobID:     jobID,
		InputType: events.InputTypeURLs,
		URLs: []string{
			"http://127.0.0.1:5173",
			"http://127.0.0.1:5173/about",
		},
		Config: models.JobConfig{Modules: []string{"axe"}},
	}
}

func assertExtractionLifecycle(
	ctx context.Context,
	t *testing.T,
	client *sharedmsg.Client,
	pipe jobstatus.JobStatusPipeline,
	jobID string,
) {
	t.Helper()

	waitForRecord(t, pipe, jobID, func(rec *status.JobRecord) bool {
		return rec.State == "EXTRACTING" && rec.InputType == "zip" && len(rec.ExpectedScanners) == 2
	})

	publishEnvelope(
		ctx,
		t,
		client,
		sharedmsg.SubjectExtractionReady,
		events.NewEnvelope(events.EventExtractionReady, jobID, "integration-test", &events.ExtractionReadyPayload{
			JobID:      jobID,
			BaseURL:    "http://localhost:8080",
			TotalPages: 3,
		}),
	)

	waitForRecord(t, pipe, jobID, func(rec *status.JobRecord) bool {
		return rec.TotalPages == 3 && rec.State == "READY_TO_SCAN"
	})
}

func assertScanLifecycle(
	ctx context.Context,
	t *testing.T,
	client *sharedmsg.Client,
	pipe jobstatus.JobStatusPipeline,
	jobID string,
) {
	t.Helper()

	publishEnvelope(
		ctx,
		t,
		client,
		sharedmsg.SubjectScanCompleted,
		events.NewEnvelope(events.EventScanCompleted, jobID, "integration-test", &events.ScanCompletedPayload{
			JobID:             jobID,
			ScannerType:       "axe",
			ResultsPath:       jobID + "/axe/results.json",
			ReportPath:        jobID + "/axe/report.html",
			TotalPagesScanned: 3,
			Summary: events.ScanSummary{
				TotalViolations: 1,
				BySeverity:      map[string]int{"critical": 1},
			},
		}),
	)

	partial := waitForRecord(t, pipe, jobID, func(rec *status.JobRecord) bool {
		return rec.State == "SCANNING" && rec.CurrentPage >= 0 && rec.TotalViolations == 1 &&
			len(rec.CompletedScanners) == 1
	})
	assertPartialScannerState(t, partial)

	publishEnvelope(
		ctx,
		t,
		client,
		sharedmsg.SubjectScanPageCompleted,
		events.NewEnvelope(events.EventScanPageCompleted, jobID, "integration-test", &events.ScanPageCompletedPayload{
			JobID:       jobID,
			ScannerType: "lighthouse",
			PageID:      "page-2",
			PageIndex:   2,
			TotalPages:  3,
		}),
	)

	waitForRecord(t, pipe, jobID, func(rec *status.JobRecord) bool {
		return rec.State == "SCANNING" && len(rec.CompletedScanners) == 1 && rec.CurrentPage >= 2
	})

	publishEnvelope(
		ctx,
		t,
		client,
		sharedmsg.SubjectScanCompleted,
		events.NewEnvelope(events.EventScanCompleted, jobID, "integration-test", &events.ScanCompletedPayload{
			JobID:             jobID,
			ScannerType:       "lighthouse",
			ResultsPath:       jobID + "/lighthouse/results.json",
			ReportPath:        jobID + "/lighthouse/report.html",
			TotalPagesScanned: 3,
			Summary: events.ScanSummary{
				TotalViolations: 1,
				BySeverity:      map[string]int{"critical": 1},
			},
		}),
	)

	waitForRecord(t, pipe, jobID, func(rec *status.JobRecord) bool {
		return rec.State == "SCANNING" && len(rec.CompletedScanners) == 2
	})
}

func assertPartialScannerState(t *testing.T, rec *status.JobRecord) {
	t.Helper()

	if len(rec.CompletedScanners) != 1 || rec.CompletedScanners[0] != "axe" {
		t.Fatalf("expected partial completion to record axe, got %+v", rec.CompletedScanners)
	}

	if len(rec.ExpectedScanners) != 2 {
		t.Fatalf("expected scanner roster to stay intact, got %+v", rec.ExpectedScanners)
	}
}

func newStatusStore(t *testing.T) *status.Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "status.db")

	store, err := status.NewStore(&status.Config{Path: dbPath})
	if err != nil {
		t.Fatalf("failed to create status store: %v", err)
	}

	t.Cleanup(func() {
		_ = store.Close()
	})

	return store
}

func waitForRecord(
	t *testing.T,
	pipe jobstatus.JobStatusPipeline,
	jobID string,
	condition func(*status.JobRecord) bool,
) *status.JobRecord {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		rec, err := pipe.Current(context.Background(), jobID)
		if err == nil && condition(rec) {
			return rec
		}

		time.Sleep(50 * time.Millisecond)
	}

	rec, err := pipe.Current(context.Background(), jobID)
	if err != nil {
		t.Fatalf("job %s not found: %v", jobID, err)
	}

	t.Fatalf("condition not met for job %s, record: %+v", jobID, rec)

	return nil
}

func publishEnvelope(
	ctx context.Context,
	t *testing.T,
	client *sharedmsg.Client,
	subject string,
	envelope *events.Envelope,
) {
	t.Helper()

	if err := client.PublishEvent(ctx, subject, envelope); err != nil {
		t.Fatalf("failed to publish %s: %v", subject, err)
	}
}

func ensureStreamsWithRetry(ctx context.Context, client *sharedmsg.Client, attempts int, delay time.Duration) error {
	var err error
	for range attempts {
		if err = client.EnsureStreams(ctx); err == nil {
			return nil
		}

		time.Sleep(delay)
	}

	return err
}

func newIntegrationFixture(t *testing.T) *integrationFixture {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	t.Cleanup(cancel)

	natsURL := startNATSServer(t)

	client, err := sharedmsg.NewClient(&sharedmsg.Config{
		URL:            natsURL,
		MaxReconnects:  5,
		ReconnectWait:  500 * time.Millisecond,
		ConnectTimeout: 10 * time.Second,
	})
	if err != nil {
		t.Fatalf("failed to create NATS client: %v", err)
	}

	t.Cleanup(func() {
		_ = client.Close()
	})

	if ensureErr := ensureStreamsWithRetry(ctx, client, 20, 250*time.Millisecond); ensureErr != nil {
		t.Fatalf("failed to ensure streams: %v", ensureErr)
	}

	service := platformmsg.NewService(client)
	pipeline := jobstatus.New(&jobstatus.Config{})
	handler := jobstatus.NewEventHandler(pipeline)

	subCtx, subCancel := context.WithCancel(ctx)
	t.Cleanup(subCancel)

	if subscribeErr := service.SubscribeToStatusEvents(subCtx, handler); subscribeErr != nil {
		t.Fatalf("failed to subscribe to events: %v", subscribeErr)
	}

	return &integrationFixture{ctx: ctx, client: client, pipe: pipeline}
}

func startNATSServer(tb testing.TB) string {
	tb.Helper()

	// In restricted sandboxes, binding to TCP ports may be disallowed. Skip instead of failing.
	listenCfg := &net.ListenConfig{}

	l, err := listenCfg.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		if errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EACCES) {
			tb.Skipf("skipping NATS integration test: cannot bind to local TCP port in this environment: %v", err)
		}

		tb.Fatalf("failed to probe TCP listen support: %v", err)
	}

	_ = l.Close()

	opts := &natsserver.Options{
		Host:      "127.0.0.1",
		Port:      -1,
		JetStream: true,
		StoreDir:  tb.TempDir(),
		NoLog:     true,
		NoSigs:    true,
	}

	srv, err := natsserver.NewServer(opts)
	if err != nil {
		tb.Fatalf("failed to create NATS server: %v", err)
	}

	go srv.Start()

	if !srv.ReadyForConnections(10 * time.Second) {
		tb.Fatalf("timed out waiting for NATS to accept connections")
	}

	tb.Cleanup(srv.Shutdown)

	return srv.ClientURL()
}
