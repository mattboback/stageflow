package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/models"
	storagepkg "github.com/mattboback/stageflow/libs/go/storage"
	"github.com/mattboback/stageflow/services/platform-api/internal/jobstatus"
	"github.com/mattboback/stageflow/services/platform-api/internal/project"
)

func TestBaselineReportKeyIsProjectScoped(t *testing.T) {
	t.Parallel()

	key, err := baselineReportKey("project-id", "job-id")
	if err != nil {
		t.Fatalf("baselineReportKey: %v", err)
	}

	if key != "projects/project-id/baselines/job-id/report.json" {
		t.Fatalf("key = %q", key)
	}

	if _, err = baselineReportKey("../project", "job-id"); err == nil {
		t.Fatal("expected unsafe project ID to be rejected")
	}
}

func TestProjectBaselineSurvivesArtifactExpiry(t *testing.T) {
	server, objectStore, _ := newTestServer(t)
	p := createTestProject(t, server, "persistent-baseline")
	seedCompletedBaselineTestJob(t, server, objectStore, p, "baseline-job")
	promoteBaselineForTest(t, server, p, "baseline-job")

	key, err := baselineReportKey(p.ID, "baseline-job")
	if err != nil {
		t.Fatalf("baseline key: %v", err)
	}

	storedKey := storageTestKey(storagepkg.BucketBaselines, key)
	if _, ok := objectStore.uploads[storedKey]; !ok {
		t.Fatalf("expected persistent baseline at %s", storedKey)
	}

	delete(objectStore.uploads, storageTestKey(storagepkg.BucketArtifacts, "baseline-job/report.json"))
	seedCompletedBaselineTestJob(t, server, objectStore, p, "current-job")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/current-job/diff", http.NoBody)
	rr := httptest.NewRecorder()
	server.handleJobDiff(rr, req, "current-job")

	if rr.Code != http.StatusOK {
		t.Fatalf("expected diff 200 after artifact expiry, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestInvalidReportCannotReplaceBaseline(t *testing.T) {
	server, objectStore, _ := newTestServer(t)
	p := createTestProject(t, server, "invalid-replacement")
	seedCompletedBaselineTestJob(t, server, objectStore, p, "old-job")
	promoteBaselineForTest(t, server, p, "old-job")
	seedCompletedBaselineTestJob(t, server, objectStore, p, "bad-job")

	badReportKey := storageTestKey(storagepkg.BucketArtifacts, "bad-job/report.json")
	objectStore.uploads[badReportKey] = []byte(`{"version":"2.0.0"}`)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/projects/"+p.Slug+"/promote",
		strings.NewReader(`{"job_id":"bad-job"}`),
	)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.handleProjectPromote(rr, req, p.Slug)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid report 400, got %d: %s", rr.Code, rr.Body.String())
	}

	updated, err := server.projectStore.GetProjectByID(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("reload project: %v", err)
	}

	if updated.BaselineJobID != "old-job" {
		t.Fatalf("invalid report replaced baseline: %q", updated.BaselineJobID)
	}

	operations, err := server.projectStore.ListBaselineOperations(context.Background())
	if err != nil {
		t.Fatalf("list baseline operations: %v", err)
	}

	if len(operations) != 0 {
		t.Fatalf("invalid permanent operation remained queued: %#v", operations)
	}
}

