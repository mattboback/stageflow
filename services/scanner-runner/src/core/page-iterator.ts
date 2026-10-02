import { scanIdentityIncludesOrigin } from './scan-context';
/**
 * Page Iterator
 *
 * Iterates through provenance pages and executes scan callbacks.
 */

import type { BrowserContext, Page } from 'playwright';

import { join } from 'node:path';

import type { BrowserManager } from './browser-manager';
import { stablePageId } from './finding-identity';

import { ensureDir, pathExists, readJson } from '../utils/fs';
import { createLogger } from '../utils/logger';
import {
	AuthHydrationError,
	defaultStorageStatePath,
	hydrateForm,
	hydrateStorageState
} from './auth-hydrator';
import { detectAuthWall } from './auth-wall';
import { attachAuthFromEnv, normalizeUrl, parseScanUrls } from './page-iterator-provenance';
import {
	redactPageEntry,
	redactPageScanResult,
	createLiteralRedactor,
	createRedactedLogger
} from './page-iterator-redaction';
import { buildTargetValidationPolicy, redactURLUserInfo } from './target-validation';
import {
	DEFAULT_WAIT_STRATEGY,
	type Issue,
	type PageEntry,
	type PageScanResult,
	type Provenance,
	type ProvenanceAuth,
	type ScanContext,
	type ScannerConfig,
	type ScannerLogger,
	type WaitStrategy
} from './types';

export type PageScanCallback = (context: ScanContext) => Promise<PageScanResult>;

export interface PageIteratorCallbacks {
	onPageStart?: (pageEntry: PageEntry, index: number, total: number) => Promise<void>;
	onPageComplete?: (result: PageScanResult, index: number, total: number) => Promise<void>;
	onPageError?: (error: Error, pageEntry: PageEntry, attempt: number) => Promise<void>;
}

export class PageIterator {
	private readonly browserManager: BrowserManager;
	private readonly config: ScannerConfig;
	private readonly logger: ScannerLogger;

	constructor(browserManager: BrowserManager, config: ScannerConfig, logger?: ScannerLogger) {
		this.browserManager = browserManager;
		this.config = config;
		this.logger = logger ?? createLogger('PageIterator');
	}

	async loadProvenance(): Promise<Provenance> {
		const provenancePath = this.config.provenancePath;

		if (!(await pathExists(provenancePath))) {
			const scanUrls = process.env.SCAN_URLS;
			if (!scanUrls) {
				throw new Error(`Provenance file not found at ${provenancePath}`);
			}

			this.logger.info('Creating provenance from SCAN_URLS environment variable');

			const urls = parseScanUrls(scanUrls).map(normalizeUrl);
			const includeOrigin = scanIdentityIncludesOrigin(urls);

			const provenance: Provenance = {
				version: '1.0.0',
				job_id: this.config.jobId,
				base_url: '',
				mode: 'live',
				// Use domcontentloaded for faster page loads. networkidle can hang on sites
				// with WebSockets, long-polling, or analytics pings. The scanner's own
				// waitForLoadState('networkidle') in scanPage provides SPA hydration time.
				default_wait_for: { type: 'domcontentloaded' as const },
				pages: urls.map((url) => ({
					id: stablePageId(url, includeOrigin),
					path: '',
					url: normalizeUrl(url)
				}))
			};

			attachAuthFromEnv(provenance, this.logger);
			return provenance;
		}

		const provenanceRaw = await readJson(provenancePath);
		const normalized = this.normalizeProvenance(provenanceRaw as Provenance);

		// Attach canonical auth only in memory when the input has no auth block.
		if (!normalized.auth) {
			attachAuthFromEnv(normalized, this.logger);
		}

		return normalized;
	}

