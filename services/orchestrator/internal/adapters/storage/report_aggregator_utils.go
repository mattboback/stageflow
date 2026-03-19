package storage

import (
	"fmt"
	"math"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"

	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
)

func flattenScannerStats(stats map[string]*report.ScannerSummary) []report.ScannerSummary {
	list := make([]report.ScannerSummary, 0, len(stats))
	for _, stat := range stats {
		if stat.Status == "" {
			stat.Status = report.ScannerStatusSkipped
		}

		list = append(list, *stat)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].Id < list[j].Id
	})

	return list
}

func flattenArtifacts(artifacts map[string]report.ReportArtifact) []report.ReportArtifact {
	if len(artifacts) == 0 {
		return nil
	}

	list := make([]report.ReportArtifact, 0, len(artifacts))
	for _, artifact := range artifacts {
		list = append(list, artifact)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].Id < list[j].Id
	})

	return list
}

func pageKey(scannerID string, index int, page report.PageSummary) string {
	if page.Id != "" {
		return "id:" + page.Id
	}

	if page.Url != "" {
		return "url:" + normalizeURL(page.Url)
	}

	if pagePath := stringValue(page.Path); pagePath != "" {
		return "path:" + pagePath
	}

	return fmt.Sprintf("scan:%s:%d", scannerID, index)
}

func derivePageID(page report.PageSummary) string {
	if page.Id != "" {
		return page.Id
	}

	if page.Url == "" {
		return ""
	}

	normalized := normalizeURL(page.Url)
	if normalized == "" {
		return ""
	}

	safe := strings.TrimPrefix(normalized, "http://")
	safe = strings.TrimPrefix(safe, "https://")
	safe = strings.ReplaceAll(safe, "/", "-")

	safe = strings.Trim(safe, "-")
	if safe == "" {
		return ""
	}

	return "url-" + safe
}

func normalizeURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return strings.TrimSpace(raw)
	}

	parsed.Fragment = ""

	parsed.Host = strings.ToLower(parsed.Host)
	if parsed.Path != "/" {
		parsed.Path = strings.TrimRight(parsed.Path, "/")
	}

	return parsed.String()
}

func derivePath(rawURL string) string {
	if rawURL == "" {
		return ""
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	if parsed.Path != "" {
		return path.Clean(parsed.Path)
	}

	return "/"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}

	return ""
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

func stringPtr(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	v := value

	return &v
}

func intPtr(value int) *int {
	v := value

	return &v
}

func floatPtr(value float64) *float64 {
	v := value

	return &v
}

func pickEarlierTime(current, candidate *time.Time) *time.Time {
	if candidate == nil {
		return current
	}

	if current == nil || candidate.Before(*current) {
		return candidate
	}

	return current
}

func pickLaterTime(current, candidate *time.Time) *time.Time {
	if candidate == nil {
		return current
	}

	if current == nil || candidate.After(*current) {
		return candidate
	}

	return current
}

func addSeverity(counts *report.SeverityCounts, severity report.IssueSeverity) {
	switch severity {
	case report.IssueSeverityCritical:
		counts.Critical++
	case report.IssueSeveritySerious:
		counts.Serious++
	case report.IssueSeverityModerate:
		counts.Moderate++
	case report.IssueSeverityMinor:
		counts.Minor++
	case report.IssueSeverityInfo:
		addInfo(counts)
	default:
		addInfo(counts)
	}
}

func addInfo(counts *report.SeverityCounts) {
	if counts.Info == nil {
		counts.Info = intPtr(0)
	}

	*counts.Info++
}

func severityInfo(counts report.SeverityCounts) int {
	if counts.Info == nil {
		return 0
	}

	return *counts.Info
}

func addSeverityTotals(total *report.SeverityCounts, add report.SeverityCounts) {
	total.Critical += add.Critical
	total.Serious += add.Serious
	total.Moderate += add.Moderate
	total.Minor += add.Minor

	info := severityInfo(*total) + severityInfo(add)
	if info > 0 {
		total.Info = intPtr(info)
	} else {
		total.Info = nil
	}
}

func addSeverityCounts(base, add *report.SeverityCounts) *report.SeverityCounts {
	if base == nil && add == nil {
		return nil
	}

	if base == nil {
		clone := *add

		return &clone
	}

	if add == nil {
		return base
	}

	base.Critical += add.Critical
	base.Serious += add.Serious
	base.Moderate += add.Moderate
	base.Minor += add.Minor

	info := severityInfo(*base) + severityInfo(*add)
	if info > 0 {
		base.Info = intPtr(info)
	} else {
		base.Info = nil
	}

	return base
}

func calculateAccessibilityScore(counts report.SeverityCounts) (score int, grade string) {
	penalty := float64(counts.Critical*10 + counts.Serious*5 + counts.Moderate*2 + counts.Minor)
	if penalty <= 0 {
		return 100, "Excellent"
	}

	scaled := 20*math.Log10(penalty+1) + penalty*0.3
	if scaled > 100 {
		scaled = 100
	}

	score = int(100 - scaled + 0.5)
	if score < 0 {
		score = 0
	}

	grade = scoreGrade(score)

	return score, grade
}

func scoreGrade(score int) string {
	switch {
	case score >= 97:
		return "A+"
	case score >= 93:
		return "A"
	case score >= 90:
		return "A-"
	case score >= 87:
		return "B+"
	case score >= 83:
		return "B"
	case score >= 80:
		return "B-"
	case score >= 77:
		return "C+"
	case score >= 73:
		return "C"
	case score >= 70:
		return "C-"
	case score >= 67:
		return "D+"
	case score >= 63:
		return "D"
	case score >= 60:
		return "D-"
	default:
		return "F"
	}
}

func durationMs(scannedAt, completedAt *time.Time) float64 {
	if scannedAt == nil || completedAt == nil {
		return 0
	}

	delta := completedAt.Sub(*scannedAt)

	return float64(delta.Milliseconds())
}
