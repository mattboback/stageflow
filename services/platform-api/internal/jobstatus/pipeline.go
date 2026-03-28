package jobstatus

import (
	"context"
	"errors"
	"time"

	"github.com/mattboback/stageflow/services/platform-api/internal/status"
)

const defaultCacheTTL = 15 * time.Minute

type Pipeline struct {
	currentReader CurrentReader
	cache         *snapshotCache
	broker        *watcherBroker
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
	}
}

func (p *Pipeline) Begin(_ context.Context, cmd BeginJob) (*status.JobRecord, error) {
	rec, err := beginSnapshot(cmd)
	if err != nil {
		return nil, err
	}

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

func (p *Pipeline) Current(ctx context.Context, jobID string) (*status.JobRecord, error) {
	if p.currentReader != nil {
		rec, err := p.currentReader.GetJob(ctx, jobID)
		if err == nil {
			return cloneJobRecord(rec), nil
		}

		if !errors.Is(err, status.ErrJobNotFound) {
			return nil, err
		}
	}

	if rec, ok := p.cache.get(jobID); ok {
		return rec, nil
	}

	return nil, status.ErrJobNotFound
}

func (p *Pipeline) Watch(ctx context.Context, jobID string, _ WatchOptions) (*status.JobRecord, Subscription, error) {
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
