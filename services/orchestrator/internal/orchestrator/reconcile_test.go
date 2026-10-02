package orchestrator

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/mattboback/stageflow/internal/artifacts"
	"github.com/mattboback/stageflow/internal/jobstore"
	"github.com/mattboback/stageflow/internal/testpostgres"
	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
	"github.com/mattboback/stageflow/libs/go/models"
	podman "github.com/mattboback/stageflow/services/orchestrator/internal/adapters/runtime"
	adapterstorage "github.com/mattboback/stageflow/services/orchestrator/internal/adapters/storage"
)

var testURL string

func TestMain(m *testing.M) {
	url, stop, err := testpostgres.PostgresForTests("reconcile")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	testURL = url
	code := m.Run()
	_ = stop()

	os.Exit(code)
}

type fakeRuntime struct {
	createEntered chan struct{}
	releaseCreate chan struct{}
	inspectErr    error

	containers []podman.ContainerInfo
	removeErr  error
	creates    int
	lists      int
	pods       int
}

func (f *fakeRuntime) CreatePod(context.Context, *podman.PodCreateRequest) (*podman.PodCreateResponse, error) {
	f.pods++
	if f.createEntered != nil {
		close(f.createEntered)
		<-f.releaseCreate
	}

	return &podman.PodCreateResponse{ID: "pod"}, nil
}
func (f *fakeRuntime) InspectPod(context.Context, string) (*podman.PodInfo, error) {
	return nil, errors.New("missing")
}
func (f *fakeRuntime) StopPod(context.Context, string) error { return nil }
func (f *fakeRuntime) RemovePod(context.Context, string, bool) error {
	if f.removeErr == nil {
		f.containers = nil
	}

	return f.removeErr
}
func (f *fakeRuntime) CreateVolume(context.Context, string) error { return nil }
func (f *fakeRuntime) InspectVolume(context.Context, string) (*podman.VolumeInfo, error) {
	return nil, errors.New("unused")
}

