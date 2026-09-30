package storage

import (
	"encoding/json"
	"os"
	"testing"

	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
	"github.com/mattboback/stageflow/libs/go/models"
)

func TestSpecialistOwnershipRequiresSuccessfulCoverage(t *testing.T) {
	for _, status := range []report.ScannerStatus{"", report.ScannerStatusSuccess, report.ScannerStatusPartial, report.ScannerStatusFailed, report.ScannerStatusSkipped} {
		t.Run(string(status), func(t *testing.T) {
			a := newReportAggregation(&Aggregator{}, &models.Job{ID: "coverage"})
			if status != "" {
				a.scannerStats["axe"] = &report.ScannerSummary{Id: "axe", Status: status}
			}

			a.issues = []report.IssueDetail{
				{Scanner: "lighthouse", RuleId: "color-contrast", Category: stringPtr("accessibility")},
				{Scanner: "seo", RuleId: "seo-missing-title"},
				{Scanner: "seo", RuleId: "seo-missing-language"},
				{Scanner: "seo", RuleId: "seo-images-missing-alt"},
				{Scanner: "seo", RuleId: "seo-heading-hierarchy"},
				{Scanner: "link-checker", RuleId: "link-checker-no-text-links"},
				// These have different semantics and must survive axe success.
				{Scanner: "seo", RuleId: "seo-missing-description"},
				{Scanner: "seo", RuleId: "seo-missing-h1"},
				{Scanner: "seo", RuleId: "seo-multiple-h1"},
				{Scanner: "seo", RuleId: "seo-missing-viewport"},
				{Scanner: "link-checker", RuleId: "link-checker-empty-links"},
				{Scanner: "lighthouse", RuleId: "meta-description", Category: stringPtr("seo")},
			}
			got := a.coverageFilteredIssues(nil)

			want := len(a.issues)
			if status == report.ScannerStatusSuccess {
				want = 6
			}

			if len(got) != want {
				t.Fatalf("status %s: got %d issues, want %d", status, len(got), want)
			}
		})
	}
}

func TestRobotsFindingOncePerOrigin(t *testing.T) {
	a := newReportAggregation(&Aggregator{}, &models.Job{})

	pages := []report.PageSummary{
		{Id: "a", Url: "https://one.example/"},
		{Id: "b", Url: "https://one.example/about"},
		{Id: "c", Url: "https://two.example/"},
	}
	for _, page := range pages {
		a.issues = append(a.issues, report.IssueDetail{Scanner: "lighthouse", RuleId: "robots-txt", PageId: page.Id})
	}

	got := a.coverageFilteredIssues(pages)
	if len(got) != 2 || got[0].PageId != "a" {
		t.Fatalf("want stable root-page representative and one per origin, got %#v", got)
	}

	a.issues[0], a.issues[1] = a.issues[1], a.issues[0]

	got = a.coverageFilteredIssues(pages)
	if len(got) != 2 || got[0].PageId != "a" {
		t.Fatalf("input order moved robots finding to another page: %#v", got)
	}
}

func TestCompleteZeroFindingReportScores100(t *testing.T) {
	a := newReportAggregation(&Aggregator{}, &models.Job{})
	a.scannerStats["seo"] = &report.ScannerSummary{Id: "seo", Status: report.ScannerStatusSuccess}

	got := a.buildReport()
	if got.Summary.Score == nil || *got.Summary.Score != 100 || stringValue(got.Summary.ScoreGrade) != "A+" {
		t.Fatalf("zero findings should score 100/A+: %#v", got.Summary)
	}

	a.scannerStats["seo"].Status = report.ScannerStatusPartial
	if partial := a.buildReport(); partial.Summary.Score != nil {
		t.Fatal("incomplete zero-finding report must not claim a score")
	}
}

