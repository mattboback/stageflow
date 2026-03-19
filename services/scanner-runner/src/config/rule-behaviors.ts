import fs from "node:fs";
import path from "node:path";

const DATA_DIR =
	process.env.SCANNER_DATA_DIR ?? path.join(process.cwd(), "data");

export type ScreenshotPolicy = "auto" | "never" | "always" | "semantic";

export interface RuleBehavior {
	screenshot?: ScreenshotPolicy;
	summary?: string;
	owner?: string;
	tags?: string[];
	displayMode?: "expanded" | "collapsed";
	references?: string[];
}

interface RuleBehaviorFile {
	defaults?: RuleBehavior;
	rules?: Record<string, RuleBehavior>;
}

const BUILT_IN_BEHAVIORS: RuleBehaviorFile = {
	defaults: {
		screenshot: "never",
		displayMode: "expanded",
	},
	rules: {
		"color-contrast": {
			screenshot: "always",
			summary: "Ensure text/background contrast meets WCAG AA ratios.",
			owner: "Design",
			tags: ["WCAG 1.4.3", "Level AA"],
		},
		"color-contrast-enhanced": {
			screenshot: "always",
			summary: "Ensure text/background contrast meets WCAG AAA ratios.",
			owner: "Design",
			tags: ["WCAG 1.4.6", "Level AAA"],
		},
		"image-alt": {
			screenshot: "always",
			summary: "Images must have alternative text describing their content.",
			owner: "Content",
			tags: ["WCAG 1.1.1", "Level A"],
		},
		"focus-visible": {
			screenshot: "auto",
			summary: "Interactive elements must have visible focus indicators.",
			owner: "Design / Frontend",
			tags: ["WCAG 2.4.7", "Level AA"],
		},
		"target-size": {
			screenshot: "auto",
			summary: "Touch targets should be at least 44x44 CSS pixels.",
			owner: "Design",
			tags: ["WCAG 2.5.5", "Level AAA"],
		},
		"link-in-text-block": {
			screenshot: "auto",
			summary:
				"Links within text must be distinguishable without relying on color alone.",
			owner: "Design",
			tags: ["WCAG 1.4.1", "Level A"],
		},

		"language-of-page": {
			screenshot: "never",
			summary: "Add a lang attribute to the <html> tag.",
			owner: "Content / Localization",
			tags: ["WCAG 3.1.1", "Level A"],
			displayMode: "collapsed",
		},
		"html-has-lang": {
			screenshot: "never",
			summary: 'Set the default page language via <html lang="...">.',
			owner: "Content / Localization",
			tags: ["WCAG 3.1.1"],
			displayMode: "collapsed",
		},
		"valid-lang": {
			screenshot: "never",
			summary: "The lang attribute must use a valid BCP 47 language tag.",
			owner: "Content",
			tags: ["WCAG 3.1.1"],
			displayMode: "collapsed",
		},
		"html-lang-valid": {
			screenshot: "never",
			summary: "The lang attribute must use a valid language code.",
			owner: "Content",
			tags: ["WCAG 3.1.1"],
			displayMode: "collapsed",
		},
		keyboard: {
			screenshot: "never",
			summary: "Every interactive element must be reachable via keyboard.",
			owner: "Frontend / Interaction",
			tags: ["WCAG 2.1.1", "Level A"],
			displayMode: "collapsed",
		},
		"landmark-one-main": {
			screenshot: "never",
			summary: "Each page should contain exactly one <main> landmark.",
			owner: "Frontend / IA",
			tags: ["WCAG 1.3.1"],
			displayMode: "collapsed",
		},
		"landmark-unique": {
			screenshot: "never",
			summary:
				"Landmarks must have unique labels when the same role appears multiple times.",
			owner: "Frontend / IA",
			tags: ["WCAG 1.3.1"],
			displayMode: "collapsed",
		},
		region: {
			screenshot: "never",
			summary: "Wrap major page areas in landmark roles.",
			owner: "Frontend / IA",
			tags: ["WCAG 2.4.1"],
			displayMode: "collapsed",
		},
		"multiple-ways": {
			screenshot: "never",
			summary: "Provide multiple navigation mechanisms.",
			owner: "Product / IA",
			tags: ["WCAG 2.4.5", "Level AA"],
			displayMode: "collapsed",
		},
		"link-name": {
			screenshot: "semantic",
			summary: "Links need discernible text or aria-labels.",
			owner: "Frontend",
			tags: ["WCAG 2.4.4"],
		},
		"button-name": {
			screenshot: "semantic",
			summary: "Buttons must have discernible accessible names.",
			owner: "Frontend",
			tags: ["WCAG 4.1.2"],
		},
		label: {
			screenshot: "semantic",
			summary: "Form inputs must have associated labels.",
			owner: "Frontend",
			tags: ["WCAG 1.3.1", "Level A"],
		},
		"form-field-multiple-labels": {
			screenshot: "semantic",
			summary: "Form fields should not have multiple labels.",
			owner: "Frontend",
			tags: ["WCAG 1.3.1"],
			displayMode: "collapsed",
		},
		"aria-valid-attr": {
			screenshot: "never",
			summary: "ARIA attributes must be valid.",
			owner: "Frontend",
			tags: ["WCAG 4.1.2"],
			displayMode: "collapsed",
		},
		"aria-valid-attr-value": {
			screenshot: "never",
			summary: "ARIA attribute values must be valid.",
			owner: "Frontend",
			tags: ["WCAG 4.1.2"],
			displayMode: "collapsed",
		},
		"aria-allowed-attr": {
			screenshot: "never",
			summary: "ARIA attributes must be allowed for the element's role.",
			owner: "Frontend",
			tags: ["WCAG 4.1.2"],
			displayMode: "collapsed",
		},
		"aria-required-attr": {
			screenshot: "never",
			summary: "Required ARIA attributes must be present.",
			owner: "Frontend",
			tags: ["WCAG 4.1.2"],
			displayMode: "collapsed",
		},
		"aria-required-children": {
			screenshot: "never",
			summary: "ARIA roles must contain required child roles.",
			owner: "Frontend",
			tags: ["WCAG 1.3.1"],
			displayMode: "collapsed",
		},
		"aria-required-parent": {
			screenshot: "never",
			summary: "ARIA roles must be contained by required parent roles.",
			owner: "Frontend",
			tags: ["WCAG 1.3.1"],
			displayMode: "collapsed",
		},
		"aria-roles": {
			screenshot: "never",
			summary: "ARIA roles must be valid.",
			owner: "Frontend",
			tags: ["WCAG 4.1.2"],
			displayMode: "collapsed",
		},
		"aria-hidden-focus": {
			screenshot: "never",
			summary:
				"Elements with aria-hidden should not contain focusable elements.",
			owner: "Frontend",
			tags: ["WCAG 4.1.2"],
			displayMode: "collapsed",
		},
		"aria-hidden-body": {
			screenshot: "never",
			summary: "aria-hidden should not be set on the document body.",
			owner: "Frontend",
			tags: ["WCAG 4.1.2"],
			displayMode: "collapsed",
		},
		"duplicate-id": {
			screenshot: "never",
			summary: "Element IDs must be unique.",
			owner: "Frontend",
			tags: ["WCAG 4.1.1"],
			displayMode: "collapsed",
		},
		"duplicate-id-active": {
			screenshot: "never",
			summary: "Active element IDs must be unique.",
			owner: "Frontend",
			tags: ["WCAG 4.1.1"],
			displayMode: "collapsed",
		},
		"duplicate-id-aria": {
			screenshot: "never",
			summary: "IDs used in ARIA must be unique.",
			owner: "Frontend",
			tags: ["WCAG 4.1.1"],
			displayMode: "collapsed",
		},
		"heading-order": {
			screenshot: "semantic",
			summary: "Heading levels should increase by one.",
			owner: "Content / IA",
			tags: ["WCAG 1.3.1"],
			displayMode: "collapsed",
		},
		"empty-heading": {
			screenshot: "semantic",
			summary: "Headings must have discernible text.",
			owner: "Content",
			tags: ["WCAG 1.3.1"],
			displayMode: "collapsed",
		},
		bypass: {
			screenshot: "never",
			summary: "Page must have a way to bypass repeated content.",
			owner: "Frontend / IA",
			tags: ["WCAG 2.4.1", "Level A"],
			displayMode: "collapsed",
		},
		"skip-link": {
			screenshot: "never",
			summary: "Skip links should be provided.",
			owner: "Frontend",
			tags: ["WCAG 2.4.1"],
			displayMode: "collapsed",
		},
		tabindex: {
			screenshot: "never",
			summary: "tabindex values should not be greater than 0.",
			owner: "Frontend",
			tags: ["WCAG 2.4.3"],
			displayMode: "collapsed",
		},
		"document-title": {
			screenshot: "never",
			summary: "Documents must have a <title> element.",
			owner: "Content",
			tags: ["WCAG 2.4.2", "Level A"],
			displayMode: "collapsed",
		},
		"meta-viewport": {
			screenshot: "never",
			summary: "Zooming and scaling should not be disabled.",
			owner: "Frontend",
			tags: ["WCAG 1.4.4"],
			displayMode: "collapsed",
		},
		"meta-refresh": {
			screenshot: "never",
			summary: "Pages should not auto-refresh.",
			owner: "Frontend",
			tags: ["WCAG 2.2.1"],
			displayMode: "collapsed",
		},
		"html-xml-lang-mismatch": {
			screenshot: "never",
			summary: "HTML lang and xml:lang attributes must match.",
			owner: "Content",
			tags: ["WCAG 3.1.1"],
			displayMode: "collapsed",
		},
		list: {
			screenshot: "never",
			summary: "Lists must be marked up correctly.",
			owner: "Frontend",
			tags: ["WCAG 1.3.1"],
			displayMode: "collapsed",
		},
		listitem: {
			screenshot: "never",
			summary: "List items must be contained in a list.",
			owner: "Frontend",
			tags: ["WCAG 1.3.1"],
			displayMode: "collapsed",
		},
		"definition-list": {
			screenshot: "never",
			summary: "Definition lists must be marked up correctly.",
			owner: "Frontend",
			tags: ["WCAG 1.3.1"],
			displayMode: "collapsed",
		},
		dlitem: {
			screenshot: "never",
			summary: "Definition list items must be contained in a definition list.",
			owner: "Frontend",
			tags: ["WCAG 1.3.1"],
			displayMode: "collapsed",
		},
		"table-fake-caption": {
			screenshot: "never",
			summary: "Tables should use <caption> instead of fake captions.",
			owner: "Frontend",
			tags: ["WCAG 1.3.1"],
			displayMode: "collapsed",
		},
		"td-headers-attr": {
			screenshot: "never",
			summary: "Table header references must be valid.",
			owner: "Frontend",
			tags: ["WCAG 1.3.1"],
			displayMode: "collapsed",
		},
		"th-has-data-cells": {
			screenshot: "never",
			summary: "Table headers must be associated with data cells.",
			owner: "Frontend",
			tags: ["WCAG 1.3.1"],
			displayMode: "collapsed",
		},
		"scope-attr-valid": {
			screenshot: "never",
			summary: "The scope attribute must be valid.",
			owner: "Frontend",
			tags: ["WCAG 1.3.1"],
			displayMode: "collapsed",
		},
		"video-caption": {
			screenshot: "never",
			summary: "Videos must have captions.",
			owner: "Content",
			tags: ["WCAG 1.2.2", "Level A"],
			displayMode: "collapsed",
		},
		"audio-caption": {
			screenshot: "never",
			summary: "Audio must have captions or transcripts.",
			owner: "Content",
			tags: ["WCAG 1.2.1"],
			displayMode: "collapsed",
		},
		"frame-title": {
			screenshot: "never",
			summary: "Frames must have accessible names.",
			owner: "Frontend",
			tags: ["WCAG 4.1.2"],
			displayMode: "collapsed",
		},
		"frame-title-unique": {
			screenshot: "never",
			summary: "Frame titles should be unique.",
			owner: "Frontend",
			tags: ["WCAG 4.1.2"],
			displayMode: "collapsed",
		},
		"server-side-image-map": {
			screenshot: "never",
			summary: "Server-side image maps should not be used.",
			owner: "Frontend",
			tags: ["WCAG 2.1.1"],
			displayMode: "collapsed",
		},
		"no-autoplay-audio": {
			screenshot: "never",
			summary: "Audio should not autoplay.",
			owner: "Frontend",
			tags: ["WCAG 1.4.2"],
			displayMode: "collapsed",
		},
		blink: {
			screenshot: "never",
			summary: "Blinking content should be avoided.",
			owner: "Design",
			tags: ["WCAG 2.2.2"],
			displayMode: "collapsed",
		},
		marquee: {
			screenshot: "never",
			summary: "Marquee elements should not be used.",
			owner: "Frontend",
			tags: ["WCAG 2.2.2"],
			displayMode: "collapsed",
		},
	},
};

