import { mkdir } from 'node:fs/promises';
import { join } from 'node:path';

import {
	type Issue,
	type IssueSeverity,
	type PageScanResult,
	type ScanContext,
	ScannerBase,
	type ScannerMetadata
} from '../../core';
import {
	AxeScreenshotService,
	type PageOverviewViolation
} from '../../screenshots/AxeScreenshotService';
import { SCANNER_VERSION } from '../version';
import {
	closeLaunchedChrome,
	launchChromeForLighthouse,
	resolveChromePath as resolveChromePathFn
} from './chrome-lifecycle';
import {
	enrichIssuesWithContext as enrichIssuesWithContextFn,
	enrichNodesWithContext as enrichNodesWithContextFn,
	extractIssuesFromResult,
	getHelpUrl as getHelpUrlFn
} from './issue-mapper';
import {
	loadLighthouseModule as loadLighthouseModuleFn,
	runLighthouseInvocation
} from './lighthouse-invoker';
import { isLighthousePrewarmEnabled } from './prewarm';
import {
	extractAuditNodes as extractAuditNodesFn,
	getAuditCategory as getAuditCategoryFn,
	getAuditNodeCount as getAuditNodeCountFn,
	mapScoreToSeverity as mapScoreToSeverityFn
} from './result-parser';
import {
	DEFAULT_LIGHTHOUSE_CATEGORIES,
	type LaunchedChrome,
	type LighthouseAudit,
	type LighthouseCategory,
	type LighthouseIssueNode,
	type LighthouseModule,
	type LighthouseOptions,
	type LighthouseResult,
	parseLighthouseOptions
} from './types';

export type { LighthouseOptions } from './types';

export class LighthouseScanner extends ScannerBase {
	readonly metadata: ScannerMetadata = {
		name: 'lighthouse',
		version: SCANNER_VERSION,
		description: 'Web quality scanner powered by Google Lighthouse'
	};

	private chrome: LaunchedChrome | null = null;
	private chromePromise: Promise<LaunchedChrome> | null = null;
	private lighthouseModulePromise: Promise<LighthouseModule> | null = null;
	private lighthouseQueue: Promise<void> = Promise.resolve();
	private screenshotService: AxeScreenshotService;
	private options: LighthouseOptions = {
		categories: DEFAULT_LIGHTHOUSE_CATEGORIES
	};

	constructor() {
		super();
		this.screenshotService = new AxeScreenshotService();
	}

	protected override async initialize(): Promise<void> {
		await super.initialize();
		this.options = parseLighthouseOptions(this.config.options);
		this.logger.info('Lighthouse options', {
			categories: this.options.categories
		});

		if (isLighthousePrewarmEnabled()) {
			this.prewarmRuntime();
		}
	}