// The shared fixture is generated and verified by the real-browser public
// scanner + formatter test, so scanner-prefixed wire rule IDs cannot drift from
// the aggregation coverage policy unnoticed.
func TestSpecialistOwnershipFromActualScannerReports(t *testing.T) {
	data, readErr := os.ReadFile("testdata/specialist-scanner-output.json")
	if readErr != nil {
		t.Fatal(readErr)
	}

	var docs map[string]report.UnifiedReportV2
	if decodeErr := json.Unmarshal(data, &docs); decodeErr != nil {
		t.Fatal(decodeErr)
	}

	for _, status := range []report.ScannerStatus{"", report.ScannerStatusSuccess, report.ScannerStatusPartial, report.ScannerStatusFailed, report.ScannerStatusSkipped} {
		t.Run(string(status), func(t *testing.T) {
			a := newReportAggregation(&Aggregator{}, &models.Job{ID: "coverage-fixture"})
			for scannerID, doc := range docs {
				a.issues = append(a.issues, doc.Issues...)
				a.absorbPages(scannerID, doc.Pages)
				a.scannerStats[scannerID] = &report.ScannerSummary{Id: scannerID, Status: report.ScannerStatusSuccess}
			}

			if status != "" {
				a.scannerStats["axe"] = &report.ScannerSummary{Id: "axe", Status: status}
			}

			assertSpecialistFixtureCoverage(t, a.buildReport(), status)
		})
	}
}

func assertSpecialistFixtureCoverage(t *testing.T, result report.UnifiedReportV2, status report.ScannerStatus) {
	t.Helper()

	present := make(map[string]bool)
	for _, issue := range result.Issues {
		present[issue.RuleId] = true
	}

	ownedRules := []string{
		"seo-missing-title",
		"seo-missing-language",
		"seo-images-missing-alt",
		"seo-heading-hierarchy",
		"link-checker-no-text-links",
	}
	for _, ruleID := range ownedRules {
		if present[ruleID] != (status != report.ScannerStatusSuccess) {
			t.Errorf("axe=%s: wire rule %s present=%v", status, ruleID, present[ruleID])
		}
	}

	if !present["seo-missing-description"] || !present["seo-missing-h1"] || !present["seo-missing-viewport"] {
		t.Error("independent SEO checks disappeared")
	}
}

func TestFailedVerificationRetainsTargetedScopeWithoutScannerContext(t *testing.T) {
	for _, identity := range []string{"", "path", "origin"} {
		t.Run(identity, func(t *testing.T) {
			job := &models.Job{ID: "targeted-failed", InputType: models.JobInputTypeZip,
				Config: models.JobConfig{
					IdentityScope: identity,
					Auth:          json.RawMessage(`{"mode":"storage_state"}`),
					Verification: &models.VerificationConfig{
						SourceJobID: "original",
						PageID:      "affected-page",
						Scanner:     "axe",
						RuleID:      "image-alt",
					},
				}}
			a := newReportAggregation(&Aggregator{}, job)
			a.scannerStats["axe"] = &report.ScannerSummary{Id: "axe", Status: report.ScannerStatusFailed}
			result := a.buildReport()

			assertFailedVerificationContext(t, result.Meta.ScanContext, identity)

			if result.Summary.Score != nil || result.Summary.ScoreGrade != nil {
				t.Fatal("targeted failed scan received a site score")
			}

			originalContext := &report.ScanContext{
				Purpose:                   report.ScanContextPurposeVerification,
				ScannerConfigFingerprints: map[string]string{"axe": "captured"},
			}

			a.scanContext = originalContext
			if a.buildReport().Meta.ScanContext != originalContext {
				t.Fatal("recorded scanner context was replaced")
			}
		})
	}

	ordinary := newReportAggregation(&Aggregator{}, &models.Job{ID: "full-failed"}).buildReport()
	if ordinary.Meta.ScanContext != nil {
		t.Fatal("untargeted run received invented verification context")
	}
}

func assertFailedVerificationContext(t *testing.T, context *report.ScanContext, identity string) {
	t.Helper()

	if context == nil || context.Purpose != report.ScanContextPurposeVerification ||
		len(context.ExpectedPages) != 1 ||
		context.ExpectedPages[0] != "affected-page" ||
		context.InputType != report.ScanContextInputTypeZip ||
		!context.Authenticated {
		t.Fatalf("failed targeted run lost scope: %#v", context)
	}

	wantIdentity := report.ScanContextIdentityScopePath
	if identity == "origin" {
		wantIdentity = report.ScanContextIdentityScopeOrigin
	}

	if context.IdentityScope != wantIdentity || context.ScannerConfigFingerprints == nil ||
		len(context.ScannerConfigFingerprints) != 0 ||
		len(context.PageConfigFingerprints) != 0 {
		t.Fatalf("fallback invented execution coverage: %#v", context)
	}
}
