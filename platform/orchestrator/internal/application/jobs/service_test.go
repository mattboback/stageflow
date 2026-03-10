package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/mattboback/stageflow/packages/shared-go/events"
	"github.com/mattboback/stageflow/packages/shared-go/models"
)

func TestServicePrepareExtractedJobStartsScanning(t *testing.T) {
	t.Parallel()

	store := &fakeJobStore{
		getJobResults: []*models.Job{
			{
				ID:        "job-123",
				State:     models.JobStateExtracting,
				InputType: models.JobInputTypeZip,
				InputPath: "staging/job-123/site.zip",
				Config: models.JobConfig{
					Modules: []string{"axe"},
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}
	runtime := &fakeRuntime{resolvedScannerTypes: []string{"axe"}}

	service := NewService(store, runtime, &fakeArtifacts{}, &fakePublisher{})
	payload := &events.ExtractionReadyPayload{
		JobID:                  "job-123",
		StageLogPath:           "job-123/extraction/stage.log",
		RecipePath:             "job-123/extraction/recipe.json",
		ProvenancePath:         "/workspace/provenance.json",
		ProvenanceArtifactPath: "job-123/provenance.json",
		TotalPages:             3,
	}

	if err := service.PrepareExtractedJob(t.Context(), payload); err != nil {
		t.Fatalf("PrepareExtractedJob() error = %v", err)
	}

	if store.recordExtractionCompleteCalls != 1 {
		t.Fatalf("RecordExtractionComplete() calls = %d, want 1", store.recordExtractionCompleteCalls)
	}

	if store.updateJobStateCalls == 0 {
		t.Fatal("expected job state transition to READY and SCANNING")
	}

	if runtime.startScannerCalls != 1 {
		t.Fatalf("start scanner calls = %d, want 1", runtime.startScannerCalls)
	}
}

func TestServiceRecordScannerFailureCompletesWithPartialResults(t *testing.T) {
	t.Parallel()

	store := &fakeJobStore{
		getJobResults: []*models.Job{
			{
				ID:                "job-fail",
				State:             models.JobStateScanning,
				ExpectedScanners:  []string{"axe", "lighthouse"},
				CompletedScanners: []string{"axe"},
				ScannerResults: map[string]*models.ScannerResult{
					"axe": {
						ScannerType: "axe",
						Success:     true,
						ResultsPath: "job-fail/axe/results.json",
						ReportPath:  "job-fail/axe/report.html",
					},
				},
			},
			{
				ID:                "job-fail",
				State:             models.JobStateScanning,
				ExpectedScanners:  []string{"axe", "lighthouse"},
				CompletedScanners: []string{"axe", "lighthouse"},
				ScannerResults: map[string]*models.ScannerResult{
					"axe": {
						ScannerType: "axe",
						Success:     true,
						ResultsPath: "job-fail/axe/results.json",
						ReportPath:  "job-fail/axe/report.html",
					},
					"lighthouse": {
						ScannerType: "lighthouse",
						Success:     false,
					},
				},
			},
		},
		recordScannerFailureAllComplete: true,
	}
	artifacts := &fakeArtifacts{reportPath: "job-fail/report.json"}
	publisher := &fakePublisher{}

	service := NewService(store, &fakeRuntime{}, artifacts, publisher)
	payload := &events.ScanFailedPayload{
		JobID:       "job-fail",
		ScannerType: "lighthouse",
		Error:       "browser crashed",
	}

	if err := service.RecordScannerFailure(t.Context(), payload); err != nil {
		t.Fatalf("RecordScannerFailure() error = %v", err)
	}

	if store.completeJobCalls != 1 {
		t.Fatalf("CompleteJob() calls = %d, want 1", store.completeJobCalls)
	}

	if publisher.completedCalls != 1 {
		t.Fatalf("PublishJobCompleted() calls = %d, want 1", publisher.completedCalls)
	}
}

func TestServiceRecordScannerCompletionWaitsForRemainingScanners(t *testing.T) {
	t.Parallel()

	store := &fakeJobStore{
		getJobResults: []*models.Job{
			{
				ID:                "job-scan",
				State:             models.JobStateScanning,
				ExpectedScanners:  []string{"axe", "lighthouse"},
				CompletedScanners: []string{},
				ScannerResults:    map[string]*models.ScannerResult{},
			},
		},
		recordScannerCompletionAllComplete: false,
	}

	service := NewService(store, &fakeRuntime{}, &fakeArtifacts{}, &fakePublisher{})
	payload := &events.ScanCompletedPayload{
		JobID:             "job-scan",
		ScannerType:       "axe",
		ResultsPath:       "job-scan/axe/results.json",
		ReportPath:        "job-scan/axe/report.html",
		TotalPagesScanned: 2,
		Summary: events.ScanSummary{
			TotalViolations: 4,
		},
	}

	if err := service.RecordScannerCompletion(t.Context(), payload); err != nil {
		t.Fatalf("RecordScannerCompletion() error = %v", err)
	}

	if store.recordScannerCompletionCalls != 1 {
		t.Fatalf("RecordScannerCompletion() calls = %d, want 1", store.recordScannerCompletionCalls)
	}

	if store.completeJobCalls != 0 {
		t.Fatalf("CompleteJob() calls = %d, want 0", store.completeJobCalls)
	}
}

type fakeJobStore struct {
	createJobIfAbsentCreated           bool
	createJobIfAbsentCalls             int
	getJobResults                      []*models.Job
	getJobCalls                        int
	recordExtractionCompleteCalls      int
	recordExtractionStartCalls         int
	recordScanStartCalls               int
	recordScanCompleteCalls            int
	updateJobStateCalls                int
	updateJobProgressCalls             int
	updateJobExtractionArtifactsCalls  int
	updateJobProvenanceCalls           int
	updateJobProvenanceKeyCalls        int
	updateJobCompletionArtifactsCalls  int
	updateJobMetricsCalls              int
	setExpectedScannersCalls           int
	recordScannerCompletionCalls       int
	recordScannerCompletionAllComplete bool
	recordScannerFailureCalls          int
	recordScannerFailureAllComplete    bool
	completeJobCalls                   int
	failJobCalls                       int
	recordInternalEventCalls           int
	lastExpectedScanners               []string
	lastStateUpdates                   []models.JobState
}

func (f *fakeJobStore) CreateJobIfAbsent(_ context.Context, _ *models.Job) (bool, error) {
	f.createJobIfAbsentCalls++
	return f.createJobIfAbsentCreated, nil
}

func (f *fakeJobStore) GetJob(_ context.Context, _ string) (*models.Job, error) {
	idx := f.getJobCalls
	if idx >= len(f.getJobResults) {
		idx = len(f.getJobResults) - 1
	}

	f.getJobCalls++

	if idx < 0 {
		return nil, errFakeLifecycleJobNotSeeded
	}

	return f.getJobResults[idx], nil
}

func (f *fakeJobStore) UpdateJobState(_ context.Context, _ string, state models.JobState) error {
	f.updateJobStateCalls++
	f.lastStateUpdates = append(f.lastStateUpdates, state)

	return nil
}

func (f *fakeJobStore) RecordExtractionComplete(_ context.Context, _ string) error {
	f.recordExtractionCompleteCalls++
	return nil
}

func (f *fakeJobStore) RecordExtractionStart(_ context.Context, _ string) error {
	f.recordExtractionStartCalls++
	return nil
}

func (f *fakeJobStore) RecordScanStart(_ context.Context, _ string) error {
	f.recordScanStartCalls++
	return nil
}

func (f *fakeJobStore) RecordScanComplete(_ context.Context, _ string) error {
	f.recordScanCompleteCalls++
	return nil
}

func (f *fakeJobStore) UpdateJobProgress(_ context.Context, _ string, _, _ int) error {
	f.updateJobProgressCalls++
	return nil
}

func (f *fakeJobStore) UpdateJobExtractionArtifacts(_ context.Context, _, _, _ string) error {
	f.updateJobExtractionArtifactsCalls++
	return nil
}

func (f *fakeJobStore) UpdateJobProvenance(_ context.Context, _, _ string) error {
	f.updateJobProvenanceCalls++
	return nil
}

func (f *fakeJobStore) UpdateJobProvenanceKey(_ context.Context, _, _ string) error {
	f.updateJobProvenanceKeyCalls++
	return nil
}

func (f *fakeJobStore) UpdateJobCompletionArtifacts(
	_ context.Context,
	_,
	_,
	_,
	_,
	_ string,
	_ int,
) error {
	f.updateJobCompletionArtifactsCalls++
	return nil
}

func (f *fakeJobStore) UpdateJobMetrics(
	_ context.Context,
	_ string,
	_, _, _, _, _, _ int,
) error {
	f.updateJobMetricsCalls++
	return nil
}

func (f *fakeJobStore) SetExpectedScanners(_ context.Context, _ string, scanners []string) error {
	f.setExpectedScannersCalls++

	f.lastExpectedScanners = append([]string{}, scanners...)

	return nil
}

func (f *fakeJobStore) RecordScannerCompletion(_ context.Context, _ string, _ *models.ScannerResult) (bool, error) {
	f.recordScannerCompletionCalls++
	return f.recordScannerCompletionAllComplete, nil
}

func (f *fakeJobStore) RecordScannerFailure(_ context.Context, _, _, _ string) (bool, error) {
	f.recordScannerFailureCalls++
	return f.recordScannerFailureAllComplete, nil
}

func (f *fakeJobStore) CompleteJob(_ context.Context, _ string) error {
	f.completeJobCalls++
	return nil
}

func (f *fakeJobStore) FailJob(_ context.Context, _, _, _, _ string) error {
	f.failJobCalls++
	return nil
}

func (f *fakeJobStore) RecordInternalEvent(_ context.Context, _ string, _ string, _ any) error {
	f.recordInternalEventCalls++
	return nil
}

type fakeRuntime struct {
	resolvedScannerTypes []string
	prepareURLJobCalls   int
	startExtractionCalls int
	startScannerCalls    int
	cleanupJobCalls      int
}

func (f *fakeRuntime) PrepareURLJob(_ context.Context, _ *models.Job) error {
	f.prepareURLJobCalls++
	return nil
}

func (f *fakeRuntime) StartExtraction(_ context.Context, _ *models.Job) error {
	f.startExtractionCalls++
	return nil
}

func (f *fakeRuntime) ResolveScannerTypes(_ []string) []string {
	return append([]string{}, f.resolvedScannerTypes...)
}

func (f *fakeRuntime) StartScanner(_ context.Context, _ *models.Job, _ *ScannerLaunchPlan) error {
	f.startScannerCalls++
	return nil
}

func (f *fakeRuntime) CleanupJob(_ context.Context, _ *models.Job) error {
	f.cleanupJobCalls++
	return nil
}

type fakeArtifacts struct {
	reportPath string
}

func (f *fakeArtifacts) BuildAggregatedReport(_ context.Context, _ *models.Job) (string, error) {
	return f.reportPath, nil
}

type fakePublisher struct {
	completedCalls int
	failedCalls    int
}

func (f *fakePublisher) PublishJobCompleted(_ context.Context, _ *events.JobCompletedPayload) error {
	f.completedCalls++
	return nil
}

func (f *fakePublisher) PublishJobFailed(_ context.Context, _ *events.JobFailedPayload) error {
	f.failedCalls++
	return nil
}