func (f *fakeRuntime) CreateContainer(
	context.Context,
	*podman.ContainerCreateRequest,
) (*podman.ContainerCreateResponse, error) {
	f.creates++
	return nil, errors.New("unexpected create")
}
func (f *fakeRuntime) InspectContainer(_ context.Context, id string) (*podman.ContainerInfo, error) {
	if f.inspectErr != nil {
		return nil, f.inspectErr
	}

	for i := range f.containers {
		if f.containers[i].ID == id {
			return &f.containers[i], nil
		}
	}

	return nil, errors.New("missing")
}
func (f *fakeRuntime) StartContainer(context.Context, string) error        { return nil }
func (f *fakeRuntime) RemoveContainer(context.Context, string, bool) error { return nil }
func (f *fakeRuntime) ListJobContainers(context.Context) ([]podman.ContainerInfo, error) {
	f.lists++
	return f.containers, nil
}
func testOrchestrator(t *testing.T) (*Orchestrator, *fakeRuntime) {
	t.Helper()

	ctx := context.Background()

	admin, err := sql.Open("pgx", testURL)
	if err != nil {
		t.Fatal(err)
	}

	schema := fmt.Sprintf("reconcile_%d", time.Now().UnixNano())
	if _, execContextErr := admin.ExecContext(ctx, "CREATE SCHEMA "+schema); execContextErr != nil {
		t.Fatal(execContextErr)
	}

	database, err := jobstore.NewDatabase(&jobstore.Config{URL: testURL + "&search_path=" + schema})
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		_ = database.Close()
		_, _ = admin.Exec("DROP SCHEMA " + schema + " CASCADE")
		_ = admin.Close()
	})
	dir := t.TempDir()

	files, err := artifacts.NewFiles(dir)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = files.Close() })

	runtime := &fakeRuntime{}
	o := NewOrchestrator(
		&Config{
			Database:       database,
			PodmanClient:   runtime,
			Files:          files,
			DataDir:        dir,
			DirectoryOwner: func(string) error { return nil },
		},
	)

	return o, runtime
}
func admit(t *testing.T, o *Orchestrator, id string) {
	t.Helper()

	job := &models.Job{
		ID:               id,
		InputType:        models.JobInputTypeURLs,
		URLs:             []string{"https://example.com"},
		ExpectedScanners: []string{"axe"},
		Config:           models.JobConfig{Modules: []string{"axe"}},
	}
	if err := o.config.Database.AdmitJob(context.Background(), job); err != nil {
		t.Fatal(err)
	}
}
func active(t *testing.T, o *Orchestrator, id string) *models.Job {
	t.Helper()
	admit(t, o, id)

	ctx := context.Background()

	job, err := o.config.Database.ClaimNext(ctx)
	if err != nil || job == nil {
		t.Fatalf("claim %v %v", job, err)
	}

	for _, state := range []models.JobState{models.JobStateReady, models.JobStateScanning} {
		if beginStageErr := o.config.Database.BeginStage(ctx, id, state); beginStageErr != nil {
			t.Fatal(beginStageErr)
		}
	}

	if setPodIDErr := o.config.Database.SetPodID(ctx, id, "pod"); setPodIDErr != nil {
		t.Fatal(setPodIDErr)
	}

	job, err = o.config.Database.GetJob(ctx, id)
	if err != nil {
		t.Fatal(err)
	}

	return job
}
func scanner(id, state string, exit int) podman.ContainerInfo {
	return podman.ContainerInfo{
		ID:       "scanner",
		State:    podman.ContainerState(state),
		ExitCode: exit,
		Labels: map[string]string{
			"managed_by":   "orchestrator",
			"job_id":       id,
			"component":    "scanner",
			"scanner_type": "axe",
		},
	}
}
func TestFailedCleanupRetainsActiveCapacity(t *testing.T) {
	o, runtime := testOrchestrator(t)
	job := active(t, o, "first")
	runtime.containers = []podman.ContainerInfo{scanner(job.ID, "running", 0)}

	admit(t, o, "second")

	if err := o.fail(context.Background(), job, "scanning", "deadline"); err != nil {
		t.Fatal(err)
	}

	runtime.removeErr = errors.New("pod removal unavailable")

	if err := o.ReconcileOnce(context.Background()); err != nil {
		t.Fatal(err)
	}

	queued, err := o.config.Database.GetJob(context.Background(), "second")
	if err != nil {
		t.Fatal(err)
	}

	if queued.ClaimedAt != nil || runtime.pods != 0 || runtime.creates != 0 {
		t.Fatal("cleanup failure released running job capacity")
	}

	runtime.removeErr = nil

	o.config.CheckDisk = func(context.Context) error { return artifacts.ErrLowDisk }
	if reconcileOnceErr := o.ReconcileOnce(context.Background()); reconcileOnceErr != nil {
		t.Fatal(reconcileOnceErr)
	}

	cleaned, _ := o.config.Database.GetJob(context.Background(), "first")
	if cleaned.CleanupPending {
		t.Fatal("successful cleanup did not release reservation")
	}

	next, err := o.config.Database.ClaimNext(context.Background())
	if err != nil || next == nil || next.ID != "second" {
		t.Fatalf("claim after cleanup %v %v", next, err)
	}
}
func TestRestartAdoptsRunningScannerWithoutAttempt(t *testing.T) {
	o, runtime := testOrchestrator(t)
	job := active(t, o, "running")
	runtime.containers = []podman.ContainerInfo{scanner(job.ID, "running", 0)}

	for range 2 {
		if err := o.ReconcileOnce(context.Background()); err != nil {
			t.Fatal(err)
		}
	}

	after, _ := o.config.Database.GetJob(context.Background(), job.ID)
	if runtime.creates != 0 || after.ScannerAttempts["axe"] != 0 || after.ScannerContainerIDs["axe"] != "scanner" ||
		runtime.lists != 2 {
		t.Fatalf("lost adoption: %#v runtime=%#v", after, runtime)
	}

	if !after.ScanStartedAt.Equal(*job.ScanStartedAt) {
		t.Fatal("restart reset stage deadline")
	}
}
func TestExitFailureOverridesCleanOutcome(t *testing.T) {
	o, runtime := testOrchestrator(t)
	job := active(t, o, "failed")
	runtime.containers = []podman.ContainerInfo{scanner(job.ID, "exited", 137)}
	runtime.containers[0].OOMKilled = true

	dir := filepath.Join(o.config.DataDir, "jobs", job.ID, "axe")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(dir, "outcome.json"),
		[]byte(`{"status":"success","pages_scanned":1,"summary":{},"timing":{}}`),
		0600,
	); err != nil {
		t.Fatal(err)
	}

	if err := o.ReconcileOnce(context.Background()); err != nil {
		t.Fatal(err)
	}

	after, _ := o.config.Database.GetJob(context.Background(), job.ID)
	if after.State != models.JobStateFailed || after.ScannerResults["axe"].Success {
		t.Fatalf("exit failure became clean pass: %#v", after)
	}
}
func TestProgressIgnoresIncompleteTrailingLine(t *testing.T) {
	o, _ := testOrchestrator(t)
	job := active(t, o, "progress")
	dir := filepath.Join(o.config.DataDir, "jobs", job.ID, "axe")
	_ = os.MkdirAll(dir, 0700)
	data := []byte("{\"pages_scanned\":2,\"total_pages\":4}\n{\"pages_scanned\":3")
	_ = os.WriteFile(filepath.Join(dir, "progress.jsonl"), data, 0600)

	if err := o.readProgress(context.Background(), job, "axe"); err != nil {
		t.Fatal(err)
	}

	after, _ := o.config.Database.GetJob(context.Background(), job.ID)
	if after.ScannerProgress["axe"] != 2 || after.TotalPages != 4 {
		raw, _ := json.Marshal(after)
		t.Fatalf("progress %s", raw)
	}
}
func TestAbsoluteDeadlineIgnoresRecentProgress(t *testing.T) {
	o, _ := testOrchestrator(t)
	start := time.Now().Add(-time.Hour)
	job := &models.Job{
		State:           models.JobStateScanning,
		ClaimedAt:       &start,
		ScanStartedAt:   &start,
		UpdatedAt:       time.Now(),
		ScannerProgress: map[string]int{"axe": 99},
	}

	expired, stage := o.expired(job, time.Now())
	if !expired || stage != "scanning" {
		t.Fatal("recent progress extended absolute scan deadline")
	}
}

