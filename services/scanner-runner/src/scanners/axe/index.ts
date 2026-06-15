import type { Page } from 'playwright';

import AxeBuilder from '@axe-core/playwright';
import { mkdir } from 'node:fs/promises';
import { join } from 'node:path';

import { getRuleBehavior } from '../../config/rule-behaviors';
import { getUserImpact } from '../../config/user-impact';
import {
	type Issue,
	type PageScanResult,
	type ScanContext,
	ScannerBase,
	type ScannerMetadata
} from '../../core';
import { extractContextSnippet } from '../../screenshots/axe/context-snippet';
import {
	AxeScreenshotService,
	type AxeViolation,
	type EnhancedScreenshotResult,
	type PageOverviewViolation,
	type ViolationCaptureFailure
} from '../../screenshots/AxeScreenshotService';
import { normalizeSeverity } from '../../utils/severity';
import { SCANNER_VERSION } from '../version';

/**
 * Default milliseconds to wait after load for dynamic content to render.
 * SPAs and JS-heavy sites often need this extra time for React/Vue/Svelte hydration.
 * Reduced from 500ms to 50ms since most modern frameworks hydrate quickly.
 */
const DEFAULT_DYNAMIC_CONTENT_WAIT_MS = 50;

/**
 * Maximum time to wait for networkidle state before proceeding with scan.
 * Prevents hanging on sites with persistent connections (WebSockets, long-polling).
 */
const NETWORKIDLE_TIMEOUT_MS = 5_000;

interface AxeCheckResult {
	id?: string;
	data?: Record<string, unknown> | null;
}

interface AxeNode {
	target?: string[];
	html?: string;
	failureSummary?: string;
	contextHtml?: string;
	ancestorPath?: string;
	any?: AxeCheckResult[];
	all?: AxeCheckResult[];
}

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

interface AxeViolationResult {
	id?: string;
	impact?: string;
	description?: string;
	help?: string;
	helpUrl?: string;
	nodes?: AxeNode[];
	tags?: string[];
}

/**
 * Options for the axe scanner.
 * Pass via SCANNER_OPTIONS environment variable as JSON.
 */
export interface AxeOptions {
	/**
	 * Milliseconds to wait after networkidle for dynamic content to render.
	 * Increase for heavy SPAs (React, Vue, Svelte) that need more hydration time.
	 * Set to 0 to skip the wait entirely.
	 * Default: 500
	 */
	dynamicContentWaitMs?: number;

	/**
	 * axe-core rules to disable. Useful for suppressing known false positives.
	 * Example: ["color-contrast", "region"]
	 */
	disabledRules?: string[];

	/**
	 * WCAG tags to limit the scan to. If not set, all rules run.
	 * Example: ["wcag2a", "wcag2aa", "wcag21aa"]
	 */
	runOnlyTags?: string[];
}

function parseAxeOptions(raw: unknown): AxeOptions {
	if (!raw || typeof raw !== 'object' || Array.isArray(raw)) {
		return { dynamicContentWaitMs: DEFAULT_DYNAMIC_CONTENT_WAIT_MS };
	}

	const record = raw as Record<string, unknown>;
	const options: AxeOptions = {};

	// Parse dynamicContentWaitMs
	const waitMs = record.dynamicContentWaitMs;
	if (typeof waitMs === 'number' && waitMs >= 0) {
		options.dynamicContentWaitMs = waitMs;
	} else {
		options.dynamicContentWaitMs = DEFAULT_DYNAMIC_CONTENT_WAIT_MS;
	}

	// Parse disabledRules
	const disabledRules = record.disabledRules;
	if (Array.isArray(disabledRules)) {
		options.disabledRules = disabledRules.filter(
			(r): r is string => typeof r === 'string' && r.length > 0
		);
	}

	// Parse runOnlyTags
	const runOnlyTags = record.runOnlyTags;
	if (Array.isArray(runOnlyTags)) {
		options.runOnlyTags = runOnlyTags.filter(
			(t): t is string => typeof t === 'string' && t.length > 0
		);
	}

	return options;
}

async function withTimeoutFallback<T>(
	operation: Promise<T>,
	timeoutMs: number,
	fallback: () => T
): Promise<T> {
	let timeout: ReturnType<typeof setTimeout> | undefined;

	try {
		return await Promise.race([
			operation,
			new Promise<T>((resolve) => {
				timeout = setTimeout(() => {
					resolve(fallback());
				}, timeoutMs);
			})
		]);
	} finally {
		if (timeout !== undefined) {
			clearTimeout(timeout);
		}
	}
}

