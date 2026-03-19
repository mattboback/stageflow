package messaging

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/mattboback/stageflow/libs/go/events"
	sharedmsg "github.com/mattboback/stageflow/libs/go/messaging"
)

// MockEventHandler records method calls for testing.
type MockEventHandler struct {
	mu                    sync.Mutex
	JobCreatedCalls       []*events.JobCreatedPayload
	ExtractionReadyCalls  []*events.ExtractionReadyPayload
	ExtractionFailedCalls []*events.ExtractionFailedPayload
	ScanPageCalls         []*events.ScanPageCompletedPayload
	ScanCompletedCalls    []*events.ScanCompletedPayload
	ScanFailedCalls       []*events.ScanFailedPayload
	ReturnError           error
}

func (m *MockEventHandler) HandleJobCreated(_ context.Context, payload *events.JobCreatedPayload) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.JobCreatedCalls = append(m.JobCreatedCalls, payload)

	return m.ReturnError
}

func (m *MockEventHandler) HandleExtractionReady(_ context.Context, payload *events.ExtractionReadyPayload) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ExtractionReadyCalls = append(m.ExtractionReadyCalls, payload)

	return m.ReturnError
}

func (m *MockEventHandler) HandleExtractionFailed(_ context.Context, payload *events.ExtractionFailedPayload) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ExtractionFailedCalls = append(m.ExtractionFailedCalls, payload)

	return m.ReturnError
}

func (m *MockEventHandler) HandleScanPageCompleted(_ context.Context, payload *events.ScanPageCompletedPayload) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ScanPageCalls = append(m.ScanPageCalls, payload)

	return m.ReturnError
}

func (m *MockEventHandler) HandleScanCompleted(_ context.Context, payload *events.ScanCompletedPayload) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ScanCompletedCalls = append(m.ScanCompletedCalls, payload)

	return m.ReturnError
}

func (m *MockEventHandler) HandleScanFailed(_ context.Context, payload *events.ScanFailedPayload) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ScanFailedCalls = append(m.ScanFailedCalls, payload)

	return m.ReturnError
}

func TestNewConsumer(t *testing.T) {
	t.Run("creates consumer with nil client and handler", func(t *testing.T) {
		consumer := NewConsumer(nil, nil)

		if consumer == nil {
			t.Fatal("NewConsumer returned nil")
		}

		if consumer.natsClient != nil {
			t.Error("expected natsClient to be nil")
		}

		if consumer.handler != nil {
			t.Error("expected handler to be nil")
		}
	})

	t.Run("creates consumer with mock handler", func(t *testing.T) {
		handler := &MockEventHandler{}
		consumer := NewConsumer(nil, handler)

		if consumer.handler == nil {
			t.Error("expected handler to be set")
		}
	})
}

