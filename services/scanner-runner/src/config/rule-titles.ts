/**
 * Rule Display Titles
 *
 * Human-readable titles and categories for accessibility rule IDs.
 * Transforms technical jargon like "aria-progressbar-name" into
 * client-friendly descriptions like "Progress Bar Missing Label".
 */

type RuleCategory =
	| 'Images & Media'
	| 'Forms & Inputs'
	| 'Navigation & Links'
	| 'Text & Readability'
	| 'Interactive Controls'
	| 'Page Structure'
	| 'Tables'
	| 'Multimedia'
	| 'Technical';

interface RuleTitleEntry {
	/** Human-readable title (e.g., "Image Missing Alternative Text") */
	title: string;
	/** Brief explanation for non-technical stakeholders */
	summary: string;
	/** Category for grouping related issues */
	category: RuleCategory;
}

/**
 * Mapping of axe-core and custom rule IDs to display information.
 */
const ruleTitles: Record<string, RuleTitleEntry> = {
	// ============================================
	// IMAGES & MEDIA
	// ============================================
	'image-alt': {
		title: 'Image Missing Alternative Text',
		summary: "Screen readers can't describe this image to blind users",
		category: 'Images & Media'
	},
	'input-image-alt': {
		title: 'Image Button Missing Label',
		summary: "Screen readers can't explain what this image button does",
		category: 'Images & Media'
	},
	'object-alt': {
		title: 'Embedded Object Missing Description',
		summary: "Screen readers can't describe this embedded content",
		category: 'Images & Media'
	},
	'svg-img-alt': {
		title: 'SVG Image Missing Label',
		summary: "Screen readers can't describe this SVG graphic",
		category: 'Images & Media'
	},
	'role-img-alt': {
		title: 'Decorative Image Missing Description',
		summary: "Images with role='img' need alternative text",
		category: 'Images & Media'
	},
	'area-alt': {
		title: 'Image Map Area Missing Label',
		summary: 'Clickable image regions need descriptive text',
		category: 'Images & Media'
	},
	'image-redundant-alt': {
		title: 'Redundant Image Description',
		summary: 'Image alt text repeats surrounding text',
		category: 'Images & Media'
	},

	// ============================================
	// FORMS & INPUTS
	// ============================================
	label: {
		title: 'Form Field Missing Label',
		summary: "Users can't tell what information to enter",
		category: 'Forms & Inputs'
	},
	'select-name': {
		title: 'Dropdown Missing Label',
		summary: "Screen readers can't describe this dropdown",
		category: 'Forms & Inputs'
	},
	'input-button-name': {
		title: 'Form Button Missing Label',
		summary: "Screen readers can't explain what this button does",
		category: 'Forms & Inputs'
	},
	'form-field-multiple-labels': {
		title: 'Form Field Has Multiple Labels',
		summary: 'Conflicting labels confuse assistive technology',
		category: 'Forms & Inputs'
	},
	autocomplete: {
		title: 'Missing Autocomplete Attribute',
		summary: "Browser autofill can't help users complete this field faster",
		category: 'Forms & Inputs'
	},
	'autocomplete-valid': {
		title: 'Invalid Autocomplete Value',
		summary: "Browser autofill won't work correctly for this field",
		category: 'Forms & Inputs'
	},

	// ============================================
	// NAVIGATION & LINKS
	// ============================================
	'link-name': {
		title: 'Link Missing Accessible Name',
		summary: "Screen readers can't tell users where this link goes",
		category: 'Navigation & Links'
	},
	'link-in-text-block': {
		title: 'Link Hard to Distinguish from Text',
		summary: 'Links rely on color alone to stand out from text',
		category: 'Navigation & Links'
	},
	bypass: {
		title: 'Missing Skip Navigation Link',
		summary: 'Keyboard users must tab through the entire menu on every page',
		category: 'Navigation & Links'
	},
	'skip-link': {
		title: 'Skip Link Not Provided',
		summary: 'No way to jump past repeated navigation',
		category: 'Navigation & Links'
	},
	tabindex: {
		title: 'Incorrect Tab Order',
		summary: 'Keyboard navigation jumps around unpredictably',
		category: 'Navigation & Links'
	},
	'focus-order-semantics': {
		title: "Focus Order Doesn't Match Visual Order",
		summary: 'Tab order is confusing for keyboard users',
		category: 'Navigation & Links'
	},
	accesskeys: {
		title: 'Conflicting Keyboard Shortcuts',
		summary: 'Access keys may conflict with assistive technology',
		category: 'Navigation & Links'
	},
	'identical-links-same-purpose': {
		title: 'Identical Links with Different Destinations',
		summary: 'Links with same text go to different places',
		category: 'Navigation & Links'
	},

	// ============================================
	// TEXT & READABILITY
	// ============================================
	'color-contrast': {
		title: 'Insufficient Color Contrast',
		summary: 'Text is hard to read against its background',
		category: 'Text & Readability'
	},
	'color-contrast-enhanced': {
		title: 'Color Contrast Below AAA Standard',
		summary: "Text contrast doesn't meet enhanced guidelines",
		category: 'Text & Readability'
	},
	'meta-viewport': {
		title: 'Zoom Disabled on Page',
		summary: "Users can't zoom in to enlarge text",
		category: 'Text & Readability'
	},
	'p-as-heading': {
		title: 'Paragraph Styled as Heading',
		summary: "Visual heading isn't marked up correctly for screen readers",
		category: 'Text & Readability'
	},

	// ============================================
	// INTERACTIVE CONTROLS
	// ============================================
	'button-name': {
		title: 'Button Missing Accessible Name',
		summary: "Screen readers can't explain what this button does",
		category: 'Interactive Controls'
	},
	'aria-progressbar-name': {
		title: 'Progress Bar Missing Label',
		summary: "Screen readers can't describe what's loading",
		category: 'Interactive Controls'
	},
	'aria-input-field-name': {
		title: 'Custom Input Missing Label',
		summary: 'ARIA input field has no accessible name',
		category: 'Interactive Controls'
	},
	'aria-toggle-field-name': {
		title: 'Toggle Control Missing Label',
		summary: "Screen readers can't describe this switch or checkbox",
		category: 'Interactive Controls'
	},
	'aria-command-name': {
		title: 'Interactive Control Missing Label',
		summary: "Screen readers can't explain this control's purpose",
		category: 'Interactive Controls'
	},
	'aria-meter-name': {
		title: 'Meter Missing Label',
		summary: "Screen readers can't describe what this meter measures",
		category: 'Interactive Controls'
	},
	'focus-visible': {
		title: 'Missing Focus Indicator',
		summary: "Keyboard users can't see which element is focused",
		category: 'Interactive Controls'
	},
	'nested-interactive': {
		title: 'Nested Interactive Elements',
		summary: 'Clicking one control may accidentally trigger another',
		category: 'Interactive Controls'
	},
	'interactive-supports-focus': {
		title: 'Interactive Element Not Keyboard Accessible',
		summary: "This control can't be reached by keyboard",
		category: 'Interactive Controls'
	},
	'scrollable-region-focusable': {
		title: 'Scrollable Area Not Keyboard Accessible',
		summary: "Keyboard users can't scroll this content",
		category: 'Interactive Controls'
	},
	keyboard: {
		title: 'Not Keyboard Accessible',
		summary: "This element can't be operated with keyboard alone",
		category: 'Interactive Controls'
	},
	'target-size': {
		title: 'Touch Target Too Small',
		summary: 'Button or link is too small to tap easily',
		category: 'Interactive Controls'
	},

	// ============================================
	// PAGE STRUCTURE
	// ============================================
	'document-title': {
		title: 'Page Missing Title',
		summary: "Browser tabs and screen readers can't identify this page",
		category: 'Page Structure'
	},
	'html-has-lang': {
		title: 'Page Language Not Specified',
		summary: 'Screen readers may mispronounce all content',
		category: 'Page Structure'
	},
	'html-lang-valid': {
		title: 'Invalid Page Language Code',
		summary: "Language code isn't recognized",
		category: 'Page Structure'
	},
	'valid-lang': {
		title: 'Invalid Language Attribute',
		summary: 'Language attribute uses unrecognized code',
		category: 'Page Structure'
	},
	'html-xml-lang-mismatch': {
		title: 'Mismatched Language Attributes',
		summary: "HTML lang and xml:lang don't match",
		category: 'Page Structure'
	},
	'language-of-page': {
		title: 'Page Language Not Declared',
		summary: 'Missing lang attribute on <html> element',
		category: 'Page Structure'
	},
	'heading-order': {
		title: 'Heading Levels Skip Numbers',
		summary: 'Headings jump from H2 to H4, breaking hierarchy',
		category: 'Page Structure'
	},
	'empty-heading': {
		title: 'Empty Heading',
		summary: 'Heading element has no text content',
		category: 'Page Structure'
	},
	'page-has-heading-one': {
		title: 'Page Missing Main Heading',
		summary: 'No H1 heading to identify page content',
		category: 'Page Structure'
	},
	'landmark-one-main': {
		title: 'Page Missing Main Landmark',
		summary: 'No <main> element to identify primary content',
		category: 'Page Structure'
	},
	'landmark-unique': {
		title: 'Duplicate Landmark Labels',
		summary: 'Multiple landmarks with same label are confusing',
		category: 'Page Structure'
	},
	region: {
		title: 'Content Outside Landmarks',
		summary: "Content isn't contained in page regions",
		category: 'Page Structure'
	},
	list: {
		title: 'Improper List Markup',
		summary: 'List not structured correctly for screen readers',
		category: 'Page Structure'
	},
	listitem: {
		title: 'List Item Outside List',
		summary: "List item isn't contained in a list element",
		category: 'Page Structure'
	},
	'definition-list': {
		title: 'Improper Definition List Markup',
		summary: 'Definition list not structured correctly',
		category: 'Page Structure'
	},
	dlitem: {
		title: 'Definition Item Outside Definition List',
		summary: "Definition item isn't contained properly",
		category: 'Page Structure'
	},
	'duplicate-id': {
		title: 'Duplicate Element ID',
		summary: 'Multiple elements share the same ID',
		category: 'Page Structure'
	},
	'duplicate-id-active': {
		title: 'Duplicate ID on Interactive Element',
		summary: 'Interactive elements share IDs, breaking functionality',
		category: 'Page Structure'
	},
	'duplicate-id-aria': {
		title: 'Duplicate ID Referenced by ARIA',
		summary: 'ARIA relationships broken by duplicate IDs',
		category: 'Page Structure'
	},

	// ============================================
	// ARIA & TECHNICAL
	// ============================================
	'aria-allowed-attr': {
		title: 'Invalid ARIA Attribute for Role',
		summary: "ARIA attribute isn't allowed on this element",
		category: 'Technical'
	},
	'aria-allowed-role': {
		title: 'Invalid Role for Element',
		summary: "This ARIA role shouldn't be used on this element",
		category: 'Technical'
	},
	'aria-hidden-focus': {
		title: 'Focusable Element Hidden from Screen Readers',
		summary: 'Hidden content can still receive keyboard focus',
		category: 'Technical'
	},
	'aria-hidden-body': {
		title: 'Entire Page Hidden from Screen Readers',
		summary: "The body element has aria-hidden='true'",
		category: 'Technical'
	},
	'aria-required-attr': {
		title: 'Missing Required ARIA Attribute',
		summary: 'ARIA role is missing a required attribute',
		category: 'Technical'
	},
	'aria-required-children': {
		title: 'Missing Required Child Elements',
		summary: 'ARIA role requires specific child elements',
		category: 'Technical'
	},
	'aria-required-parent': {
		title: 'Missing Required Parent Element',
		summary: 'ARIA role must be inside a specific parent',
		category: 'Technical'
	},
	'aria-roles': {
		title: 'Invalid ARIA Role',
		summary: "This ARIA role isn't recognized",
		category: 'Technical'
	},
	'aria-valid-attr': {
		title: 'Unknown ARIA Attribute',
		summary: "This ARIA attribute isn't recognized",
		category: 'Technical'
	},
	'aria-valid-attr-value': {
		title: 'Invalid ARIA Attribute Value',
		summary: 'ARIA attribute has an invalid value',
		category: 'Technical'
	},

	// ============================================
	// TABLES
	// ============================================
	'td-headers-attr': {
		title: 'Table Cell Missing Header Reference',
		summary: "Data cells don't reference their headers correctly",
		category: 'Tables'
	},
	'th-has-data-cells': {
		title: 'Table Header Without Data Cells',
		summary: "Table header isn't associated with any data",
		category: 'Tables'
	},
	'table-duplicate-name': {
		title: 'Tables Have Identical Names',
		summary: 'Multiple tables share the same caption',
		category: 'Tables'
	},
	'scope-attr-valid': {
		title: 'Invalid Table Scope Attribute',
		summary: "Table header scope value isn't valid",
		category: 'Tables'
	},
	'table-fake-caption': {
		title: 'Fake Table Caption',
		summary: 'Table uses non-standard caption approach',
		category: 'Tables'
	},

	// ============================================
	// MULTIMEDIA
	// ============================================
	'video-caption': {
		title: 'Video Missing Captions',
		summary: "Deaf users can't access the audio content",
		category: 'Multimedia'
	},
	'audio-caption': {
		title: 'Audio Missing Transcript',
		summary: "Deaf users can't access this audio content",
		category: 'Multimedia'
	},
	'no-autoplay-audio': {
		title: 'Audio Autoplays',
		summary: 'Auto-playing audio disrupts screen reader users',
		category: 'Multimedia'
	},
	blink: {
		title: 'Blinking Content',
		summary: 'Blinking content can trigger seizures or distract users',
		category: 'Multimedia'
	},
	marquee: {
		title: 'Scrolling Marquee Content',
		summary: 'Moving text is hard to read and distracting',
		category: 'Multimedia'
	},
	'meta-refresh': {
		title: 'Page Auto-Refreshes',
		summary: 'Users may lose their place when page reloads',
		category: 'Multimedia'
	},

	// ============================================
	// FRAMES
	// ============================================
	'frame-title': {
		title: 'Frame Missing Title',
		summary: "Screen readers can't describe this embedded content",
		category: 'Page Structure'
	},
	'frame-title-unique': {
		title: 'Frames Have Identical Titles',
		summary: 'Multiple frames share the same title',
		category: 'Page Structure'
	},
	'server-side-image-map': {
		title: 'Server-Side Image Map',
		summary: "Keyboard users can't use server-side image maps",
		category: 'Images & Media'
	}
};

/**
 * Get the human-readable title for a rule ID.
 * Falls back to humanizing the rule ID if not found.
 */
export function getRuleTitle(ruleId: string): string {
	const entry = ruleTitles[ruleId];
	if (entry) {
		return entry.title;
	}
	// Fallback: humanize the rule ID
	return humanizeRuleId(ruleId);
}

/**
 * Check if a rule ID has a curated title entry.
 */
export function hasRuleTitle(ruleId: string): boolean {
	return Boolean(ruleTitles[ruleId]);
}

/**
 * Get the category for a rule ID.
 */
export function getRuleCategory(ruleId: string): string {
	return ruleTitles[ruleId]?.category ?? 'Technical';
}

/**
 * Convert a rule ID to a human-readable string.
 * "aria-progressbar-name" -> "Aria Progressbar Name"
 */
function humanizeRuleId(ruleId: string): string {
	return ruleId.replace(/[-_]/g, ' ').replace(/\b\w/g, (char) => char.toUpperCase());
}
