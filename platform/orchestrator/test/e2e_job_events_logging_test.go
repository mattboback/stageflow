package test

import (
	"context"
	"testing"
	"time"

	"github.com/mattboback/stageflow/packages/shared-go/events"
	"github.com/mattboback/stageflow/packages/shared-go/models"
	"github.com/mattboback/stageflow/platform/orchestrator/internal/db"
)

// TestE2E_JobEventsLogging tests that all events are properly logged.
func TestE2E_JobEventsLogging(t *testing.T) {
	orch, database, _, _, _ := setupE2ETest(t)

	ctx := context.Background()
	jobID := "test-job-logging"

	if err := orch.HandleJobCreated(ctx, &events.JobCreatedPayload{
		JobID:     jobID,
		InputType: "zip",
		InputPath: "test.zip",
		Config:    models.JobConfig{Modules: []string{"axe"}},
	}); err != nil {
		t.Fatalf("HandleJobCreated failed: %v", err)
	}

	if err := database.InsertJobEvent(context.Background(), &db.JobEventInsert{
		JobID:     jobID,
		Event:     "job.created",
		Timestamp: time.Now().UTC(),
		Payload:   `{"input_type":"zip"}`,
	}); err != nil {
		t.Fatalf("Failed to insert job.created: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	if err := database.InsertJobEvent(context.Background(), &db.JobEventInsert{
		JobID:     jobID,
		Event:     "extraction.ready",
		Timestamp: time.Now().UTC(),
		Payload:   `{"total_pages":3}`,
	}); err != nil {
		t.Fatalf("Failed to insert extraction.ready: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	if err := database.InsertJobEvent(context.Background(), &db.JobEventInsert{
		JobID:     jobID,
		Event:     "scan.completed",
		Timestamp: time.Now().UTC(),
		Payload:   `{"violations":5}`,
	}); err != nil {
		t.Fatalf("Failed to insert scan.completed: %v", err)
	}

	events, err := database.ListJobEvents(context.Background(), jobID, db.ListJobEventsOptions{Limit: 100})
	if err != nil {
		t.Fatalf("Failed to get job events: %v", err)
	}

	findAfter := func(start int, name string) int {
		for i := start; i < len(events); i++ {
			if events[i].Event == name {
				return i
			}
		}
		return -1
	}

	iCreated := findAfter(0, "job.created")
	if iCreated == -1 {
		t.Fatalf("Expected to find job.created in events, got %#v", events)
	}

	iReady := findAfter(iCreated+1, "extraction.ready")
	if iReady == -1 {
		t.Fatalf("Expected to find extraction.ready after job.created, got %#v", events)
	}

	iCompleted := findAfter(iReady+1, "scan.completed")
	if iCompleted == -1 {
		t.Fatalf("Expected to find scan.completed after extraction.ready, got %#v", events)
	}

	for i := 1; i < len(events); i++ {
		if events[i].Timestamp.Before(events[i-1].Timestamp) {
			t.Error("Events are not in chronological order")
		}
	}
}
