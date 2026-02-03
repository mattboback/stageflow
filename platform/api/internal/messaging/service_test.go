package messaging_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/mattboback/stageflow/packages/shared-go/events"
	"github.com/mattboback/stageflow/platform/api/internal/messaging"
	"github.com/mattboback/stageflow/platform/api/internal/sse"
)

type mockStatusStore struct {
	called map[string]bool
}

func newMockStatusStore() *mockStatusStore {
	return &mockStatusStore{
		called: make(map[string]bool),
	}
}

func (m *mockStatusStore) HandleJobCreated(ctx context.Context, p *events.JobCreatedPayload) error {
	m.called["HandleJobCreated"] = true
	return nil
}

func (m *mockStatusStore) HandleExtractionReady(ctx context.Context, p *events.ExtractionReadyPayload) error {
	m.called["HandleExtractionReady"] = true
	return nil
}

func (m *mockStatusStore) HandleExtractionFailed(ctx context.Context, p *events.ExtractionFailedPayload) error {
	m.called["HandleExtractionFailed"] = true
	return nil
}

func (m *mockStatusStore) HandleScanPageCompleted(ctx context.Context, p *events.ScanPageCompletedPayload) error {
	m.called["HandleScanPageCompleted"] = true
	return nil
}

func (m *mockStatusStore) HandleScanCompleted(ctx context.Context, p *events.ScanCompletedPayload) error {
	m.called["HandleScanCompleted"] = true
	return nil
}

func (m *mockStatusStore) HandleScanFailed(ctx context.Context, p *events.ScanFailedPayload) error {
	m.called["HandleScanFailed"] = true
	return nil
}

func (m *mockStatusStore) HandleJobCompleted(ctx context.Context, p *events.JobCompletedPayload) error {
	m.called["HandleJobCompleted"] = true
	return nil
}

func (m *mockStatusStore) HandleJobFailed(ctx context.Context, p *events.JobFailedPayload) error {
	m.called["HandleJobFailed"] = true
	return nil
}

func TestSSEBroadcastHandler_HandleJobCreated(t *testing.T) {
	hub := sse.NewHub()
	store := newMockStatusStore()
	handler := messaging.NewSSEBroadcastHandler(store, hub)

	jobID := "job-123"
	client := hub.Subscribe(jobID)
	defer hub.Unsubscribe(client)

	payload := &events.JobCreatedPayload{
		JobID: jobID,
	}

	err := handler.HandleJobCreated(context.Background(), payload)
	if err != nil {
		t.Fatalf("HandleJobCreated failed: %v", err)
	}

	if !store.called["HandleJobCreated"] {
		t.Error("Store.HandleJobCreated was not called")
	}

	select {
	case msg := <-client.Events:
		var evt map[string]any
		if err := json.Unmarshal(msg, &evt); err != nil {
			t.Fatalf("Failed to unmarshal SSE message: %v", err)
		}
		if evt["type"] != "status" || evt["state"] != "PENDING" {
			t.Errorf("Unexpected SSE message: %v", evt)
		}
	case <-time.After(time.Second):
		t.Error("Timed out waiting for SSE message")
	}
}

func TestSSEBroadcastHandler_HandleExtractionReady(t *testing.T) {
	hub := sse.NewHub()
	store := newMockStatusStore()
	handler := messaging.NewSSEBroadcastHandler(store, hub)

	jobID := "job-456"
	client := hub.Subscribe(jobID)
	defer hub.Unsubscribe(client)

	payload := &events.ExtractionReadyPayload{
		JobID:      jobID,
		TotalPages: 5,
	}

	err := handler.HandleExtractionReady(context.Background(), payload)
	if err != nil {
		t.Fatalf("HandleExtractionReady failed: %v", err)
	}

	if !store.called["HandleExtractionReady"] {
		t.Error("Store.HandleExtractionReady was not called")
	}

	select {
	case msg := <-client.Events:
		var evt map[string]any
		if err := json.Unmarshal(msg, &evt); err != nil {
			t.Fatalf("Failed to unmarshal SSE message: %v", err)
		}
		if evt["type"] != "status" || evt["state"] != "READY_TO_SCAN" {
			t.Errorf("Unexpected SSE message: %v", evt)
		}
		if total, ok := evt["totalPages"].(float64); !ok || total != 5 {
			t.Errorf("Expected totalPages=5, got %v", evt["totalPages"])
		}
	case <-time.After(time.Second):
		t.Error("Timed out waiting for SSE message")
	}
}

func TestSSEBroadcastHandler_HandleScanPageCompleted(t *testing.T) {
	hub := sse.NewHub()
	store := newMockStatusStore()
	handler := messaging.NewSSEBroadcastHandler(store, hub)

	jobID := "job-789"
	client := hub.Subscribe(jobID)
	defer hub.Unsubscribe(client)

	payload := &events.ScanPageCompletedPayload{
		JobID:      jobID,
		PageIndex:  1,
		TotalPages: 2,
	}

	err := handler.HandleScanPageCompleted(context.Background(), payload)
	if err != nil {
		t.Fatalf("HandleScanPageCompleted failed: %v", err)
	}

	if !store.called["HandleScanPageCompleted"] {
		t.Error("Store.HandleScanPageCompleted was not called")
	}

	select {
	case msg := <-client.Events:
		var evt map[string]any
		if err := json.Unmarshal(msg, &evt); err != nil {
			t.Fatalf("Failed to unmarshal SSE message: %v", err)
		}
		if evt["type"] != "progress" || evt["state"] != "SCANNING" {
			t.Errorf("Unexpected SSE message: %v", evt)
		}
		progress, ok := evt["progress"].(map[string]any)
		if !ok {
			t.Fatalf("progress is not a map: %v", evt["progress"])
		}
		if cp, ok := progress["currentPage"].(float64); !ok || cp != 1 {
			t.Errorf("Expected currentPage=1, got %v", progress["currentPage"])
		}
	case <-time.After(time.Second):
		t.Error("Timed out waiting for SSE message")
	}
}