//nolint:gocognit,gocyclo // Test function covers multiple interface scenarios
func TestEventHandlerInterface(t *testing.T) {
	t.Run("MockEventHandler implements EventHandler", func(_ *testing.T) {
		var _ EventHandler = (*MockEventHandler)(nil)
	})

	t.Run("MockEventHandler records JobCreated calls", func(t *testing.T) {
		handler := &MockEventHandler{}
		payload := &events.JobCreatedPayload{
			JobID:     "job-123",
			InputType: "zip",
			InputPath: "staging/job-123/archive.zip",
		}

		err := handler.HandleJobCreated(context.Background(), payload)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(handler.JobCreatedCalls) != 1 {
			t.Fatalf("expected 1 call, got %d", len(handler.JobCreatedCalls))
		}

		if handler.JobCreatedCalls[0].JobID != "job-123" {
			t.Errorf("JobID = %q, want %q", handler.JobCreatedCalls[0].JobID, "job-123")
		}
	})

	t.Run("MockEventHandler records ExtractionReady calls", func(t *testing.T) {
		handler := &MockEventHandler{}
		payload := &events.ExtractionReadyPayload{
			JobID:          "job-456",
			ProvenancePath: "/workspace/provenance.json",
			BaseURL:        "http://localhost:8080",
			TotalPages:     5,
		}

		err := handler.HandleExtractionReady(context.Background(), payload)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(handler.ExtractionReadyCalls) != 1 {
			t.Fatalf("expected 1 call, got %d", len(handler.ExtractionReadyCalls))
		}

		if handler.ExtractionReadyCalls[0].TotalPages != 5 {
			t.Errorf("TotalPages = %d, want %d", handler.ExtractionReadyCalls[0].TotalPages, 5)
		}
	})

	t.Run("MockEventHandler records ExtractionFailed calls", func(t *testing.T) {
		handler := &MockEventHandler{}
		payload := &events.ExtractionFailedPayload{
			JobID: "job-789",
			Error: "corrupted archive",
		}

		err := handler.HandleExtractionFailed(context.Background(), payload)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(handler.ExtractionFailedCalls) != 1 {
			t.Fatalf("expected 1 call, got %d", len(handler.ExtractionFailedCalls))
		}

		if handler.ExtractionFailedCalls[0].Error != "corrupted archive" {
			t.Errorf("Error = %q, want %q", handler.ExtractionFailedCalls[0].Error, "corrupted archive")
		}
	})

	t.Run("MockEventHandler records ScanCompleted calls", func(t *testing.T) {
		handler := &MockEventHandler{}
		payload := &events.ScanCompletedPayload{
			JobID:             "job-scan-1",
			ScannerType:       "axe",
			ResultsPath:       "job-scan-1/results.json",
			ReportPath:        "job-scan-1/report.html",
			TotalPagesScanned: 10,
			Summary: events.ScanSummary{
				TotalViolations: 25,
				BySeverity: map[string]int{
					"critical": 5,
					"serious":  10,
					"minor":    10,
				},
			},
		}

		err := handler.HandleScanCompleted(context.Background(), payload)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(handler.ScanCompletedCalls) != 1 {
			t.Fatalf("expected 1 call, got %d", len(handler.ScanCompletedCalls))
		}

		if handler.ScanCompletedCalls[0].Summary.TotalViolations != 25 {
			t.Errorf("TotalViolations = %d, want 25", handler.ScanCompletedCalls[0].Summary.TotalViolations)
		}
	})

	t.Run("MockEventHandler records ScanPageCompleted calls", func(t *testing.T) {
		handler := &MockEventHandler{}
		payload := &events.ScanPageCompletedPayload{
			JobID:      "job-scan-page",
			PageID:     "page-1",
			PageIndex:  3,
			TotalPages: 10,
		}

		err := handler.HandleScanPageCompleted(context.Background(), payload)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(handler.ScanPageCalls) != 1 {
			t.Fatalf("expected 1 call, got %d", len(handler.ScanPageCalls))
		}

		if handler.ScanPageCalls[0].PageIndex != 3 {
			t.Errorf("PageIndex = %d, want 3", handler.ScanPageCalls[0].PageIndex)
		}
	})

	t.Run("MockEventHandler records ScanFailed calls", func(t *testing.T) {
		handler := &MockEventHandler{}
		payload := &events.ScanFailedPayload{
			JobID: "job-scan-fail",
			Error: "browser crashed",
		}

		err := handler.HandleScanFailed(context.Background(), payload)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(handler.ScanFailedCalls) != 1 {
			t.Fatalf("expected 1 call, got %d", len(handler.ScanFailedCalls))
		}

		if handler.ScanFailedCalls[0].Error != "browser crashed" {
			t.Errorf("Error = %q, want %q", handler.ScanFailedCalls[0].Error, "browser crashed")
		}
	})

	t.Run("MockEventHandler returns configured error", func(t *testing.T) {
		handler := &MockEventHandler{
			ReturnError: context.Canceled,
		}

		err := handler.HandleJobCreated(context.Background(), &events.JobCreatedPayload{})
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	})
}

func TestConsumerSubscriptionMethods(t *testing.T) {
	// These tests verify the Consumer structure
	// Actual subscription testing requires integration tests with NATS
	t.Run("Consumer Start returns error with nil client", func(t *testing.T) {
		handler := &MockEventHandler{}
		consumer := NewConsumer(nil, handler)

		err := consumer.Start(context.Background())
		if err == nil {
			t.Fatal("expected error with nil client, got nil")
		}

		if !errors.Is(err, sharedmsg.ErrNilClient) {
			t.Fatalf("expected ErrNilClient, got %v", err)
		}
	})
}

