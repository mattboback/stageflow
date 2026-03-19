package api

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/mattboback/stageflow/libs/go/events"
	"github.com/mattboback/stageflow/libs/go/models"
	"github.com/mattboback/stageflow/platform/api/internal/status"
)

type pendingJobCache struct {
	mu            sync.Mutex
	ttl           time.Duration
	sweepInterval time.Duration
	lastSweep     time.Time
	jobs          map[string]*pendingJobEntry
}

type pendingJobEntry struct {
	record    *status.JobRecord
	expiresAt time.Time
}

const defaultPendingJobCacheTTL = 2 * time.Minute

func newPendingJobCache() *pendingJobCache {
	return &pendingJobCache{
		ttl:           defaultPendingJobCacheTTL,
		sweepInterval: 30 * time.Second,
		jobs:          make(map[string]*pendingJobEntry),
	}
}

func (c *pendingJobCache) putFromPayload(payload *events.JobCreatedPayload) {
	if c == nil || payload == nil || payload.JobID == "" {
		return
	}

	state := models.JobStatePending

	switch payload.InputType {
	case models.JobInputTypeZip:
		state = models.JobStateExtracting
	case models.JobInputTypeURLs:
		state = models.JobStateScanning
	}

	now := time.Now().UTC()
	totalPages := 0

	if payload.InputType == models.JobInputTypeURLs {
		totalPages = len(payload.URLs)
	}

	rec := &status.JobRecord{
		JobID:             payload.JobID,
		State:             state,
		InputType:         payload.InputType,
		CreatedAt:         now,
		UpdatedAt:         now,
		CurrentPage:       0,
		TotalPages:        totalPages,
		ExpectedScanners:  cloneStringSlice(payload.Config.Modules),
		CompletedScanners: []string{},
	}

	c.mu.Lock()
	c.sweepExpiredLocked(now)
	c.jobs[payload.JobID] = &pendingJobEntry{
		record:    rec,
		expiresAt: now.Add(c.ttl),
	}
	c.mu.Unlock()
}

func (c *pendingJobCache) sweepExpiredLocked(now time.Time) {
	if c == nil {
		return
	}

	if c.sweepInterval > 0 && !c.lastSweep.IsZero() && now.Sub(c.lastSweep) < c.sweepInterval {
		return
	}

	for jobID, entry := range c.jobs {
		if entry == nil || now.After(entry.expiresAt) {
			delete(c.jobs, jobID)
		}
	}

	c.lastSweep = now
}

func (c *pendingJobCache) get(jobID string) (*status.JobRecord, bool) {
	if c == nil || jobID == "" {
		return nil, false
	}

	now := time.Now().UTC()

	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.jobs[jobID]
	if !ok {
		return nil, false
	}

	if now.After(entry.expiresAt) {
		delete(c.jobs, jobID)

		return nil, false
	}

	return cloneJobRecord(entry.record), true
}

func (c *pendingJobCache) delete(jobID string) {
	if c == nil || jobID == "" {
		return
	}

	c.mu.Lock()
	delete(c.jobs, jobID)
	c.mu.Unlock()
}

func (s *Server) loadJobRecord(ctx context.Context, jobID string) (*status.JobRecord, error) {
	if s.statusReader == nil {
		if pending, ok := s.pendingJobs.get(jobID); ok {
			return pending, nil
		}

		return nil, status.ErrJobNotFound
	}

	rec, err := s.statusReader.GetJob(ctx, jobID)
	if err == nil {
		s.pendingJobs.delete(jobID)

		return rec, nil
	}

	if errors.Is(err, status.ErrJobNotFound) {
		if pending, ok := s.pendingJobs.get(jobID); ok {
			return pending, nil
		}
	}

	return nil, err
}

func cloneJobRecord(rec *status.JobRecord) *status.JobRecord {
	if rec == nil {
		return nil
	}

	cloned := *rec
	cloned.ExpectedScanners = cloneStringSlice(rec.ExpectedScanners)
	cloned.CompletedScanners = cloneStringSlice(rec.CompletedScanners)

	if rec.ScannerArtifacts != nil {
		cloned.ScannerArtifacts = make(map[string]*status.ScannerArtifactRecord, len(rec.ScannerArtifacts))
		for scannerType, artifact := range rec.ScannerArtifacts {
			if artifact == nil {
				continue
			}

			copyArtifact := *artifact
			cloned.ScannerArtifacts[scannerType] = &copyArtifact
		}
	}

	return &cloned
}

func cloneStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	cloned := make([]string, len(values))
	copy(cloned, values)

	return cloned
}
