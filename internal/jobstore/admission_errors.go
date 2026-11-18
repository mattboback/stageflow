package jobstore

import "errors"

// ErrQueueFull indicates that transactional admission reached the pending-job limit.
var ErrQueueFull = errors.New("pending job queue is full")

var ErrCleanupPending = errors.New("job cleanup is pending")
var ErrJobBusy = errors.New("job is running")
