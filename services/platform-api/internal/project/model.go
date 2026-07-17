package project

import (
	"errors"
	"time"
)

const (
	BaselineOperationPromote       = "promote"
	BaselineOperationBackfill      = "backfill"
	BaselineOperationDeleteObject  = "delete_object"
	BaselineOperationDeleteProject = "delete_project"

	BaselineOperationObjectPending = "object_pending"
	BaselineOperationCommitPending = "commit_pending"
)

// BaselineOperation is a durable, idempotently replayable object-store
// mutation paired with its project database transition.
type BaselineOperation struct {
	ID            int64
	Kind          string
	State         string
	ProjectID     string
	JobID         string
	PreviousJobID string
	SourceKey     string
	Attempts      int
	LastError     string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

var (
	ErrBaselineOperationPending = errors.New("baseline operation already pending")
	ErrProjectDeletionPending   = errors.New("project deletion already pending")
	ErrBaselineSuperseded       = errors.New("baseline promotion superseded")
)

// Project represents a registered scan target with baseline tracking.
type Project struct {
	ID            string    `json:"id"`
	Slug          string    `json:"slug"`
	Name          string    `json:"name"`
	URLs          []string  `json:"urls"`
	Scanners      []string  `json:"scanners,omitempty"`
	BaselineJobID string    `json:"baseline_job_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Update holds mutable fields for PATCH operations.
// Nil pointer = field not included in the update.
type Update struct {
	Name     *string  `json:"name,omitempty"`
	URLs     []string `json:"urls,omitempty"`
	Scanners []string `json:"scanners,omitempty"`
}
