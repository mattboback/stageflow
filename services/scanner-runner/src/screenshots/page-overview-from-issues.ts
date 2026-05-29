import type { Page } from 'playwright';

import type { Issue } from '../core/types';
import type { AxeScreenshotService } from './AxeScreenshotService';
import type { PageOverviewResult, PageOverviewViolation } from './axe/types';

interface MetadataNode {
	target?: unknown;
	selector?: unknown;
}

/**
 * Build the page-overview violation list any scanner needs from its issues.
 *
 * Every issue becomes one "violation" keyed by its rule id. Each occurrence
 * node contributes a target selector (preferring `metadata.nodes[].target`,
 * falling back to `metadata.nodes[].selector`). Issues without a resolvable
 * selector still appear in the screenshot — they just get no element overlay,
 * which is exactly the page-level evidence fallback we want for page-global
 * findings (headers, meta tags, etc.).
 */
export function issuesToPageOverviewViolations(issues: Issue[]): PageOverviewViolation[] {
	return issues.map((issue) => {
		const rawNodes = Array.isArray(issue.metadata?.nodes)
			? (issue.metadata.nodes as MetadataNode[])
			: [];

		const nodes = rawNodes
			.map((node) => {
				const target = Array.isArray(node?.target)
					? node.target.filter((t): t is string => typeof t === 'string' && t.trim().length > 0)
					: typeof node?.selector === 'string' && node.selector.trim().length > 0
						? [node.selector]
						: [];
				return target.length > 0 ? { target } : null;
			})
			.filter((node): node is { target: string[] } => node !== null);

		return {
			id: issue.id,
			impact: issue.severity,
			...(nodes.length > 0 ? { nodes } : {})
		};
	});
}

/**
 * Shared page-overview capture used by every scanner that can locate its
 * findings in the live DOM. Wraps the (already scanner-agnostic)
 * AxeScreenshotService.capturePageOverview with the issue→violation mapping and
 * a timeout so a slow/hostile page can never wedge a scan.
 */
export async function capturePageOverviewFromIssues(params: {
	service: Pick<AxeScreenshotService, 'capturePageOverview'>;
	page: Page;
	issues: Issue[];
	screenshotsDir: string;
	pageId: string;
	scannerId: string;
	timeoutMs?: number;
	logger?: { warn(message: string, meta?: unknown): void };
}): Promise<PageOverviewResult | null> {
	const { service, page, issues, screenshotsDir, pageId, scannerId, logger } = params;
	const timeoutMs = params.timeoutMs ?? 30_000;
	const violations = issuesToPageOverviewViolations(issues);

	try {
		return await Promise.race([
			service.capturePageOverview(page, violations, screenshotsDir, pageId, { scannerId }),
			new Promise<never>((_, reject) => {
				setTimeout(() => {
					reject(new Error('Page overview capture timed out'));
				}, timeoutMs);
			})
		]);
	} catch (error) {
		logger?.warn('Page overview capture failed or timed out, continuing without it', {
			scannerId,
			pageId,
			error: error instanceof Error ? error.message : String(error)
		});
		return null;
	}
}
