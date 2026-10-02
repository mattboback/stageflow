package aggregate

// ruleEquivalences identifies proven cross-scanner concerns for specialist ownership.
// Format: "scanner:ruleId" -> canonicalID.
//
// The aggregation ownership predicate uses known axe rule IDs to recognize
// Lighthouse accessibility findings when older reports omit their category.
// New overlap mappings require a fixture proving replacement coverage.
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
