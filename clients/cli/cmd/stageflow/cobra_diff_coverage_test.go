package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mattboback/stageflow/clients/cli/internal/diffrender"
	"github.com/mattboback/stageflow/clients/cli/internal/render"
	"github.com/mattboback/stageflow/clients/cli/internal/testsupport"
	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
)

type diffCoverageCase struct {
	name     string
	pages    bool
	scanners bool
}

type diffCoverageGate struct {
	name     string
	flags    []string
	noNew    bool
	wantExit int
}

func TestDiffIncompleteCoverageReasonsAndGates(t *testing.T) {
	gates := []diffCoverageGate{
		{name: "default", wantExit: 2},
		{name: "incomplete overrides regression", flags: []string{"--fail-on-regression"}, wantExit: 2},
		{name: "accepted", flags: []string{"--allow-partial"}},
		{name: "accepted below threshold", flags: []string{"--allow-partial", "--fail-on-new=serious"}},
		{name: "accepted new issue", flags: []string{"--allow-partial", "--fail-on-new=any"}, wantExit: 1},
		{name: "accepted regression", flags: []string{"--allow-partial", "--fail-on-regression"}, wantExit: 1},
		{name: "incomparable score drop", flags: []string{"--allow-partial", "--fail-on-regression"}, noNew: true},
	}

	for _, coverage := range []diffCoverageCase{
		{name: "page only", pages: true},
		{name: "scanner only", scanners: true},
		{name: "combined", pages: true, scanners: true},
	} {
		t.Run(coverage.name, func(t *testing.T) {
			for _, gate := range gates {
				t.Run(gate.name, func(t *testing.T) {
					runIncompleteDiffCoverageCase(t, coverage, gate)
				})
			}
		})
	}
}

func runIncompleteDiffCoverageCase(t *testing.T, coverage diffCoverageCase, gate diffCoverageGate) {
	t.Helper()

	baseline, current := incompleteDiffCoverageReports(coverage, gate.noNew)
	paths := writeCoverageReports(t, baseline, current)

	for _, format := range []string{"text", "json"} {
		args := append([]string{"stageflow", "diff", paths[0], paths[1], "--format=" + format}, gate.flags...)
		stdout, stderr, exit := runCLI(t, args...)

		if exit != gate.wantExit {
			t.Fatalf("%s: exit=%d want=%d stdout=%q stderr=%q", format, exit, gate.wantExit, stdout, stderr)
		}

		assertIncompleteDiffStderr(t, coverage, gate.wantExit, stderr)
		assertIncompleteDiffResult(t, format, gate.noNew, stdout)
	}
}

func incompleteDiffCoverageReports(
	coverage diffCoverageCase,
	noNew bool,
) (report.UnifiedReportV2, report.UnifiedReportV2) {
	baseline := testsupport.SampleReport("baseline")
	current := testsupport.SampleReport("current")
	baseline.Issues = baseline.Issues[:1]
	current.Summary.Score = productGateScore(50)

	if noNew {
		current.Issues = current.Issues[:1]
	}

	if coverage.pages {
		baseline.Pages = append(baseline.Pages, report.PageSummary{Id: "page-2", Url: "https://example.com/omitted"})
		omitted := baseline.Issues[0]
		omitted.Id, omitted.PageId, omitted.PageUrl = "omitted-issue", "page-2", "https://example.com/omitted"
		baseline.Issues = append(baseline.Issues, omitted)
	}

	if coverage.scanners {
		baseline.Scanners = append(baseline.Scanners, report.ScannerSummary{
			Id: "unavailable", Status: report.ScannerStatusSuccess,
		})
		current.Scanners = append(current.Scanners, report.ScannerSummary{
			Id: "unavailable", Status: report.ScannerStatusPartial,
		})
		skipped := baseline.Issues[0]
		skipped.Id, skipped.Scanner = "skipped-issue", "unavailable"
		baseline.Issues = append(baseline.Issues, skipped)
	}

	baseline.Summary.TotalIssues = len(baseline.Issues)
	current.Summary.TotalIssues = len(current.Issues)

	return baseline, current
}

func assertIncompleteDiffStderr(t *testing.T, coverage diffCoverageCase, wantExit int, stderr string) {
	t.Helper()

	if wantExit != 2 {
		if stderr != "" {
			t.Fatalf("accepted comparison emitted an error: %q", stderr)
		}

		return
	}

	if !strings.Contains(stderr, "incomplete comparison: ") || !strings.Contains(stderr, "use --allow-partial") {
		t.Fatalf("missing incomplete comparison explanation: %q", stderr)
	}

	if coverage.pages && !strings.Contains(stderr, "pages not compared: 1") {
		t.Fatalf("missing page reason: %q", stderr)
	}

	if coverage.scanners && !strings.Contains(stderr, "scanners not compared: unavailable") {
		t.Fatalf("missing scanner reason: %q", stderr)
	}
}

func assertIncompleteDiffResult(t *testing.T, format string, noNew bool, stdout string) {
	t.Helper()

	if format == "text" {
		if !strings.Contains(stdout, "Score: N/A") || !strings.Contains(stdout, "Fixed issues: 0") {
			t.Fatalf("partial comparison reported a score or false fixes: %q", stdout)
		}

		return
	}

	var result diffrender.Envelope

	testsupport.RequireNoErr(t, json.Unmarshal([]byte(stdout), &result))

	if result.Baseline.Score != nil || result.Current.Score != nil || result.Delta.ScoreDelta != nil ||
		len(result.Fixed) != 0 || result.Delta.FixedIssues != 0 || result.Regressed == noNew {
		t.Fatalf("partial comparison produced a score, false fixes or wrong regression: %+v", result)
	}
}

func writeCoverageReports(t *testing.T, baseline, current report.UnifiedReportV2) [2]string {
	t.Helper()

	dir := t.TempDir()
	paths := [2]string{filepath.Join(dir, "baseline.json"), filepath.Join(dir, "current.json")}

	for i, doc := range []report.UnifiedReportV2{baseline, current} {
		data, err := json.Marshal(render.ReportEnvelope{Schema: "stageflow-cli/report@v1", Report: doc})
		testsupport.RequireNoErr(t, err)
		testsupport.RequireNoErr(t, os.WriteFile(paths[i], data, 0o600))
	}

	return paths
}
