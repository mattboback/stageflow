package apiclient

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
	Auth                *JobAuthInput             `json:"auth,omitempty"`
}

// JobAuthInput is the optional authentication block carried alongside a URL job
// submission. Discriminated by mode, it mirrors the shape of Provenance.auth in
// libs/contracts/provenance/schema/provenance.schema.json with one wire-level
// adaptation: storage_state ships a base64-encoded blob inline; the platform-api
// uploads it to MinIO before publishing the job, and the rest of the pipeline
// references it via artifact_key. Form recipes are inlined as-is.
type JobAuthInput struct {
	Mode         string                    `json:"mode"`
	StorageState *JobAuthStorageStateInput `json:"storage_state,omitempty"`
	Form         *JobAuthFormRecipe        `json:"form,omitempty"`
}

// JobAuthStorageStateInput carries a captured Playwright storage-state file
// inline as base64. It is ephemeral on the wire: the platform-api validates and
// uploads to MinIO under the job's prefix, and only the artifact_key is
// published/persisted afterwards.
type JobAuthStorageStateInput struct {
	ContentBase64 string `json:"content_b64"`
}

// JobAuthFormRecipe is a declarative login replay. Step values are either
// literal strings or {from_env: NAME} references resolved at scanner-runner
// execution time against an explicit allow-list derived from this recipe.
// Resolved credential values never appear in stored Provenance, the unified
// report, the scan stage log, or NATS payloads.
type JobAuthFormRecipe struct {
	LoginURL string           `json:"login_url"`
	Steps    []map[string]any `json:"steps"`
	Success  map[string]any   `json:"success"`
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