func TestTransientInspectDoesNotFailScanner(t *testing.T) {
	for _, fault := range []error{context.DeadlineExceeded, &podman.APIError{StatusCode: 503}} {
		t.Run(fault.Error(), func(t *testing.T) {
			o, runtime := testOrchestrator(t)
			job := active(t, o, "inspect")
			runtime.containers = []podman.ContainerInfo{scanner(job.ID, "exited", 0)}
			runtime.inspectErr = fault

			if err := o.ReconcileOnce(context.Background()); err == nil {
				t.Fatal("lost transient inspection error")
			}

			after, _ := o.config.Database.GetJob(context.Background(), job.ID)
			if after.State != models.JobStateScanning || len(after.ScannerResults) != 0 {
				t.Fatal("transient failure settled scanner")
			}

			runtime.inspectErr = nil

			if err := o.ReconcileOnce(context.Background()); err != nil {
				t.Fatal(err)
			}

			after, _ = o.config.Database.GetJob(context.Background(), job.ID)
			if after.State != models.JobStateFailed {
				t.Fatal("zero exit without outcome was clean")
			}
		})
	}
}
func TestInvalidOutcomeFailsCoverage(t *testing.T) {
	for _, data := range []string{`{"status":"success","pages_scanned":1,"summary":{},"timing":{}}`, string(make([]byte, maxOutcomeBytes+1)), `not-json`} {
		t.Run(fmt.Sprintf("bytes-%d", len(data)), func(t *testing.T) {
			o, runtime := testOrchestrator(t)
			job := active(t, o, "invalid")
			runtime.containers = []podman.ContainerInfo{scanner(job.ID, "exited", 0)}
			dir := filepath.Join(o.config.DataDir, "jobs", job.ID, "axe")
			_ = os.MkdirAll(dir, 0700)
			_ = os.WriteFile(filepath.Join(dir, "outcome.json"), []byte(data), 0600)

			if err := o.ReconcileOnce(context.Background()); err != nil {
				t.Fatal(err)
			}

			after, _ := o.config.Database.GetJob(context.Background(), job.ID)
			if after.State != models.JobStateFailed || after.ScannerResults["axe"].Success {
				t.Fatal("invalid outcome became clean pass")
			}
		})
	}
}

type interruptedFiles struct {
	adapterstorage.JobFiles
	failReport bool
}

func (s *interruptedFiles) Write(ctx context.Context, key string, reader io.Reader) error {
	if s.failReport && filepath.Base(key) == "report.json" {
		return errors.New("interrupted report write")
	}

	return s.JobFiles.Write(ctx, key, reader)
}

