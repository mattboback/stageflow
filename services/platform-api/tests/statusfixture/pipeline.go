// Package jobstatus provides in-memory snapshots solely for API handler tests.
package jobstatus

import (
	"context"
	"sync"

	"github.com/mattboback/stageflow/internal/jobstore"
	"github.com/mattboback/stageflow/libs/go/models"
)

type Pipeline struct {
	mu   sync.Mutex
	jobs map[string]*jobstore.JobRecord
}

func New(_ *Config) *Pipeline { return &Pipeline{jobs: make(map[string]*jobstore.JobRecord)} }
func (p *Pipeline) GetJob(_ context.Context, id string) (*jobstore.JobRecord, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	rec := p.jobs[id]
	if rec == nil {
		return nil, jobstore.ErrJobNotFound
	}

	return cloneJobRecord(rec), nil
}
func (p *Pipeline) Apply(_ context.Context, signal Signal) (*jobstore.JobRecord, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	id := signalJobID(signal)

	next, _, err := reduceSnapshot(p.jobs[id], signal)
	if err != nil {
		return nil, err
	}

	if next.ScannerStatuses == nil {
		next.ScannerStatuses = make(map[string]models.ScannerStatus)
	}

	if v := signal.ScanPageCompleted; v != nil {
		next.ScannerStatuses[v.ScannerType] = models.ScannerStatus{
			State:        "running",
			PagesScanned: v.PageIndex,
			TotalPages:   v.TotalPages,
		}
	}

	if v := signal.ScanCompleted; v != nil {
		next.ScannerStatuses[v.ScannerType] = models.ScannerStatus{
			State:        "completed",
			PagesScanned: v.TotalPagesScanned,
			TotalPages:   next.TotalPages,
			Violations:   v.Summary.TotalViolations,
		}
	}

	if v := signal.ScanFailed; v != nil {
		next.ScannerStatuses[v.ScannerType] = models.ScannerStatus{State: "failed", Error: v.Error}
	}

	p.jobs[id] = next

	return cloneJobRecord(next), nil
}

// Put replaces a snapshot to model committed database admission in HTTP tests.
func (p *Pipeline) Put(rec *jobstore.JobRecord) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.jobs[rec.JobID] = cloneJobRecord(rec)
}

// Delete removes the durable record; no events can replay it after queue retirement.
func (p *Pipeline) Delete(id string) { p.mu.Lock(); defer p.mu.Unlock(); delete(p.jobs, id) }