//nolint:gocognit // Test function covers multiple payload structure validations
func TestEventPayloadStructures(t *testing.T) {
	// Test that our event payload types have expected fields
	t.Run("JobCreatedPayload fields", func(t *testing.T) {
		payload := events.JobCreatedPayload{
			JobID:     "job-1",
			InputType: "urls",
			URLs:      []string{"https://example.com"},
		}

		if payload.JobID == "" {
			t.Error("JobID should not be empty")
		}

		if payload.InputType != "urls" {
			t.Errorf("InputType = %q, want %q", payload.InputType, "urls")
		}

		if len(payload.URLs) != 1 || payload.URLs[0] != "https://example.com" {
			t.Errorf("URLs = %v, want [https://example.com]", payload.URLs)
		}
	})

	t.Run("ExtractionReadyPayload fields", func(t *testing.T) {
		payload := events.ExtractionReadyPayload{
			JobID:          "job-2",
			ProvenancePath: "/workspace/provenance.json",
			BaseURL:        "http://localhost:8080",
			TotalPages:     5,
		}

		if payload.ProvenancePath == "" {
			t.Error("ProvenancePath should not be empty")
		}

		if payload.JobID == "" {
			t.Error("JobID should not be empty")
		}

		if payload.BaseURL == "" {
			t.Error("BaseURL should not be empty")
		}

		if payload.TotalPages != 5 {
			t.Errorf("TotalPages = %d, want %d", payload.TotalPages, 5)
		}
	})

	t.Run("ScanCompletedPayload fields", func(t *testing.T) {
		payload := events.ScanCompletedPayload{
			JobID:             "job-3",
			ScannerType:       "axe",
			ResultsPath:       "job-3/results.json",
			ReportPath:        "job-3/report.html",
			TotalPagesScanned: 5,
			Summary: events.ScanSummary{
				TotalViolations: 12,
			},
		}

		if payload.ResultsPath == "" || payload.ReportPath == "" {
			t.Error("artifact paths should not be empty")
		}

		if payload.JobID == "" {
			t.Error("JobID should not be empty")
		}

		if payload.ScannerType == "" {
			t.Error("ScannerType should not be empty")
		}

		if payload.TotalPagesScanned != 5 {
			t.Errorf("TotalPagesScanned = %d, want %d", payload.TotalPagesScanned, 5)
		}
	})
}

func TestMockHandlerConcurrency(t *testing.T) {
	t.Run("concurrent handler calls are safe", func(t *testing.T) {
		handler := &MockEventHandler{}
		ctx := context.Background()

		var wg sync.WaitGroup
		for i := range 100 {
			wg.Add(1)

			go func(i int) {
				defer wg.Done()

				switch i % 6 {
				case 0:
					_ = handler.HandleJobCreated(ctx, &events.JobCreatedPayload{JobID: "job"})
				case 1:
					_ = handler.HandleExtractionReady(ctx, &events.ExtractionReadyPayload{JobID: "job"})
				case 2:
					_ = handler.HandleExtractionFailed(ctx, &events.ExtractionFailedPayload{JobID: "job"})
				case 3:
					_ = handler.HandleScanPageCompleted(ctx, &events.ScanPageCompletedPayload{
						JobID:      "job",
						PageID:     "page-1",
						PageIndex:  1,
						TotalPages: 1,
					})
				case 4:
					_ = handler.HandleScanCompleted(ctx, &events.ScanCompletedPayload{JobID: "job"})
				case 5:
					_ = handler.HandleScanFailed(ctx, &events.ScanFailedPayload{JobID: "job"})
				}
			}(i)
		}

		wg.Wait()

		total := len(handler.JobCreatedCalls) +
			len(handler.ExtractionReadyCalls) +
			len(handler.ExtractionFailedCalls) +
			len(handler.ScanPageCalls) +
			len(handler.ScanCompletedCalls) +
			len(handler.ScanFailedCalls)

		if total != 100 {
			t.Errorf("expected 100 total calls, got %d", total)
		}
	})
}
