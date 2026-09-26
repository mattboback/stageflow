package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
	"github.com/mattboback/stageflow/clients/cli/internal/diffrender"
	"github.com/mattboback/stageflow/clients/cli/internal/render"
	"github.com/mattboback/stageflow/clients/cli/internal/testsupport"
)

func TestDiffLiveTarget_NormalizesAndAllowsPrivateTargetsForLocalAPI(t *testing.T) {
	server, captured := newCapturingScanAPI(t)
	defer server.Close()

	baselinePath := writeDiffBaselineReport(t)

	stdout, stderr, exitCode := runCLI(
		t,
		"stageflow",
		"--api",
		server.URL,
		"diff",
		baselinePath,
		"127.0.0.1:5173",
		"--no-stream",
	)
	if exitCode != 0 {
		t.Fatalf("exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
	}

	if !strings.Contains(stderr, "Detected private/loopback targets") {
		t.Fatalf("expected private-target notice in stderr, got %q", stderr)
	}

	captured.mu.Lock()
	defer captured.mu.Unlock()

	testsupport.RequireEqual(t, captured.hits, 1, "captured.hits")
	testsupport.RequireDeepEqual(t, captured.body.URLs, []string{"http://127.0.0.1:5173"}, "captured.body.URLs")
	testsupport.RequireEqual(t, captured.body.AllowPrivateTargets, true, "captured.body.AllowPrivateTargets")
}

func TestDiffLiveTarget_RejectsPrivateTargetsForRemoteAPI(t *testing.T) {
	baselinePath := writeDiffBaselineReport(t)

	for _, target := range []string{
		"http://127.0.0.1:5173",
		"http://10.0.0.42:3000",
	} {
		t.Run(target, func(t *testing.T) {
			stdout, stderr, exitCode := runCLI(
				t,
				"stageflow",
				"--api",
				"https://stageflow.example",
				"diff",
				baselinePath,
				target,
				"--no-stream",
			)
			if exitCode != 2 {
				t.Fatalf("exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
			}

			if !strings.Contains(stderr, "refusing to submit private/loopback targets") {
				t.Fatalf("expected private-target refusal in stderr, got %q", stderr)
			}
		})
	}
}

func TestDiffCurrentTarget_PrefersExistingSchemelessFile(t *testing.T) {
	dir := t.TempDir()

	cleanup := withWorkingDir(t, dir)
	defer cleanup()

	baselinePath := writeDiffBaselineReport(t)
	writeDiffReportFile(t, filepath.Join(dir, "example.com"), "job-current")

	stdout, stderr, exitCode := runCLI(t, "stageflow", "diff", baselinePath, "example.com")
	if exitCode != 0 {
		t.Fatalf("exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
	}

	if strings.Contains(stderr, "Job submitted") {
		t.Fatalf("expected existing file to be used without live scan, got stderr=%q", stderr)
	}
}

func writeDiffBaselineReport(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "baseline.json")
	writeDiffReportFile(t, path, "job-baseline")

	return path
}

func writeDiffReportFile(t *testing.T, path, jobID string) {
	t.Helper()

	env := render.ReportEnvelope{
		Schema: "stageflow-cli/report@v1",
		Job:    render.JobMeta{ID: jobID, State: apiclient.JobStateDone},
		Report: testsupport.SampleReport(jobID),
	}

	data, err := json.Marshal(env)
	testsupport.RequireNoErr(t, err)

	err = os.WriteFile(path, data, 0o600)
	testsupport.RequireNoErr(t, err)
}

func TestDiffRegressionStatusIndependentOfExitPolicy(t *testing.T) {
	for _, tt := range []struct {
		name      string
		newIssue  bool
		scoreDrop bool
		flags     []string
		wantExit  int
	}{
		{name: "new issue without gate", newIssue: true},
		{name: "new issue below gate", newIssue: true, flags: []string{"--fail-on-new=serious"}},
		{name: "new issue matching gate", newIssue: true, flags: []string{"--fail-on-new=any"}, wantExit: 1},
		{name: "score drop without gate", scoreDrop: true},
		{name: "score drop with new issue gate", scoreDrop: true, flags: []string{"--fail-on-new=any"}},
		{name: "score drop with regression gate", scoreDrop: true, flags: []string{"--fail-on-regression"}, wantExit: 1},
		{name: "unchanged with regression gate", flags: []string{"--fail-on-regression"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			baseline := render.ReportEnvelope{
				Schema: "stageflow-cli/report@v1",
				Report: testsupport.SampleReport("baseline"),
			}

			current := render.ReportEnvelope{
				Schema: "stageflow-cli/report@v1",
				Report: testsupport.SampleReport("current"),
			}
			if tt.newIssue {
				baseline.Report.Issues = baseline.Report.Issues[:1]
				baseline.Report.Summary.TotalIssues = 1
			}

			if tt.scoreDrop {
				score := *baseline.Report.Summary.Score - 5
				current.Report.Summary.Score = &score
			}

			dir := t.TempDir()
			paths := []string{filepath.Join(dir, "baseline.json"), filepath.Join(dir, "current.json")}

			for i, env := range []render.ReportEnvelope{baseline, current} {
				data, err := json.Marshal(env)
				testsupport.RequireNoErr(t, err)
				testsupport.RequireNoErr(t, os.WriteFile(paths[i], data, 0o600))
			}

			args := append([]string{"stageflow", "diff", paths[0], paths[1], "--format=json"}, tt.flags...)

			stdout, stderr, exit := runCLI(t, args...)
			if exit != tt.wantExit {
				t.Fatalf("exit=%d, want %d; stderr=%q", exit, tt.wantExit, stderr)
			}

			var result diffrender.Envelope
			testsupport.RequireNoErr(t, json.Unmarshal([]byte(stdout), &result))
			testsupport.RequireEqual(t, result.Regressed, tt.newIssue || tt.scoreDrop, "regressed")
		})
	}
}

func TestDiffRejectsInvalidReportInputs(t *testing.T) {
	rawReport, err := json.Marshal(testsupport.SampleReport("raw"))
	testsupport.RequireNoErr(t, err)

	for _, tt := range []struct{ name, data string }{
		{"empty object", `{}`},
		{"null", `null`},
		{"raw report", string(rawReport)},
		{"missing report", `{"schema":"stageflow-cli/report@v1"}`},
		{"null report", `{"schema":"stageflow-cli/report@v1","report":null}`},
		{"empty report", `{"schema":"stageflow-cli/report@v1","report":{}}`},
		{"wrong schema", `{"schema":"stageflow/diff@v1","report":` + string(rawReport) + `}`},
		{"missing schema", `{"report":` + string(rawReport) + `}`},
		{"malformed JSON", `{`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			valid := writeDiffBaselineReport(t)
			invalid := filepath.Join(t.TempDir(), "invalid.json")
			testsupport.RequireNoErr(t, os.WriteFile(invalid, []byte(tt.data), 0o600))

			for _, paths := range [][2]string{{invalid, valid}, {valid, invalid}} {
				stdout, stderr, exit := runCLI(t, "stageflow", "diff", paths[0], paths[1], "--format=json")
				if exit != 2 || stdout != "" || !strings.Contains(stderr, "parse ") {
					t.Fatalf("exit=%d stdout=%q stderr=%q", exit, stdout, stderr)
				}
			}
		})
	}
}

func TestDiffRequiresCompleteRenderedReport(t *testing.T) {
	for _, tt := range []struct {
		name     string
		opts     render.Options
		empty    bool
		wantExit int
	}{
		{name: "capped", opts: render.Options{MaxIssues: 1}, wantExit: 2},
		{name: "severity filtered", opts: render.Options{Severities: []string{"critical"}}, wantExit: 2},
		{name: "category filtered", opts: render.Options{Categories: []string{"accessibility"}}, wantExit: 2},
		{name: "summary only", opts: render.Options{SummaryOnly: true}, wantExit: 2},
		{name: "complete", opts: render.Options{MaxIssues: 0}},
		{name: "empty", empty: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			doc := testsupport.SampleReport("rendered")
			doc.Issues[0].Category = testsupport.StringPtr("accessibility")

			doc.Issues[1].Category = testsupport.StringPtr("seo")
			if tt.empty {
				doc.Issues = doc.Issues[:0]
				doc.Summary.TotalIssues = 0
			}

			var output bytes.Buffer

			opts := tt.opts
			opts.Format = render.FormatJSON
			testsupport.RequireNoErr(t, render.UnifiedReport(
				&output,
				"http://localhost:8080",
				apiclient.JobStatus{ID: "rendered", State: apiclient.JobStateDone},
				doc,
				opts,
			))

			renderedPath := filepath.Join(t.TempDir(), "rendered.json")
			testsupport.RequireNoErr(t, os.WriteFile(renderedPath, output.Bytes(), 0o600))

			completePath := writeDiffBaselineReport(t)
			for _, paths := range [][2]string{{renderedPath, completePath}, {completePath, renderedPath}} {
				stdout, stderr, exit := runCLI(t, "stageflow", "diff", paths[0], paths[1], "--format=json")
				if exit != tt.wantExit {
					t.Fatalf("exit=%d, want %d; stdout=%q stderr=%q", exit, tt.wantExit, stdout, stderr)
				}

				if tt.wantExit == 2 && (stdout != "" || !strings.Contains(stderr, "--format json --max-issues 0")) {
					t.Fatalf("expected rejection with recovery instructions; stdout=%q stderr=%q", stdout, stderr)
				}
			}
		})
	}
}
