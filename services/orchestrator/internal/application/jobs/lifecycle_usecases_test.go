package jobs

import (
	"errors"
	"testing"

	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/messaging"
	"github.com/mattboback/stageflow/libs/go/models"
)

var errFakeLifecycleJobNotSeeded = errors.New("fake job store GetJob called without a seeded result")

func TestServiceCreateJobDuplicatePendingRetriesWorkflow(t *testing.T) {
	t.Parallel()

	store := &fakeJobStore{
		createJobIfAbsentCreated: false,
		getJobResults: []*models.Job{
			{
				ID:        "job-dup",
				State:     models.JobStatePending,
				InputType: models.JobInputTypeZip,
				InputPath: "staging/job-dup/test.zip",
				Config: models.JobConfig{
					Modules: []string{"axe"},
				},
			},
		},
	}
	runtime := &fakeRuntime{}

	service := NewService(store, runtime, &fakeArtifacts{}, &fakePublisher{})
	payload := &events.JobCreatedPayload{
		JobID:     "job-dup",
		InputType: string(models.JobInputTypeZip),
		InputPath: "staging/job-dup/test.zip",
		Config: models.JobConfig{
			Modules: []string{"axe"},
		},
	}

	if err := service.CreateJob(t.Context(), payload); err != nil {
		t.Fatalf("CreateJob() error = %v", err)
	}

	if store.createJobIfAbsentCalls != 1 {
		t.Fatalf("CreateJobIfAbsent() calls = %d, want 1", store.createJobIfAbsentCalls)
	}

	if runtime.createJobPodCalls != 1 {
		t.Fatalf("CreateJobPod() calls = %d, want 1", runtime.createJobPodCalls)
	}

	if runtime.startExtractionWorkerCalls != 1 {
		t.Fatalf("StartExtractionWorker() calls = %d, want 1", runtime.startExtractionWorkerCalls)
	}
}

func TestServiceCreateJobCarriesBrowserEngineFromPayload(t *testing.T) {
	t.Parallel()

	var operationLog []string

	store := &fakeJobStore{
		createJobIfAbsentCreated: true,
		operationLog:             &operationLog,
	}
	runtime := &fakeRuntime{
		createJobPodID:       "pod-engine",
		resolvedScannerTypes: []string{"axe"},
		allowLoopbackTargets: true,
		operationLog:         &operationLog,
	}

	service := NewService(store, runtime, &fakeArtifacts{}, &fakePublisher{})
	payload := &events.JobCreatedPayload{
		JobID:     "job-engine",
		InputType: string(models.JobInputTypeURLs),
		URLs:      []string{"https://example.com"},
		Config: models.JobConfig{
			Modules:        []string{"axe"},
			Browser:        "firefox",
			HighlightStyle: "solid",
		},
	}

	if err := service.CreateJob(t.Context(), payload); err != nil {
		t.Fatalf("CreateJob() error = %v", err)
	}

	if store.lastCreatedJob == nil {
		t.Fatal("CreateJobIfAbsent was not called with a job")
	}

	// Regression: the orchestrator rebuilds JobConfig field-by-field from the
	// event payload, so a new Config field must be copied here too — Browser was
	// dropped while HighlightStyle was carried, sending every scan to Chromium.
	if got := store.lastCreatedJob.Config.Browser; got != "firefox" {
		t.Fatalf("persisted Config.Browser = %q, want %q", got, "firefox")
	}

	if got := store.lastCreatedJob.Config.HighlightStyle; got != "solid" {
		t.Fatalf("persisted Config.HighlightStyle = %q, want %q", got, "solid")
	}
}

