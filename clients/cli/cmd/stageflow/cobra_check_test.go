package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/mattboback/stageflow/clients/cli/internal/apiclient"
	"github.com/mattboback/stageflow/clients/cli/internal/diffrender"
	"github.com/mattboback/stageflow/clients/cli/internal/testsupport"
	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
)

func checkTestConfiguration() checkConfiguration {
	enabled := true

	return checkConfiguration{
		URLs:           []string{"https://example.com"},
		Scanners:       []checkScanner{{ID: "axe", Enabled: &enabled}},
		Browser:        "chromium",
		HighlightStyle: "solid",
		Profile:        "staging",
	}
}

func writeCheckProject(t *testing.T, configuration checkConfiguration, baseline *report.UnifiedReportV2) string {
	t.Helper()

	project := map[string]any{
		"schema":  "stageflow/local-project@v2",
		"project": map[string]any{"id": "project-1", "configuration": configuration},
	}

	if baseline != nil {
		fingerprint := "identity-v2:browser-generated"

		project["baseline"] = map[string]any{
			"projectId":         "project-1",
			"jobId":             baseline.Meta.JobId,
			"configFingerprint": fingerprint,
			"report":            baseline,
		}
	}

	data, err := json.Marshal(project)
	testsupport.RequireNoErr(t, err)
	path := filepath.Join(t.TempDir(), "project.json")
	testsupport.RequireNoErr(t, os.WriteFile(path, data, 0o600))

	return path
}

func checkTestReport(jobID string) report.UnifiedReportV2 {
	doc := testsupport.SampleReport(jobID)
	profile := report.ScanProfileStaging
	doc.Meta.Profile = &profile
	doc.Meta.ScanContext = &report.ScanContext{
		InputType:     report.ScanContextInputTypeUrls,
		Purpose:       report.ScanContextPurposeFull,
		IdentityScope: report.ScanContextIdentityScopePath,
		ExpectedPages: []string{"page-1"},
		ScannerConfigFingerprints: map[string]string{
			"axe": "default",
		},
		ScannerConfigurations:     map[string]map[string]any{"axe": {}},
		Viewport:                  &report.ScanContextViewport{Width: 1280, Height: 720},
		VerificationEligiblePages: map[string]bool{"page-1": true},
	}
	doc.Scanners[0].Pages = []report.ScannerPageCoverage{
		{PageId: "page-1", Status: report.ScannerPageCoverageStatusSuccess},
	}

	return doc
}

func TestCheckReplaysProjectAndSavesFullReport(t *testing.T) {
	configuration := checkTestConfiguration()
	configuration.URLs = append(configuration.URLs, "https://example.com/about")
	configuration.Scanners[0].Config = map[string]any{
		"disabledRules":        []any{},
		"runOnlyTags":          []any{"wcag2a"},
		"dynamicContentWaitMs": float64(500),
	}
	projectPath := writeCheckProject(t, configuration, nil)
	doc := checkTestReport("check-job")
	doc.Meta.ScanContext.ScannerConfigurations["axe"] = configuration.Scanners[0].Config
	doc.Pages = append(doc.Pages, report.PageSummary{Id: "page-2", Url: configuration.URLs[1]})
	doc.Meta.ScanContext.ExpectedPages = append(doc.Meta.ScanContext.ExpectedPages, "page-2")
	doc.Meta.ScanContext.VerificationEligiblePages["page-2"] = true
	doc.Scanners[0].Pages = append(
		doc.Scanners[0].Pages,
		report.ScannerPageCoverage{PageId: "page-2", Status: report.ScannerPageCoverageStatusSuccess},
	)

	backend := newCLIAPIServerWithReport(t, "check-job", doc)
	defer backend.Close()

	var submitted apiclient.SubmitJobRequest

	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if err := json.NewDecoder(r.Body).Decode(&submitted); err != nil {
				t.Error(err)
			}

			writeJSONResponse(t, w, http.StatusAccepted, apiclient.SubmitJobResponse{JobID: "check-job"})

			return
		}

		backend.Config.Handler.ServeHTTP(w, r)
	}))
	defer proxy.Close()

	output := filepath.Join(t.TempDir(), "run")

	stdout, stderr, code := runCLI(
		t,
		"stageflow",
		"--api",
		proxy.URL,
		"check",
		"--project",
		projectPath,
		"--output",
		output,
	)
	if code != 0 {
		t.Fatalf("exit=%d stdout=%s stderr=%s", code, stdout, stderr)
	}

	if !reflect.DeepEqual(submitted.URLs, configuration.URLs) ||
		!reflect.DeepEqual(submitted.ScannerConfigs["axe"], configuration.Scanners[0].Config) ||
		submitted.Profile != "staging" ||
		!submitted.Screenshot ||
		submitted.Auth != nil ||
		submitted.AllowPrivateTargets {
		t.Fatalf("configuration changed: %+v", submitted)
	}

	saved, err := loadReportFile(filepath.Join(output, "report.json"))
	testsupport.RequireNoErr(t, err)

	if len(saved.Report.Issues) != len(doc.Issues) {
		t.Fatal("report truncated")
	}

	if _, err = os.Stat(filepath.Join(output, "summary.md")); err != nil {
		t.Fatal(err)
	}

	if _, err = os.Stat(filepath.Join(output, "diff.json")); !os.IsNotExist(err) {
		t.Fatal("first run unexpectedly promoted a baseline")
	}
}