	async iteratePages(
		provenance: Provenance,
		scanCallback: PageScanCallback,
		callbacks?: PageIteratorCallbacks
	): Promise<PageScanResult[]> {
		const pages = provenance.pages.filter((p) => !p.skip);
		const results: PageScanResult[] = [];
		const activePromises = new Set<Promise<void>>();
		const targetValidationPolicy = buildTargetValidationPolicy(provenance);

		const redact = createLiteralRedactor(provenance);

		this.logger.info('Starting page iteration', {
			totalPages: pages.length,
			concurrency: this.config.concurrency,
			allowedTargetOrigins: targetValidationPolicy.allowedOrigins.map(redact),
			authMode: provenance.auth?.mode ?? 'none'
		});

		let authHydrated = false;
		let storageStatePath: string | undefined;
		if (provenance.auth?.mode === 'storage_state') {
			try {
				const result = await hydrateStorageState({
					auth: provenance.auth,
					destPath: defaultStorageStatePath(this.config.resultsDir),
					logger: this.logger
				});
				storageStatePath = result.storageStatePath;
				authHydrated = true;
			} catch (err) {
				return this.synthesizeAuthFailureResults(pages, err, provenance.auth, redact, callbacks);
			}
		}

		const context = await this.browserManager.createContext(provenance.default_viewport, {
			...(storageStatePath !== undefined ? { storageStatePath } : {})
		});

		try {
			if (provenance.auth?.mode === 'form') {
				try {
					await hydrateForm({
						auth: provenance.auth,
						context,
						browserManager: this.browserManager,
						targetValidationPolicy,
						redact,
						logger: this.logger
					});
				} catch (err) {
					return await this.synthesizeAuthFailureResults(
						pages,
						err,
						provenance.auth,
						redact,
						callbacks
					);
				}
				authHydrated = true;
			}

			for (const [i, pageEntry] of pages.entries()) {
				const promise = this.processPage(
					context,
					pageEntry,
					i,
					pages.length,
					provenance,
					authHydrated,
					targetValidationPolicy,
					redact,
					scanCallback,
					callbacks
				).then((result) => {
					results.push(result);
					activePromises.delete(promise);
				});

				activePromises.add(promise);

				if (activePromises.size >= this.config.concurrency) {
					await Promise.race(activePromises);
				}
			}

			await Promise.all(activePromises);
			this.logger.info('All pages processed', { resultCount: results.length });
		} finally {
			this.logger.info('Closing browser context');
			await closeWithTimeout({
				close: () => context.close(),
				label: 'browser context',
				timeoutMs: 10_000,
				logger: this.logger,
				redactError: (value) => redact(value)
			});
			this.logger.info('Browser context closed');
		}

		// Create index map for O(1) lookups instead of O(n) findIndex per comparison
		const pageIndexMap = new Map(pages.map((p, i) => [p.id, i]));
		results.sort((a, b) => {
			const indexA = pageIndexMap.get(a.pageId) ?? 0;
			const indexB = pageIndexMap.get(b.pageId) ?? 0;
			return indexA - indexB;
		});

		return results;
	}

	private async synthesizeAuthFailureResults(
		pages: PageEntry[],
		error: unknown,
		auth: ProvenanceAuth,
		redact: (value: string) => string,
		callbacks?: PageIteratorCallbacks
	): Promise<PageScanResult[]> {
		const rawError =
			error instanceof AuthHydrationError
				? error
				: new AuthHydrationError(
						`Auth hydration failed: ${error instanceof Error ? error.message : String(error)}`,
						{
							mode: auth.mode,
							...(auth.mode === 'form' ? { loginUrl: auth.login_url } : {}),
							cause: error
						}
					);

		const hydrationError = new AuthHydrationError(redact(rawError.message), {
			mode: rawError.mode,
			...(rawError.loginUrl !== undefined
				? { loginUrl: redact(redactURLUserInfo(rawError.loginUrl)) }
				: {}),
			...(rawError.postLoginUrl !== undefined
				? { postLoginUrl: redact(redactURLUserInfo(rawError.postLoginUrl)) }
				: {})
		});

		this.logger.error('Auth hydration failed; skipping page iteration', {
			mode: hydrationError.mode,
			loginUrl: hydrationError.loginUrl,
			postLoginUrl: hydrationError.postLoginUrl,
			error: hydrationError.message
		});

		const now = new Date().toISOString();
		const issueMetadata: Record<string, unknown> = {
			mode: hydrationError.mode
		};
		if (hydrationError.loginUrl !== undefined) {
			issueMetadata.loginUrl = hydrationError.loginUrl;
		}
		if (hydrationError.postLoginUrl !== undefined) {
			issueMetadata.postLoginUrl = hydrationError.postLoginUrl;
		}

		const results: PageScanResult[] = [];
		for (const [i, pageEntry] of pages.entries()) {
			const issue: Issue = {
				id: 'auth-hydration-failed',
				scanner: this.config.scannerName,
				severity: 'critical',
				category: 'auth',
				title: 'Authentication hydration failed',
				description: hydrationError.message,
				metadata: issueMetadata
			};

			const result: PageScanResult = {
				pageId: redact(pageEntry.id),
				url: redact(pageEntry.url),
				path: redact(pageEntry.path),
				success: false,
				issues: [issue],
				durationMs: 0,
				startedAt: now,
				finishedAt: now,
				error: hydrationError.message,
				retryable: false
			};

			if (callbacks?.onPageComplete) {
				await callbacks.onPageComplete(result, i, pages.length);
			}

			results.push(result);
		}

		return results;
	}

