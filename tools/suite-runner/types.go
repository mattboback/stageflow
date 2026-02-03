package main

// Suite defines the YAML contract for multi-domain submissions.
type Suite struct {
	Domains     []string   `yaml:"domains"`
	Modules     []string   `yaml:"modules"`
	Screenshot  bool       `yaml:"screenshot"`
	Thresholds  Thresholds `yaml:"thresholds"`
	TimeoutSec  int        `yaml:"timeout_seconds"`
	StreamRetry int        `yaml:"stream_retry_seconds"`
}

// Thresholds represent simple gating rules for accessibility counts.
type Thresholds struct {
	MaxCritical *int `yaml:"max_critical"`
	MaxSerious  *int `yaml:"max_serious"`
	MaxTotal    *int `yaml:"max_total"`
}

// submitResponse matches /api/v1/jobs/urls response.
type submitResponse struct {
	JobID string `json:"job_id"`
}

// jobStatus mirrors the platform API JobStatus payload enough to evaluate outcomes.
type jobStatus struct {
	ID        string           `json:"id"`
	State     string           `json:"state"`
	Error     string           `json:"error"`
	Artifacts *artifactPayload `json:"artifacts"`
}

type artifactPayload struct {
	ResultsJSON string `json:"results_json"`
	ReportHTML  string `json:"report_html"`
}

// resultsSummary extracts the fields we need from results.json.
type resultsSummary struct {
	Summary struct {
		TotalViolations int            `json:"totalViolations"`
		ByImpact        map[string]int `json:"byImpact"`
	} `json:"summary"`
}

type jobOutcome struct {
	Domain          string
	JobID           string
	State           string
	Error           string
	TotalViolations int
	Critical        int
	Serious         int
	Passed          bool
}
