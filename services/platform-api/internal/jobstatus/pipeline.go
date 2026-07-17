package jobstatus

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/mattboback/stageflow/services/platform-api/internal/status"
)

const defaultCacheTTL = 15 * time.Minute

type Pipeline struct {
	currentReader CurrentReader
	cache         *snapshotCache
	broker        *watcherBroker
	jobLocksMu    sync.Mutex
	jobLocks      map[string]*pipelineJobLock
}

type pipelineJobLock struct {
	mu   sync.Mutex
	refs int
}

func New(cfg *Config) *Pipeline {
	if cfg == nil {
		cfg = &Config{}
	}

	cacheTTL := cfg.CacheTTL
	if cacheTTL <= 0 {
		cacheTTL = defaultCacheTTL
	}

	return &Pipeline{
		currentReader: cfg.CurrentReader,
		cache:         newSnapshotCache(cacheTTL),
		broker:        newWatcherBroker(),
		jobLocks:      make(map[string]*pipelineJobLock),
	}
}

func (p *Pipeline) Begin(_ context.Context, cmd BeginJob) (*status.JobRecord, error) {
	rec, err := beginSnapshot(cmd)
	if err != nil {
		return nil, err
	}

	unlock := p.lockJob(rec.JobID)
	defer unlock()

	if current, ok := p.cache.get(rec.JobID); ok {
		return current, nil
	}

	p.cache.put(rec)

	return cloneJobRecord(rec), nil
}

func (p *Pipeline) Apply(ctx context.Context, signal Signal) (*status.JobRecord, error) {
	jobID := signalJobID(signal)
	if jobID == "" {
		return nil, errors.New("jobstatus: signal job ID is required")
	}

	// Loading, reducing, and replacing a snapshot is one logical operation.
	// Serialize it per job so events for independent jobs still run in parallel
	// while concurrent scanner events cannot overwrite one another.
	unlock := p.lockJob(jobID)
	defer unlock()

	base, err := p.loadBaseSnapshot(ctx, jobID, signal.ObservedAt)
	if err != nil {
		return nil, err
	}

	next, changed, err := reduceSnapshot(base, signal)
	if err != nil {
		return nil, err
	}

	if !changed {
		return cloneJobRecord(next), nil
	}

	p.cache.put(next)
	p.broker.Publish(Change{
		JobID:      jobID,
		Snapshot:   next,
		Signal:     signal,
		ObservedAt: normalizeObservedAt(signal.ObservedAt),
	})

	return cloneJobRecord(next), nil
}

func (p *Pipeline) lockJob(jobID string) func() {
	p.jobLocksMu.Lock()

	lock := p.jobLocks[jobID]
	if lock == nil {
		lock = &pipelineJobLock{}
		p.jobLocks[jobID] = lock
	}

	lock.refs++
	p.jobLocksMu.Unlock()

	lock.mu.Lock()

	return func() {
		lock.mu.Unlock()

		p.jobLocksMu.Lock()

		lock.refs--
		if lock.refs == 0 {
			delete(p.jobLocks, jobID)
		}
		p.jobLocksMu.Unlock()
	}
}

// Current returns the most recent projection for a job. It checks the
// in-process cache first (which tracks real-time NATS event projections),
// falling back to the currentReader (orchestrator HTTP API) for cold-start
// or cache-miss scenarios. This ordering is consistent with loadBaseSnapshot
// and ensures SSE initial status reflects the latest projected state rather
// than a potentially stale orchestrator snapshot.
func (p *Pipeline) Current(ctx context.Context, jobID string) (*status.JobRecord, error) {
	if rec, ok := p.cache.get(jobID); ok {
		return rec, nil
	}

	if p.currentReader != nil {
		rec, err := p.currentReader.GetJob(ctx, jobID)
		if err == nil {
			return cloneJobRecord(rec), nil
		}

		if !errors.Is(err, status.ErrJobNotFound) {
			return nil, err
		}
	}

	return nil, status.ErrJobNotFound
}

func (p *Pipeline) Watch(ctx context.Context, jobID string) (*status.JobRecord, Subscription, error) {
	sub := p.broker.Subscribe(jobID)

	if done := ctx.Done(); done != nil {
		go func() {
			<-done

			_ = sub.Close()
		}()
	}

	rec, err := p.Current(ctx, jobID)
	if err != nil {
		_ = sub.Close()

		return nil, nil, err
	}

	return rec, sub, nil
}

func (p *Pipeline) loadBaseSnapshot(
	ctx context.Context,
	jobID string,
	observedAt time.Time,
) (*status.JobRecord, error) {
	if rec, ok := p.cache.get(jobID); ok {
		return rec, nil
	}

	if p.currentReader != nil {
		rec, err := p.currentReader.GetJob(ctx, jobID)
		if err == nil {
			return cloneJobRecord(rec), nil
		}

		if !errors.Is(err, status.ErrJobNotFound) {
			return nil, err
		}
	}

	observedAt = normalizeObservedAt(observedAt)

	return &status.JobRecord{
		JobID:     jobID,
		CreatedAt: observedAt,
		UpdatedAt: observedAt,
	}, nil
}