func TestServiceCreateJobURLPreparesLifecycleBeforeStartingScanners(t *testing.T) {
	t.Parallel()

	var operationLog []string

	store := &fakeJobStore{
		createJobIfAbsentCreated: true,
		operationLog:             &operationLog,
	}
	runtime := &fakeRuntime{
		createJobPodID:       "pod-url-123",
		resolvedScannerTypes: []string{"axe"},
		allowLoopbackTargets: true,
		operationLog:         &operationLog,
	}

	service := NewService(store, runtime, &fakeArtifacts{}, &fakePublisher{})
	payload := &events.JobCreatedPayload{
		JobID:     "job-url",
		InputType: string(models.JobInputTypeURLs),
		URLs:      []string{"https://example.com"},
		Config: models.JobConfig{
			Modules: []string{"axe"},
		},
	}

	if err := service.CreateJob(t.Context(), payload); err != nil {
		t.Fatalf("CreateJob() error = %v", err)
	}

	got, want := store.lastStateUpdates, []models.JobState{models.JobStateReady, models.JobStateScanning}
	if len(got) != len(want) {
		t.Fatalf("UpdateJobState() calls = %v, want %v", got, want)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("UpdateJobState() calls = %v, want %v", got, want)
		}
	}

	if store.updateJobPodIDCalls != 1 {
		t.Fatalf("UpdateJobPodID() calls = %d, want 1", store.updateJobPodIDCalls)
	}

	assertOperationOrder(
		t,
		operationLog,
		"runtime.CreateJobPod",
		"store.UpdateJobPodID",
		"store.UpdateJobState:"+string(models.JobStateReady),
	)

	if store.updateJobProvenanceKeyCalls != 1 {
		t.Fatalf("UpdateJobProvenanceKey() calls = %d, want 1", store.updateJobProvenanceKeyCalls)
	}

	if runtime.createJobPodCalls != 1 {
		t.Fatalf("CreateJobPod() calls = %d, want 1", runtime.createJobPodCalls)
	}

	if runtime.startScannerCalls != 1 {
		t.Fatalf("StartScanner() calls = %d, want 1", runtime.startScannerCalls)
	}
}

func TestServiceCreateJobZipAllocatesPodBeforePersistingExtracting(t *testing.T) {
	t.Parallel()

	var operationLog []string

	store := &fakeJobStore{
		createJobIfAbsentCreated: true,
		operationLog:             &operationLog,
	}
	runtime := &fakeRuntime{operationLog: &operationLog}

	service := NewService(store, runtime, &fakeArtifacts{}, &fakePublisher{})
	payload := &events.JobCreatedPayload{
		JobID:     "job-zip",
		InputType: string(models.JobInputTypeZip),
		InputPath: "staging/job-zip/site.zip",
		Config: models.JobConfig{
			Modules: []string{"axe"},
		},
	}

	if err := service.CreateJob(t.Context(), payload); err != nil {
		t.Fatalf("CreateJob() error = %v", err)
	}

	assertOperationOrder(
		t,
		operationLog,
		"runtime.CreateJobPod",
		"store.UpdateJobPodID",
		"store.UpdateJobState:EXTRACTING",
		"store.RecordExtractionStart",
		"runtime.StartExtractionWorker",
	)
}

func TestServiceCreateJobURLFailsWhenLoopbackTargetsNeedHostNetworking(t *testing.T) {
	t.Parallel()

	store := &fakeJobStore{
		createJobIfAbsentCreated: true,
		getJobResults: []*models.Job{
			{
				ID:        "job-url-loopback",
				State:     models.JobStatePending,
				InputType: models.JobInputTypeURLs,
				URLs:      []string{"http://127.0.0.1:3000"},
			},
		},
	}
	runtime := &fakeRuntime{}
	publisher := &fakePublisher{}

	service := NewService(store, runtime, &fakeArtifacts{}, publisher)
	payload := &events.JobCreatedPayload{
		JobID:     "job-url-loopback",
		InputType: string(models.JobInputTypeURLs),
		URLs:      []string{"http://127.0.0.1:3000"},
		Config: models.JobConfig{
			Modules: []string{"axe"},
		},
	}

	err := service.CreateJob(t.Context(), payload)
	if err == nil {
		t.Fatal("CreateJob() error = nil, want non-nil")
	}

	if store.failJobCalls != 1 {
		t.Fatalf("FailJob() calls = %d, want 1", store.failJobCalls)
	}

	if publisher.failedCalls != 1 {
		t.Fatalf("PublishJobFailed() calls = %d, want 1", publisher.failedCalls)
	}

	if runtime.createJobPodCalls != 0 {
		t.Fatalf("CreateJobPod() calls = %d, want 0", runtime.createJobPodCalls)
	}
}