//nolint:gocyclo // Verifies fault, restart, partial coverage and terminal replay in one lifecycle.
func TestCompletingRetryAfterRestartPublishesOneReport(t *testing.T) {
	o, runtime := testOrchestrator(t)
	job := active(t, o, "aggregate")
	ctx := context.Background()
	document := report.UnifiedReportV2{
		Version:  "2.2.0",
		Meta:     report.ReportMeta{JobId: job.ID},
		Pages:    []report.PageSummary{{Id: "page", Url: "https://example.com/"}},
		Scanners: []report.ScannerSummary{{Id: "axe", Status: report.ScannerStatusSuccess}},
	}

	data, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}

	key := job.ID + "/axe/results.json"
	if writeErr := o.config.Files.Write(ctx, key, bytes.NewReader(data)); writeErr != nil {
		t.Fatal(writeErr)
	}

	if recordScannerResultErr := o.config.Database.RecordScannerResult(
		ctx,
		job.ID,
		"partial",
		&models.ScannerResult{
			ScannerType:  "axe",
			Success:      true,
			ResultsPath:  key,
			PagesScanned: 1,
			Error:        "a page audit failed",
		},
	); recordScannerResultErr != nil {
		t.Fatal(recordScannerResultErr)
	}

	if beginStageErr := o.config.Database.BeginStage(ctx, job.ID, models.JobStateCompleting); beginStageErr != nil {
		t.Fatal(beginStageErr)
	}

	files := &interruptedFiles{JobFiles: o.config.Files, failReport: true}

	o.aggregator = adapterstorage.NewAggregator(files, nil)
	if reconcileOnceErr := o.ReconcileOnce(ctx); reconcileOnceErr == nil {
		t.Fatal("missing interrupted aggregation error")
	}

	after, _ := o.config.Database.GetJob(ctx, job.ID)
	if after.State != models.JobStateCompleting {
		t.Fatal("failed report write settled terminal")
	}

	files.failReport = false
	restarted := NewOrchestrator(&o.config)

	restarted.config.CheckDisk = func(context.Context) error { return artifacts.ErrLowDisk }
	if reconcileOnceErr := restarted.ReconcileOnce(ctx); reconcileOnceErr != nil {
		t.Fatal(reconcileOnceErr)
	}

	after, _ = o.config.Database.GetJob(ctx, job.ID)
	if after.State != models.JobStateDone || after.ReportJSONKey != job.ID+"/report.json" || runtime.creates != 0 {
		t.Fatalf("recovery %#v", after)
	}

	reader, err := o.config.Files.Open(ctx, after.ReportJSONKey)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	var published report.UnifiedReportV2
	if decodeErr := json.NewDecoder(reader).Decode(&published); decodeErr != nil {
		t.Fatal(decodeErr)
	}

	if published.Scanners[0].Status != report.ScannerStatusPartial || published.Summary.Score != nil {
		t.Fatal("partial outcome was rendered as complete coverage")
	}

	if reconcileOnceErr := restarted.ReconcileOnce(ctx); reconcileOnceErr != nil {
		t.Fatal(reconcileOnceErr)
	}

	again, _ := o.config.Database.GetJob(ctx, job.ID)
	if !again.CompletedAt.Equal(*after.CompletedAt) {
		t.Fatal("replay changed terminal timestamp")
	}
}

func TestOutcomeFIFOIsRejectedWithoutBlocking(t *testing.T) {
	o, _ := testOrchestrator(t)
	admit(t, o, "fifo")
	dir := filepath.Join(o.config.DataDir, "jobs", "fifo", "axe")

	_ = os.MkdirAll(dir, 0700)
	if err := syscall.Mkfifo(filepath.Join(dir, "outcome.json"), 0600); err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)

	go func() { _, err := o.readOutcome("fifo", "axe"); done <- err }()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("FIFO accepted")
		}
	case <-time.After(time.Second):
		t.Fatal("outcome reader blocked on FIFO")
	}
}
func TestOrphanRetentionPreservesDurableAndFreshDirectories(t *testing.T) {
	o, _ := testOrchestrator(t)
	admit(t, o, "durable")
	root := filepath.Join(o.config.DataDir, "jobs")
	old := time.Now().Add(-2 * time.Hour)

	for _, id := range []string{"orphan", "durable", "fresh"} {
		dir := filepath.Join(root, id)

		_ = os.MkdirAll(dir, 0700)
		if id != "fresh" {
			_ = os.Chtimes(dir, old, old)
		}
	}

	if err := o.retainOrphans(context.Background(), time.Now()); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(root, "orphan")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("old orphan retained")
	}

	for _, id := range []string{"durable", "fresh"} {
		if _, err := os.Stat(filepath.Join(root, id)); err != nil {
			t.Fatalf("removed %s", id)
		}
	}
}

