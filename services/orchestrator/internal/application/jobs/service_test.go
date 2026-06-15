package jobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mattboback/stageflow/libs/go/events"
	sharedmsg "github.com/mattboback/stageflow/libs/go/messaging"
	"github.com/mattboback/stageflow/libs/go/models"
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
		BaseURL:                "http://127.0.0.1:8080",
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
		claimJobCompletionResult:        true,
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

func TestServiceRecordScannerCompletionDoesNotCompleteWithoutOwnership(t *testing.T) {
	t.Parallel()

	store := &fakeJobStore{
		getJobResults: []*models.Job{
			{
				ID:               "job-race",
				State:            models.JobStateScanning,
				ExpectedScanners: []string{"axe"},
				ScannerResults: map[string]*models.ScannerResult{
					"axe": {
						ScannerType: "axe",
						Success:     true,
						ResultsPath: "job-race/axe/results.json",
						ReportPath:  "job-race/axe/report.html",
					},
				},
			},
			{
				ID:               "job-race",
				State:            models.JobStateScanning,
				ExpectedScanners: []string{"axe"},
				ScannerResults: map[string]*models.ScannerResult{
					"axe": {
						ScannerType: "axe",
						Success:     true,
						ResultsPath: "job-race/axe/results.json",
						ReportPath:  "job-race/axe/report.html",
					},
				},
			},
		},
		recordScannerCompletionAllComplete: true,
		claimJobCompletionResult:           false,
	}
	publisher := &fakePublisher{}

	service := NewService(store, &fakeRuntime{}, &fakeArtifacts{reportPath: "job-race/report.json"}, publisher)
	payload := &events.ScanCompletedPayload{
		JobID:             "job-race",
		ScannerType:       "axe",
		ResultsPath:       "job-race/axe/results.json",
		ReportPath:        "job-race/axe/report.html",
		TotalPagesScanned: 1,
		Summary:           events.ScanSummary{TotalViolations: 0, BySeverity: map[string]int{}},
	}

	if err := service.RecordScannerCompletion(t.Context(), payload); err != nil {
		t.Fatalf("RecordScannerCompletion() error = %v", err)
	}

	if store.claimJobCompletionCalls != 1 {
		t.Fatalf("ClaimJobCompletion() calls = %d, want 1", store.claimJobCompletionCalls)
	}

	if store.completeJobCalls != 0 {
		t.Fatalf("CompleteJobWithTerminalEvent() calls = %d, want 0", store.completeJobCalls)
	}

	if publisher.completedCalls != 0 {
		t.Fatalf("PublishJobCompleted() calls = %d, want 0", publisher.completedCalls)
	}
}

func TestServiceCompleteJobCompletingPublishesPendingTerminalEvent(t *testing.T) {
	t.Parallel()

	store := &fakeJobStore{
		pendingTerminalEvents: []TerminalEvent{
			{
				Event: events.EventJobCompleted,
				JobCompleted: &events.JobCompletedPayload{
					JobID:  "job-completing",
					Status: events.JobStatusSuccess,
				},
			},
		},
	}
	artifacts := &fakeArtifacts{reportPath: "job-completing/report.json"}
	publisher := &fakePublisher{}
	service := NewService(store, &fakeRuntime{}, artifacts, publisher)

	job := &models.Job{
		ID:    "job-completing",
		State: models.JobStateCompleting,
	}

	if err := service.CompleteJob(t.Context(), job); err != nil {
		t.Fatalf("CompleteJob() error = %v", err)
	}

	if store.listUnpublishedTerminalEventsCalls != 1 {
		t.Fatalf("ListUnpublishedTerminalEvents() calls = %d, want 1", store.listUnpublishedTerminalEventsCalls)
	}

	if publisher.completedCalls != 1 {
		t.Fatalf("PublishJobCompleted() calls = %d, want 1", publisher.completedCalls)
	}

	if store.markTerminalEventPublishedCalls != 1 {
		t.Fatalf("MarkTerminalEventPublished() calls = %d, want 1", store.markTerminalEventPublishedCalls)
	}

	if artifacts.buildAggregatedReportCalls != 0 {
		t.Fatalf("BuildAggregatedReport() calls = %d, want 0", artifacts.buildAggregatedReportCalls)
	}

	if store.completeJobCalls != 0 {
		t.Fatalf("CompleteJobWithTerminalEvent() calls = %d, want 0", store.completeJobCalls)
	}
}

