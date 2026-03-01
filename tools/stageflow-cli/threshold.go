package main

import (
	"fmt"
	"strings"

	report "github.com/mattboback/stageflow/packages/contracts/report/generated/go"
)

type thresholdResult struct {
	Evaluated bool
	Passed    bool
	Detail    string
}

func evaluateThresholds(
	doc report.UnifiedReportV2,
	maxCritical, maxSerious, maxTotal int,
) thresholdResult {
	if maxCritical < 0 && maxSerious < 0 && maxTotal < 0 {
		return thresholdResult{Passed: true}
	}

	critical := doc.Summary.BySeverity.Critical
	if maxCritical >= 0 && critical > maxCritical {
		return thresholdResult{
			Evaluated: true,
			Passed:    false,
			Detail:    fmt.Sprintf("critical: %d > %d", critical, maxCritical),
		}
	}

	serious := doc.Summary.BySeverity.Serious
	if maxSerious >= 0 && serious > maxSerious {
		return thresholdResult{
			Evaluated: true,
			Passed:    false,
			Detail:    fmt.Sprintf("serious: %d > %d", serious, maxSerious),
		}
	}

	total := doc.Summary.TotalIssues
	if maxTotal >= 0 && total > maxTotal {
		return thresholdResult{
			Evaluated: true,
			Passed:    false,
			Detail:    fmt.Sprintf("total: %d > %d", total, maxTotal),
		}
	}

	return thresholdResult{
		Evaluated: true,
		Passed:    true,
		Detail:    formatThresholdSummary(maxCritical, maxSerious, maxTotal),
	}
}

func formatThresholdSummary(maxCritical, maxSerious, maxTotal int) string {
	parts := make([]string, 0, 3)
	if maxCritical >= 0 {
		parts = append(parts, fmt.Sprintf("critical<=%d", maxCritical))
	}

	if maxSerious >= 0 {
		parts = append(parts, fmt.Sprintf("serious<=%d", maxSerious))
	}

	if maxTotal >= 0 {
		parts = append(parts, fmt.Sprintf("total<=%d", maxTotal))
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, ", ")
}
