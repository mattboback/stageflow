package statussource

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mattboback/stageflow/packages/shared-go/models"
	"github.com/mattboback/stageflow/platform/api/internal/status"
)

func TestClientGetJobNotFound(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := NewClient(&Config{BaseURL: server.URL, Timeout: time.Second})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = client.GetJob(context.Background(), "missing")
	if !errors.Is(err, status.ErrJobNotFound) {
		t.Fatalf("expected status.ErrJobNotFound, got %v", err)
	}
}

func TestClientGetJobMapsRecord(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"job": &models.Job{
				ID:              "job-1",
				State:           models.JobStateScanning,
				InputType:       models.JobInputTypeURLs,
				CreatedAt:       now,
				UpdatedAt:       now,
				Error:           "warning",
				ErrorDetails:    "detail",
				LastStage:       "scanning",
				TotalPages:      12,
				CurrentPage:     5,
				TotalViolations: 7,
				ReportJSONKey:   "job-1/report.json",
				ReportKey:       "job-1/report.html",
				ScanStageLogKey: "job-1/stage.log",
				ScanRecipeKey:   "job-1/recipe.json",
				ProvenanceKey:   "job-1/provenance.json",
				ScannerResults: map[string]*models.ScannerResult{
					"axe": {
						ScannerType: "axe",
						ResultsPath: "job-1/axe/results.json",
						ReportPath:  "job-1/axe/report.html",
					},
				},
			},
		})
	}))
	defer server.Close()

	client, err := NewClient(&Config{BaseURL: server.URL, Timeout: time.Second})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	rec, err := client.GetJob(context.Background(), "job-1")
	if err != nil {
		t.Fatalf("GetJob: %v", err)
	}

	if rec.JobID != "job-1" || rec.CurrentPage != 5 || rec.TotalPages != 12 {
		t.Fatalf("unexpected record fields: %+v", rec)
	}

	axe := rec.ScannerArtifacts["axe"]
	if axe == nil || axe.ResultsKey != "job-1/axe/results.json" {
		t.Fatalf("expected scanner artifact mapping, got %+v", rec.ScannerArtifacts)
	}
}
