import type { IssueDetail } from '../types/unified-report';

/**
 * Classifies an aggregated issue so the UI can present it correctly.
 *
 * - `element`  — has at least one occurrence with a concrete DOM target
 *                (selector, ancestorPath, contextHtml, or html). The user
 *                can be shown a screenshot crop, selector, and HTML.
 * - `url`      — has occurrences but they describe a URL/link/page-level
 *                target rather than a DOM element (e.g. broken-link
 *                scanners). No DOM evidence available, but the URL itself
 *                is the evidence.
 * - `manual`   — Lighthouse "manual" / informational audits and other
 *                page-level checks that explicitly require human review.
 *                No DOM occurrence is reported.
 * - `page`     — page-level finding without DOM target and not flagged as
 *                manual review (e.g. missing meta tag, header policy).
 * - `scanner`  — scanner-level diagnostic (no occurrences, no page).
 */
export type IssueKind = 'element' | 'url' | 'manual' | 'page' | 'scanner';

const MANUAL_RULE_HINTS = [
	'manual',
	'must-be-verified',
	'should-be-verified',
	'requires manual',
	'manual review',
	'manually verified'
];

const URL_SCANNER_HINTS = ['link-checker', 'linkchecker', 'broken-link', 'url'];

function hasElementTarget(issue: IssueDetail): boolean {
	const occurrences = issue.occurrences ?? [];
	if (occurrences.length === 0) return false;
	return occurrences.some((occ) => {
		if (!occ) return false;
		return Boolean(
			occ.selector ||
				occ.ancestorPath ||
				occ.contextHtml ||
				occ.html ||
				occ.elementId ||
				occ.boundingBox
		);
	});
}

function looksLikeManualReview(issue: IssueDetail): boolean {
	const occurrences = issue.occurrences ?? [];
	if (occurrences.length > 0) return false;

	const haystack =
		`${issue.title ?? ''} ${issue.description ?? ''} ${issue.ruleId ?? ''}`.toLowerCase();
	if (MANUAL_RULE_HINTS.some((hint) => haystack.includes(hint))) return true;

	// Lighthouse informational/manual audits show up as severity=info with no
	// occurrences. Treat them as manual review by default — they are usually
	// "verify X" items rather than concrete failures.
	if (issue.scanner === 'lighthouse' && issue.severity === 'info') return true;

	return false;
}

function looksLikeUrlIssue(issue: IssueDetail): boolean {
	if (URL_SCANNER_HINTS.some((hint) => issue.scanner?.toLowerCase().includes(hint))) return true;
	const occurrences = issue.occurrences ?? [];
	if (occurrences.length === 0) return false;
	return occurrences.every(
		(occ) =>
			!!occ && !occ.selector && !occ.ancestorPath && !occ.contextHtml && !occ.html && !occ.elementId
	);
}

export function getIssueKind(issue: IssueDetail): IssueKind {
	if (hasElementTarget(issue)) return 'element';
	if (looksLikeManualReview(issue)) return 'manual';
	if (looksLikeUrlIssue(issue)) return 'url';
	if (!issue.pageId) return 'scanner';
	return 'page';
}

export function isManualReviewIssue(issue: IssueDetail): boolean {
	return getIssueKind(issue) === 'manual';
}

export function getIssueKindLabel(kind: IssueKind): string {
	switch (kind) {
		case 'element':
			return 'Element issue';
		case 'url':
			return 'URL issue';
		case 'manual':
			return 'Manual review';
		case 'page':
			return 'Page issue';
		case 'scanner':
			return 'Scanner diagnostic';
	}
}

/**
 * Rewrites titles for manual-review issues so they don't read like a pass.
 * Lighthouse manual audits ship as positive-sounding statements
 * ("Interactive controls are keyboard focusable") that imply success when in
 * fact the scanner is asking a human to verify them.
 */
export function rewriteIssueTitle(issue: IssueDetail): string {
	const original = issue.title ?? issue.ruleId ?? 'Issue';
	if (!isManualReviewIssue(issue)) return original;
	const trimmed = original.trim();
	if (/^(verify|review|manual|check|confirm)\b/i.test(trimmed)) return trimmed;
	return `Verify: ${trimmed}`;
}