func TestServiceCompleteJobCompletingRedeliveryResumesWithoutPendingTerminalEvent(t *testing.T) {
	t.Parallel()

	store := &fakeJobStore{}
	artifacts := &fakeArtifacts{reportPath: "job-redelivery/report.json"}
	publisher := &fakePublisher{}
	service := NewService(store, &fakeRuntime{}, artifacts, publisher)
	ctx := sharedmsg.WithReceivedEventMeta(t.Context(), &sharedmsg.ReceivedEventMeta{Deliveries: 2})

	job := &models.Job{
		ID:    "job-redelivery",
		State: models.JobStateCompleting,
		ScannerResults: map[string]*models.ScannerResult{
			"axe": {
				ScannerType: "axe",
				Success:     true,
				ReportPath:  "job-redelivery/axe/report.html",
			},
		},
	}

	if err := service.CompleteJob(ctx, job); err != nil {
		t.Fatalf("CompleteJob() error = %v", err)
	}

	if store.claimJobCompletionCalls != 0 {
		t.Fatalf("ClaimJobCompletion() calls = %d, want 0", store.claimJobCompletionCalls)
	}

	if artifacts.buildAggregatedReportCalls != 1 {
		t.Fatalf("BuildAggregatedReport() calls = %d, want 1", artifacts.buildAggregatedReportCalls)
	}

	if store.completeJobCalls != 1 {
		t.Fatalf("CompleteJobWithTerminalEvent() calls = %d, want 1", store.completeJobCalls)
	}

	if publisher.completedCalls != 1 {
		t.Fatalf("PublishJobCompleted() calls = %d, want 1", publisher.completedCalls)
	}

	if store.markTerminalEventPublishedCalls != 1 {
		t.Fatalf("MarkTerminalEventPublished() calls = %d, want 1", store.markTerminalEventPublishedCalls)
	}
}

func TestServiceCompleteJobCompletingFirstDeliveryWithoutPendingTerminalEventNoops(t *testing.T) {
	t.Parallel()

	store := &fakeJobStore{}
	artifacts := &fakeArtifacts{reportPath: "job-first-delivery/report.json"}
	publisher := &fakePublisher{}
	service := NewService(store, &fakeRuntime{}, artifacts, publisher)
	ctx := sharedmsg.WithReceivedEventMeta(t.Context(), &sharedmsg.ReceivedEventMeta{Deliveries: 1})

	job := &models.Job{
		ID:    "job-first-delivery",
		State: models.JobStateCompleting,
	}

	if err := service.CompleteJob(ctx, job); err != nil {
		t.Fatalf("CompleteJob() error = %v", err)
	}

	if store.listUnpublishedTerminalEventsCalls != 1 {
		t.Fatalf("ListUnpublishedTerminalEvents() calls = %d, want 1", store.listUnpublishedTerminalEventsCalls)
	}

	if store.claimJobCompletionCalls != 0 {
		t.Fatalf("ClaimJobCompletion() calls = %d, want 0", store.claimJobCompletionCalls)
	}

	if artifacts.buildAggregatedReportCalls != 0 {
		t.Fatalf("BuildAggregatedReport() calls = %d, want 0", artifacts.buildAggregatedReportCalls)
	}

	if store.completeJobCalls != 0 {
		t.Fatalf("CompleteJobWithTerminalEvent() calls = %d, want 0", store.completeJobCalls)
	}

	if publisher.completedCalls != 0 {
		t.Fatalf("PublishJobCompleted() calls = %d, want 0", publisher.completedCalls)
	}
}

