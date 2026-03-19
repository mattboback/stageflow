/**
 * User Impact Statements
 *
 * Human-readable descriptions of how accessibility issues affect real users.
 * These help non-technical stakeholders understand the importance of fixes.
 */

export type UserGroup =
	| "blind"
	| "low-vision"
	| "motor"
	| "cognitive"
	| "deaf"
	| "vestibular"
	| "all";

export interface UserImpact {
	statement: string;
	affectedGroups: UserGroup[];
	severity: "blocking" | "degraded" | "inconvenient";
	userStory?: string;
}

const userGroupLabels: Record<UserGroup, string> = {
	blind: "Blind users",
	"low-vision": "Low vision users",
	motor: "Motor-impaired users",
	cognitive: "Cognitive disability users",
	deaf: "Deaf/hard of hearing users",
	vestibular: "Vestibular disorder users",
	all: "All users",
};

const userGroupIcons: Record<UserGroup, string> = {
	blind: "👁️",
	"low-vision": "🔍",
	motor: "🖐️",
	cognitive: "🧠",
	deaf: "👂",
	vestibular: "💫",
	all: "👥",
};

/**
 * Mapping of axe-core rule IDs to user impact information.
 * Based on common accessibility violations and their real-world effects.
 */
const ruleImpacts: Record<string, UserImpact> = {
	"image-alt": {
		statement:
			"Screen reader users cannot understand what this image shows or represents.",
		affectedGroups: ["blind"],
		severity: "blocking",
		userStory:
			"When I navigate to this image, my screen reader just says 'image' or reads the filename, leaving me confused about what content I'm missing.",
	},
	"input-image-alt": {
		statement:
			"Screen reader users cannot determine what action this image button performs.",
		affectedGroups: ["blind"],
		severity: "blocking",
		userStory:
			"I cannot tell what this button does, so I might accidentally submit a form or trigger an unwanted action.",
	},
	"object-alt": {
		statement:
			"Screen reader users have no way to understand this embedded content.",
		affectedGroups: ["blind"],
		severity: "blocking",
	},
	"svg-img-alt": {
		statement:
			"Screen reader users cannot understand the meaning of this SVG graphic.",
		affectedGroups: ["blind"],
		severity: "blocking",
	},
	"role-img-alt": {
		statement:
			"Screen reader users cannot access the information conveyed by this image.",
		affectedGroups: ["blind"],
		severity: "blocking",
	},

	"link-name": {
		statement:
			"Screen reader users cannot determine where this link goes or what it does.",
		affectedGroups: ["blind"],
		severity: "blocking",
		userStory:
			"My screen reader announces 'link' but doesn't tell me what the link is for. I have to guess or skip it entirely.",
	},
	"button-name": {
		statement:
			"Screen reader users cannot determine what action this button performs.",
		affectedGroups: ["blind"],
		severity: "blocking",
		userStory:
			"I hear 'button' but have no idea what clicking it will do. I might accidentally delete something or navigate away.",
	},
	"link-in-text-block": {
		statement:
			"Low vision users may not be able to distinguish links from regular text.",
		affectedGroups: ["low-vision"],
		severity: "degraded",
	},

	label: {
		statement:
			"Screen reader users cannot identify what information to enter in this form field.",
		affectedGroups: ["blind", "cognitive"],
		severity: "blocking",
		userStory:
			"When I focus on this input, I don't know what information is expected. Is it my email? Phone number? I have to guess.",
	},
	"input-button-name": {
		statement:
			"Screen reader users cannot determine what this form button does.",
		affectedGroups: ["blind"],
		severity: "blocking",
	},
	"select-name": {
		statement:
			"Screen reader users cannot understand what this dropdown is for.",
		affectedGroups: ["blind"],
		severity: "blocking",
	},
	"form-field-multiple-labels": {
		statement:
			"Screen reader users hear conflicting labels, causing confusion about what to enter.",
		affectedGroups: ["blind", "cognitive"],
		severity: "degraded",
	},
	autocomplete: {
		statement:
			"Users with motor or cognitive disabilities cannot use browser autofill to complete this form faster.",
		affectedGroups: ["motor", "cognitive"],
		severity: "inconvenient",
	},

	"color-contrast": {
		statement:
			"Low vision users and users in bright sunlight cannot read this text.",
		affectedGroups: ["low-vision", "all"],
		severity: "degraded",
		userStory:
			"The text is too faint for me to read comfortably. I have to strain my eyes or zoom in significantly.",
	},
	"color-contrast-enhanced": {
		statement:
			"Users with moderate vision impairment struggle to read this text.",
		affectedGroups: ["low-vision"],
		severity: "degraded",
	},

	tabindex: {
		statement:
			"Keyboard users experience a confusing, illogical navigation order.",
		affectedGroups: ["motor", "blind"],
		severity: "degraded",
		userStory:
			"When I press Tab, focus jumps around unpredictably instead of following the visual order of the page.",
	},
	"focus-order-semantics": {
		statement:
			"Keyboard users cannot navigate this page in a logical sequence.",
		affectedGroups: ["motor", "blind"],
		severity: "degraded",
	},
	bypass: {
		statement:
			"Keyboard users must tab through the entire navigation on every page before reaching main content.",
		affectedGroups: ["motor", "blind"],
		severity: "degraded",
		userStory:
			"On every page, I have to press Tab 30+ times just to get past the header. It's exhausting.",
	},
	"scrollable-region-focusable": {
		statement:
			"Keyboard users cannot scroll this content area without using a mouse.",
		affectedGroups: ["motor", "blind"],
		severity: "blocking",
	},
	accesskeys: {
		statement:
			"Keyboard shortcut conflicts may prevent users from using their assistive technology.",
		affectedGroups: ["motor", "blind"],
		severity: "degraded",
	},

	"aria-allowed-role": {
		statement:
			"Screen reader users receive incorrect information about this element's purpose.",
		affectedGroups: ["blind"],
		severity: "degraded",
	},
	"aria-hidden-focus": {
		statement:
			"Screen reader users can focus on content they cannot hear, causing confusion.",
		affectedGroups: ["blind"],
		severity: "degraded",
	},
	"aria-hidden-body": {
		statement:
			"The entire page is hidden from screen readers, making it completely inaccessible.",
		affectedGroups: ["blind"],
		severity: "blocking",
	},
	"aria-required-attr": {
		statement:
			"Screen readers cannot properly convey this element's state or behavior.",
		affectedGroups: ["blind"],
		severity: "degraded",
	},
	"aria-required-children": {
		statement:
			"Screen readers cannot properly navigate this component structure.",
		affectedGroups: ["blind"],
		severity: "degraded",
	},
	"aria-required-parent": {
		statement:
			"Screen readers misinterpret this element due to missing parent context.",
		affectedGroups: ["blind"],
		severity: "degraded",
	},
	"aria-roles": {
		statement:
			"Screen readers announce an invalid role, confusing users about this element's purpose.",
		affectedGroups: ["blind"],
		severity: "degraded",
	},
	"aria-valid-attr": {
		statement:
			"Screen readers may ignore or misinterpret this element due to invalid ARIA.",
		affectedGroups: ["blind"],
		severity: "degraded",
	},
	"aria-valid-attr-value": {
		statement:
			"Screen readers receive incorrect information about this element's state.",
		affectedGroups: ["blind"],
		severity: "degraded",
	},
	"aria-progressbar-name": {
		statement:
			"Screen reader users cannot tell what this progress indicator is tracking.",
		affectedGroups: ["blind"],
		severity: "degraded",
		userStory:
			"I hear there's a progress bar, but I don't know if it's a file upload, page loading, or something else.",
	},
	"aria-input-field-name": {
		statement:
			"Screen reader users cannot identify the purpose of this input field.",
		affectedGroups: ["blind"],
		severity: "blocking",
	},
	"aria-toggle-field-name": {
		statement:
			"Screen reader users cannot determine what this toggle controls.",
		affectedGroups: ["blind"],
		severity: "blocking",
	},
	"aria-command-name": {
		statement:
			"Screen reader users cannot determine what action this control performs.",
		affectedGroups: ["blind"],
		severity: "blocking",
	},
	"aria-meter-name": {
		statement:
			"Screen reader users cannot understand what this meter is measuring.",
		affectedGroups: ["blind"],
		severity: "degraded",
	},

	"document-title": {
		statement:
			"Screen reader users cannot quickly identify which page or tab they are on.",
		affectedGroups: ["blind", "cognitive"],
		severity: "degraded",
		userStory:
			"When I switch between tabs, I can't tell which one is which because they all have the same title.",
	},
	"html-has-lang": {
		statement: "Screen readers may mispronounce all content on this page.",
		affectedGroups: ["blind"],
		severity: "degraded",
		userStory:
			"My screen reader is pronouncing words with the wrong accent, making it hard to understand.",
	},
	"html-lang-valid": {
		statement:
			"Screen readers use incorrect pronunciation rules for this page's content.",
		affectedGroups: ["blind"],
		severity: "degraded",
	},
	"valid-lang": {
		statement: "Screen readers mispronounce this section of content.",
		affectedGroups: ["blind"],
		severity: "inconvenient",
	},
	region: {
		statement:
			"Screen reader users cannot quickly navigate to different page sections.",
		affectedGroups: ["blind"],
		severity: "inconvenient",
	},
	landmark: {
		statement:
			"Screen reader users must listen to the entire page to find specific content.",
		affectedGroups: ["blind"],
		severity: "inconvenient",
	},

	"heading-order": {
		statement:
			"Screen reader users cannot understand the page's content hierarchy.",
		affectedGroups: ["blind", "cognitive"],
		severity: "degraded",
		userStory:
			"The heading structure is confusing. I can't tell which content belongs to which section.",
	},
	"empty-heading": {
		statement:
			"Screen reader users hear 'heading' but receive no information about the section.",
		affectedGroups: ["blind"],
		severity: "degraded",
	},
	"page-has-heading-one": {
		statement: "Screen reader users cannot find the main content of the page.",
		affectedGroups: ["blind"],
		severity: "degraded",
	},

	"td-headers-attr": {
		statement:
			"Screen reader users cannot understand which header applies to this data cell.",
		affectedGroups: ["blind"],
		severity: "degraded",
	},
	"th-has-data-cells": {
		statement: "Screen reader users cannot properly navigate this data table.",
		affectedGroups: ["blind"],
		severity: "degraded",
	},
	"table-duplicate-name": {
		statement:
			"Screen reader users cannot distinguish between similar tables on the page.",
		affectedGroups: ["blind"],
		severity: "inconvenient",
	},
	"scope-attr-valid": {
		statement:
			"Screen reader users may misunderstand table header relationships.",
		affectedGroups: ["blind"],
		severity: "degraded",
	},

	"frame-title": {
		statement:
			"Screen reader users cannot identify the purpose of this embedded content.",
		affectedGroups: ["blind"],
		severity: "degraded",
	},
	"frame-title-unique": {
		statement:
			"Screen reader users cannot distinguish between multiple embedded frames.",
		affectedGroups: ["blind"],
		severity: "inconvenient",
	},

	"meta-refresh": {
		statement:
			"Users with cognitive disabilities may lose their place when the page auto-refreshes.",
		affectedGroups: ["cognitive", "blind"],
		severity: "degraded",
	},
	"meta-viewport": {
		statement: "Low vision users cannot zoom in to make text readable.",
		affectedGroups: ["low-vision"],
		severity: "blocking",
		userStory:
			"I need to zoom in to 200% to read comfortably, but this page prevents me from zooming.",
	},
	"no-autoplay-audio": {
		statement:
			"Screen reader users cannot hear their assistive technology over auto-playing audio.",
		affectedGroups: ["blind"],
		severity: "blocking",
	},

	list: {
		statement: "Screen reader users cannot navigate this list efficiently.",
		affectedGroups: ["blind"],
		severity: "inconvenient",
	},
	listitem: {
		statement:
			"Screen reader users receive incorrect information about list structure.",
		affectedGroups: ["blind"],
		severity: "inconvenient",
	},
	"definition-list": {
		statement:
			"Screen reader users cannot understand term-definition relationships.",
		affectedGroups: ["blind"],
		severity: "inconvenient",
	},
	dlitem: {
		statement:
			"Screen reader users receive malformed definition list information.",
		affectedGroups: ["blind"],
		severity: "inconvenient",
	},

	"duplicate-id": {
		statement:
			"Screen readers may skip content or announce it incorrectly due to duplicate IDs.",
		affectedGroups: ["blind"],
		severity: "degraded",
	},
	"duplicate-id-active": {
		statement:
			"Screen readers cannot properly associate labels with form controls.",
		affectedGroups: ["blind"],
		severity: "blocking",
	},
	"duplicate-id-aria": {
		statement:
			"ARIA relationships are broken, causing screen readers to give incorrect information.",
		affectedGroups: ["blind"],
		severity: "degraded",
	},

	"nested-interactive": {
		statement:
			"Keyboard users may not be able to activate all interactive elements.",
		affectedGroups: ["motor", "blind"],
		severity: "blocking",
	},
	"interactive-supports-focus": {
		statement: "Keyboard users cannot access this interactive element at all.",
		affectedGroups: ["motor", "blind"],
		severity: "blocking",
	},

	"video-caption": {
		statement: "Deaf users cannot access the audio content of this video.",
		affectedGroups: ["deaf"],
		severity: "blocking",
		userStory:
			"I can see the video but have no idea what's being said. I'm missing crucial information.",
	},
	"audio-caption": {
		statement: "Deaf users cannot access this audio content.",
		affectedGroups: ["deaf"],
		severity: "blocking",
	},
};

/**
 * Default impact for rules not in the mapping
 */
const defaultImpact: UserImpact = {
	statement: "Some users may experience difficulty using this feature.",
	affectedGroups: ["all"],
	severity: "inconvenient",
};

/**
 * Get user impact information for a given rule ID
 */
export function getUserImpact(ruleId: string | undefined): UserImpact {
	if (!ruleId) {
		return defaultImpact;
	}
	return ruleImpacts[ruleId] ?? defaultImpact;
}

/**
 * Get icon for a user group
 */
export function getUserGroupIcon(group: UserGroup): string {
	return userGroupIcons[group];
}

/**
 * Format affected groups as a readable string
 */
export function formatAffectedGroups(groups: UserGroup[]): string {
	if (groups.includes("all")) {
		return "All users";
	}

	const labels = groups.map((group) => userGroupLabels[group]);
	if (labels.length === 0) {
		return "";
	}
	if (labels.length === 1) {
		return labels[0] ?? "";
	}
	if (labels.length === 2) {
		return `${labels[0]} and ${labels[1]}`;
	}

	const last = labels.pop();
	if (!last) {
		return labels.join(", ");
	}
	return `${labels.join(", ")}, and ${last}`;
}