	private normalizeProvenance(provenance: Provenance): Provenance {
		const includeOrigin = scanIdentityIncludesOrigin(
			provenance.pages.map((page) => this.resolvePageUrl(provenance.base_url, page))
		);
		const requestedPaths: unknown = process.env.SCAN_PAGE_PATHS
			? JSON.parse(process.env.SCAN_PAGE_PATHS)
			: null;
		if (
			requestedPaths !== null &&
			(!Array.isArray(requestedPaths) || requestedPaths.some((item) => typeof item !== 'string'))
		) {
			throw new Error('Invalid SCAN_PAGE_PATHS');
		}
		const selectedPages = Array.isArray(requestedPaths)
			? provenance.pages.filter((page) =>
					requestedPaths.includes(new URL(this.resolvePageUrl(provenance.base_url, page)).pathname)
				)
			: provenance.pages;
		if (selectedPages.length === 0) {
			throw new Error('No requested pages exist in this fresh build');
		}
		const normalizedPages = selectedPages.map((page) => {
			const { wait_for: _pageWaitFor, ...pageWithoutWaitFor } = page;
			const waitFor = page.wait_for ?? provenance.default_wait_for;

			return {
				...pageWithoutWaitFor,
				id: stablePageId(this.resolvePageUrl(provenance.base_url, page), includeOrigin),
				url: this.resolvePageUrl(provenance.base_url, page),
				...(waitFor !== undefined ? { wait_for: waitFor } : {})
			};
		});

		return {
			...provenance,
			pages: normalizedPages
		};
	}

	private resolvePageUrl(baseUrl: string, page: PageEntry): string {
		const pageUrl = page.url as string | undefined;
		const pagePath = page.path as string | undefined;
		if (pageUrl) {
			return normalizeUrl(pageUrl);
		}

		if (baseUrl) {
			try {
				const normalizedBase = baseUrl.endsWith('/') ? baseUrl : `${baseUrl}/`;
				return new URL(pagePath ?? '', normalizedBase).toString();
			} catch {
				return normalizeUrl(pagePath ?? '');
			}
		}

		return normalizeUrl(pagePath ?? '');
	}