func TestServicePublishesPendingTerminalEventForDuplicateTerminalScanEvent(t *testing.T) {
	t.Parallel()

	store := &fakeJobStore{
		getJobResults: []*models.Job{
			{
				ID:    "job-terminal",
				State: models.JobStateDone,
			},
		},
		pendingTerminalEvents: []TerminalEvent{
			{
				Event: events.EventJobCompleted,
				JobCompleted: &events.JobCompletedPayload{
					JobID:  "job-terminal",
					Status: events.JobStatusSuccess,
				},
			},
		},
	}
	publisher := &fakePublisher{}

	service := NewService(store, &fakeRuntime{}, &fakeArtifacts{}, publisher)
	payload := &events.ScanCompletedPayload{
		JobID:             "job-terminal",
		ScannerType:       "axe",
		ResultsPath:       "job-terminal/axe/results.json",
		ReportPath:        "job-terminal/axe/report.html",
		TotalPagesScanned: 1,
		Summary:           events.ScanSummary{TotalViolations: 0, BySeverity: map[string]int{}},
	}

	if err := service.RecordScannerCompletion(t.Context(), payload); err != nil {
		t.Fatalf("RecordScannerCompletion() error = %v", err)
	}

	if publisher.completedCalls != 1 {
		t.Fatalf("PublishJobCompleted() calls = %d, want 1", publisher.completedCalls)
	}

	if store.markTerminalEventPublishedCalls != 1 {
		t.Fatalf("MarkTerminalEventPublished() calls = %d, want 1", store.markTerminalEventPublishedCalls)
	}
}

func TestServiceStartScanningFailsJobWhenScannerLaunchFails(t *testing.T) {
	t.Parallel()

	store := &fakeJobStore{
		getJobResults: []*models.Job{
			{
				ID:        "job-launch-fail",
				State:     models.JobStateScanning,
				InputType: models.JobInputTypeURLs,
				URLs:      []string{"https://example.com"},
				Config: models.JobConfig{
					Modules: []string{"axe"},
				},
			},
		},
	}
	runtime := &fakeRuntime{
		resolvedScannerTypes: []string{"axe"},
		startScannerErr:      errors.New("image not known"),
	}
	publisher := &fakePublisher{}

	service := NewService(store, runtime, &fakeArtifacts{}, publisher)
	job := &models.Job{
		ID:        "job-launch-fail",
		State:     models.JobStateReady,
		InputType: models.JobInputTypeURLs,
		URLs:      []string{"https://example.com"},
		Config: models.JobConfig{
			Modules: []string{"axe"},
		},
	}

	err := service.StartScanning(t.Context(), job)
	if err == nil {
		t.Fatal("StartScanning() error = nil, want non-nil")
	}

	if store.failJobCalls != 1 {
		t.Fatalf("FailJob() calls = %d, want 1", store.failJobCalls)
	}

	if publisher.failedCalls != 1 {
		t.Fatalf("PublishJobFailed() calls = %d, want 1", publisher.failedCalls)
	}

	if runtime.cleanupJobCalls != 1 {
		t.Fatalf("CleanupJob() calls = %d, want 1", runtime.cleanupJobCalls)
	}

	if runtime.startScannerCalls != 1 {
		t.Fatalf("StartScanner() calls = %d, want 1", runtime.startScannerCalls)
	}
}