	async scanPage(context: ScanContext): Promise<PageScanResult> {
		const { page, pageEntry, resultsDir } = context;
		const startedAt = new Date().toISOString();
		const hrStart = process.hrtime.bigint();

		try {
			await mkdir(resultsDir, { recursive: true });
			const screenshotsDir = join(resultsDir, 'screenshots');
			await mkdir(screenshotsDir, { recursive: true });

			const lhResult = await this.runLighthouse(page, pageEntry.url);

			// Lighthouse runs in its own Chrome instance. The Playwright page may have gone stale
			// while waiting in the serialized queue (especially with concurrency > 1). Re-navigate
			// to ensure we have a fresh, responsive page for context enrichment and screenshots.
			// Use 'domcontentloaded' instead of 'networkidle' for speed - we just need the DOM ready.
			try {
				this.logger.debug('Re-navigating Playwright page after Lighthouse', {
					url: pageEntry.url
				});

				// Prefer BrowserManager navigation to reuse runtime target validation.
				const browserManager = this.browserManager;
				if (browserManager) {
					await browserManager.navigateToPage(page, pageEntry.url, {
						type: 'domcontentloaded'
					});
				} else {
					await page.goto(pageEntry.url, {
						waitUntil: 'domcontentloaded',
						timeout: 15_000
					});
				}
			} catch (navError) {
				// If re-navigation fails, log it but continue - we'll try enrichment/screenshots anyway
				this.logger.warn(
					'Failed to re-navigate page after Lighthouse, continuing with stale page',
					{
						url: pageEntry.url,
						error: navError instanceof Error ? navError.message : String(navError)
					}
				);
			}

			const issues = this.extractIssues(lhResult);

			// Context enrichment and screenshots use Playwright page operations that can hang
			// if the page is unresponsive. Wrap with timeouts as a safety net.
			const enrichmentTimeout = 15_000; // Reduced from 30s
			try {
				await Promise.race([
					this.enrichIssuesWithContext(page, issues),
					new Promise<never>((_, reject) =>
						setTimeout(() => {
							reject(new Error('Context enrichment timed out'));
						}, enrichmentTimeout)
					)
				]);
			} catch (enrichError) {
				this.logger.warn('Context enrichment failed or timed out, continuing without enrichment', {
					error: enrichError instanceof Error ? enrichError.message : String(enrichError)
				});
			}

			const pageOverviewViolations: PageOverviewViolation[] = issues.map((issue) => {
				const issueNodes = (issue.metadata?.nodes as LighthouseIssueNode[] | undefined)?.map(
					(n) => ({
						...(n.target !== undefined ? { target: n.target } : {})
					})
				);

				return {
					id: issue.id,
					impact: issue.severity,
					...(issueNodes !== undefined ? { nodes: issueNodes } : {})
				};
			});

			const screenshotTimeout = 30_000; // Reduced from 60s
			let pageOverview: Awaited<ReturnType<typeof this.screenshotService.capturePageOverview>> =
				null;
			try {
				pageOverview = await Promise.race([
					this.screenshotService.capturePageOverview(
						page,
						pageOverviewViolations,
						screenshotsDir,
						pageEntry.id,
						{ scannerId: this.metadata.name }
					),
					new Promise<never>((_, reject) =>
						setTimeout(() => {
							reject(new Error('Screenshot capture timed out'));
						}, screenshotTimeout)
					)
				]);
			} catch (screenshotError) {
				this.logger.warn('Screenshot capture failed or timed out, continuing without screenshot', {
					error:
						screenshotError instanceof Error ? screenshotError.message : String(screenshotError)
				});
			}

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
					...lhResult,
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

	private async runLighthouse(
		_page: import('playwright').Page,
		url: string
	): Promise<LighthouseResult> {
		const lighthouseModule = await this.loadLighthouseModule();
		const lighthouse = lighthouseModule.default;

		return this.runSerialized(async () => {
			const chrome = await this.ensureChrome();
			const categories = this.options.categories ?? DEFAULT_LIGHTHOUSE_CATEGORIES;

			return runLighthouseInvocation({
				url,
				port: chrome.port,
				categories,
				lighthouse,
				logger: this.logger,
				scanStageLogger: this.scanStageLogger
			});
		});
	}

	protected override async cleanup(): Promise<void> {
		await this.closeChrome();
		await super.cleanup();
	}

	private async runSerialized<T>(fn: () => Promise<T>): Promise<T> {
		const previous = this.lighthouseQueue;
		let release!: () => void;
		const current = new Promise<void>((resolve) => {
			release = resolve;
		});
		this.lighthouseQueue = previous.then(() => current);

		await previous;
		try {
			return await fn();
		} finally {
			release();
		}
	}

	private resolveChromePath(): string {
		return resolveChromePathFn();
	}

	private async ensureChrome(): Promise<LaunchedChrome> {
		if (this.chrome) {
			return this.chrome;
		}

		if (this.chromePromise) {
			return this.chromePromise;
		}

		this.chromePromise = this.launchChrome();

		try {
			const chrome = await this.chromePromise;
			this.chrome = chrome;

			return chrome;
		} catch (error) {
			this.chromePromise = null;
			throw error;
		}
	}

	private async launchChrome(): Promise<LaunchedChrome> {
		const chromePath = this.resolveChromePath();
		return launchChromeForLighthouse({
			chromePath,
			logger: this.logger,
			scanStageLogger: this.scanStageLogger
		});
	}

	private loadLighthouseModule(): Promise<LighthouseModule> {
		this.lighthouseModulePromise ??= loadLighthouseModuleFn();

		return this.lighthouseModulePromise;
	}

	private prewarmRuntime(): void {
		this.logger.info('Prewarming Lighthouse runtime');
		void this.loadLighthouseModule();
		void this.ensureChrome().catch((error: unknown) => {
			this.logger.warn('Failed to prewarm Lighthouse runtime', {
				error: error instanceof Error ? error.message : String(error)
			});
		});
	}

	private async closeChrome(): Promise<void> {
		if (!this.chrome) {
			return;
		}

		const chrome = this.chrome;
		this.chrome = null;
		this.chromePromise = null;

		await closeLaunchedChrome({
			chrome,
			logger: this.logger,
			scanStageLogger: this.scanStageLogger
		});
	}

	private extractIssues(lhResult: LighthouseResult): Issue[] {
		return extractIssuesFromResult({
			lhResult,
			scannerName: this.metadata.name,
			logger: this.logger,
			scanStageLogger: this.scanStageLogger
		});
	}

	private getAuditNodeCount(audit: LighthouseAudit): number {
		return getAuditNodeCountFn(audit);
	}

	private extractAuditNodes(audit: LighthouseAudit): LighthouseIssueNode[] {
		return extractAuditNodesFn(audit);
	}

	private async enrichNodesWithContext(
		page: import('playwright').Page,
		nodes: LighthouseIssueNode[]
	): Promise<LighthouseIssueNode[]> {
		return enrichNodesWithContextFn(page, nodes);
	}

	private async enrichIssuesWithContext(
		page: import('playwright').Page,
		issues: Issue[]
	): Promise<void> {
		return enrichIssuesWithContextFn(page, issues);
	}

	private getAuditCategory(
		auditId: string,
		categories: Record<string, LighthouseCategory>
	): string | null {
		return getAuditCategoryFn(auditId, categories);
	}

	private mapScoreToSeverity(score: number | null): IssueSeverity {
		return mapScoreToSeverityFn(score);
	}

	private getHelpUrl(auditId: string): string {
		return getHelpUrlFn(auditId);
	}
}
