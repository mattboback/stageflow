package diff

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
)

type certaintyCase struct {
	Name            string                           `json:"name"`
	Rules           []report.RuleEvaluation          `json:"rules"`
	CurrentSubjects []string                         `json:"currentSubjects"`
	OmitScanner     bool                             `json:"omitScanner"`
	ScannerStatus   report.ScannerStatus             `json:"scannerStatus"`
	PageStatus      report.ScannerPageCoverageStatus `json:"pageStatus"`
	Expected        struct {
		Fixed     int  `json:"fixed"`
		Unchanged int  `json:"unchanged"`
		New       int  `json:"new"`
		Limited   bool `json:"limited"`
	} `json:"expected"`
}

func TestSharedComparisonCertainty(t *testing.T) {
	var fixture struct {
		Baseline report.UnifiedReportV2 `json:"baseline"`
		Cases    []certaintyCase        `json:"cases"`
	}

	data, readErr := os.ReadFile(filepath.Join("..", "..", "..", "qa", "fixtures", "comparison-certainty.json"))
	if readErr != nil {
		t.Fatal(readErr)
	}

	if parseErr := json.Unmarshal(data, &fixture); parseErr != nil {
		t.Fatal(parseErr)
	}

	for _, scenario := range fixture.Cases {
		t.Run(scenario.Name, func(t *testing.T) {
			current := certaintyCurrent(t, fixture.Baseline, scenario)
			result := ComputeDiff("original-job", fixture.Baseline, "current-job", current)
			assertCertaintyDelta(t, result, scenario.Expected.Fixed, scenario.Expected.New, scenario.Expected.Unchanged)

			reverse := ComputeDiff("current-job", current, "original-job", fixture.Baseline)
			assertCertaintyDelta(
				t,
				reverse,
				scenario.Expected.New,
				scenario.Expected.Fixed,
				scenario.Expected.Unchanged,
			)

			limited := len(result.NotComparedPages) > 0 || len(result.SkippedScanners) > 0
			if limited != scenario.Expected.Limited {
				t.Fatalf("limited=%v, want %v", limited, scenario.Expected.Limited)
			}

			if limited && (result.Delta.ScoreDelta != nil || reverse.Delta.ScoreDelta != nil) {
				t.Fatal("incomplete comparison exposed score delta")
			}
		})
	}
}

func assertCertaintyDelta(t *testing.T, result Result, fixed, newIssues, unchanged int) {
	t.Helper()

	if result.Delta.FixedIssues != fixed || result.Delta.UnchangedIssues != unchanged ||
		result.Delta.NewIssues != newIssues {
		t.Fatalf("unexpected delta: %+v", result.Delta)
	}
}

func certaintyCurrent(t *testing.T, baseline report.UnifiedReportV2, scenario certaintyCase) report.UnifiedReportV2 {
	t.Helper()

	encoded, marshalErr := json.Marshal(baseline)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}

	var current report.UnifiedReportV2
	if parseErr := json.Unmarshal(encoded, &current); parseErr != nil {
		t.Fatal(parseErr)
	}

	current.Issues = nil
	if len(scenario.CurrentSubjects) > 0 {
		issue := baseline.Issues[0]
		issue.Id = "renamed-issue"
		issue.ElementCount = len(scenario.CurrentSubjects)
		issue.Occurrences = nil

		for _, subject := range scenario.CurrentSubjects {
			issue.Occurrences = append(issue.Occurrences, report.IssueOccurrence{ElementId: strPtr(subject)})
		}

		current.Issues = []report.IssueDetail{issue}
	}

	scanner := &current.Scanners[0]
	if scenario.ScannerStatus != "" {
		scanner.Status = scenario.ScannerStatus
	}

	if scenario.PageStatus != "" {
		scanner.Pages[0].Status = scenario.PageStatus
	}

	scanner.Pages[0].Rules = scenario.Rules
	if scenario.OmitScanner {
		current.Scanners = nil
	}

	return current
}