func TestCheckActionableGateAndIncompleteCoverage(t *testing.T) {
	for _, name := range []string{"unchanged", "actionable", "expected", "partial", "missing context"} {
		t.Run(name, func(t *testing.T) {
			baseline, current := checkTestReport("baseline"), checkTestReport("current")
			baseline.Issues = baseline.Issues[:1]
			baseline.Summary.TotalIssues = 1
			baseline.Assessment = &report.ReportAssessment{
				Profile: report.ScanProfileStaging,
				Score:   productGateScore(100),
			}
			current.Assessment = &report.ReportAssessment{
				Profile:            report.ScanProfileStaging,
				Score:              productGateScore(100),
				DispositionByIssue: map[string]report.FindingDisposition{},
			}
			want := 0

			switch name {
			case "unchanged":
				current.Issues = current.Issues[:1]
				current.Summary.TotalIssues = 1
			case "actionable":
				want = 1
			case "expected":
				current.Assessment.DispositionByIssue[current.Issues[1].Id] = report.FindingDispositionExpected
			case "partial":
				current.Scanners[0].Status = report.ScannerStatusPartial
				want = 2
			case "missing context":
				current.Meta.ScanContext = nil
				want = 2
			}

			server := newCLIAPIServerWithReport(t, "current", current)
			defer server.Close()

			output := filepath.Join(t.TempDir(), "run")

			_, stderr, code := runCLI(
				t,
				"stageflow",
				"--api",
				server.URL,
				"check",
				"--project",
				writeCheckProject(t, checkTestConfiguration(), &baseline),
				"--output",
				output,
			)
			if code != want {
				t.Fatalf("exit=%d want=%d stderr=%s", code, want, stderr)
			}

			bytes, err := os.ReadFile(filepath.Join(output, "diff.json"))
			testsupport.RequireNoErr(t, err)

			var result diffrender.Envelope
			testsupport.RequireNoErr(t, json.Unmarshal(bytes, &result))

			if name == "expected" && (result.Regressed || len(result.New) != 1) {
				t.Fatal("expected finding must stay in raw diff without failing gate")
			}
		})
	}
}

