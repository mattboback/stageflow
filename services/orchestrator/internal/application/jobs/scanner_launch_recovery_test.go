package jobs

import (
	"testing"
	"time"

	"github.com/mattboback/stageflow/libs/go/models"
)

func TestReconcileScanningJobsRestartsIncompleteLaunch(t *testing.T) {
	t.Parallel()

	store := &fakeJobStore{
		listJobsByStateResults: []*models.Job{
			{
				ID:               "job-restart",
				State:            models.JobStateScanning,
				InputType:        models.JobInputTypeURLs,
				URLs:             []string{"https://example.com"},
				PodID:            "pod-restart",
				Config:           models.JobConfig{Modules: []string{"axe"}},
				ExpectedScanners: []string{"axe"},
				CreatedAt:        time.Now(),
				UpdatedAt:        time.Now(),
			},
		},
	}
	runtime := &fakeRuntime{resolvedScannerTypes: []string{"axe"}}
	service := NewService(store, runtime, &fakeArtifacts{}, &fakePublisher{})

	if err := service.ReconcileScanningJobs(t.Context()); err != nil {
		t.Fatalf("ReconcileScanningJobs() error = %v", err)
	}

	if store.prepareScannerLaunchesCalls != 1 {
		t.Fatalf("PrepareScannerLaunches() calls = %d, want 1", store.prepareScannerLaunchesCalls)
	}

	if store.claimScannerRecoveryCalls != 1 {
		t.Fatalf("ClaimScannerLaunchRecovery() calls = %d, want 1", store.claimScannerRecoveryCalls)
	}

	if runtime.startScannerCalls != 1 {
		t.Fatalf("StartScanner() calls = %d, want 1", runtime.startScannerCalls)
	}

	if store.markScannerLaunchedCalls != 1 {
		t.Fatalf("MarkScannerLaunched() calls = %d, want 1", store.markScannerLaunchedCalls)
	}
}

func TestReconcileScanningJobsSkipsCompletedScanner(t *testing.T) {
	t.Parallel()

	store := &fakeJobStore{
		listJobsByStateResults: []*models.Job{
			{
				ID:                "job-completed-scanner",
				State:             models.JobStateScanning,
				PodID:             "pod-completed-scanner",
				Config:            models.JobConfig{Modules: []string{"axe"}},
				ExpectedScanners:  []string{"axe"},
				CompletedScanners: []string{"axe"},
			},
		},
	}
	runtime := &fakeRuntime{resolvedScannerTypes: []string{"axe"}}
	service := NewService(store, runtime, &fakeArtifacts{}, &fakePublisher{})

	if err := service.ReconcileScanningJobs(t.Context()); err != nil {
		t.Fatalf("ReconcileScanningJobs() error = %v", err)
	}

	if store.claimScannerRecoveryCalls != 0 {
		t.Fatalf("ClaimScannerLaunchRecovery() calls = %d, want 0", store.claimScannerRecoveryCalls)
	}

	if runtime.startScannerCalls != 0 {
		t.Fatalf("StartScanner() calls = %d, want 0", runtime.startScannerCalls)
	}
}

func TestStartScanningDuplicateDoesNotRelaunchClaimedScanner(t *testing.T) {
	t.Parallel()

	claim := false
	store := &fakeJobStore{claimScannerLaunchResult: &claim}
	runtime := &fakeRuntime{resolvedScannerTypes: []string{"axe"}}
	service := NewService(store, runtime, &fakeArtifacts{}, &fakePublisher{})
	job := &models.Job{
		ID:               "job-redelivery",
		State:            models.JobStateScanning,
		InputType:        models.JobInputTypeURLs,
		URLs:             []string{"https://example.com"},
		PodID:            "pod-redelivery",
		Config:           models.JobConfig{Modules: []string{"axe"}},
		ExpectedScanners: []string{"axe"},
	}

	if err := service.StartScanning(t.Context(), job); err != nil {
		t.Fatalf("StartScanning() duplicate error = %v", err)
	}

	if store.claimScannerLaunchCalls != 1 {
		t.Fatalf("ClaimScannerLaunch() calls = %d, want 1", store.claimScannerLaunchCalls)
	}

	if runtime.startScannerCalls != 0 {
		t.Fatalf("StartScanner() calls = %d, want 0", runtime.startScannerCalls)
	}

	if store.markScannerLaunchedCalls != 0 {
		t.Fatalf("MarkScannerLaunched() calls = %d, want 0", store.markScannerLaunchedCalls)
	}
}
