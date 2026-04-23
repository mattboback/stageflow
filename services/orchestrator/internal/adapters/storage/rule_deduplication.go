package storage

import report "github.com/mattboback/stageflow/libs/contracts/report/generated/go"

// RuleEquivalence defines equivalent rules across different scanners.
// When multiple scanners flag the same underlying issue, we keep only
// the version from the primary (highest priority) scanner.
type RuleEquivalence struct {
	CanonicalID string   // The canonical rule identifier
	Rules       []string // Scanner-prefixed rule IDs: "scanner:ruleId"
}

// scannerPriority defines which scanner's version to keep when duplicates are found.
// Higher priority scanners take precedence.
var scannerPriority = map[string]int{
	"axe":              100, // Accessibility specialist - most detailed
	"lighthouse":       80,  // Good accessibility coverage but broader scope
	"security-headers": 90,  // Unique domain, no overlap expected
	"seo":              60,  // Often overlaps with accessibility
	"link-checker":     70,  // Unique domain, no overlap expected
}

// ruleEquivalences maps canonical rule IDs to their cross-scanner equivalents.
// Format: "scanner:ruleId" -> canonicalID.
//
// Maintenance: when adding a scanner that overlaps with axe or lighthouse
// accessibility audits, add entries here. The canonical ID should match the
// axe-core rule ID where possible.
var ruleEquivalences = map[string]string{
	// Image alt text issues
	"axe:image-alt":          "image-alt",
	"lighthouse:image-alt":   "image-alt",
	"seo:images-missing-alt": "image-alt",

	// Color contrast issues
	"axe:color-contrast":        "color-contrast",
	"lighthouse:color-contrast": "color-contrast",

	// Document language
	"axe:html-has-lang":        "html-has-lang",
	"lighthouse:html-has-lang": "html-has-lang",

	// Language validity
	"axe:html-lang-valid":        "html-lang-valid",
	"lighthouse:html-lang-valid": "html-lang-valid",

	// Valid lang attribute
	"axe:valid-lang":        "valid-lang",
	"lighthouse:valid-lang": "valid-lang",

	// Link text issues
	"axe:link-name":        "link-name",
	"lighthouse:link-name": "link-name",

	// Form labels
	"axe:label":        "form-label",
	"lighthouse:label": "form-label",

	// Heading order
	"axe:heading-order":        "heading-order",
	"lighthouse:heading-order": "heading-order",
	"seo:heading-hierarchy":    "heading-order",

	// Meta viewport
	"axe:meta-viewport":        "meta-viewport",
	"lighthouse:meta-viewport": "meta-viewport",

	// Button name
	"axe:button-name":        "button-name",
	"lighthouse:button-name": "button-name",

	// Frame title
	"axe:frame-title":        "frame-title",
	"lighthouse:frame-title": "frame-title",

	// Document title
	"axe:document-title":        "document-title",
	"lighthouse:document-title": "document-title",
	"seo:missing-title":         "document-title",

	// Meta description (SEO-focused but can overlap)
	"seo:missing-meta-description": "meta-description",
	"lighthouse:meta-description":  "meta-description",

	// Duplicate IDs
	"axe:duplicate-id":        "duplicate-id",
	"axe:duplicate-id-active": "duplicate-id",
	"axe:duplicate-id-aria":   "duplicate-id",
	"lighthouse:duplicate-id": "duplicate-id",

	// ARIA attribute rules
	"axe:aria-allowed-attr":        "aria-allowed-attr",
	"lighthouse:aria-allowed-attr": "aria-allowed-attr",

	"axe:aria-hidden-body":        "aria-hidden-body",
	"lighthouse:aria-hidden-body": "aria-hidden-body",

	"axe:aria-hidden-focus":        "aria-hidden-focus",
	"lighthouse:aria-hidden-focus": "aria-hidden-focus",

	"axe:aria-required-attr":        "aria-required-attr",
	"lighthouse:aria-required-attr": "aria-required-attr",

	"axe:aria-roles":        "aria-roles",
	"lighthouse:aria-roles": "aria-roles",

	"axe:aria-valid-attr-value":        "aria-valid-attr-value",
	"lighthouse:aria-valid-attr-value": "aria-valid-attr-value",

	"axe:aria-valid-attr":        "aria-valid-attr",
	"lighthouse:aria-valid-attr": "aria-valid-attr",

	// Navigation and structure
	"axe:bypass":        "bypass",
	"lighthouse:bypass": "bypass",

	// Form and input rules
	"axe:input-image-alt":        "input-image-alt",
	"lighthouse:input-image-alt": "input-image-alt",

	"axe:form-field-multiple-labels":        "form-field-multiple-labels",
	"lighthouse:form-field-multiple-labels": "form-field-multiple-labels",

	// List structure
	"axe:list":        "list",
	"lighthouse:list": "list",

	"axe:listitem":        "listitem",
	"lighthouse:listitem": "listitem",

	// Media and objects
	"axe:object-alt":        "object-alt",
	"lighthouse:object-alt": "object-alt",

	"axe:video-caption":        "video-caption",
	"lighthouse:video-caption": "video-caption",

	// Keyboard and focus
	"axe:tabindex":        "tabindex",
	"lighthouse:tabindex": "tabindex",

	"axe:accesskeys":        "accesskeys",
	"lighthouse:accesskeys": "accesskeys",

	// Table structure
	"axe:td-headers-attr":        "td-headers-attr",
	"lighthouse:td-headers-attr": "td-headers-attr",

	"axe:th-has-data-cells":        "th-has-data-cells",
	"lighthouse:th-has-data-cells": "th-has-data-cells",
}