let cachedBehavior: RuleBehaviorFile | null = null;

function loadBehaviorFile(
	filePath: string | undefined,
): RuleBehaviorFile | null {
	if (!filePath) {
		return null;
	}
	try {
		const resolved = path.isAbsolute(filePath)
			? filePath
			: path.resolve(process.cwd(), filePath);
		if (!fs.existsSync(resolved)) {
			return null;
		}
		const raw = fs.readFileSync(resolved, "utf-8");
		const parsed = JSON.parse(raw) as RuleBehaviorFile;
		return parsed;
	} catch (err) {
		console.warn(`[RuleBehavior] Failed to load config from ${filePath}:`, err);
		return null;
	}
}

function mergeBehaviorFiles(
	base: RuleBehaviorFile,
	override?: RuleBehaviorFile | null,
): RuleBehaviorFile {
	if (!override) {
		return { ...base };
	}
	const mergedDefaults: RuleBehavior = {
		...(base.defaults ?? {}),
		...(override.defaults ?? {}),
	};
	const mergedRules: Record<string, RuleBehavior> = { ...(base.rules ?? {}) };
	for (const [ruleId, behavior] of Object.entries(override.rules ?? {})) {
		mergedRules[ruleId] = {
			...(mergedRules[ruleId] ?? {}),
			...behavior,
		};
	}
	return { defaults: mergedDefaults, rules: mergedRules };
}

