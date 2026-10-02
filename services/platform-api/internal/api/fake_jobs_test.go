package api

import (
	"context"
	"maps"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/mattboback/stageflow/internal/jobstore"
	"github.com/mattboback/stageflow/libs/go/models"
)

// fakeJobs stands in for the jobs table in handler tests. It keeps the rows
// PostgreSQL would hold and serves them through the real jobstore.Snapshot
// projection, so a handler reads what it would read in production. A test moves
// a job along by editing its row, the way a reconcile tick does.
type fakeJobs struct {
	mu       sync.Mutex
	jobs     map[string]*models.Job
	admitted []*models.Job // every job AdmitJob accepted, as the handler built it

	err       error                       // returned by AdmitJob
	deleteErr error                       // returned by DeleteJob
	onAdmit   func(job *models.Job) error // runs before a job is stored
}

func newFakeJobs() *fakeJobs {
	return &fakeJobs{jobs: make(map[string]*models.Job)}
}

func (f *fakeJobs) AdmitJob(_ context.Context, job *models.Job) error {
	if f.err != nil {
		return f.err
	}

	if f.onAdmit != nil {
		if err := f.onAdmit(job); err != nil {
			return err
		}
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.admitted = append(f.admitted, cloneJob(job))
	f.jobs[job.ID] = storedJob(job)

	return nil
}

func (f *fakeJobs) DeleteJob(_ context.Context, id string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	delete(f.jobs, id)

	return nil
}

func (f *fakeJobs) GetJob(_ context.Context, id string) (*jobstore.JobRecord, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	job, ok := f.jobs[id]
	if !ok {
		return nil, jobstore.ErrJobNotFound
	}

	return jobstore.Snapshot(cloneJob(job)), nil
}

// put stores a row as the orchestrator would have written it.
func (f *fakeJobs) put(job *models.Job) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.jobs[job.ID] = storedJob(job)
}

// finish marks a stored job DONE, as the terminal transition does. An empty
// reportKey models a job whose report was never published.
func (f *fakeJobs) finish(t *testing.T, id, reportKey string) {
	t.Helper()

	f.mu.Lock()
	defer f.mu.Unlock()

	job, ok := f.jobs[id]
	if !ok {
		t.Fatalf("finish: job %s is not stored", id)
	}

	now := time.Now().UTC()
	job.State = models.JobStateDone
	job.ReportJSONKey = reportKey
	job.CompletedAt = &now
	job.UpdatedAt = now
}

// scanningURLJob is a claimed one-page URL job in the middle of its scan.
func scanningURLJob(id string, scanners ...string) *models.Job {
	now := time.Now().UTC()

	return &models.Job{
		ID:               id,
		State:            models.JobStateScanning,
		InputType:        models.JobInputTypeURLs,
		URLs:             []string{"https://example.com"},
		Config:           models.JobConfig{Modules: scanners},
		ExpectedScanners: scanners,
		TotalPages:       1,
		CreatedAt:        now,
		UpdatedAt:        now,
		ClaimedAt:        &now,
		ScanStartedAt:    &now,
	}
}

// storedJob returns a row the way the jobs table hands it back: one scanner per
// requested module, and the provenance location every job has.
func storedJob(job *models.Job) *models.Job {
	stored := cloneJob(job)
	if len(stored.ExpectedScanners) == 0 {
		stored.ExpectedScanners = slices.Clone(stored.Config.Modules)
	}

	stored.ProvenanceKey = stored.ID + "/provenance.json"

	return stored
}

// cloneJob copies everything a test or a reader could mutate, so a stream
// reading a snapshot never races a test editing the row.
func cloneJob(job *models.Job) *models.Job {
	clone := *job
	clone.URLs = slices.Clone(job.URLs)
	clone.ExpectedScanners = slices.Clone(job.ExpectedScanners)
	clone.CompletedScanners = slices.Clone(job.CompletedScanners)
	clone.Config.Modules = slices.Clone(job.Config.Modules)
	clone.Config.PagePaths = slices.Clone(job.Config.PagePaths)
	clone.Config.Auth = slices.Clone(job.Config.Auth)
	clone.ScannerAttempts = maps.Clone(job.ScannerAttempts)
	clone.ScannerStates = maps.Clone(job.ScannerStates)
	clone.ScannerContainerIDs = maps.Clone(job.ScannerContainerIDs)
	clone.ScannerProgress = maps.Clone(job.ScannerProgress)

	if job.ScannerResults != nil {
		clone.ScannerResults = make(map[string]*models.ScannerResult, len(job.ScannerResults))
		for name, result := range job.ScannerResults {
			copied := *result
			clone.ScannerResults[name] = &copied
		}
	}

	return &clone
}

func testJobs(server *Server) *fakeJobs {
	return server.statusReader.(*fakeJobs)
}