func TestServiceHandleScanPageCompletedPersistsProgress(t *testing.T) {
	t.Parallel()

	store := &fakeJobStore{
		getJobResults: []*models.Job{
			{
				ID:    "job-page",
				State: models.JobStateScanning,
			},
		},
	}

	service := NewService(store, &fakeRuntime{}, &fakeArtifacts{}, &fakePublisher{})
	payload := &events.ScanPageCompletedPayload{
		JobID:       "job-page",
		ScannerType: "axe",
		PageID:      "page-2",
		PageIndex:   2,
		TotalPages:  5,
	}

	if err := service.HandleScanPageCompleted(t.Context(), payload); err != nil {
		t.Fatalf("HandleScanPageCompleted() error = %v", err)
	}

	if store.updateJobProgressCalls != 1 {
		t.Fatalf("UpdateJobProgress() calls = %d, want 1", store.updateJobProgressCalls)
	}
}

func TestServiceRunURLJobReusesExistingPod(t *testing.T) {
	t.Parallel()

	store := &fakeJobStore{}
	runtime := &fakeRuntime{resolvedScannerTypes: []string{"axe"}}

	service := NewService(store, runtime, &fakeArtifacts{}, &fakePublisher{})
	job := &models.Job{
		ID:        "job-url-existing-pod",
		State:     models.JobStatePending,
		InputType: models.JobInputTypeURLs,
		URLs:      []string{"https://example.com"},
		PodID:     "pod-existing",
		Config: models.JobConfig{
			Modules: []string{"axe"},
		},
	}

	if err := service.RunURLJob(t.Context(), job); err != nil {
		t.Fatalf("RunURLJob() error = %v", err)
	}

	if runtime.createJobPodCalls != 0 {
		t.Fatalf("CreateJobPod() calls = %d, want 0", runtime.createJobPodCalls)
	}

	if store.updateJobPodIDCalls != 0 {
		t.Fatalf("UpdateJobPodID() calls = %d, want 0", store.updateJobPodIDCalls)
	}

	if runtime.startScannerCalls != 1 {
		t.Fatalf("StartScanner() calls = %d, want 1", runtime.startScannerCalls)
	}
}

func TestServiceRunURLJobIgnoresTerminalStates(t *testing.T) {
	t.Parallel()

	for _, state := range []models.JobState{models.JobStateDone, models.JobStateFailed} {
		t.Run(string(state), func(t *testing.T) {
			t.Parallel()

			store := &fakeJobStore{}
			runtime := &fakeRuntime{resolvedScannerTypes: []string{"axe"}}

			service := NewService(store, runtime, &fakeArtifacts{}, &fakePublisher{})
			job := &models.Job{
				ID:        "job-terminal-" + string(state),
				State:     state,
				InputType: models.JobInputTypeURLs,
				URLs:      []string{"https://example.com"},
				Config: models.JobConfig{
					Modules: []string{"axe"},
				},
			}

			if err := service.RunURLJob(t.Context(), job); err != nil {
				t.Fatalf("RunURLJob() error = %v", err)
			}

			if runtime.createJobPodCalls != 0 {
				t.Fatalf("CreateJobPod() calls = %d, want 0", runtime.createJobPodCalls)
			}

			if runtime.startScannerCalls != 0 {
				t.Fatalf("StartScanner() calls = %d, want 0", runtime.startScannerCalls)
			}
		})
	}
}

func TestServiceStartScanningRejectsInvalidTransition(t *testing.T) {
	t.Parallel()

	store := &fakeJobStore{}
	runtime := &fakeRuntime{resolvedScannerTypes: []string{"axe"}}

	service := NewService(store, runtime, &fakeArtifacts{}, &fakePublisher{})
	job := &models.Job{
		ID:        "job-invalid-scan",
		State:     models.JobStateDone,
		InputType: models.JobInputTypeZip,
		InputPath: "staging/job-invalid-scan/site.zip",
		Config: models.JobConfig{
			Modules: []string{"axe"},
		},
	}

	err := service.StartScanning(t.Context(), job)
	if err == nil {
		t.Fatal("StartScanning() error = nil, want non-nil")
	}

	if store.setExpectedScannersCalls != 0 {
		t.Fatalf("SetExpectedScanners() calls = %d, want 0", store.setExpectedScannersCalls)
	}

	if runtime.startScannerCalls != 0 {
		t.Fatalf("StartScanner() calls = %d, want 0", runtime.startScannerCalls)
	}
}

// Pod setup used to be the one failure in this path that returned bare, so a job
// whose pod could never be created was left in PENDING with no terminal state and
// no cleanup. These pin both halves of the replacement: keep retrying while the
// delivery budget lasts, then record the failure on the last attempt.

