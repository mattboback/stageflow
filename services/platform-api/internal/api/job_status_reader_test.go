package api

import (
	"testing"
	"time"

	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/models"
)

func TestPendingJobCachePutFromPayloadSweepsExpiredEntries(t *testing.T) {
	t.Parallel()

	cache := newPendingJobCache()
	cache.sweepInterval = 0

	now := time.Now().UTC()

	cache.mu.Lock()
	cache.jobs["expired-job"] = &pendingJobEntry{
		record:    nil,
		expiresAt: now.Add(-time.Minute),
	}
	cache.jobs["active-job"] = &pendingJobEntry{
		record:    nil,
		expiresAt: now.Add(time.Minute),
	}
	cache.mu.Unlock()

	cache.putFromPayload(&events.JobCreatedPayload{
		JobID:     "new-job",
		InputType: models.JobInputTypeURLs,
		URLs:      []string{"https://example.com"},
	})

	cache.mu.Lock()
	defer cache.mu.Unlock()

	if _, ok := cache.jobs["expired-job"]; ok {
		t.Fatal("expected expired cache entry to be swept during put")
	}

	if _, ok := cache.jobs["active-job"]; !ok {
		t.Fatal("expected non-expired cache entry to remain after sweep")
	}

	if _, ok := cache.jobs["new-job"]; !ok {
		t.Fatal("expected new cache entry to be stored")
	}
}