export class AxeScanner extends ScannerBase {
	readonly metadata: ScannerMetadata = {
		name: 'axe',
		version: SCANNER_VERSION,
		description: 'Accessibility scanner powered by axe-core'
	};

	private screenshotService: AxeScreenshotService;
	private options: AxeOptions = {
		dynamicContentWaitMs: DEFAULT_DYNAMIC_CONTENT_WAIT_MS
	};

	constructor() {
		super();
		this.screenshotService = new AxeScreenshotService();
	}

	protected override async initialize(): Promise<void> {
		await super.initialize();
		this.options = parseAxeOptions(this.config.options);
		this.logger.info('Axe options', {
			dynamicContentWaitMs: this.options.dynamicContentWaitMs,
			disabledRules: this.options.disabledRules,
			runOnlyTags: this.options.runOnlyTags
		});
	}

	async scanPage(context: ScanContext): Promise<PageScanResult> {
		const { page, pageEntry, resultsDir, logger } = context;
		const startedAt = new Date().toISOString();
		const hrStart = process.hrtime.bigint();

		try {
			// Wait for networkidle with timeout to handle sites with persistent connections.
			// If timeout is reached, proceed anyway - the page is likely ready enough for scanning.
			try {
				await withTimeoutFallback(
					page.waitForLoadState('networkidle'),
					NETWORKIDLE_TIMEOUT_MS,
					() => undefined
				);
			} catch {
				// Ignore networkidle timeout - page may still be scannable
				logger.debug('Networkidle wait timed out, proceeding with scan', {
					url: pageEntry.url,
					timeoutMs: NETWORKIDLE_TIMEOUT_MS
				});
			}

			const waitMs = this.options.dynamicContentWaitMs ?? DEFAULT_DYNAMIC_CONTENT_WAIT_MS;
			if (waitMs > 0) {
				await page.waitForTimeout(waitMs);
			}

			logger.debug('Running axe-core analysis', {
				url: pageEntry.url,
				dynamicContentWaitMs: waitMs
			});

			let axe = new AxeBuilder({ page });

			// Apply disabled rules if configured
			if (this.options.disabledRules && this.options.disabledRules.length > 0) {
				axe = axe.disableRules(this.options.disabledRules);
			}

			// Apply runOnly tags if configured
			if (this.options.runOnlyTags && this.options.runOnlyTags.length > 0) {
				axe = axe.withTags(this.options.runOnlyTags);
			}

			const axeResults = await axe.analyze();

			logger.info('Axe analysis complete', {
				url: pageEntry.url,
				violations: axeResults.violations.length,
				passes: axeResults.passes.length,
				incomplete: axeResults.incomplete.length,
				inapplicable: axeResults.inapplicable.length
			});

			if (axeResults.violations.length > 0) {
				logger.info('Axe violations found', {
					violationIds: axeResults.violations.map((v) => ({
						id: v.id,
						impact: v.impact,
						nodes: v.nodes.length
					}))
				});
			}

			const violations = axeResults.violations as AxeViolationResult[];
			const reportableIncompleteResults = this.getReportableIncompleteResults(
				axeResults.incomplete as AxeViolationResult[]
			);

			await mkdir(resultsDir, { recursive: true });
			const screenshotsDir = join(resultsDir, 'screenshots');
			await mkdir(screenshotsDir, { recursive: true });

			const issues: Issue[] = [];

			// Process violations in parallel batches for better performance
			const BATCH_SIZE = 5;
			const SCREENSHOT_TIMEOUT = 10_000;
			const ENRICHMENT_TIMEOUT = 5_000;

			const processViolation = async (violation: AxeViolationResult) => {
				const [screenshotResult, enrichedNodes] = await Promise.all([
					withTimeoutFallback(
						this.captureViolationScreenshot(page, violation as AxeViolation, screenshotsDir),
						SCREENSHOT_TIMEOUT,
						() => null
					),
					withTimeoutFallback(
						this.enrichNodesWithContext(page, violation.nodes ?? []),
						ENRICHMENT_TIMEOUT,
						() => [...(violation.nodes ?? [])]
					)
				]);
				return { violation, screenshotResult, enrichedNodes };
			};

			// Process in batches to avoid overwhelming the browser
			for (let i = 0; i < violations.length; i += BATCH_SIZE) {
				const batch = violations.slice(i, i + BATCH_SIZE);
				const results = await Promise.allSettled(batch.map(processViolation));

				for (const result of results) {
					if (result.status === 'fulfilled') {
						const { violation, screenshotResult, enrichedNodes } = result.value;
						const issue = this.mapViolationToIssue(violation, screenshotResult, enrichedNodes);
						issues.push(issue);
					}
				}
			}

			const incompleteIssues = await this.mapIncompleteResultsToIssues(
				page,
				reportableIncompleteResults
			);
			if (incompleteIssues.length > 0) {
				logger.info('Axe incomplete results promoted to review issues', {
					issueCount: incompleteIssues.length,
					ruleIds: [...new Set(incompleteIssues.map((issue) => issue.id))]
				});
				issues.push(...incompleteIssues);
			}

			// Capture page overview screenshot with all violations highlighted
			const pageOverviewViolations: PageOverviewViolation[] = [
				...violations,
				// Incomplete results are split into one issue per node by mapIncompleteResultsToIssues.
				// Flatten them into single-node violations here so each gets its own fingerprint,
				// matching the IDs generated by WebServerFormatter.
				...reportableIncompleteResults.flatMap((v) => {
					const nodes = v.nodes ?? [];
					if (nodes.length <= 1) {
						return [
							{
								id: v.id ?? 'unknown',
								...(v.impact !== undefined ? { impact: v.impact } : {}),
								...(v.nodes !== undefined ? { nodes: v.nodes } : {})
							}
						];
					}
					return nodes.map((node) => ({
						id: v.id ?? 'unknown',
						...(v.impact !== undefined ? { impact: v.impact } : {}),
						nodes: [node]
					}));
				})
			].map((v) => ({
				id: v.id ?? 'unknown',
				...(v.impact !== undefined ? { impact: v.impact } : {}),
				...(v.nodes !== undefined ? { nodes: v.nodes } : {})
			}));

			const pageOverview = await this.screenshotService.capturePageOverview(
				page,
				pageOverviewViolations,
				screenshotsDir,
				pageEntry.id,
				{ scannerId: this.metadata.name }
			);

			const finishedAt = new Date().toISOString();
			const durationNs = process.hrtime.bigint() - hrStart;
			const durationMs = Number(durationNs) / 1e6;

			return {
				pageId: pageEntry.id,
				url: pageEntry.url,
				path: pageEntry.path,
				success: true,
				issues,
				durationMs: Math.round(durationMs * 100) / 100,
				startedAt,
				finishedAt,
				artifacts: pageOverview ? [pageOverview.screenshotPath] : [],
				rawResults: {
					...axeResults,
					pageOverview: pageOverview
						? {
								screenshotFilename: pageOverview.screenshotFilename,
								pageWidth: pageOverview.pageWidth,
								pageHeight: pageOverview.pageHeight,
								elements: pageOverview.elements
							}
						: null
				}
			};
		} catch (error) {
			const finishedAt = new Date().toISOString();
			const durationNs = process.hrtime.bigint() - hrStart;
			const durationMs = Number(durationNs) / 1e6;

			return {
				pageId: pageEntry.id,
				url: pageEntry.url,
				path: pageEntry.path,
				success: false,
				issues: [],
				durationMs: Math.round(durationMs * 100) / 100,
				startedAt,
				finishedAt,
				error: error instanceof Error ? error.message : String(error)
			};
		}
	}

