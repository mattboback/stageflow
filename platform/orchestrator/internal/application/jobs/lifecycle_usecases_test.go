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

	if runtime.startExtractionCalls != 1 {
		t.Fatalf("StartExtraction() calls = %d, want 1", runtime.startExtractionCalls)
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
