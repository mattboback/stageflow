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

	"github.com/mattboback/stageflow/packages/shared-go/events"
	sharedmsg "github.com/mattboback/stageflow/packages/shared-go/messaging"
	"github.com/mattboback/stageflow/packages/shared-go/models"
	platformmsg "github.com/mattboback/stageflow/platform/api/internal/messaging"
	"github.com/mattboback/stageflow/platform/api/internal/status"
)

func TestServiceIntegrationWithNATS(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

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

	if err := ensureStreamsWithRetry(ctx, client, 20, 250*time.Millisecond); err != nil {
		t.Fatalf("failed to ensure streams: %v", err)
	}

	store := newStatusStore(t)
	service := platformmsg.NewService(client)

	subCtx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)
	if err := service.SubscribeToStatusEvents(subCtx, store); err != nil {
		t.Fatalf("failed to subscribe to events: %v", err)
	}

	jobID := "nats-integration-job"
	jobCreated := &events.JobCreatedPayload{
		JobID:     jobID,
		InputType: "zip",
		InputPath: "staging/" + jobID + "/input.zip",
		Config:    models.JobConfig{Modules: []string{"axe"}},
	}

	createdEnv := events.NewEnvelope(events.EventJobCreated, jobID, "integration-test", jobCreated)
	if err := service.PublishJobCreated(ctx, createdEnv); err != nil {
		t.Fatalf("failed to publish job.created: %v", err)
	}

	waitForRecord(t, store, jobID, func(rec *status.JobRecord) bool {
		return rec.State == "EXTRACTING" && rec.InputType == "zip"
	})

	ready := &events.ExtractionReadyPayload{
		JobID:      jobID,
		BaseURL:    "http://localhost:8080",
		TotalPages: 3,
	}
	publishEnvelope(ctx, t, client, sharedmsg.SubjectExtractionReady, events.EventExtractionReady, jobID, ready)
	waitForRecord(t, store, jobID, func(rec *status.JobRecord) bool {
		return rec.TotalPages == 3 && rec.State == "READY_TO_SCAN"
	})

	scanCompleted := &events.ScanCompletedPayload{
		JobID:             jobID,
		ScannerType:       "axe",
		ResultsPath:       jobID + "/axe/results.json",
		ReportPath:        jobID + "/axe/report.html",
		TotalPagesScanned: 3,
		Summary: events.ScanSummary{
			TotalViolations: 1,
			BySeverity:      map[string]int{"critical": 1},
		},
	}
	publishEnvelope(ctx, t, client, sharedmsg.SubjectScanCompleted, events.EventScanCompleted, jobID, scanCompleted)

	waitForRecord(t, store, jobID, func(rec *status.JobRecord) bool {
		return rec.State == "COMPLETING" && rec.CurrentPage >= 0
	})

	jobCompleted := &events.JobCompletedPayload{
		JobID:  jobID,
		Status: "success",
		Artifacts: events.ArtifactLocations{
			ReportJSON: jobID + "/report.json",
			ReportHTML: jobID + "/report.html",
		},
	}
	publishEnvelope(ctx, t, client, sharedmsg.SubjectJobCompleted, events.EventJobCompleted, jobID, jobCompleted)

	final := waitForRecord(t, store, jobID, func(rec *status.JobRecord) bool {
		return rec.State == "DONE" && rec.ReportJSONKey != "" && rec.ReportKey != ""
	})

	if final.ReportJSONKey == "" || final.ReportKey == "" {
		t.Fatalf("expected artifacts to be set, got %+v", final)
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

func waitForRecord(t *testing.T, store *status.Store, jobID string, condition func(*status.JobRecord) bool) *status.JobRecord { //nolint:unparam // keep jobID for clearer call sites
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		rec, err := store.GetJob(context.Background(), jobID)
		if err == nil && condition(rec) {
			return rec
		}
		time.Sleep(50 * time.Millisecond)
	}

	rec, err := store.GetJob(context.Background(), jobID)
	if err != nil {
		t.Fatalf("job %s not found: %v", jobID, err)
	}
	t.Fatalf("condition not met for job %s, record: %+v", jobID, rec)

	return nil
}

func publishEnvelope(ctx context.Context, t *testing.T, client *sharedmsg.Client, subject, eventName, jobID string, payload interface{}) {
	t.Helper()
	envelope := events.NewEnvelope(eventName, jobID, "integration-test", payload)
	if err := client.PublishEvent(ctx, subject, envelope); err != nil {
		t.Fatalf("failed to publish %s: %v", subject, err)
	}
}

func ensureStreamsWithRetry(ctx context.Context, client *sharedmsg.Client, attempts int, delay time.Duration) error {
	var err error
	for i := 0; i < attempts; i++ {
		if err = client.EnsureStreams(ctx); err == nil {
			return nil
		}
		time.Sleep(delay)
	}
	return err
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
