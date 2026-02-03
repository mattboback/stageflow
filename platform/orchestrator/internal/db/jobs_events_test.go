package db

import (
	"context"
	"testing"
	"time"

	"github.com/mattboback/stageflow/packages/shared-go/models"
)

func TestInsertJobEvent(t *testing.T) {
	db := setupTestDB(t)

	job := &models.Job{
		ID:        "job-123",
		State:     models.JobStatePending,
		InputType: "zip",
		InputPath: "test.zip",
		Config:    models.JobConfig{Modules: []string{"axe"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mustCreateJob(t, db, job)

	if err := db.InsertJobEvent(context.Background(), &JobEventInsert{
		JobID:     "job-123",
		Event:     "job.created",
		Timestamp: time.Now().UTC(),
		Payload:   `{"test":"payload"}`,
		RequestID: "req-1",
		RunID:     "run-1",
		Producer:  "test",
	}); err != nil {
		t.Fatalf("Failed to insert job event: %v", err)
	}

	events, err := db.ListJobEvents(context.Background(), "job-123", ListJobEventsOptions{Limit: 10})
	if err != nil {
		t.Fatalf("Failed to get job events: %v", err)
	}

	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}

	if events[0].Event != "job.created" {
		t.Errorf("Expected event job.created, got %s", events[0].Event)
	}
	if events[0].RequestID != "req-1" {
		t.Errorf("Expected request_id req-1, got %s", events[0].RequestID)
	}
	if events[0].RunID != "run-1" {
		t.Errorf("Expected run_id run-1, got %s", events[0].RunID)
	}
}

func TestListJobEvents(t *testing.T) {
	db := setupTestDB(t)

	job := &models.Job{
		ID:        "job-123",
		State:     models.JobStatePending,
		InputType: "zip",
		InputPath: "test.zip",
		Config:    models.JobConfig{Modules: []string{"axe"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mustCreateJob(t, db, job)

	mustLogEvent(t, db, "job-123", "job.created", `{"test":"payload1"}`)
	mustLogEvent(t, db, "job-123", "extraction.ready", `{"test":"payload2"}`)
	mustLogEvent(t, db, "job-123", "scan.completed", `{"test":"payload3"}`)

	events, err := db.ListJobEvents(context.Background(), "job-123", ListJobEventsOptions{Limit: 10})
	if err != nil {
		t.Fatalf("Failed to get job events: %v", err)
	}

	if len(events) != 3 {
		t.Errorf("Expected 3 events, got %d", len(events))
	}

	if events[0].Event != "job.created" {
		t.Errorf("Expected first event to be job.created, got %s", events[0].Event)
	}
	if events[2].Event != "scan.completed" {
		t.Errorf("Expected last event to be scan.completed, got %s", events[2].Event)
	}
}