	private async captureViolationScreenshot(
		page: Page,
		violation: AxeViolation,
		screenshotsDir: string
	): Promise<EnhancedScreenshotResult | null> {
		try {
			const outcome = await this.screenshotService.captureViolationScreenshot(
				page,
				violation,
				screenshotsDir
			);
			if (outcome.status === 'captured') {
				this.logScreenshotFallbacks(violation.id, outcome.fallbacks);

				return outcome.screenshot;
			}

			if (outcome.status === 'failed') {
				this.logScreenshotFallbacks(violation.id, outcome.fallbacks);
				this.logger.debug('Screenshot capture failed', {
					ruleId: violation.id,
					step: outcome.failure.step,
					reason: outcome.failure.reason,
					message: outcome.failure.message
				});

				return null;
			}

			this.logger.debug('Screenshot capture skipped', {
				ruleId: violation.id,
				reason: outcome.reason
			});

			return null;
		} catch {
			this.logger.debug('Screenshot capture threw unexpected error', {
				ruleId: violation.id
			});

			return null;
		}
	}

	private getReportableIncompleteResults(
		incompleteResults: AxeViolationResult[]
	): AxeViolationResult[] {
		return incompleteResults.filter(
			(result) => this.shouldReportIncompleteResult(result) && (result.nodes?.length ?? 0) > 0
		);
	}

	private shouldReportIncompleteResult(result: AxeViolationResult): boolean {
		return this.isColorContrastRule(result.id);
	}

