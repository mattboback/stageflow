package jobstatus

import (
	"sync"
	"time"

	"github.com/mattboback/stageflow/services/platform-api/internal/status"
)

type snapshotCache struct {
	mu            sync.Mutex
	ttl           time.Duration
	sweepInterval time.Duration
	lastSweep     time.Time
	jobs          map[string]*snapshotCacheEntry
}

type snapshotCacheEntry struct {
	record    *status.JobRecord
	expiresAt time.Time
}

func newSnapshotCache(ttl time.Duration) *snapshotCache {
	return &snapshotCache{
		ttl:           ttl,
		sweepInterval: 30 * time.Second,
		jobs:          make(map[string]*snapshotCacheEntry),
	}
}

func (c *snapshotCache) get(jobID string) (*status.JobRecord, bool) {
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

func (c *snapshotCache) put(record *status.JobRecord) {
	if c == nil || record == nil || record.JobID == "" {
		return
	}

	now := time.Now().UTC()

	c.mu.Lock()
	c.sweepExpiredLocked(now)
	c.jobs[record.JobID] = &snapshotCacheEntry{
		record:    cloneJobRecord(record),
		expiresAt: now.Add(c.ttl),
	}
	c.mu.Unlock()
}

func (c *snapshotCache) delete(jobID string) {
	if c == nil || jobID == "" {
		return
	}

	c.mu.Lock()
	delete(c.jobs, jobID)
	c.mu.Unlock()
}

func (c *snapshotCache) sweepExpiredLocked(now time.Time) {
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
