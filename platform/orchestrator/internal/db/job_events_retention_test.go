package db

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/mattboback/stageflow/packages/shared-go/models"
)

func TestPruneJobEventsBefore_RemovesOnlyOlderRows(t *testing.T) {
	db := setupTestDB(t)

	job := &models.Job{
		ID:        "job-prune",
		State:     models.JobStatePending,
		InputType: "zip",
		InputPath: "test.zip",
		Config:    models.JobConfig{Modules: []string{"axe"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mustCreateJob(t, db, job)

	now := time.Now().UTC()
	oldTimestamp := now.Add(-72 * time.Hour)
	newTimestamp := now.Add(-30 * time.Minute)

	for i := 0; i < 5; i++ {
		if err := db.InsertJobEvent(context.Background(), &JobEventInsert{
			JobID:     job.ID,
			Event:     "old.event",
			Timestamp: oldTimestamp.Add(time.Duration(i) * time.Second),
			Payload:   "{}",
		}); err != nil {
			t.Fatalf("insert old event: %v", err)
		}
	}

	for i := 0; i < 2; i++ {
		if err := db.InsertJobEvent(context.Background(), &JobEventInsert{
			JobID:     job.ID,
			Event:     "new.event",
			Timestamp: newTimestamp.Add(time.Duration(i) * time.Second),
			Payload:   "{}",
		}); err != nil {
			t.Fatalf("insert new event: %v", err)
		}
	}

	deleted, err := db.PruneJobEventsBefore(context.Background(), now.Add(-24*time.Hour), 2)
	if err != nil {
		t.Fatalf("PruneJobEventsBefore returned error: %v", err)
	}

	if deleted != 5 {
		t.Fatalf("expected 5 rows deleted, got %d", deleted)
	}

	remaining, err := db.ListJobEvents(context.Background(), job.ID, ListJobEventsOptions{Limit: 100})
	if err != nil {
		t.Fatalf("ListJobEvents failed: %v", err)
	}

	if len(remaining) != 2 {
		t.Fatalf("expected 2 remaining events, got %d", len(remaining))
	}

	for _, event := range remaining {
		if event.Event != "new.event" {
			t.Fatalf("expected only new events to remain, got %q", event.Event)
		}
	}
}

func TestPruneJobEventsBefore_RespectsCanceledContext(t *testing.T) {
	db := setupTestDB(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := db.PruneJobEventsBefore(ctx, time.Now(), 100); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestStartJobEventsPruner_PrunesInBackground(t *testing.T) {
	db := setupTestDB(t)

	job := &models.Job{
		ID:        "job-pruner-loop",
		State:     models.JobStatePending,
		InputType: "zip",
		InputPath: "test.zip",
		Config:    models.JobConfig{Modules: []string{"axe"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mustCreateJob(t, db, job)

	oldTimestamp := time.Now().UTC().Add(-96 * time.Hour)
	if err := db.InsertJobEvent(context.Background(), &JobEventInsert{
		JobID:     job.ID,
		Event:     "old.event",
		Timestamp: oldTimestamp,
		Payload:   "{}",
	}); err != nil {
		t.Fatalf("insert old event: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := db.StartJobEventsPruner(ctx, JobEventsPrunerConfig{
		Retention: 48 * time.Hour,
		Interval:  10 * time.Millisecond,
		BatchSize: 10,
		Logger:    logger,
	}); err != nil {
		t.Fatalf("StartJobEventsPruner returned error: %v", err)
	}

	deadline := time.Now().Add(500 * time.Millisecond)
	for {
		events, err := db.ListJobEvents(context.Background(), job.ID, ListJobEventsOptions{Limit: 10})
		if err != nil {
			t.Fatalf("ListJobEvents failed: %v", err)
		}

		if len(events) == 0 {
			return
		}

		if time.Now().After(deadline) {
			t.Fatalf("expected background pruner to delete old events, still have %d", len(events))
		}

		time.Sleep(10 * time.Millisecond)
	}
}
