package db

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/mattboback/stageflow/libs/go/models"
)

func TestPendingIntakeVisibleBeforePublishAndDeliveryWaits(t *testing.T) {
	store := setupTestDB(t)
	job := &models.Job{
		ID:               "seed",
		State:            models.JobStatePending,
		InputType:        "urls",
		URLs:             []string{"https://example.com"},
		Config:           models.JobConfig{Modules: []string{"axe"}},
		ExpectedScanners: []string{"axe"},
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	delivery := make(chan error, 1)

	err := store.PublishPendingJob(context.Background(), job, func(ctx context.Context) error {
		snapshot, readErr := (StatusReader{Database: store}).GetJob(ctx, job.ID)
		if readErr != nil || snapshot.State != models.JobStatePending || len(snapshot.ExpectedScanners) != 1 {
			t.Fatalf("status before publish: %#v %v", snapshot, readErr)
		}

		go func() { _, createErr := store.CreateJobIfAbsent(context.Background(), job); delivery <- createErr }()

		select {
		case deliveryErr := <-delivery:
			t.Fatalf("delivery escaped intake lock: %v", deliveryErr)
		case <-time.After(50 * time.Millisecond):
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	select {
	case deliveryErr := <-delivery:
		if deliveryErr != nil {
			t.Fatal(deliveryErr)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("delivery stayed locked")
	}
}

func TestPendingIntakePublishFailureScrubsBeforeDelayedDelivery(t *testing.T) {
	store := setupTestDB(t)
	job := &models.Job{
		ID:        "failed-seed",
		State:     models.JobStatePending,
		InputType: "urls",
		Config:    models.JobConfig{Auth: json.RawMessage(`{"mode":"form","password":"publication-secret"}`)},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	publishErr := errors.New("lost acknowledgement")
	if err := store.PublishPendingJob(
		context.Background(),
		job,
		func(context.Context) error { return publishErr },
	); !errors.Is(
		err,
		publishErr,
	) {
		t.Fatalf("publish error %v", err)
	}

	row, err := store.GetJob(context.Background(), job.ID)
	if err != nil {
		t.Fatal(err)
	}

	raw, _ := json.Marshal(row.Config)
	if row.State != models.JobStateFailed || strings.Contains(string(raw), "publication-secret") {
		t.Fatalf("unsafe failed intake %#v %s", row, string(raw))
	}

	created, err := store.CreateJobIfAbsent(context.Background(), job)
	if err != nil || created {
		t.Fatalf("late delivery recreated failed seed: %v %v", created, err)
	}
}

func TestPendingIntakeCancellationScrubsSeed(t *testing.T) {
	store := setupTestDB(t)
	store.db.SetMaxOpenConns(1)

	ctx, cancel := context.WithCancel(context.Background())
	job := &models.Job{
		ID:        "canceled-seed",
		State:     models.JobStatePending,
		InputType: "urls",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := store.PublishPendingJob(ctx, job, func(context.Context) error { cancel(); return ctx.Err() })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation error %v", err)
	}

	row, err := store.GetJob(context.Background(), job.ID)
	if err != nil || row.State != models.JobStateFailed {
		t.Fatalf("canceled seed %#v %v", row, err)
	}
}