func TestServiceStartScanningCleansUpAfterPartialScannerLaunchFailure(t *testing.T) {
	t.Parallel()

	store := &fakeJobStore{
		getJobResults: []*models.Job{
			{
				ID:     "job-partial-launch-fail",
				State:  models.JobStateScanning,
				PodID:  "pod-partial",
				Config: models.JobConfig{Modules: []string{"axe", "lighthouse", "seo"}},
			},
		},
	}
	runtime := &fakeRuntime{
		resolvedScannerTypes:  []string{"axe", "lighthouse", "seo"},
		startScannerErr:       errors.New("lighthouse image not known"),
		startScannerErrOnCall: 2,
	}
	publisher := &fakePublisher{}

	service := NewService(store, runtime, &fakeArtifacts{}, publisher)
	job := &models.Job{
		ID:     "job-partial-launch-fail",
		State:  models.JobStateReady,
		PodID:  "pod-partial",
		Config: models.JobConfig{Modules: []string{"axe", "lighthouse", "seo"}},
	}

	err := service.StartScanning(t.Context(), job)
	if err == nil {
		t.Fatal("StartScanning() error = nil, want non-nil")
	}

	if runtime.startScannerCalls != 2 {
		t.Fatalf("StartScanner() calls = %d, want 2", runtime.startScannerCalls)
	}

	if store.failJobCalls != 1 {
		t.Fatalf("FailJob() calls = %d, want 1", store.failJobCalls)
	}

	if runtime.cleanupJobCalls != 1 {
		t.Fatalf("CleanupJob() calls = %d, want 1", runtime.cleanupJobCalls)
	}

	if publisher.failedCalls != 1 {
		t.Fatalf("PublishJobFailed() calls = %d, want 1", publisher.failedCalls)
	}
}

type fakeJobStore struct {
	createJobIfAbsentCreated           bool
	createJobIfAbsentCalls             int
	getJobResults                      []*models.Job
	getJobCalls                        int
	claimJobCompletionCalls            int
	claimJobCompletionResult           bool
	recordExtractionCompleteCalls      int
	recordExtractionStartCalls         int
	recordScanStartCalls               int
	recordScanCompleteCalls            int
	updateJobStateCalls                int
	updateJobProgressCalls             int
	updateJobExtractionArtifactsCalls  int
	updateJobProvenanceCalls           int
	updateJobProvenanceKeyCalls        int
	updateJobPodIDCalls                int
	updateJobCompletionArtifactsCalls  int
	updateJobMetricsCalls              int
	setExpectedScannersCalls           int
	recordScannerCompletionCalls       int
	recordScannerCompletionAllComplete bool
	recordScannerFailureCalls          int
	recordScannerFailureAllComplete    bool
	completeJobCalls                   int
	failJobCalls                       int
	listUnpublishedTerminalEventsCalls int
	markTerminalEventPublishedCalls    int
	pendingTerminalEvents              []TerminalEvent
	recordInternalEventCalls           int
	lastExpectedScanners               []string
	lastStateUpdates                   []models.JobState
	lastPodID                          string
	lastCreatedJob                     *models.Job
	operationLog                       *[]string
}

func (f *fakeJobStore) CreateJobIfAbsent(_ context.Context, job *models.Job) (bool, error) {
	f.createJobIfAbsentCalls++
	f.lastCreatedJob = job

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
	f.recordOperation("store.UpdateJobState:" + string(state))

	return nil
}

func (f *fakeJobStore) ClaimJobCompletion(_ context.Context, _ string) (bool, error) {
	f.claimJobCompletionCalls++
	f.recordOperation("store.ClaimJobCompletion")

	return f.claimJobCompletionResult, nil
}

func (f *fakeJobStore) RecordExtractionComplete(_ context.Context, _ string) error {
	f.recordExtractionCompleteCalls++
	return nil
}

func (f *fakeJobStore) RecordExtractionStart(_ context.Context, _ string) error {
	f.recordExtractionStartCalls++
	f.recordOperation("store.RecordExtractionStart")

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
	f.recordOperation("store.UpdateJobProvenanceKey")

	return nil
}

