/**
 * Pure mapping from axe-core results to StageFlow issues. Everything here is
 * side-effect free; the AxeScanner class owns page access and screenshots.
 */

import { getRuleBehavior } from '../../config/rule-behaviors';
import { getUserImpact } from '../../config/user-impact';
import type { Issue } from '../../core';
import type { EnhancedScreenshotResult } from '../../screenshots/AxeScreenshotService';
import { normalizeSeverity } from '../../utils/severity';
import type { AxeNode, AxeViolationResult } from './types';

/**
 * The fields of axe's color-contrast check data worth forwarding to the report.
 * Powers the in-report "Verify contrast" tool: pre-filled colors, measured
 * ratio, and the messageKey explaining why automatic verification failed.
 */
const CONTRAST_DATA_FIELDS = [
	'fgColor',
	'bgColor',
	'contrastRatio',
	'expectedContrastRatio',
	'fontSize',
	'fontWeight',
	'messageKey'
] as const;

export function isColorContrastRule(ruleId: string | undefined): boolean {
	return ruleId === 'color-contrast' || ruleId === 'color-contrast-enhanced';
}

/**
 * axe messageKeys that mean "there is no text here whose contrast could fail",
 * as opposed to "the contrast could not be determined".
 *
 * `nonBmp` fires on elements whose content is only non-text characters — icon
 * glyphs, arrows, decorative dividers. Asking a human to manually verify the
 * contrast of a character that carries no information is busywork, and it drowns
 * the findings that do need a look: scanning StageFlow's own landing page
 * produced 14 of these findings, 12 of them `nonBmp`.
 */
const NON_VERIFIABLE_CONTRAST_MESSAGE_KEYS = new Set(['nonBmp']);

function isVerifiableContrastIncomplete(node: AxeNode): boolean {
	const messageKey = extractContrastData(node)?.messageKey;

	return typeof messageKey !== 'string' || !NON_VERIFIABLE_CONTRAST_MESSAGE_KEYS.has(messageKey);
}

/**
 * Narrows axe's `incomplete` results to the color-contrast nodes a human could
 * actually adjudicate. Filters per node rather than per rule, because one
 * color-contrast result mixes genuinely-ambiguous nodes with vacuous ones.
 */
export function getReportableIncompleteResults(
	incompleteResults: AxeViolationResult[]
): AxeViolationResult[] {
	return incompleteResults
		.filter((result) => isColorContrastRule(result.id))
		.map((result) => ({
			...result,
			nodes: (result.nodes ?? []).filter(isVerifiableContrastIncomplete)
		}))
		.filter((result) => result.nodes.length > 0);
}

/**
 * Pull axe's contrast check data (fgColor, bgColor, measured ratio, messageKey, …)
 * off a node. Axe stores it on the node's check results, which the upstream
 * types don't model but the runtime objects always carry.
 */
export function extractContrastData(
	node: AxeNode | undefined
): Record<string, unknown> | undefined {
	const checks = [...(node?.any ?? []), ...(node?.all ?? [])];
	const data = checks.find(
		(check) => check.data && typeof check.data === 'object' && !Array.isArray(check.data)
	)?.data;
	if (!data) {
		return undefined;
	}

	const picked: Record<string, unknown> = {};
	for (const field of CONTRAST_DATA_FIELDS) {
		const value = data[field];
		if (value !== undefined && value !== null) {
			picked[field] = value;
		}
	}
	return Object.keys(picked).length > 0 ? picked : undefined;
}

export function extractCategory(tags?: string[]): string {
	if (!tags || tags.length === 0) {
		return 'accessibility';
	}

	const wcagTag = tags.find((t) => t.startsWith('wcag'));
	if (wcagTag) {
		return wcagTag;
	}

	if (tags.includes('best-practice')) {
		return 'best-practice';
	}

	const categoryTags = [
		'cat.color',
		'cat.forms',
		'cat.keyboard',
		'cat.language',
		'cat.name-role-value',
		'cat.parsing',
		'cat.semantics',
		'cat.sensory-and-visual-cues',
		'cat.structure',
		'cat.tables',
		'cat.text-alternatives',
		'cat.time-and-media'
	];

	for (const tag of tags) {
		if (categoryTags.includes(tag)) {
			return tag.replace('cat.', '');
		}
	}

	return 'accessibility';
}

