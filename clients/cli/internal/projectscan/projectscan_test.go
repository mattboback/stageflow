package projectscan

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
	"github.com/mattboback/stageflow/clients/cli/internal/exitcode"
	"github.com/mattboback/stageflow/clients/cli/internal/render"
	"github.com/mattboback/stageflow/clients/cli/internal/testsupport"
	"github.com/mattboback/stageflow/libs/go/diff"
)

func TestInterpretDiffErrorMissingBaseline(t *testing.T) {
	t.Parallel()

	state, matched := interpretDiffError(
		"demo-site",
		"job-123",
		errors.New(`API request failed with status 404: {"error":"No baseline set for project"}`),
	)
	if !matched {
		t.Fatal("expected missing-baseline error to match")
	}
	testsupport.RequireEqual(t, state.baseline.Status, BaselineStatusMissing, "baseline status")
	testsupport.RequireEqual(
		t,
		state.baseline.PromoteCommand,
		"Promote this scan: stageflow project promote demo-site --job-id job-123",
		"promote command",
	)
}

func TestInterpretDiffErrorCurrentBaseline(t *testing.T) {
	t.Parallel()

	state, matched := interpretDiffError(
		"demo-site",
		"job-123",
		errors.New(`API request failed with status 400: {"error":"Cannot diff against self"}`),
	)
	if !matched {
		t.Fatal("expected current-baseline error to match")
	}
	testsupport.RequireEqual(t, state.baseline.Status, BaselineStatusCurrent, "baseline status")
	testsupport.RequireEqual(
		t,
		state.baseline.Message,
		"This scan is the current baseline. Run a new scan to see a diff.",
		"baseline message",
	)
}

func TestWriteResultJSONCombinesReportDiffAndDecision(t *testing.T) {
	t.Parallel()

	scoreDelta := -10
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI != "/api/v1/jobs/job%2F123/diff" {
			t.Errorf("RequestURI = %q", r.RequestURI)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(diff.Result{
			Schema:  "stageflow/diff@v1",
			Delta:   diff.Delta{ScoreDelta: &scoreDelta, NewIssues: 1},
			New:     testsupport.SampleReport("job/123").Issues[:1],
			Current: diff.ScanMeta{JobID: "job/123"},
		}); err != nil {
			t.Errorf("encode diff: %v", err)
		}
	}))
	defer server.Close()

	reportDoc := testsupport.SampleReport("job/123")
	status := apiclient.JobStatus{ID: "job/123", State: apiclient.JobStateDone}
	var stdout bytes.Buffer

	err := WriteResult(
		context.Background(),
		apiclient.NewClient(server.URL, "", server.Client()),
		status,
		reportDoc,
		Options{
			APIBaseURL: server.URL,
			Slug:       "demo-site",
			JobID:      "job/123",
			Format:     render.FormatJSON,
			Report:     render.Options{Format: render.FormatJSON, SummaryOnly: true},
			Stdout:     &stdout,
			Stderr:     &bytes.Buffer{},
		},
	)
	var exitErr exitcode.Error
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Fatalf("error = %v", err)
	}

	var payload Envelope
	testsupport.RequireNoErr(t, json.Unmarshal(stdout.Bytes(), &payload))
	testsupport.RequireEqual(t, payload.Schema, "stageflow-cli/project-scan@v1", "schema")
	testsupport.RequireEqual(t, payload.Project.Baseline.Status, BaselineStatusAvailable, "baseline status")
	testsupport.RequireEqual(t, payload.Decision.Regressed, true, "regressed")
	testsupport.RequireEqual(t, payload.Decision.Passed, false, "passed")
	if payload.Diff == nil {
		t.Fatal("expected diff")
	}
}