function loadConfig(): RuleBehaviorFile {
	if (cachedBehavior) {
		return cachedBehavior;
	}
	const envPath = process.env.RULE_BEHAVIOR_CONFIG_PATH;
	const dataDirPath = path.join(DATA_DIR, "rule-behaviors.json");
	const override = loadBehaviorFile(envPath) ?? loadBehaviorFile(dataDirPath);
	cachedBehavior = mergeBehaviorFiles(BUILT_IN_BEHAVIORS, override);
	return cachedBehavior;
}

export function getRuleBehavior(ruleId?: string): RuleBehavior {
	const config = loadConfig();
	const defaults = config.defaults ?? { screenshot: "auto" };
	if (!ruleId) {
		return { ...defaults };
	}
	const rule = config.rules?.[ruleId];
	return {
		...defaults,
		...(rule ?? {}),
	};
}

export function getScreenshotPolicy(ruleId?: string): ScreenshotPolicy {
	return getRuleBehavior(ruleId).screenshot ?? "auto";
}

export function shouldCaptureScreenshot(ruleId?: string): boolean {
	const policy = getScreenshotPolicy(ruleId);
	return policy === "auto" || policy === "always";
}

export function shouldCaptureSemanticScreenshot(ruleId?: string): boolean {
	return getScreenshotPolicy(ruleId) === "semantic";
}
