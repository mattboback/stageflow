package statussource

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mattboback/stageflow/libs/go/models"
	"github.com/mattboback/stageflow/services/platform-api/internal/status"
)

// Config configures the orchestrator status source.
type Config struct {
	BaseURL string
	Timeout time.Duration
	Token   string
}

// Client fetches job status snapshots from orchestrator's admin API.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewClient creates an orchestrator-backed status source.
func NewClient(cfg *Config) (*Client, error) {
	if cfg == nil {
		cfg = &Config{}
	}

	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		return nil, errors.New("orchestrator status source base URL is required")
	}

	if _, err := url.ParseRequestURI(baseURL); err != nil {
		return nil, fmt.Errorf("invalid orchestrator status source URL: %w", err)
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   strings.TrimSpace(cfg.Token),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// GetJob fetches a single job from orchestrator and maps it to the API status record.
func (c *Client) GetJob(ctx context.Context, jobID string) (*status.JobRecord, error) {
	if strings.TrimSpace(jobID) == "" {
		return nil, status.ErrJobNotFound
	}

	escapedJobID := url.PathEscape(jobID)
	endpoint := c.baseURL + "/api/v1/jobs/" + escapedJobID

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("create orchestrator status request: %w", err)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request orchestrator status: %w", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode == http.StatusNotFound {
		return nil, status.ErrJobNotFound
	}

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
		if readErr != nil {
			return nil, fmt.Errorf(
				"orchestrator status request failed with %d and unreadable body: %w",
				resp.StatusCode,
				readErr,
			)
		}

		return nil, fmt.Errorf(
			"orchestrator status request failed with %d: %s",
			resp.StatusCode,
			strings.TrimSpace(string(body)),
		)
	}

	var payload struct {
		Job *models.Job `json:"job"`
	}

	if decodeErr := json.NewDecoder(resp.Body).Decode(&payload); decodeErr != nil {
		return nil, fmt.Errorf("decode orchestrator status response: %w", decodeErr)
	}

	if payload.Job == nil {
		return nil, status.ErrJobNotFound
	}

	return toJobRecord(payload.Job), nil
}

func toJobRecord(job *models.Job) *status.JobRecord {
	rec := &status.JobRecord{
		JobID:                 job.ID,
		State:                 job.State,
		InputType:             job.InputType,
		CreatedAt:             job.CreatedAt,
		UpdatedAt:             job.UpdatedAt,
		CompletedAt:           job.CompletedAt,
		Error:                 job.Error,
		LastStage:             job.LastStage,
		LastErrorDetails:      job.ErrorDetails,
		TotalPages:            job.TotalPages,
		CurrentPage:           job.CurrentPage,
		TotalViolations:       job.TotalViolations,
		ReportJSONKey:         job.ReportJSONKey,
		ReportKey:             job.ReportKey,
		ScanStageLogKey:       job.ScanStageLogKey,
		ScanRecipeKey:         job.ScanRecipeKey,
		ExtractionStageLogKey: job.ExtractionStageLogKey,
		ExtractionRecipeKey:   job.ExtractionRecipeKey,
		ProvenanceKey:         job.ProvenanceKey,
		ExpectedScanners:      status.CloneStrings(job.ExpectedScanners),
		CompletedScanners:     status.CloneStrings(job.CompletedScanners),
	}

	if len(job.ScannerResults) > 0 {
		rec.ScannerArtifacts = make(map[string]*status.ScannerArtifactRecord, len(job.ScannerResults))
		for scannerType, result := range job.ScannerResults {
			if result == nil {
				continue
			}

			rec.ScannerArtifacts[scannerType] = &status.ScannerArtifactRecord{
				ScannerType: result.ScannerType,
				ResultsKey:  result.ResultsPath,
				ReportKey:   result.ReportPath,
				StageLogKey: result.StageLogPath,
				RecipeKey:   result.RecipePath,
			}
		}
	}

	return rec
}