func TestPromotionCopyFailureKeepsPointerAndReconciles(t *testing.T) {
	server, objectStore, _ := newTestServer(t)

	p := createTestProject(t, server, "copy-retry")
	if err := server.projectStore.SetBaseline(context.Background(), p.ID, "old-job"); err != nil {
		t.Fatalf("set old baseline: %v", err)
	}

	seedCompletedBaselineTestJob(t, server, objectStore, p, "new-job")
	faults := &baselineFaultStorage{
		Client:    objectStore,
		uploadErr: errors.New("baseline bucket unavailable"),
	}
	server.config.Storage = faults

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/projects/"+p.Slug+"/promote",
		strings.NewReader(`{"job_id":"new-job"}`),
	)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.handleProjectPromote(rr, req, p.Slug)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected queued 503, got %d: %s", rr.Code, rr.Body.String())
	}

	updated, err := server.projectStore.GetProjectByID(context.Background(), p.ID)
	if err != nil || updated.BaselineJobID != "old-job" {
		t.Fatalf("baseline changed after copy failure: project=%#v err=%v", updated, err)
	}

	operations, err := server.projectStore.ListBaselineOperations(context.Background())
	if err != nil || len(operations) != 1 || operations[0].Kind != project.BaselineOperationPromote {
		t.Fatalf("pending promotion = %#v, err=%v", operations, err)
	}

	faults.uploadErr = nil

	if _, err = server.ReconcileProjectBaselines(context.Background()); err != nil {
		t.Fatalf("reconcile promotion: %v", err)
	}

	updated, err = server.projectStore.GetProjectByID(context.Background(), p.ID)
	if err != nil || updated.BaselineJobID != "new-job" {
		t.Fatalf("queued promotion not committed: project=%#v err=%v", updated, err)
	}
}

func TestPromotionCommitFailureReplaysWithoutReupload(t *testing.T) {
	server, objectStore, _ := newTestServer(t)
	p := createTestProject(t, server, "commit-replay")
	seedCompletedBaselineTestJob(t, server, objectStore, p, "new-job")

	faults := &baselineFaultStorage{Client: objectStore}
	server.config.Storage = faults
	underlying := server.projectStore
	server.projectStore = &failingBaselineCommitStore{
		ProjectStore: underlying,
		err:          errors.New("sqlite commit unavailable"),
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/projects/"+p.Slug+"/promote",
		strings.NewReader(`{"job_id":"new-job"}`),
	)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.handleProjectPromote(rr, req, p.Slug)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected queued 503, got %d: %s", rr.Code, rr.Body.String())
	}

	operations, err := underlying.ListBaselineOperations(context.Background())
	if err != nil || len(operations) != 1 || operations[0].State != project.BaselineOperationCommitPending {
		t.Fatalf("commit-pending operation = %#v, err=%v", operations, err)
	}

	uploadCount := faults.uploadCount
	server.projectStore = underlying
	faults.uploadErr = errors.New("unexpected reupload")

	if _, err = server.ReconcileProjectBaselines(context.Background()); err != nil {
		t.Fatalf("reconcile commit-pending operation: %v", err)
	}

	if faults.uploadCount != uploadCount {
		t.Fatalf("commit replay reuploaded: before=%d after=%d", uploadCount, faults.uploadCount)
	}
}

func TestReplacementAndProjectDeletionAreDurablyRetried(t *testing.T) {
	t.Run("replacement cleanup", func(t *testing.T) {
		server, objectStore, _ := newTestServer(t)
		p := createTestProject(t, server, "replace-cleanup")
		seedCompletedBaselineTestJob(t, server, objectStore, p, "old-job")
		promoteBaselineForTest(t, server, p, "old-job")
		seedCompletedBaselineTestJob(t, server, objectStore, p, "new-job")

		faults := &baselineFaultStorage{Client: objectStore, deleteErr: errors.New("delete unavailable")}
		server.config.Storage = faults
		promoteBaselineForTest(t, server, p, "new-job")

		operations, err := server.projectStore.ListBaselineOperations(context.Background())
		if err != nil || len(operations) != 1 || operations[0].Kind != project.BaselineOperationDeleteObject {
			t.Fatalf("pending replacement cleanup = %#v, err=%v", operations, err)
		}

		faults.deleteErr = nil

		if _, err = server.ReconcileProjectBaselines(context.Background()); err != nil {
			t.Fatalf("reconcile replacement cleanup: %v", err)
		}

		oldKey, _ := baselineReportKey(p.ID, "old-job")
		if _, ok := objectStore.uploads[storageTestKey(storagepkg.BucketBaselines, oldKey)]; ok {
			t.Fatal("replaced baseline object remained after reconciliation")
		}
	})

	t.Run("project deletion", func(t *testing.T) {
		server, objectStore, _ := newTestServer(t)
		p := createTestProject(t, server, "delete-retry")
		seedCompletedBaselineTestJob(t, server, objectStore, p, "baseline-job")
		promoteBaselineForTest(t, server, p, "baseline-job")

		faults := &baselineFaultStorage{Client: objectStore, deleteErr: errors.New("delete unavailable")}
		server.config.Storage = faults
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/"+p.Slug, http.NoBody)
		rr := httptest.NewRecorder()
		server.handleDeleteProject(rr, req, p.Slug)

		if rr.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected queued 503, got %d: %s", rr.Code, rr.Body.String())
		}

		if _, err := server.projectStore.GetProjectByID(context.Background(), p.ID); err != nil {
			t.Fatalf("project removed before object cleanup: %v", err)
		}

		faults.deleteErr = nil

		if _, err := server.ReconcileProjectBaselines(context.Background()); err != nil {
			t.Fatalf("reconcile project deletion: %v", err)
		}

		_, err := server.projectStore.GetProjectByID(context.Background(), p.ID)
		if !errors.Is(err, project.ErrNotFound) {
			t.Fatalf("project remained after reconciliation: %v", err)
		}
	})
}

