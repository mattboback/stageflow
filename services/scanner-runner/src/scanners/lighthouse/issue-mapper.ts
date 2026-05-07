import type { ScanStageLogger } from '../../core/scan-stage-logger';
import type { Issue, ScannerLogger } from '../../core/types';
import type { LighthouseIssueNode, LighthouseResult } from './types';

import { extractContextSnippet } from '../../screenshots/axe/context-snippet';
import {
	extractAuditNodes,
	getAuditCategory,
	getAuditNodeCount,
	mapScoreToSeverity
} from './result-parser';

export function getHelpUrl(auditId: string): string {
	const docsBase = 'https://developer.chrome.com/docs/lighthouse';
	const accessibilityAudits: Record<string, string> = {
		accesskeys: `${docsBase}/accessibility/accesskeys/`,
		'aria-allowed-attr': `${docsBase}/accessibility/aria-allowed-attr/`,
		'aria-hidden-body': `${docsBase}/accessibility/aria-hidden-body/`,
		'aria-hidden-focus': `${docsBase}/accessibility/aria-hidden-focus/`,
		'aria-required-attr': `${docsBase}/accessibility/aria-required-attr/`,
		'aria-roles': `${docsBase}/accessibility/aria-roles/`,
		'aria-valid-attr-value': `${docsBase}/accessibility/aria-valid-attr-value/`,
		'aria-valid-attr': `${docsBase}/accessibility/aria-valid-attr/`,
		'button-name': `${docsBase}/accessibility/button-name/`,
		bypass: `${docsBase}/accessibility/bypass/`,
		'color-contrast': `${docsBase}/accessibility/color-contrast/`,
		'document-title': `${docsBase}/accessibility/document-title/`,
		'duplicate-id-aria': `${docsBase}/accessibility/duplicate-id-aria/`,
		'form-field-multiple-labels': `${docsBase}/accessibility/form-field-multiple-labels/`,
		'frame-title': `${docsBase}/accessibility/frame-title/`,
		'heading-order': `${docsBase}/accessibility/heading-order/`,
		'html-has-lang': `${docsBase}/accessibility/html-has-lang/`,
		'html-lang-valid': `${docsBase}/accessibility/html-lang-valid/`,
		'image-alt': `${docsBase}/accessibility/image-alt/`,
		'input-image-alt': `${docsBase}/accessibility/input-image-alt/`,
		label: `${docsBase}/accessibility/label/`,
		'link-name': `${docsBase}/accessibility/link-name/`,
		list: `${docsBase}/accessibility/list/`,
		listitem: `${docsBase}/accessibility/listitem/`,
		'meta-viewport': `${docsBase}/accessibility/meta-viewport/`,
		'object-alt': `${docsBase}/accessibility/object-alt/`,
		tabindex: `${docsBase}/accessibility/tabindex/`,
		'td-headers-attr': `${docsBase}/accessibility/td-headers-attr/`,
		'th-has-data-cells': `${docsBase}/accessibility/th-has-data-cells/`,
		'valid-lang': `${docsBase}/accessibility/valid-lang/`,
		'video-caption': `${docsBase}/accessibility/video-caption/`
	};

	return accessibilityAudits[auditId] ?? `${docsBase}/overview/`;
}

