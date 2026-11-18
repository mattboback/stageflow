import { lighthouseRuleEvaluations } from './rule-evaluations';
import { mkdir } from 'node:fs/promises';
import { join } from 'node:path';
import {
	chromium,
	type Browser,
	type BrowserContext as PlaywrightBrowserContext
} from 'playwright';

import {
	type Issue,
	type PageScanResult,
	type ScanContext,
	ScannerBase,
	type ScannerMetadata
} from '../../core';
import { createRedactedLogger } from '../../core/page-iterator-redaction';
import type { ScannerLogger } from '../../core/types';
import type { TargetValidationPolicy } from '../../core/target-validation';
import {
	AxeScreenshotService,
	type PageOverviewViolation
} from '../../screenshots/axe-screenshot-service';
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

type BrowserStorageState = Awaited<ReturnType<PlaywrightBrowserContext['storageState']>>;

function hasStorageStateEntries(storageState: BrowserStorageState): boolean {
	return (
		storageState.cookies.length > 0 ||
		storageState.origins.some((origin) => origin.localStorage.length > 0)
	);
}

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
	}

	async scanPage(context: ScanContext): Promise<PageScanResult> {
		const { page, pageEntry, resultsDir } = context;
		const redact = context.redact ?? ((value: string) => value);
		const logger = createRedactedLogger(this.logger, redact);
		const startedAt = new Date().toISOString();
		const hrStart = process.hrtime.bigint();

		try {
			await mkdir(resultsDir, { recursive: true });
			const screenshotsDir = join(resultsDir, 'screenshots');
			await mkdir(screenshotsDir, { recursive: true });

			const lhResult = await this.runLighthouse(page, pageEntry.url, logger);
			await this.renavigateAfterLighthouse(
				page,
				pageEntry.url,
				context.targetValidationPolicy,
				logger,
				redact
			);

			const issues = this.extractIssues(lhResult, logger);
			await this.enrichWithTimeout(page, issues, logger);

			const pageOverview = await this.captureScreenshotWithTimeout(
				page,
				issues,
				screenshotsDir,
				pageEntry.id,
				logger
			);

			const finishedAt = new Date().toISOString();
			const durationMs = Number(process.hrtime.bigint() - hrStart) / 1e6;

			return {
				pageId: pageEntry.id,
				url: pageEntry.url,
				path: pageEntry.path,
				success: true,
				finalUrl: lhResult.finalUrl,
				rules: lighthouseRuleEvaluations(lhResult, pageEntry.id),
				issues,
				durationMs: Math.round(durationMs * 100) / 100,
				startedAt,
				finishedAt,
				artifacts: pageOverview ? [pageOverview.screenshotPath] : [],
				// Only the category scores are aggregated; the full result stays out of
				// memory once this page's issues are built.
				rawResults: {
					categories: lhResult.categories,
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

	// The policy carries a ZIP job's loopback origin; without it the re-navigation
	// to the extracted site would be blocked as a private address.
	private async renavigateAfterLighthouse(
		page: import('playwright').Page,
		url: string,
		targetValidationPolicy: TargetValidationPolicy = { allowedOrigins: [] },
		logger: ScannerLogger = this.logger,
		redact: (value: string) => string = (value) => value
	): Promise<void> {
		try {
			logger.debug('Re-navigating Playwright page after Lighthouse', { url });
			const browserManager = this.browserManager;
			if (browserManager) {
				await browserManager.navigateToPage(
					page,
					url,
					{ type: 'domcontentloaded' },
					targetValidationPolicy,
					redact
				);
			} else {
				await page.goto(url, { waitUntil: 'domcontentloaded', timeout: 15_000 });
			}
		} catch (navError) {
			logger.warn('Failed to re-navigate page after Lighthouse, continuing with stale page', {
				url,
				error: navError instanceof Error ? navError.message : String(navError)
			});
		}
	}

	private async enrichWithTimeout(
		page: import('playwright').Page,
		issues: Issue[],
		logger: ScannerLogger = this.logger
	): Promise<void> {
		try {
			await Promise.race([
				enrichIssuesWithContextFn(page, issues),
				new Promise<never>((_, reject) => {
					setTimeout(() => {
						reject(new Error('Context enrichment timed out'));
					}, 15_000);
				})
			]);
		} catch (enrichError) {
			logger.warn('Context enrichment failed or timed out, continuing without enrichment', {
				error: enrichError instanceof Error ? enrichError.message : String(enrichError)
			});
		}
	}

	private async captureScreenshotWithTimeout(
		page: import('playwright').Page,
		issues: Issue[],
		screenshotsDir: string,
		pageId: string,
		logger: ScannerLogger = this.logger
	): Promise<Awaited<ReturnType<typeof this.screenshotService.capturePageOverview>> | null> {
		const violations: PageOverviewViolation[] = issues.map((issue) => {
			const issueNodes = (issue.metadata?.nodes as LighthouseIssueNode[] | undefined)?.map((n) => ({
				...(n.target !== undefined ? { target: n.target } : {})
			}));
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
				new Promise<never>((_, reject) => {
					setTimeout(() => {
						reject(new Error('Screenshot capture timed out'));
					}, 30_000);
				})
			]);
		} catch (screenshotError) {
			logger.warn('Screenshot capture failed or timed out, continuing without screenshot', {
				error: screenshotError instanceof Error ? screenshotError.message : String(screenshotError)
			});
			return null;
		}
	}

	// --- Chrome and Lighthouse lifecycle ---

	private async runLighthouse(
		page: import('playwright').Page,
		url: string,
		logger: ScannerLogger = this.logger
	): Promise<LighthouseResult> {
		const lighthouseModule = await this.loadLighthouseModule();
		const lighthouse = lighthouseModule.default;

		return this.runSerialized(async () => {
			const chrome = await this.ensureChrome();
			const categories = this.options.categories ?? DEFAULT_LIGHTHOUSE_CATEGORIES;
			const cdpBrowser = await this.hydrateLighthouseBrowserState(page, chrome.port, logger);
			const disableStorageReset = cdpBrowser !== null;

			try {
				return await runLighthouseInvocation({
					url,
					port: chrome.port,
					categories,
					disableStorageReset,
					lighthouse,
					logger
				});
			} finally {
				if (cdpBrowser !== null) {
					await this.closeCdpConnection(cdpBrowser, logger);
				}
			}
		});
	}

	private async hydrateLighthouseBrowserState(
		page: import('playwright').Page,
		port: number,
		logger: ScannerLogger = this.logger
	): Promise<Browser | null> {
		const storageState = await page.context().storageState();
		if (!hasStorageStateEntries(storageState)) {
			return null;
		}

		const browser = await chromium.connectOverCDP(`http://127.0.0.1:${port}`, {
			timeout: 10_000
		});
		const [defaultContext] = browser.contexts();
		if (!defaultContext) {
			await this.closeCdpConnection(browser, logger);
			throw new Error(
				'Unable to hydrate Lighthouse auth state: CDP browser has no default context'
			);
		}

		await defaultContext.setStorageState(storageState);
		logger.debug('Hydrated Lighthouse browser state from Playwright context', {
			cookies: storageState.cookies.length,
			origins: storageState.origins.length
		});

		return browser;
	}

	private async closeCdpConnection(
		browser: Browser,
		logger: ScannerLogger = this.logger
	): Promise<void> {
		try {
			await browser.close({ reason: 'Lighthouse auth state hydration complete' });
		} catch (error) {
			logger.warn('Failed to close Lighthouse CDP connection', {
				error: error instanceof Error ? error.message : String(error)
			});
		}
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
			logger: this.logger
		});
	}

	private loadLighthouseModule(): Promise<LighthouseModule> {
		this.lighthouseModulePromise ??= loadLighthouseModuleFn();

		return this.lighthouseModulePromise;
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
			logger: this.logger
		});
	}

	private extractIssues(lhResult: LighthouseResult, logger: ScannerLogger = this.logger): Issue[] {
		return extractIssuesFromResult({
			lhResult,
			scannerName: this.metadata.name,
			logger
		});
	}
}