func TestReconcileRepairsInvalidPersistentBaseline(t *testing.T) {
	server, objectStore, _ := newTestServer(t)
	p := createTestProject(t, server, "repair-baseline")
	seedCompletedBaselineTestJob(t, server, objectStore, p, "baseline-job")
	promoteBaselineForTest(t, server, p, "baseline-job")

	key, err := baselineReportKey(p.ID, "baseline-job")
	if err != nil {
		t.Fatalf("baseline key: %v", err)
	}

	storedKey := storageTestKey(storagepkg.BucketBaselines, key)
	objectStore.uploads[storedKey] = []byte(`{"version":"2.0.0"}`)

	summary, err := server.ReconcileProjectBaselines(context.Background())
	if err != nil {
		t.Fatalf("repair persistent baseline: %v", err)
	}

	if summary.LegacyCopied != 1 {
		t.Fatalf("summary = %#v", summary)
	}

	if _, err = decodeBaselineReport(objectStore.uploads[storedKey], "baseline-job"); err != nil {
		t.Fatalf("persistent baseline was not repaired: %v", err)
	}
}

func TestReconcileBackfillsLegacyBaseline(t *testing.T) {
	server, objectStore, _ := newTestServer(t)
	p := createTestProject(t, server, "legacy-backfill")
	seedCompletedBaselineTestJob(t, server, objectStore, p, "legacy-job")

	if err := server.projectStore.SetBaseline(context.Background(), p.ID, "legacy-job"); err != nil {
		t.Fatalf("set legacy baseline: %v", err)
	}

	summary, err := server.ReconcileProjectBaselines(context.Background())
	if err != nil {
		t.Fatalf("reconcile legacy baseline: %v", err)
	}

	if summary.LegacyCopied != 1 || len(summary.MissingLegacyProjects) != 0 {
		t.Fatalf("summary = %#v", summary)
	}

	if server.legacySweepDue.Load() {
		t.Fatal("successful legacy sweep remained scheduled on every journal retry tick")
	}
}