func TestCheckRejectsUnsupportedInputBeforeSubmission(t *testing.T) {
	for _, name := range []string{"private", "credentials", "no scanners", "secret", "browser", "profile", "existing output", "verification baseline", "auth baseline", "changed configuration"} {
		t.Run(name, func(t *testing.T) {
			configuration := checkTestConfiguration()

			var baseline *report.UnifiedReportV2

			output := filepath.Join(t.TempDir(), "run")

			switch name {
			case "private":
				configuration.URLs = []string{"http://127.0.0.1"}
			case "credentials":
				configuration.URLs = []string{"https://user:secret@example.com"}
			case "no scanners":
				disabled := false
				configuration.Scanners[0].Enabled = &disabled
			case "secret":
				configuration.Scanners[0].Config = map[string]any{"nested": map[string]any{"api_key": "secret"}}
			case "browser":
				configuration.Browser = "firefox"
			case "profile":
				configuration.Profile = "guess"
			case "existing output":
				testsupport.RequireNoErr(t, os.Mkdir(output, 0o700))
			case "verification baseline", "auth baseline", "changed configuration":
				doc := checkTestReport("baseline")

				baseline = &doc
				if name == "verification baseline" {
					baseline.Meta.ScanContext.Purpose = report.ScanContextPurposeVerification
				}

				if name == "auth baseline" {
					baseline.Meta.ScanContext.Authenticated = true
				}
			}

			path := writeCheckProject(t, configuration, baseline)
			if name == "changed configuration" {
				data, err := os.ReadFile(path)
				testsupport.RequireNoErr(t, err)
				testsupport.RequireNoErr(
					t,
					os.WriteFile(
						path,
						[]byte(
							strings.ReplaceAll(
								string(data),
								`"highlightStyle":"solid","profile":"staging"`,
								`"highlightStyle":"solid","profile":"production"`,
							),
						),
						0o600,
					),
				)
			}

			requests := 0

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				requests++

				http.Error(w, "unexpected", http.StatusInternalServerError)
			}))
			defer server.Close()

			_, stderr, code := runCLI(
				t,
				"stageflow",
				"--api",
				server.URL,
				"check",
				"--project",
				path,
				"--output",
				output,
			)
			if code != 2 || requests != 0 {
				t.Fatalf("exit=%d requests=%d stderr=%s", code, requests, stderr)
			}
		})
	}
}

func TestCheckBaselineUsesActualRecordedSettings(t *testing.T) {
	for _, name := range []string{"matching", "manual", "profile", "pages", "scanner config", "viewport", "custom page", "missing coverage", "scanner set"} {
		t.Run(name, func(t *testing.T) {
			configuration := checkTestConfiguration()
			configuration.Scanners[0].Config = map[string]any{
				"custom":        map[string]any{"文字": "<&\u2028", "number": 1e-7},
				"disabledRules": []any{},
			}
			baseline := checkTestReport("baseline")
			baseline.Meta.ScanContext.ScannerConfigurations["axe"] = configuration.Scanners[0].Config

			switch name {
			case "manual":
				baseline.Scanners[0].Pages[0].Rules = []report.RuleEvaluation{
					{RuleId: "color-contrast", Outcome: report.RuleEvaluationOutcomeIncomplete},
				}
			case "profile":
				profile := report.ScanProfileProduction
				baseline.Meta.Profile = &profile
			case "pages":
				baseline.Pages[0].Url = "https://example.com/different"
			case "scanner config":
				baseline.Meta.ScanContext.ScannerConfigurations["axe"] = map[string]any{}
			case "viewport":
				baseline.Meta.ScanContext.Viewport.Width = 1000
			case "custom page":
				baseline.Meta.ScanContext.VerificationEligiblePages["page-1"] = false
			case "missing coverage":
				baseline.Scanners[0].Pages = nil
			case "scanner set":
				baseline.Scanners[0].Id = "seo"
			}

			path := writeCheckProject(t, configuration, &baseline)
			for _, schema := range []string{"stageflow/local-project@v2", "stageflow/local-project@v1"} {
				data, err := os.ReadFile(path)
				testsupport.RequireNoErr(t, err)

				data = []byte(strings.ReplaceAll(string(data), "stageflow/local-project@v2", schema))
				testsupport.RequireNoErr(t, os.WriteFile(path, data, 0o600))

				_, _, err = loadCheckProject(path)
				if (err == nil) != (name == "matching" || name == "manual") {
					t.Fatalf("schema %s case %s error %v", schema, name, err)
				}
			}
		})
	}
}