export function mapIncompleteNodeToIssue(
	result: AxeViolationResult,
	node: AxeNode,
	nodeIndex: number,
	scannerName: string
): Issue {
	const ruleId = result.id ?? 'unknown';
	const behavior = getRuleBehavior(ruleId);
	const userImpact = getUserImpact(ruleId);
	const selector = node.target?.[0];
	const category = extractCategory(result.tags);
	const contrastData = extractContrastData(node);

	const location = {
		...(selector !== undefined ? { selector } : {}),
		...(node.html !== undefined ? { html: node.html } : {})
	};

	return {
		id: ruleId,
		scanner: scannerName,
		// Always moderate, never axe's impact. axe reports color-contrast at
		// `serious`, but an *incomplete* result means the check did not finish — the
		// text is unverified, not failing. Inheriting `serious` conflated the two and
		// made `--fail-on serious` unusable: it failed the build on ambiguity a human
		// still has to adjudicate. axe's own impact stays in metadata below.
		severity: 'moderate',
		category,
		title: 'Color contrast needs manual verification',
		description:
			'axe-core could not determine the background color for this text, so StageFlow cannot treat it as a pass. Review the actual foreground and background contrast.',
		...(Object.keys(location).length > 0 ? { location } : {}),
		...(result.helpUrl !== undefined ? { helpUrl: result.helpUrl } : {}),
		metadata: {
			impact: result.impact,
			tags: result.tags,
			nodeCount: 1,
			nodes: [
				{
					target: node.target,
					html: node.html,
					failureSummary: node.failureSummary,
					contextHtml: node.contextHtml,
					ancestorPath: node.ancestorPath
				}
			],
			axeIncomplete: true,
			...(contrastData !== undefined ? { contrastData } : {}),
			incompleteNodeIndex: nodeIndex,
			ruleBehavior: behavior,
			userImpact: {
				statement: userImpact.statement,
				affectedGroups: userImpact.affectedGroups,
				severity: userImpact.severity,
				userStory: userImpact.userStory
			}
		}
	};
}

export function mapViolationToIssue(
	violation: AxeViolationResult,
	screenshotResult: EnhancedScreenshotResult | null | undefined,
	enrichedNodes: AxeNode[],
	scannerName: string
): Issue {
	const severity = normalizeSeverity(violation.impact, 'info');
	const behavior = getRuleBehavior(violation.id);
	const userImpact = getUserImpact(violation.id);
	const nodes = violation.nodes ?? [];

	const primaryNode = enrichedNodes[0] ?? nodes[0];
	const selector = primaryNode?.target?.[0];

	const category = extractCategory(violation.tags);
	// First node only: a pragmatic prefill for the report's contrast verifier;
	// other occurrences may have different measured colors.
	const contrastData = isColorContrastRule(violation.id)
		? extractContrastData(nodes[0])
		: undefined;

	const location = {
		...(selector !== undefined ? { selector } : {}),
		...(primaryNode?.html !== undefined ? { html: primaryNode.html } : {})
	};

	return {
		id: violation.id ?? 'unknown',
		scanner: scannerName,
		severity,
		category,
		title: violation.help ?? violation.id ?? 'Accessibility Issue',
		description: behavior.summary ?? violation.description ?? '',
		...(Object.keys(location).length > 0 ? { location } : {}),
		...(violation.helpUrl !== undefined ? { helpUrl: violation.helpUrl } : {}),
		...(screenshotResult?.screenshot !== undefined
			? { screenshot: screenshotResult.screenshot }
			: {}),
		metadata: {
			impact: violation.impact,
			tags: violation.tags,
			nodeCount: nodes.length,
			nodes: enrichedNodes.slice(0, 5).map((node) => ({
				target: node.target,
				html: node.html,
				failureSummary: node.failureSummary,
				contextHtml: node.contextHtml,
				ancestorPath: node.ancestorPath
			})),
			ruleBehavior: behavior,
			...(contrastData !== undefined ? { contrastData } : {}),
			friendlyNode: screenshotResult?.friendlyNode,
			locationInfo: screenshotResult?.locationInfo,
			thumbnail: screenshotResult?.thumbnail,
			elementBounds: screenshotResult?.elementBounds,
			// User impact for stakeholder communication
			userImpact: {
				statement: userImpact.statement,
				affectedGroups: userImpact.affectedGroups,
				severity: userImpact.severity,
				userStory: userImpact.userStory
			}
		}
	};
}