func newPodCreateFailureService(
	t *testing.T,
	jobID string,
) (*Service, *fakeJobStore, *fakeRuntime, *fakePublisher) {
	t.Helper()

	store := &fakeJobStore{
		createJobIfAbsentCreated: true,
		// FailJob re-reads the job before recording the terminal state.
		getJobResults: []*models.Job{{ID: jobID, State: models.JobStatePending}},
	}
	runtime := &fakeRuntime{createJobPodErr: errors.New("podman: timeout awaiting response headers")}
	publisher := &fakePublisher{}

	return NewService(store, runtime, &fakeArtifacts{}, publisher), store, runtime, publisher
}

func zipJobCreatedPayload(jobID string) *events.JobCreatedPayload {
	return &events.JobCreatedPayload{
		JobID:     jobID,
		InputType: string(models.JobInputTypeZip),
		InputPath: "staging/" + jobID + "/site.zip",
		Config:    models.JobConfig{Modules: []string{"axe"}},
	}
}

func TestPodCreateFailureRetriesWhileDeliveryBudgetRemains(t *testing.T) {
	t.Parallel()

	service, store, runtime, publisher := newPodCreateFailureService(t, "job-retry")

	// No delivery metadata at all: the conservative default, and what
	// reconciliation sweeps and direct calls look like.
	err := service.CreateJob(t.Context(), zipJobCreatedPayload("job-retry"))
	if err == nil {
		t.Fatal("expected the pod failure to be returned so the message is redelivered")
	}

	if store.failJobCalls != 0 {
		t.Fatalf("failJobCalls = %d, want 0: the job must keep its remaining attempts", store.failJobCalls)
	}

	if publisher.failedCalls != 0 {
		t.Fatalf("failedCalls = %d, want 0", publisher.failedCalls)
	}

	if runtime.cleanupJobCalls != 0 {
		t.Fatalf("cleanupJobCalls = %d, want 0 before the budget is spent", runtime.cleanupJobCalls)
	}
}

func TestPodCreateFailureFailsTheJobOnTheFinalDelivery(t *testing.T) {
	t.Parallel()

	service, store, runtime, publisher := newPodCreateFailureService(t, "job-final")

	ctx := messaging.WithReceivedEventMeta(
		t.Context(),
		&messaging.ReceivedEventMeta{Deliveries: messaging.MaxDeliver},
	)

	if err := service.CreateJob(ctx, zipJobCreatedPayload("job-final")); err == nil {
		t.Fatal("expected the setup error to be reported")
	}

	if store.failJobCalls != 1 {
		t.Fatalf("failJobCalls = %d, want 1: the last attempt must record a terminal state", store.failJobCalls)
	}

	if publisher.failedCalls != 1 {
		t.Fatalf("failedCalls = %d, want 1: the client needs to see the job stop", publisher.failedCalls)
	}

	// Cleanup must run even though no pod ID was ever recorded -- Podman can create
	// the pod and lose the response, and nothing else would remove it. The count is
	// deliberately not pinned: failExtractionSetup cleans up, and FailJob cleans up
	// again after transitioning, which is intentional belt-and-braces because
	// FailJob returns early for an already-terminal job before reaching its own
	// cleanup.
	if runtime.cleanupJobCalls < 1 {
		t.Fatal("expected cleanup to run so a pod created behind a lost response is removed")
	}
}

func TestPodCreateFailureFailsURLJobsOnTheFinalDelivery(t *testing.T) {
	t.Parallel()

	service, store, runtime, _ := newPodCreateFailureService(t, "job-url-final")

	ctx := messaging.WithReceivedEventMeta(
		t.Context(),
		&messaging.ReceivedEventMeta{Deliveries: messaging.MaxDeliver},
	)

	payload := &events.JobCreatedPayload{
		JobID:     "job-url-final",
		InputType: "urls",
		URLs:      []string{"https://example.com"},
		Config:    models.JobConfig{Modules: []string{"axe"}},
	}

	if err := service.CreateJob(ctx, payload); err == nil {
		t.Fatal("expected the setup error to be reported")
	}

	if store.failJobCalls != 1 {
		t.Fatalf("failJobCalls = %d, want 1 for the URL path too", store.failJobCalls)
	}

	if runtime.cleanupJobCalls < 1 {
		t.Fatal("expected cleanup to run for the URL path too")
	}
}
