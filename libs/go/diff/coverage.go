package diff

import (
	"net/url"
	"reflect"
	"sort"

	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
)

func reportPages(doc report.UnifiedReportV2) map[string]bool {
	pages := make(map[string]bool)
	for _, page := range doc.Pages {
		pages[page.Id] = true
	}

	for _, issue := range doc.Issues {
		pages[issue.PageId] = true
	}

	if doc.Meta.ScanContext != nil {
		for _, id := range doc.Meta.ScanContext.ExpectedPages {
			pages[id] = true
		}
	}

	return pages
}

func commonPages(baseline, current report.UnifiedReportV2) (map[string]bool, []string) {
	before, after := reportPages(baseline), reportPages(current)
	common := make(map[string]bool)
	excluded := make(map[string]struct{})

	for id := range before {
		if after[id] {
			common[id] = true
		} else {
			excluded[id] = struct{}{}
		}
	}

	for id := range after {
		if !before[id] {
			excluded[id] = struct{}{}
		}
	}

	if baseline.Meta.ScanContext != nil && current.Meta.ScanContext != nil {
		for id := range common {
			left, right := baseline.Meta.ScanContext.PageConfigFingerprints[id], current.Meta.ScanContext.PageConfigFingerprints[id]
			if left != right || baseline.Meta.ScanContext.Authenticated != current.Meta.ScanContext.Authenticated {
				delete(common, id)
				excluded[id] = struct{}{}
			}
		}
	}

	for _, id := range unevaluatedPages(baseline, current, common) {
		excluded[id] = struct{}{}
	}

	return common, sortedKeys(excluded)
}

func contextualScannerExclusions(baseline, current report.UnifiedReportV2, skipped map[string]struct{}) {
	if baseline.Meta.ScanContext == nil || current.Meta.ScanContext == nil {
		return
	}

	before := make(map[string]report.ScannerSummary)
	for _, scanner := range baseline.Scanners {
		before[scanner.Id] = scanner
	}

	for _, scanner := range current.Scanners {
		left, ok := before[scanner.Id]
		if !ok || !reflect.DeepEqual(left.ToolVersion, scanner.ToolVersion) ||
			baseline.Meta.ScanContext.ScannerConfigFingerprints[scanner.Id] != current.Meta.ScanContext.ScannerConfigFingerprints[scanner.Id] {
			skipped[scanner.Id] = struct{}{}
		}
	}
}

func sameScoreContext(baseline, current report.UnifiedReportV2) bool {
	if !reflect.DeepEqual(profile(baseline), profile(current)) ||
		!reflect.DeepEqual(baseline.Meta.PolicyVersion, current.Meta.PolicyVersion) {
		return false
	}

	left, right := baseline.Meta.ScanContext, current.Meta.ScanContext
	if left == nil || right == nil {
		return left == nil && right == nil
	}

	a, b := append([]string{}, left.ExpectedPages...), append([]string{}, right.ExpectedPages...)
	sort.Strings(a)
	sort.Strings(b)

	return reflect.DeepEqual(a, b) && left.IdentityScope == right.IdentityScope &&
		left.Authenticated == right.Authenticated && left.InputType == right.InputType &&
		left.Purpose == right.Purpose &&
		left.Purpose != report.ScanContextPurposeVerification &&
		reflect.DeepEqual(left.Viewport, right.Viewport)
}

func profile(doc report.UnifiedReportV2) report.ScanProfile {
	if doc.Meta.Profile != nil {
		return *doc.Meta.Profile
	}

	return report.ScanProfileProduction
}

func evaluatedOnBoth(baseline, current report.UnifiedReportV2, issue report.IssueDetail) bool {
	// Older reports have no rule outcomes. Keep their page-level comparison,
	// but never fall back to it when either side records rule-level evidence.
	requireRules := issue.RuleId != "" && (hasRuleCoverage(baseline, issue) || hasRuleCoverage(current, issue))
	before, left := evaluatedCoverage(baseline, issue, requireRules)

	after, right := evaluatedCoverage(current, issue, requireRules)
	if !before || !after {
		return false
	}

	if left != nil && right != nil && coverageLocation(baseline, *left) != coverageLocation(current, *right) {
		return false
	}

	return true
}

func evaluatedCoverage(doc report.UnifiedReportV2, issue report.IssueDetail, requireRules bool) (bool, *string) {
	for _, scanner := range doc.Scanners {
		if scanner.Id != issue.Scanner {
			continue
		}

		if len(scanner.Pages) == 0 {
			return doc.Meta.ScanContext == nil && !requireRules, nil
		}

		for _, page := range scanner.Pages {
			if page.PageId == issue.PageId {
				return page.Status == report.ScannerPageCoverageStatusSuccess &&
					(!requireRules || len(page.Rules) > 0) && ruleEvaluated(page, issue.RuleId), page.FinalUrl
			}
		}

		return false, nil
	}
	// Legacy in-memory callers may have no scanner summaries. Real new reports
	// require explicit successful coverage from the responsible scanner.
	return doc.Meta.ScanContext == nil && !requireRules, nil
}

func unevaluatedPages(baseline, current report.UnifiedReportV2, common map[string]bool) []string {
	excluded := []string{}

	for id := range common {
		for _, scanner := range baseline.Scanners {
			if !evaluatedOnBoth(baseline, current, report.IssueDetail{Scanner: scanner.Id, PageId: id}) {
				excluded = append(excluded, id)
				break
			}
		}
	}

	for _, doc := range []report.UnifiedReportV2{baseline, current} {
		for _, issue := range doc.Issues {
			if common[issue.PageId] && !evaluatedOnBoth(baseline, current, issue) {
				excluded = append(excluded, issue.PageId)
			}
		}
	}

	return excluded
}

func coverageLocation(doc report.UnifiedReportV2, location string) string {
	context := doc.Meta.ScanContext
	if context == nil ||
		(context.IdentityScope == report.ScanContextIdentityScopeOrigin && context.InputType != report.ScanContextInputTypeZip) {
		return location
	}

	parsed, err := url.Parse(location)
	if err != nil {
		return location
	}

	return parsed.EscapedPath() + "?" + parsed.RawQuery
}

func scoresComparable(baseline, current report.UnifiedReportV2, skipped map[string]struct{}, excluded []string) bool {
	return len(skipped) == 0 && len(excluded) == 0 && len(IncompleteScanners(baseline.Scanners)) == 0 &&
		len(IncompleteScanners(current.Scanners)) == 0 &&
		sameScoreContext(baseline, current)
}

func hasRuleCoverage(doc report.UnifiedReportV2, issue report.IssueDetail) bool {
	for _, scanner := range doc.Scanners {
		if scanner.Id == issue.Scanner {
			for _, page := range scanner.Pages {
				if page.PageId == issue.PageId {
					return len(page.Rules) > 0
				}
			}
		}
	}

	return false
}

func ruleEvaluated(page report.ScannerPageCoverage, ruleID string) bool {
	if ruleID == "" || len(page.Rules) == 0 {
		return true
	}

	found := false

	for _, rule := range page.Rules {
		if rule.RuleId == ruleID {
			if rule.Outcome == report.RuleEvaluationOutcomeError ||
				rule.Outcome == report.RuleEvaluationOutcomeIncomplete {
				return false
			}

			found = true
		}
	}

	return found
}
