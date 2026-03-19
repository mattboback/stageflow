package main

import (
	"encoding/json"
	"time"
)

// SubmitJobRequest represents the payload for POST /api/v1/jobs/urls.
type SubmitJobRequest struct {
	URLs                []string                  `json:"urls"`
	Modules             []string                  `json:"modules,omitempty"`
	Screenshot          bool                      `json:"screenshot"`
	AllowPrivateTargets bool                      `json:"allow_private_targets,omitempty"`
	ScannerConfigs      map[string]map[string]any `json:"scanner_configs,omitempty"`
}

// SubmitJobResponse represents the response from POST /api/v1/jobs/urls.
type SubmitJobResponse struct {
	JobID   string `json:"job_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// JobStatus represents the subset of GET /api/v1/jobs/{id} used by the CLI.
type JobStatus struct {
	ID                string          `json:"id"`
	State             string          `json:"state"`
	Error             string          `json:"error,omitempty"`
	Progress          *JobProgress    `json:"progress,omitempty"`
	ExpectedScanners  []string        `json:"expected_scanners,omitempty"`
	CompletedScanners []string        `json:"completed_scanners,omitempty"`
	RemainingScanners []string        `json:"remaining_scanners,omitempty"`
	Artifacts         json.RawMessage `json:"artifacts,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

type JobProgress struct {
	CurrentPage int `json:"current_page"`
	TotalPages  int `json:"total_pages"`
	Percentage  int `json:"percentage"`
}

// ScannersResponse matches GET /api/v1/scanners.
type ScannersResponse struct {
	Scanners   []ScannerInfo `json:"scanners"`
	Total      int           `json:"total"`
	Enabled    int           `json:"enabled"`
	Categories []string      `json:"categories,omitempty"`
}

type ScannerInfo struct {
	ID           string              `json:"id"`
	Name         string              `json:"name"`
	Version      string              `json:"version"`
	Description  string              `json:"description,omitempty"`
	Categories   []string            `json:"categories"`
	Aliases      []string            `json:"aliases,omitempty"`
	Image        string              `json:"image,omitempty"`
	Enabled      bool                `json:"enabled"`
	BuiltIn      bool                `json:"builtIn"`
	Capabilities ScannerCapabilities `json:"capabilities"`
}

type ScannerCapabilities struct {
	OutputFormats        []string `json:"outputFormats,omitempty"`
	SupportsScreenshots  bool     `json:"supportsScreenshots"`
	SupportsConcurrency  bool     `json:"supportsConcurrency"`
	RequiresBrowser      bool     `json:"requiresBrowser"`
	SupportsOffline      bool     `json:"supportsOffline"`
	MaxConcurrency       int      `json:"maxConcurrency,omitempty"`
	EstimatedTimePerPage int      `json:"estimatedTimePerPage,omitempty"`
}