export function extractIssuesFromResult(deps: {
	lhResult: LighthouseResult;
	scannerName: string;
	logger: ScannerLogger;
	scanStageLogger: ScanStageLogger | null;
}): Issue[] {
	const { lhResult, scannerName, logger, scanStageLogger } = deps;
	const issues: Issue[] = [];

	const auditEntries = Object.entries(lhResult.audits ?? {});
	logger.info('Lighthouse audit extraction', {
		totalAudits: auditEntries.length,
		categories: Object.keys(lhResult.categories ?? {})
	});

	if (auditEntries.length === 0) {
		logger.warn('Lighthouse returned no audits');
		return issues;
	}

	let skippedPassed = 0;
	let skippedNotApplicable = 0;
	let skippedInformative = 0;
	const skippedManual = 0;

	for (const [auditId, audit] of auditEntries) {
		// Skip passed audits, informational, or not applicable
		if (audit.score === 1) {
			skippedPassed++;
			continue;
		}
		if (audit.scoreDisplayMode === 'informative') {
			skippedInformative++;
			continue;
		}
		if (audit.scoreDisplayMode === 'manual') {
			const category = getAuditCategory(auditId, lhResult.categories ?? {});
			const nodeCount = getAuditNodeCount(audit);
			const nodes = extractAuditNodes(audit);

			issues.push({
				id: auditId,
				scanner: scannerName,
				severity: 'info',
				category: category ?? 'manual-review',
				title: audit.title,
				description: `Manual verification required: ${audit.description}`,
				helpUrl: getHelpUrl(auditId),
				metadata: {
					score: audit.score,
					scoreDisplayMode: audit.scoreDisplayMode,
					displayValue: audit.displayValue,
					numericValue: audit.numericValue,
					numericUnit: audit.numericUnit,
					details: audit.details,
					nodeCount,
					nodes: nodes.slice(0, 5),
					lighthouseManual: true
				}
			});
			continue;
		}
		if (audit.score === null || audit.scoreDisplayMode === 'notApplicable') {
			skippedNotApplicable++;
			continue;
		}

		const category = getAuditCategory(auditId, lhResult.categories ?? {});

		const severity = mapScoreToSeverity(audit.score);

		const nodeCount = getAuditNodeCount(audit);
		const nodes = extractAuditNodes(audit);

		const issue: Issue = {
			id: auditId,
			scanner: scannerName,
			severity,
			category: category ?? 'general',
			title: audit.title,
			description: audit.description,
			helpUrl: getHelpUrl(auditId),
			metadata: {
				score: audit.score,
				scoreDisplayMode: audit.scoreDisplayMode,
				displayValue: audit.displayValue,
				numericValue: audit.numericValue,
				numericUnit: audit.numericUnit,
				details: audit.details,
				nodeCount,
				nodes: nodes.slice(0, 5)
			}
		};

		issues.push(issue);
	}

	logger.info('Lighthouse issue extraction complete', {
		issuesFound: issues.length,
		skippedPassed,
		skippedNotApplicable,
		skippedInformative,
		skippedManual
	});

	// Record extraction stats to stage log
	scanStageLogger?.recordEvent('lighthouse_extraction', {
		issuesFound: issues.length,
		skippedPassed,
		skippedNotApplicable,
		skippedInformative,
		skippedManual,
		totalProcessed: auditEntries.length
	});

	return issues;
}

export async function enrichIssuesWithContext(
	page: import('playwright').Page,
	issues: Issue[]
): Promise<void> {
	const BATCH_SIZE = 10;
	const NODE_TIMEOUT = 2_000;

	// Flatten all enrichment tasks
	const tasks: { issue: Issue; node: LighthouseIssueNode }[] = [];
	for (const issue of issues) {
		const metadata = issue.metadata;
		if (!metadata || typeof metadata !== 'object' || Array.isArray(metadata)) {
			continue;
		}

		const nodes = metadata.nodes as LighthouseIssueNode[] | undefined;
		if (!Array.isArray(nodes)) {
			continue;
		}

		// Limit to first 5 nodes per issue
		for (const node of nodes.slice(0, 5)) {
			tasks.push({ issue, node });
		}
	}

	// Process in batches
	for (let i = 0; i < tasks.length; i += BATCH_SIZE) {
		const batch = tasks.slice(i, i + BATCH_SIZE);
		await Promise.allSettled(
			batch.map(async ({ node }) => {
				const selector = node.selector ?? node.target?.[0];
				if (!selector) {
					return;
				}

				try {
					const ctx = await Promise.race([
						extractContextSnippet(page, selector),
						new Promise<undefined>((resolve) => {
							setTimeout(() => {
								resolve(undefined);
							}, NODE_TIMEOUT);
						})
					]);
					if (ctx) {
						node.contextHtml = ctx.contextHtml;
						node.ancestorPath = ctx.ancestorPath;
					}
				} catch {
					/* ignore individual failures */
				}
			})
		);
	}
}
