package jobs

import (
	"errors"
	"testing"

	"github.com/mattboback/stageflow/packages/shared-go/events"
	"github.com/mattboback/stageflow/packages/shared-go/models"
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

	service := NewService(store, runtime, &fakeArtifacts{}, &fakePublisher{}, ScannerLaunchPlannerConfig{})
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

	service := NewService(store, &fakeRuntime{}, &fakeArtifacts{}, &fakePublisher{}, ScannerLaunchPlannerConfig{})
	payload := &events.ScanPageCompletedPayload{
		JobID:      "job-page",
		PageIndex:  2,
		TotalPages: 5,
	}

	if err := service.HandleScanPageCompleted(t.Context(), payload); err != nil {
		t.Fatalf("HandleScanPageCompleted() error = %v", err)
	}

	if store.updateJobProgressCalls != 1 {
		t.Fatalf("UpdateJobProgress() calls = %d, want 1", store.updateJobProgressCalls)
	}
}
