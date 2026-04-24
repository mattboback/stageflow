package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"
)

func TestDeduplicateIssues_NoEquivalents(t *testing.T) {
	issues := []report.IssueDetail{
		{Id: "1", Scanner: "axe", RuleId: "unique-rule", PageId: "page1", Severity: report.IssueSeverityCritical},
		{Id: "2", Scanner: "seo", RuleId: "another-rule", PageId: "page1", Severity: report.IssueSeveritySerious},
	}

	result := deduplicateIssues(issues)

	assert.Len(t, result, 2)
}

func TestDeduplicateIssues_ImageAltDuplicates(t *testing.T) {
	issues := []report.IssueDetail{
		{
			Id:       "1",
			Scanner:  "axe",
			RuleId:   "image-alt",
			PageId:   "page1",
			Severity: report.IssueSeverityCritical,
			Title:    "Images must have alternative text",
		},
		{
			Id:       "2",
			Scanner:  "seo",
			RuleId:   "images-missing-alt",
			PageId:   "page1",
			Severity: report.IssueSeveritySerious,
			Title:    "Images Missing Alt Text",
		},
		{
			Id:       "3",
			Scanner:  "lighthouse",
			RuleId:   "image-alt",
			PageId:   "page1",
			Severity: report.IssueSeveritySerious,
			Title:    "Image elements have alt attributes",
		},
	}

	result := deduplicateIssues(issues)

	// Should keep only the axe version (highest priority)
	require.Len(t, result, 1)
	assert.Equal(t, "axe", result[0].Scanner)
	assert.Equal(t, "image-alt", result[0].RuleId)

	// Should have metadata about other scanners
	alsoDetectedBy, ok := result[0].ScannerData["alsoDetectedBy"].([]string)
	require.True(t, ok, "alsoDetectedBy should be a string slice")
	assert.Len(t, alsoDetectedBy, 2)
	assert.Contains(t, alsoDetectedBy, "seo")
	assert.Contains(t, alsoDetectedBy, "lighthouse")
}

func TestDeduplicateIssues_DifferentPages(t *testing.T) {
	issues := []report.IssueDetail{
		{Id: "1", Scanner: "axe", RuleId: "image-alt", PageId: "page1", Severity: report.IssueSeverityCritical},
		{Id: "2", Scanner: "seo", RuleId: "images-missing-alt", PageId: "page2", Severity: report.IssueSeveritySerious},
	}

	result := deduplicateIssues(issues)

	// Different pages should not be deduplicated
	assert.Len(t, result, 2)
}

func TestDeduplicateIssues_ColorContrastDuplicates(t *testing.T) {
	issues := []report.IssueDetail{
		{
			Id:       "1",
			Scanner:  "lighthouse",
			RuleId:   "color-contrast",
			PageId:   "page1",
			Severity: report.IssueSeveritySerious,
		},
		{Id: "2", Scanner: "axe", RuleId: "color-contrast", PageId: "page1", Severity: report.IssueSeverityCritical},
	}

	result := deduplicateIssues(issues)

	// Should keep axe version (higher priority than lighthouse)
	require.Len(t, result, 1)
	assert.Equal(t, "axe", result[0].Scanner)
	assert.Equal(t, report.IssueSeverityCritical, result[0].Severity)
}

func TestDeduplicateIssues_MixedDuplicatesAndUnique(t *testing.T) {
	issues := []report.IssueDetail{
		// Duplicate: image-alt
		{Id: "1", Scanner: "axe", RuleId: "image-alt", PageId: "page1", Severity: report.IssueSeverityCritical},
		{Id: "2", Scanner: "seo", RuleId: "images-missing-alt", PageId: "page1", Severity: report.IssueSeveritySerious},
		// Unique: security header
		{
			Id:       "3",
			Scanner:  "security-headers",
			RuleId:   "missing-csp",
			PageId:   "page1",
			Severity: report.IssueSeveritySerious,
		},
		// Unique: broken link
		{
			Id:       "4",
			Scanner:  "link-checker",
			RuleId:   "broken-link",
			PageId:   "page1",
			Severity: report.IssueSeverityModerate,
		},
	}

	result := deduplicateIssues(issues)

	// Should have 3 issues: deduplicated image-alt + 2 unique
	assert.Len(t, result, 3)

	// Verify we have one of each unique scanner
	scanners := make(map[string]bool)
	for _, issue := range result {
		scanners[issue.Scanner] = true
	}

	assert.True(t, scanners["axe"], "axe issue should be present")
	assert.True(t, scanners["security-headers"], "security-headers issue should be present")
	assert.True(t, scanners["link-checker"], "link-checker issue should be present")
	assert.False(t, scanners["seo"], "seo issue should be deduplicated")
}

func TestDeduplicateIssues_EmptyInput(t *testing.T) {
	result := deduplicateIssues(nil)
	assert.Empty(t, result)

	result = deduplicateIssues([]report.IssueDetail{})
	assert.Empty(t, result)
}

func TestGetScannerPriority(t *testing.T) {
	assert.Equal(t, 100, getScannerPriority("axe"))
	assert.Equal(t, 80, getScannerPriority("lighthouse"))
	assert.Equal(t, 60, getScannerPriority("seo"))
	assert.Equal(t, 50, getScannerPriority("unknown-scanner"))
}

func TestDeduplicateIssues_AriaRuleDuplicates(t *testing.T) {
	issues := []report.IssueDetail{
		{
			Id:       "1",
			Scanner:  "axe",
			RuleId:   "aria-required-attr",
			PageId:   "page1",
			Severity: report.IssueSeverityCritical,
			Title:    "ARIA attributes must conform to valid names",
		},
		{
			Id:       "2",
			Scanner:  "lighthouse",
			RuleId:   "aria-required-attr",
			PageId:   "page1",
			Severity: report.IssueSeveritySerious,
			Title:    "Elements with ARIA roles have required attributes",
		},
	}

	result := deduplicateIssues(issues)

	require.Len(t, result, 1)
	assert.Equal(t, "axe", result[0].Scanner)

	alsoDetectedBy, ok := result[0].ScannerData["alsoDetectedBy"].([]string)
	require.True(t, ok)
	assert.Equal(t, []string{"lighthouse"}, alsoDetectedBy)
}

func TestAllEquivalencesHaveCounterpart(t *testing.T) {
	// Every canonical ID in the equivalences map should have at least two
	// scanner-prefixed entries. A single entry means the mapping is useless.
	canonicalCounts := make(map[string]int)
	for _, canonicalID := range ruleEquivalences {
		canonicalCounts[canonicalID]++
	}

	for canonicalID, count := range canonicalCounts {
		if count < 2 {
			t.Errorf("canonical rule %q has only %d entry — needs at least 2 for dedup to work", canonicalID, count)
		}
	}
}