	private async processPage(
		context: BrowserContext,
		pageEntry: PageEntry,
		index: number,
		total: number,
		provenance: Provenance,
		authHydrated: boolean,
		targetValidationPolicy: ReturnType<typeof buildTargetValidationPolicy>,
		redact: (value: string) => string,
		scanCallback: PageScanCallback,
		callbacks?: PageIteratorCallbacks
	): Promise<PageScanResult> {
		const startedAt = new Date().toISOString();
		const hrStart = process.hrtime.bigint();
		const safePageEntry = redactPageEntry(pageEntry, redact);

		this.logger.info(`Scanning page ${index + 1}/${total}`, {
			pageId: redact(pageEntry.id),
			url: redact(pageEntry.url)
		});

		if (callbacks?.onPageStart) {
			await callbacks.onPageStart(safePageEntry, index, total);
		}

		let lastError: Error | null = null;
		let result: PageScanResult | null = null;

		for (let attempt = 1; attempt <= this.config.maxRetries; attempt++) {
			let page: Page | null = null;

			try {
				page = await context.newPage();

				if (pageEntry.viewport) {
					await page.setViewportSize(pageEntry.viewport);
				}

				const waitStrategy: WaitStrategy =
					pageEntry.wait_for ?? provenance.default_wait_for ?? DEFAULT_WAIT_STRATEGY;

				await this.browserManager.navigateToPage(
					page,
					pageEntry.url,
					waitStrategy,
					targetValidationPolicy,
					redact
				);

				const authWallIssue = await detectAuthWall({
					page,
					requestedUrl: pageEntry.url,
					scannerName: this.config.scannerName,
					authConfigured: provenance.auth !== undefined,
					authHydrated,
					...(provenance.auth?.mode !== undefined ? { authMode: provenance.auth.mode } : {})
				});

				if (pageEntry.pre_scan_actions?.length) {
					await this.browserManager.executePreScanActions(page, pageEntry.pre_scan_actions, {
						maskInputValues: true
					});
				}

				const pageResultsDir = join(this.config.resultsDir, pageEntry.id);
				await ensureDir(pageResultsDir);

				const scanContext: ScanContext = {
					page,
					context,
					pageEntry,
					resultsDir: pageResultsDir,
					config: this.config,
					logger: createRedactedLogger(this.logger, redact),
					redact,
					targetValidationPolicy
				};

				result = await scanCallback(scanContext);
				this.browserManager.assertPageHealthy(page);
				const navigationStatus = this.browserManager.getNavigationStatus(page);
				if (navigationStatus !== undefined && navigationStatus >= 400) {
					result.success = false;
					result.error = `The intended page returned HTTP ${navigationStatus}.`;
					result.retryable = false;
					if (result.rules) {
						result.rules = result.rules.map((rule) => ({ ...rule, outcome: 'error' }));
					}
				}
				if (authWallIssue) {
					result.issues = [...result.issues, authWallIssue];
				}
				result = redactPageScanResult(result, redact);

				if (!result.success) {
					const retryable = result.retryable !== false && attempt < this.config.maxRetries;
					const errorMessage = result.error ?? 'Scan returned unsuccessful result';
					lastError = new Error(errorMessage);

					this.logger.warn(
						`Page scan reported unsuccessful result (attempt ${attempt}/${this.config.maxRetries})`,
						{
							pageId: redact(pageEntry.id),
							error: errorMessage,
							retryable
						}
					);

					if (callbacks?.onPageError) {
						await callbacks.onPageError(lastError, safePageEntry, attempt);
					}

					if (retryable) {
						result = null;
						continue;
					}
				}

				break;
			} catch (err) {
				lastError = new Error(redact(err instanceof Error ? err.message : String(err)));

				this.logger.warn(`Page scan failed (attempt ${attempt}/${this.config.maxRetries})`, {
					pageId: redact(pageEntry.id),
					error: lastError.message
				});

				if (callbacks?.onPageError) {
					await callbacks.onPageError(lastError, safePageEntry, attempt);
				}

				if (attempt === this.config.maxRetries) {
					this.logger.error(`Giving up on page after ${this.config.maxRetries} attempts`, {
						pageId: redact(pageEntry.id)
					});
				}
			} finally {
				const pageToClose = page;
				if (pageToClose) {
					const closed = await closeWithTimeout({
						close: () => this.browserManager.closePage(pageToClose),
						label: 'page',
						timeoutMs: 5_000,
						logger: this.logger,
						meta: { pageId: redact(pageEntry.id) },
						redactError: redact
					});
					if (!closed) {
						lastError = new Error('Page request handlers did not close cleanly.');
						result = null;
					}
					try {
						this.browserManager.assertPageHealthy(pageToClose);
					} catch (err) {
						lastError = new Error(redact(err instanceof Error ? err.message : String(err)));
						result = null;
					}
				}
			}
		}

		const finishedAt = new Date().toISOString();
		const durationNs = process.hrtime.bigint() - hrStart;
		const durationMs = Number(durationNs) / 1e6;

		if (result) {
			result.startedAt = startedAt;
			result.finishedAt = finishedAt;
			result.durationMs = Math.round(durationMs * 100) / 100;
		} else {
			result = {
				pageId: redact(pageEntry.id),
				url: redact(pageEntry.url),
				path: redact(pageEntry.path),
				success: false,
				issues: [],
				durationMs: Math.round(durationMs * 100) / 100,
				startedAt,
				finishedAt,
				error: lastError?.message ?? 'Unknown error'
			};
		}
		result = redactPageScanResult(result, redact);

		if (callbacks?.onPageComplete) {
			await callbacks.onPageComplete(result, index, total);
		}

		this.logger.info(`Completed page ${index + 1}/${total}`, {
			pageId: redact(pageEntry.id),
			success: result.success,
			issues: result.issues.length,
			durationMs: result.durationMs
		});

		return result;
	}
}

async function closeWithTimeout(opts: {
	close: () => Promise<void>;
	label: string;
	timeoutMs: number;
	logger: ScannerLogger;
	meta?: Record<string, unknown>;
	redactError?: (value: string) => string;
}): Promise<boolean> {
	const { close, label, timeoutMs, logger, meta, redactError } = opts;

	const closePromise = close()
		.then(() => true)
		.catch((err: unknown) => {
			logger.warn(`Failed to close ${label} gracefully`, {
				...meta,
				error: redactError
					? redactError(err instanceof Error ? err.message : String(err))
					: err instanceof Error
						? err.message
						: String(err)
			});
			return false;
		});

	let timeoutHandle!: ReturnType<typeof setTimeout>;
	const timeoutPromise = new Promise<'timeout'>((resolve) => {
		timeoutHandle = setTimeout(() => {
			resolve('timeout');
		}, timeoutMs);
	});

	const winner = await Promise.race([closePromise, timeoutPromise]);

	clearTimeout(timeoutHandle);

	if (winner === 'timeout') {
		logger.warn(`Timed out closing ${label}`, {
			...meta,
			timeoutMs
		});
	}
	return winner === true;
}