func TestMissingLegacyBaselineReturnsConflict(t *testing.T) {
	server, objectStore, _ := newTestServer(t)
	p := createTestProject(t, server, "missing-legacy")
	seedCompletedBaselineTestJob(t, server, objectStore, p, "legacy-job")

	if err := server.projectStore.SetBaseline(context.Background(), p.ID, "legacy-job"); err != nil {
		t.Fatalf("set legacy baseline: %v", err)
	}

	delete(objectStore.uploads, storageTestKey(storagepkg.BucketArtifacts, "legacy-job/report.json"))

	summary, err := server.ReconcileProjectBaselines(context.Background())
	if err != nil {
		t.Fatalf("missing legacy baseline should be degraded, not fatal: %v", err)
	}

	if len(summary.MissingLegacyProjects) != 1 || summary.MissingLegacyProjects[0] != p.Slug {
		t.Fatalf("summary = %#v", summary)
	}

	seedCompletedBaselineTestJob(t, server, objectStore, p, "current-job")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/current-job/diff", http.NoBody)
	rr := httptest.NewRecorder()
	server.handleJobDiff(rr, req, "current-job")

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestBackgroundBaselineReconcilerRetriesOperation(t *testing.T) {
	server, objectStore, _ := newTestServer(t)
	p := createTestProject(t, server, "background-retry")
	seedCompletedBaselineTestJob(t, server, objectStore, p, "baseline-job")
	promoteBaselineForTest(t, server, p, "baseline-job")

	faults := &baselineFaultStorage{Client: objectStore, deleteErr: errors.New("temporary delete failure")}
	server.config.Storage = faults
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/"+p.Slug, http.NoBody)
	rr := httptest.NewRecorder()
	server.handleDeleteProject(rr, req, p.Slug)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected queued 503, got %d", rr.Code)
	}

	faults.deleteErr = nil

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		server.WaitForBaselineReconciler()
	}()

	server.startBaselineReconciler(ctx, 5*time.Millisecond)

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		_, err := server.projectStore.GetProjectByID(context.Background(), p.ID)
		if errors.Is(err, project.ErrNotFound) {
			return
		}

		time.Sleep(5 * time.Millisecond)
	}

	t.Fatal("background reconciler did not finish the queued deletion")
}

func TestBackgroundBaselineReconcilerResumesLegacySweepAfterJournalFailure(t *testing.T) {
	server, objectStore, _ := newTestServer(t)
	journalProject := createTestProject(t, server, "journal-first")
	legacyProject := createTestProject(t, server, "legacy-after-journal")
	seedCompletedBaselineTestJob(t, server, objectStore, journalProject, "journal-job")
	seedCompletedBaselineTestJob(t, server, objectStore, legacyProject, "legacy-job")

	if err := server.projectStore.SetBaseline(
		context.Background(),
		legacyProject.ID,
		"legacy-job",
	); err != nil {
		t.Fatalf("set legacy baseline: %v", err)
	}

	uploaded := make(chan string, 4)
	faults := &baselineFaultStorage{
		Client:    objectStore,
		uploadErr: errors.New("temporary baseline outage"),
		uploaded:  uploaded,
	}
	server.config.Storage = faults

	if err := server.promoteBaseline(
		context.Background(),
		journalProject.ID,
		"",
		"journal-job",
		"journal-job/report.json",
	); !errors.Is(err, errBaselineOperationQueued) {
		t.Fatalf("queue journal operation: %v", err)
	}

	if _, err := server.ReconcileProjectBaselines(context.Background()); err == nil {
		t.Fatal("startup reconciliation unexpectedly succeeded during storage outage")
	}

	faults.uploadErr = nil

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		server.WaitForBaselineReconciler()
	}()

	server.startBaselineReconciler(ctx, 5*time.Millisecond)

	wantKey, err := baselineReportKey(legacyProject.ID, "legacy-job")
	if err != nil {
		t.Fatalf("legacy baseline key: %v", err)
	}

	wantKey = storageTestKey(storagepkg.BucketBaselines, wantKey)

	timeout := time.NewTimer(time.Second)
	defer timeout.Stop()

	for {
		select {
		case key := <-uploaded:
			if key == wantKey {
				return
			}
		case <-timeout.C:
			t.Fatal("background reconciler repaired the journal but did not resume the legacy sweep")
		}
	}
}