	private isColorContrastRule(ruleId: string | undefined): boolean {
		return ruleId === 'color-contrast' || ruleId === 'color-contrast-enhanced';
	}

	/**
	 * Pull axe's contrast check data (fgColor, bgColor, measured ratio, messageKey, …)
	 * off a node. Axe stores it on the node's check results, which the upstream
	 * types don't model but the runtime objects always carry.
	 */
	private extractContrastData(node: AxeNode | undefined): Record<string, unknown> | undefined {
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

	private async mapIncompleteResultsToIssues(
		page: Page,
		incompleteResults: AxeViolationResult[]
	): Promise<Issue[]> {
		const ENRICHMENT_TIMEOUT = 5_000;
		const issues: Issue[] = [];

		for (const result of incompleteResults) {
			const nodes = result.nodes ?? [];
			const enrichedNodes = await withTimeoutFallback(
				this.enrichNodesWithContext(page, nodes),
				ENRICHMENT_TIMEOUT,
				() => [...nodes]
			);
			nodes.forEach((node, index) => {
				issues.push(this.mapIncompleteNodeToIssue(result, enrichedNodes[index] ?? node, index));
			});
		}

		return issues;
	}

	private mapIncompleteNodeToIssue(
		result: AxeViolationResult,
		node: AxeNode,
		nodeIndex: number
	): Issue {
		const ruleId = result.id ?? 'unknown';
		const behavior = getRuleBehavior(ruleId);
		const userImpact = getUserImpact(ruleId);
		const selector = node.target?.[0];
		const category = this.extractCategory(result.tags);
		const contrastData = this.extractContrastData(node);

		const location = {
			...(selector !== undefined ? { selector } : {}),
			...(node.html !== undefined ? { html: node.html } : {})
		};

		return {
			id: ruleId,
			scanner: this.metadata.name,
			severity: normalizeSeverity(result.impact, 'moderate'),
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

	private logScreenshotFallbacks(
		ruleID: string | undefined,
		fallbacks: ViolationCaptureFailure[]
	): void {
		for (const fallback of fallbacks) {
			this.logger.debug('Screenshot strategy fallback', {
				ruleId: ruleID,
				step: fallback.step,
				reason: fallback.reason,
				message: fallback.message
			});
		}
	}

	/**
	 * Enrich violation nodes with DOM context snippets.
	 * Extracts contextHtml and ancestorPath for each node (up to 5).
	 */
	private async enrichNodesWithContext(
		page: import('playwright').Page,
		nodes: AxeNode[]
	): Promise<AxeNode[]> {
		const enrichedNodes: AxeNode[] = [];

		// Only process first 5 nodes (same limit as in mapViolationToIssue)
		const nodesToProcess = nodes.slice(0, 5);

		for (const node of nodesToProcess) {
			const selector = node.target?.[0];
			if (!selector) {
				// No selector to extract context from, keep original node
				enrichedNodes.push({ ...node });
				continue;
			}

			try {
				const contextResult = await extractContextSnippet(page, selector);
				enrichedNodes.push({
					...node,
					...(contextResult?.contextHtml !== undefined
						? { contextHtml: contextResult.contextHtml }
						: {}),
					...(contextResult?.ancestorPath !== undefined
						? { ancestorPath: contextResult.ancestorPath }
						: {})
				});
			} catch {
				// Failed to extract context, keep original node
				enrichedNodes.push({ ...node });
			}
		}

		return enrichedNodes;
	}

	private mapViolationToIssue(
		violation: AxeViolationResult,
		screenshotResult: EnhancedScreenshotResult | null | undefined,
		enrichedNodes: AxeNode[]
	): Issue {
		const severity = normalizeSeverity(violation.impact, 'info');
		const behavior = getRuleBehavior(violation.id);
		const userImpact = getUserImpact(violation.id);
		const nodes = violation.nodes ?? [];

		const primaryNode = enrichedNodes[0] ?? nodes[0];
		const selector = primaryNode?.target?.[0];

		const category = this.extractCategory(violation.tags);
		// First node only: a pragmatic prefill for the report's contrast verifier;
		// other occurrences may have different measured colors.
		const contrastData = this.isColorContrastRule(violation.id)
			? this.extractContrastData(nodes[0])
			: undefined;

		const location = {
			...(selector !== undefined ? { selector } : {}),
			...(primaryNode?.html !== undefined ? { html: primaryNode.html } : {})
		};

		return {
			id: violation.id ?? 'unknown',
			scanner: this.metadata.name,
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

	private extractCategory(tags?: string[]): string {
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
}
