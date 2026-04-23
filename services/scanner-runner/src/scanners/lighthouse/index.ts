import { mkdir } from 'node:fs/promises';
import { join } from 'node:path';

import {
	type Issue,
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
	extractIssuesFromResult
} from './issue-mapper';
import {
	loadLighthouseModule as loadLighthouseModuleFn,
	runLighthouseInvocation
} from './lighthouse-invoker';
import { isLighthousePrewarmEnabled } from './prewarm';
import {
	DEFAULT_LIGHTHOUSE_CATEGORIES,
	type LaunchedChrome,
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
			await this.renavigateAfterLighthouse(page, pageEntry.url);

			const issues = this.extractIssues(lhResult);
			await this.enrichWithTimeout(page, issues);

			const pageOverview = await this.captureScreenshotWithTimeout(
				page,
				issues,
				screenshotsDir,
				pageEntry.id
			);

			const finishedAt = new Date().toISOString();
			const durationMs = Number(process.hrtime.bigint() - hrStart) / 1e6;

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
			const durationMs = Number(process.hrtime.bigint() - hrStart) / 1e6;

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

	// --- scanPage helpers ---

	private async renavigateAfterLighthouse(
		page: import('playwright').Page,
		url: string
	): Promise<void> {
		try {
			this.logger.debug('Re-navigating Playwright page after Lighthouse', { url });
			const browserManager = this.browserManager;
			if (browserManager) {
				await browserManager.navigateToPage(page, url, { type: 'domcontentloaded' });
			} else {
				await page.goto(url, { waitUntil: 'domcontentloaded', timeout: 15_000 });
			}
		} catch (navError) {
			this.logger.warn(
				'Failed to re-navigate page after Lighthouse, continuing with stale page',
				{
					url,
					error: navError instanceof Error ? navError.message : String(navError)
				}
			);
		}
	}

	private async enrichWithTimeout(
		page: import('playwright').Page,
		issues: Issue[]
	): Promise<void> {
		try {
			await Promise.race([
				enrichIssuesWithContextFn(page, issues),
				new Promise<never>((_, reject) =>
					setTimeout(() => reject(new Error('Context enrichment timed out')), 15_000)
				)
			]);
		} catch (enrichError) {
			this.logger.warn('Context enrichment failed or timed out, continuing without enrichment', {
				error: enrichError instanceof Error ? enrichError.message : String(enrichError)
			});
		}
	}

	private async captureScreenshotWithTimeout(
		page: import('playwright').Page,
		issues: Issue[],
		screenshotsDir: string,
		pageId: string
	): Promise<Awaited<ReturnType<typeof this.screenshotService.capturePageOverview>> | null> {
		const violations: PageOverviewViolation[] = issues.map((issue) => {
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

		try {
			return await Promise.race([
				this.screenshotService.capturePageOverview(page, violations, screenshotsDir, pageId, {
					scannerId: this.metadata.name
				}),
				new Promise<never>((_, reject) =>
					setTimeout(() => reject(new Error('Screenshot capture timed out')), 30_000)
				)
			]);
		} catch (screenshotError) {
			this.logger.warn('Screenshot capture failed or timed out, continuing without screenshot', {
				error:
					screenshotError instanceof Error ? screenshotError.message : String(screenshotError)
			});
			return null;
		}
	}

	// --- Chrome and Lighthouse lifecycle ---

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

	private async enrichIssuesWithContext(
		page: import('playwright').Page,
		issues: Issue[]
	): Promise<void> {
		return enrichIssuesWithContextFn(page, issues);
	}
}