func TestBackgroundBaselineReconcilerDoesNotLetPermanentOperationStarveLegacySweep(t *testing.T) {
	server, objectStore, _ := newTestServer(t)
	blockedProject := createTestProject(t, server, "blocked-operation")
	legacyProject := createTestProject(t, server, "legacy-independent")
	seedCompletedBaselineTestJob(t, server, objectStore, blockedProject, "blocked-job")
	seedCompletedBaselineTestJob(t, server, objectStore, legacyProject, "independent-job")

	if err := server.projectStore.SetBaseline(
		context.Background(),
		legacyProject.ID,
		"independent-job",
	); err != nil {
		t.Fatalf("set legacy baseline: %v", err)
	}

	blockedPath, err := baselineReportKey(blockedProject.ID, "blocked-job")
	if err != nil {
		t.Fatalf("blocked baseline key: %v", err)
	}

	wantPath, err := baselineReportKey(legacyProject.ID, "independent-job")
	if err != nil {
		t.Fatalf("independent baseline key: %v", err)
	}

	uploaded := make(chan string, 4)
	faults := &baselineFaultStorage{
		Client:       objectStore,
		uploadErr:    errors.New("object-specific permission failure"),
		uploadErrFor: storageTestKey(storagepkg.BucketBaselines, blockedPath),
		uploaded:     uploaded,
	}
	server.config.Storage = faults

	if err = server.promoteBaseline(
		context.Background(),
		blockedProject.ID,
		"",
		"blocked-job",
		"blocked-job/report.json",
	); !errors.Is(err, errBaselineOperationQueued) {
		t.Fatalf("queue blocked operation: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		server.WaitForBaselineReconciler()
	}()

	server.startBaselineReconciler(ctx, 5*time.Millisecond)

	timeout := time.NewTimer(time.Second)
	defer timeout.Stop()

	wantUpload := storageTestKey(storagepkg.BucketBaselines, wantPath)

	for {
		select {
		case key := <-uploaded:
			if key == wantUpload {
				return
			}
		case <-timeout.C:
			t.Fatal("permanently blocked operation starved an independent legacy baseline")
		}
	}
}

type baselineFaultStorage struct {
	storagepkg.Client
	uploadErr    error
	uploadErrFor string
	deleteErr    error
	uploadCount  int
	uploaded     chan<- string
}

func (s *baselineFaultStorage) UploadFile(
	ctx context.Context,
	bucket, path string,
	reader io.Reader,
	size int64,
) error {
	s.uploadCount++

	key := storageTestKey(bucket, path)
	if s.uploadErr != nil && (s.uploadErrFor == "" || s.uploadErrFor == key) {
		return s.uploadErr
	}

	if err := s.Client.UploadFile(ctx, bucket, path, reader, size); err != nil {
		return err
	}

	if s.uploaded != nil {
		select {
		case s.uploaded <- key:
		default:
		}
	}

	return nil
}

func (s *baselineFaultStorage) DeleteFile(ctx context.Context, bucket, path string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}

	return s.Client.DeleteFile(ctx, bucket, path)
}

type failingBaselineCommitStore struct {
	ProjectStore
	err error
}

func (s *failingBaselineCommitStore) CompleteBaselinePromotion(
	context.Context,
	project.BaselineOperation,
) error {
	return s.err
}

func seedCompletedBaselineTestJob(
	t *testing.T,
	server *Server,
	objectStore *fakeStorage,
	p *project.Project,
	jobID string,
) {
	t.Helper()

	if err := server.projectStore.RecordProjectJob(context.Background(), p.ID, jobID); err != nil {
		t.Fatalf("record project job: %v", err)
	}

	applyTestSignal(t, server, jobstatus.Signal{
		Kind:       jobstatus.SignalJobCreated,
		ObservedAt: time.Now().UTC(),
		JobCreated: &events.JobCreatedPayload{
			JobID:     jobID,
			InputType: models.JobInputTypeURLs,
			URLs:      []string{"https://example.com"},
		},
	})

	reportKey := jobID + "/report.json"

	data, err := json.Marshal(buildStructuredScreenshotReport(jobID, time.Now().UTC()))
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}

	objectStore.uploads[storageTestKey(storagepkg.BucketArtifacts, reportKey)] = data
	completeJobWithArtifacts(t, server.jobStatus, jobID, events.ArtifactLocations{
		ReportJSON: reportKey,
		ReportHTML: jobID + "/report.html",
	})
}

func promoteBaselineForTest(t *testing.T, server *Server, p *project.Project, jobID string) {
	t.Helper()

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/projects/"+p.Slug+"/promote",
		strings.NewReader(fmt.Sprintf(`{"job_id":%q}`, jobID)),
	)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.handleProjectPromote(rr, req, p.Slug)

	if rr.Code != http.StatusOK {
		t.Fatalf("promote %s: status=%d body=%s", jobID, rr.Code, rr.Body.String())
	}
}

func storageTestKey(bucket, path string) string {
	return fmt.Sprintf("%s::%s", bucket, path)
}
