package main

import (
	"testing"

	"github.com/mattboback/stageflow/clients/cli/internal/diffrender"
	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
)

func TestProductDiffGateDefaultsRawAndActionableIsExplicit(t *testing.T) {
	issues := []report.IssueDetail{{Id: "expected"}, {Id: "review"}, {Id: "actionable"}}
	before := report.UnifiedReportV2{Assessment: &report.ReportAssessment{Score: productGateScore(80)}}
	after := report.UnifiedReportV2{
		Assessment: &report.ReportAssessment{
			Score: productGateScore(100),
			DispositionByIssue: map[string]report.FindingDisposition{
				"expected":   report.FindingDispositionExpected,
				"review":     report.FindingDispositionNeedsReview,
				"actionable": report.FindingDispositionActionable,
			},
		},
	}
	raw := diffrender.Envelope{New: issues, Delta: diffrender.Delta{NewIssues: 3, ScoreDelta: productGateScore(-10)}}

	for _, tc := range []struct {
		policy       string
		count, score int
	}{{"", 3, -10}, {"raw", 3, -10}, {"actionable", 1, 20}} {
		t.Run(tc.policy, func(t *testing.T) {
			gated, err := actionableDiffGate(tc.policy, before, after, raw)
			if err != nil {
				t.Fatal(err)
			}

			if gated.Delta.NewIssues != tc.count || len(gated.New) != tc.count || gated.Delta.ScoreDelta == nil ||
				*gated.Delta.ScoreDelta != tc.score {
				t.Fatalf("wrong policy result: %+v", gated)
			}

			if raw.Delta.NewIssues != 3 || len(raw.New) != 3 || *raw.Delta.ScoreDelta != -10 {
				t.Fatal("gate policy mutated raw diff")
			}
		})
	}

	if _, err := actionableDiffGate("invalid", before, after, raw); err == nil {
		t.Fatal("unknown gate policy accepted")
	}
}

func TestProductActionableGateDoesNotRecreateIncompatibleScoreDelta(t *testing.T) {
	before := report.UnifiedReportV2{Assessment: &report.ReportAssessment{Score: productGateScore(80)}}
	after := report.UnifiedReportV2{Assessment: &report.ReportAssessment{Score: productGateScore(100)}}

	gated, err := actionableDiffGate("actionable", before, after, diffrender.Envelope{})
	if err != nil {
		t.Fatal(err)
	}

	if gated.Delta.ScoreDelta != nil {
		t.Fatal("assessment scores bypassed incompatible comparison context")
	}
}

func productGateScore(value int) *int { return &value }