// deduplicateIssues removes duplicate issues that are flagged by multiple scanners.
// It keeps the issue from the highest-priority scanner and adds metadata about
// which other scanners also detected the issue.
func deduplicateIssues(issues []report.IssueDetail) []report.IssueDetail {
	if len(issues) == 0 {
		return issues
	}

	// Group issues by (pageId, canonicalRuleId)
	type issueKey struct {
		pageID      string
		canonicalID string
	}

	groups := make(map[issueKey][]report.IssueDetail)
	standalone := make([]report.IssueDetail, 0, len(issues))

	for _, issue := range issues {
		ruleKey := issue.Scanner + ":" + issue.RuleId
		canonicalID, hasEquivalent := ruleEquivalences[ruleKey]

		if !hasEquivalent {
			// No known equivalent - keep as-is
			standalone = append(standalone, issue)

			continue
		}

		key := issueKey{pageID: issue.PageId, canonicalID: canonicalID}
		groups[key] = append(groups[key], issue)
	}

	// For each group, keep only the highest-priority scanner's version
	result := make([]report.IssueDetail, 0, len(issues))
	result = append(result, standalone...)

	for _, group := range groups {
		if len(group) == 1 {
			result = append(result, group[0])

			continue
		}

		// Find highest priority issue
		best := group[0]
		bestPriority := getScannerPriority(best.Scanner)
		otherScanners := make([]string, 0, len(group)-1)

		for i := 1; i < len(group); i++ {
			issue := group[i]
			priority := getScannerPriority(issue.Scanner)

			if priority > bestPriority {
				otherScanners = append(otherScanners, best.Scanner)
				best = issue
				bestPriority = priority
			} else {
				otherScanners = append(otherScanners, issue.Scanner)
			}
		}

		// Add metadata about other scanners that detected this issue
		if best.ScannerData == nil {
			best.ScannerData = make(map[string]interface{})
		}

		best.ScannerData["alsoDetectedBy"] = otherScanners

		result = append(result, best)
	}

	return result
}

// getScannerPriority returns the priority for a scanner (higher = more authoritative).
func getScannerPriority(scanner string) int {
	if priority, ok := scannerPriority[scanner]; ok {
		return priority
	}

	return 50 // Default priority for unknown scanners
}
