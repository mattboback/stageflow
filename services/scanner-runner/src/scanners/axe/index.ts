import type { Page } from 'playwright';

import AxeBuilder from '@axe-core/playwright';
import { mkdir } from 'node:fs/promises';
import { join } from 'node:path';

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
} from '../../screenshots/axe-screenshot-service';
import { SCANNER_VERSION } from '../version';
import {
	getReportableIncompleteResults,
	mapIncompleteNodeToIssue,
	mapViolationToIssue
} from './issue-mapper';
import {
	DEFAULT_DYNAMIC_CONTENT_WAIT_MS,
	NETWORKIDLE_TIMEOUT_MS,
	parseAxeOptions,
	withTimeoutFallback,
	type AxeOptions
} from './options';
import type { AxeNode, AxeViolationResult } from './types';

export type { AxeOptions } from './options';

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
			const reportableIncompleteResults = getReportableIncompleteResults(
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
						const issue = mapViolationToIssue(
							violation,
							screenshotResult,
							enrichedNodes,
							this.metadata.name
						);
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
				issues.push(
					mapIncompleteNodeToIssue(result, enrichedNodes[index] ?? node, index, this.metadata.name)
				);
			});
		}

		return issues;
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
	private async enrichNodesWithContext(page: Page, nodes: AxeNode[]): Promise<AxeNode[]> {
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
}