func TestConcurrentSchedulersDoNotRaceLaunchOwnership(t *testing.T) {
	o, runtime := testOrchestrator(t)
	admit(t, o, "leased")

	runtime.createEntered = make(chan struct{})
	runtime.releaseCreate = make(chan struct{})
	second := NewOrchestrator(&o.config)

	firstDone := make(chan error, 1)
	go func() { firstDone <- o.ReconcileOnce(context.Background()) }()

	select {
	case <-runtime.createEntered:
	case <-time.After(5 * time.Second):
		t.Fatal("first scheduler did not claim")
	}

	if err := second.ReconcileOnce(context.Background()); err != nil {
		t.Fatal(err)
	}

	close(runtime.releaseCreate)

	if err := <-firstDone; err != nil {
		t.Fatal(err)
	}

	if runtime.pods != 1 || runtime.lists != 1 {
		t.Fatalf("concurrent scheduler repeated runtime side effects: pods=%d lists=%d", runtime.pods, runtime.lists)
	}
}

func TestDurableOutcomeSuppliesSuccessfulAndFailedTotals(t *testing.T) {
	for _, status := range []string{"success", "failed"} {
		t.Run(status, func(t *testing.T) { testDurableOutcome(t, status, false) })
	}

	// A scanner image from before 2.4c may still be running when the
	// orchestrator restarts on a new image.
	t.Run("legacy publish timing", func(t *testing.T) { testDurableOutcome(t, "success", true) })
}
func testDurableOutcome(t *testing.T, status string, legacyPublishTiming bool) {
	t.Helper()
	o, runtime := testOrchestrator(t)
	job := active(t, o, "outcome")
	runtime.containers = []podman.ContainerInfo{scanner(job.ID, "exited", 0)}

	data, err := os.ReadFile("../../../../libs/contracts/outcome/fixtures/outcome.json")
	if err != nil {
		t.Fatal(err)
	}

	var outcome map[string]any
	if decodeErr := json.Unmarshal(data, &outcome); decodeErr != nil {
		t.Fatal(decodeErr)
	}

	outcome["status"] = status
	if status == "failed" {
		outcome["error"] = "scanner stopped after a page"
	}

	if legacyPublishTiming {
		timing, ok := outcome["timing"].(map[string]any)
		if !ok {
			t.Fatal("outcome fixture has no timing object")
		}

		timing["publish_completed_ms"] = 0
	}

	data, _ = json.Marshal(outcome)
	dir := filepath.Join(o.config.DataDir, "jobs", job.ID, "axe")
	_ = os.MkdirAll(dir, 0700)
	_ = os.WriteFile(filepath.Join(dir, "outcome.json"), data, 0600)

	doc := report.UnifiedReportV2{
		Version: "2.2.0",
		Meta:    report.ReportMeta{JobId: job.ID},
		Summary: report.ReportSummary{TotalIssues: 3},
		Pages:   []report.PageSummary{{Id: "p", Url: "https://example.com/"}},
		Issues: []report.IssueDetail{
			{Id: "1", Scanner: "axe", RuleId: "test1", Severity: report.IssueSeveritySerious, PageId: "p"},
			{Id: "2", Scanner: "axe", RuleId: "test2", Severity: report.IssueSeveritySerious, PageId: "p"},
			{Id: "3", Scanner: "axe", RuleId: "test3", Severity: report.IssueSeveritySerious, PageId: "p"},
		},
	}
	for i := range doc.Issues {
		doc.Issues[i].Description = "description"
		doc.Issues[i].Title = "test finding"
		doc.Issues[i].PageUrl = "https://example.com/"
		doc.Issues[i].ElementCount = 1
	}

	raw, _ := json.Marshal(doc)
	_ = os.WriteFile(filepath.Join(dir, "results.json"), raw, 0644)

	if reconcileErr := o.ReconcileOnce(context.Background()); reconcileErr != nil {
		t.Fatal(reconcileErr)
	}

	after, _ := o.config.Database.GetJob(context.Background(), job.ID)

	result := after.ScannerResults["axe"]
	if result.PagesScanned != 1 || result.TotalMs != 50 {
		t.Fatalf("durable outcome metadata lost: %#v", result)
	}

	expected := models.JobStateDone
	if status == "failed" {
		expected = models.JobStateFailed
	}

	if after.State != expected {
		t.Fatalf("state %s expected %s result=%#v", after.State, expected, result)
	}
}
