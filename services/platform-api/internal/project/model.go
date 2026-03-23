package project

import "time"

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