func (f *fakeJobStore) UpdateJobPodID(_ context.Context, _, podID string) error {
	f.updateJobPodIDCalls++
	f.lastPodID = podID
	f.recordOperation("store.UpdateJobPodID")

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

func (f *fakeJobStore) CompleteJobWithTerminalEvent(_ context.Context, _ string, _ *events.JobCompletedPayload) error {
	f.completeJobCalls++
	return nil
}

func (f *fakeJobStore) FailJobWithTerminalEvent(
	_ context.Context,
	_, _, _, _ string,
	_ *events.JobFailedPayload,
) error {
	f.failJobCalls++
	return nil
}

func (f *fakeJobStore) ListUnpublishedTerminalEvents(_ context.Context, _ string) ([]TerminalEvent, error) {
	f.listUnpublishedTerminalEventsCalls++
	return append([]TerminalEvent{}, f.pendingTerminalEvents...), nil
}

func (f *fakeJobStore) MarkTerminalEventPublished(_ context.Context, _, _ string) error {
	f.markTerminalEventPublishedCalls++
	return nil
}

func (f *fakeJobStore) RecordInternalEvent(_ context.Context, _ string, _ string, _ any) error {
	f.recordInternalEventCalls++
	return nil
}

func (f *fakeJobStore) recordOperation(op string) {
	if f.operationLog == nil {
		return
	}

	*f.operationLog = append(*f.operationLog, op)
}

type fakeRuntime struct {
	resolvedScannerTypes       []string
	allowLoopbackTargets       bool
	createJobPodID             string
	startScannerErr            error
	startScannerErrOnCall      int
	createJobPodCalls          int
	startExtractionWorkerCalls int
	startScannerCalls          int
	cleanupJobCalls            int
	operationLog               *[]string
}

func (f *fakeRuntime) AllowsLoopbackTargets() bool {
	return f.allowLoopbackTargets
}

func (f *fakeRuntime) PodNetnsMode() string {
	if f.allowLoopbackTargets {
		return PodNetnsModeHost
	}

	return PodNetnsModeBridge
}

func (f *fakeRuntime) CreateJobPod(_ context.Context, _ *models.Job) (string, error) {
	f.createJobPodCalls++
	f.recordOperation("runtime.CreateJobPod")

	if f.createJobPodID == "" {
		return "pod-123", nil
	}

	return f.createJobPodID, nil
}

func (f *fakeRuntime) StartExtractionWorker(_ context.Context, _ *models.Job) error {
	f.startExtractionWorkerCalls++
	f.recordOperation("runtime.StartExtractionWorker")

	return nil
}

func (f *fakeRuntime) ResolveScannerTypes(_ []string) []string {
	return append([]string{}, f.resolvedScannerTypes...)
}

func (f *fakeRuntime) StartScanner(_ context.Context, _ *models.Job, _ *ScannerLaunchPlan) error {
	f.startScannerCalls++
	if f.startScannerErr != nil && (f.startScannerErrOnCall == 0 || f.startScannerCalls == f.startScannerErrOnCall) {
		return f.startScannerErr
	}

	return nil
}

func (f *fakeRuntime) CleanupJob(_ context.Context, _ *models.Job) error {
	f.cleanupJobCalls++
	return nil
}

func (f *fakeRuntime) recordOperation(op string) {
	if f.operationLog == nil {
		return
	}

	*f.operationLog = append(*f.operationLog, op)
}

type fakeArtifacts struct {
	reportPath                 string
	buildAggregatedReportCalls int
}

func (f *fakeArtifacts) BuildAggregatedReport(_ context.Context, _ *models.Job) (string, error) {
	f.buildAggregatedReportCalls++
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

func assertOperationOrder(t *testing.T, operations []string, ordered ...string) {
	t.Helper()

	lastIndex := -1

	for _, op := range ordered {
		index := -1

		for i, candidate := range operations {
			if candidate == op {
				index = i
				break
			}
		}

		if index == -1 {
			t.Fatalf("operation %q missing from log %v", op, operations)
		}

		if index <= lastIndex {
			t.Fatalf("operation order %v violated by log %v", ordered, operations)
		}

		lastIndex = index
	}
}
