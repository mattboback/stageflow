/**
 * Page Iterator
 *
 * Iterates through provenance pages and executes scan callbacks.
 */

import type { BrowserContext, Page } from 'playwright';

import { dirname, join } from 'node:path';

import type { BrowserManager } from './browser-manager';

import { ensureDir, pathExists, readJson, writeJson } from '../utils/fs';
import { createLogger } from '../utils/logger';
import {
	AuthHydrationError,
	defaultStorageStatePath,
	hydrateForm,
	hydrateStorageState
} from './auth-hydrator';
import { detectAuthWall } from './auth-wall';
import { collectFromEnvReferences, createSecretsResolver } from './secrets-resolver';
import { buildTargetValidationPolicy } from './target-validation';
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
	type StorageProvider,
	type WaitStrategy
} from './types';

export type PageScanCallback = (context: ScanContext) => Promise<PageScanResult>;

export interface PageIteratorAuditEvent {
	type: string;
	details?: Record<string, unknown>;
}

export interface PageIteratorCallbacks {
	onPageStart?: (pageEntry: PageEntry, index: number, total: number) => Promise<void>;
	onPageComplete?: (result: PageScanResult, index: number, total: number) => Promise<void>;
	onPageError?: (error: Error, pageEntry: PageEntry, attempt: number) => Promise<void>;
	onAuditEvent?: (event: PageIteratorAuditEvent) => void;
}

interface PageIteratorOptions {
	storageProvider?: StorageProvider;
}

export class PageIterator {
	private readonly browserManager: BrowserManager;
	private readonly config: ScannerConfig;
	private readonly logger: ScannerLogger;
	private readonly storageProvider: StorageProvider | undefined;

	constructor(
		browserManager: BrowserManager,
		config: ScannerConfig,
		logger?: ScannerLogger,
		options?: PageIteratorOptions
	) {
		this.browserManager = browserManager;
		this.config = config;
		this.logger = logger ?? createLogger('PageIterator');
		this.storageProvider = options?.storageProvider;
	}

	async loadProvenance(): Promise<Provenance> {
		const provenancePath = this.config.provenancePath;

		if (!(await pathExists(provenancePath))) {
			const scanUrls = process.env.SCAN_URLS;
			if (!scanUrls) {
				throw new Error(`Provenance file not found at ${provenancePath}`);
			}

			this.logger.info('Creating provenance from SCAN_URLS environment variable');

			const urls = parseScanUrls(scanUrls);

			const provenance: Provenance = {
				version: '1.0.0',
				job_id: this.config.jobId,
				base_url: '',
				mode: 'live',
				// Use domcontentloaded for faster page loads. networkidle can hang on sites
				// with WebSockets, long-polling, or analytics pings. The scanner's own
				// waitForLoadState('networkidle') in scanPage provides SPA hydration time.
				default_wait_for: { type: 'domcontentloaded' as const },
				pages: urls.map((url, index) => ({
					id: `url-${index + 1}`,
					path: '',
					url: normalizeUrl(url)
				}))
			};

			attachAuthFromEnv(provenance, this.logger);

			await ensureDir(dirname(provenancePath));
			await writeJson(provenancePath, provenance, { spaces: 2 });
			return provenance;
		}

		const provenanceRaw = await readJson(provenancePath);
		const normalized = this.normalizeProvenance(provenanceRaw as Provenance);

		// The orchestrator may also emit PROVENANCE_AUTH_JSON for ZIP-input
		// jobs that opt into authenticated scanning post-extraction. If the
		// loaded Provenance does not already carry an auth block, attach the
		// orchestrator's; if it does, leave it alone (the file is the source
		// of truth in that case).
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

		const allowList = collectFromEnvReferences(provenance);
		const secretsResolver = createSecretsResolver({ allowList });

		this.logger.info('Starting page iteration', {
			totalPages: pages.length,
			concurrency: this.config.concurrency,
			allowedTargetOrigins: targetValidationPolicy.allowedOrigins,
			authMode: provenance.auth?.mode ?? 'none',
			fromEnvAllowList: allowList
		});

		let authHydrated = false;
		let storageStatePath: string | undefined;
		if (provenance.auth?.mode === 'storage_state') {
			try {
				const result = await hydrateStorageState({
					auth: provenance.auth,
					storageProvider: this.requireStorageProvider(),
					bucket: this.config.storage.bucket,
					destPath: defaultStorageStatePath(this.config.resultsDir),
					logger: this.logger
				});
				storageStatePath = result.storageStatePath;
				authHydrated = true;
				callbacks?.onAuditEvent?.({
					type: 'auth_hydrated',
					details: {
						mode: 'storage_state',
						artifact_key: provenance.auth.artifact_key
					}
				});
			} catch (err) {
				return this.synthesizeAuthFailureResults(pages, err, provenance.auth, callbacks);
			}
		}

		const context = await this.browserManager.createContext(provenance.default_viewport, {
			...(storageStatePath !== undefined ? { storageStatePath } : {})
		});

		try {
			if (provenance.auth?.mode === 'form') {
				try {
					const { postLoginUrl } = await hydrateForm({
						auth: provenance.auth,
						context,
						browserManager: this.browserManager,
						targetValidationPolicy,
						secretsResolver,
						logger: this.logger
					});
					callbacks?.onAuditEvent?.({
						type: 'auth_hydrated',
						details: {
							mode: 'form',
							login_url: provenance.auth.login_url,
							post_login_url: postLoginUrl
						}
					});
				} catch (err) {
					return await this.synthesizeAuthFailureResults(pages, err, provenance.auth, callbacks);
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
					secretsResolver,
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
				logger: this.logger
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

	private requireStorageProvider(): StorageProvider {
		if (!this.storageProvider) {
			throw new AuthHydrationError(
				'StorageProvider not configured: cannot resolve storage_state artifact.',
				{ mode: 'storage_state' }
			);
		}
		return this.storageProvider;
	}

	private async synthesizeAuthFailureResults(
		pages: PageEntry[],
		error: unknown,
		auth: ProvenanceAuth,
		callbacks?: PageIteratorCallbacks
	): Promise<PageScanResult[]> {
		const hydrationError =
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

		this.logger.error('Auth hydration failed; skipping page iteration', {
			mode: hydrationError.mode,
			loginUrl: hydrationError.loginUrl,
			postLoginUrl: hydrationError.postLoginUrl,
			error: hydrationError.message
		});

		callbacks?.onAuditEvent?.({
			type: 'auth_hydration_failed',
			details: {
				mode: hydrationError.mode,
				...(hydrationError.loginUrl !== undefined ? { login_url: hydrationError.loginUrl } : {}),
				...(hydrationError.postLoginUrl !== undefined
					? { post_login_url: hydrationError.postLoginUrl }
					: {}),
				error: hydrationError.message
			}
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
				pageId: pageEntry.id,
				url: pageEntry.url,
				path: pageEntry.path,
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
		const normalizedPages = provenance.pages.map((page) => {
			const { wait_for: _pageWaitFor, ...pageWithoutWaitFor } = page;
			const waitFor = page.wait_for ?? provenance.default_wait_for;

			return {
				...pageWithoutWaitFor,
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
		secretsResolver: ReturnType<typeof createSecretsResolver>,
		scanCallback: PageScanCallback,
		callbacks?: PageIteratorCallbacks
	): Promise<PageScanResult> {
		const startedAt = new Date().toISOString();
		const hrStart = process.hrtime.bigint();

		this.logger.info(`Scanning page ${index + 1}/${total}`, {
			pageId: pageEntry.id,
			url: pageEntry.url
		});

		if (callbacks?.onPageStart) {
			await callbacks.onPageStart(pageEntry, index, total);
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
					targetValidationPolicy
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
					await this.browserManager.executePreScanActions(
						page,
						pageEntry.pre_scan_actions,
						secretsResolver
					);
				}

				const pageResultsDir = join(this.config.resultsDir, pageEntry.id);
				await ensureDir(pageResultsDir);

				const scanContext: ScanContext = {
					page,
					context,
					pageEntry,
					resultsDir: pageResultsDir,
					config: this.config,
					logger: this.logger,
					targetValidationPolicy
				};

				result = await scanCallback(scanContext);
				if (authWallIssue) {
					result.issues = [...result.issues, authWallIssue];
				}

				if (!result.success) {
					const retryable = result.retryable !== false && attempt < this.config.maxRetries;
					const errorMessage = result.error ?? 'Scan returned unsuccessful result';
					lastError = new Error(errorMessage);

					this.logger.warn(
						`Page scan reported unsuccessful result (attempt ${attempt}/${this.config.maxRetries})`,
						{
							pageId: pageEntry.id,
							error: errorMessage,
							retryable
						}
					);

					if (callbacks?.onPageError) {
						await callbacks.onPageError(lastError, pageEntry, attempt);
					}

					if (retryable) {
						result = null;
						continue;
					}
				}

				break;
			} catch (err) {
				lastError = err instanceof Error ? err : new Error(String(err));

				this.logger.warn(`Page scan failed (attempt ${attempt}/${this.config.maxRetries})`, {
					pageId: pageEntry.id,
					error: lastError.message
				});

				if (callbacks?.onPageError) {
					await callbacks.onPageError(lastError, pageEntry, attempt);
				}

				if (attempt === this.config.maxRetries) {
					this.logger.error(`Giving up on page after ${this.config.maxRetries} attempts`, {
						pageId: pageEntry.id
					});
				}
			} finally {
				const pageToClose = page;
				if (pageToClose) {
					await closeWithTimeout({
						close: () => pageToClose.close(),
						label: 'page',
						timeoutMs: 5_000,
						logger: this.logger,
						meta: { pageId: pageEntry.id }
					});
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
				pageId: pageEntry.id,
				url: pageEntry.url,
				path: pageEntry.path,
				success: false,
				issues: [],
				durationMs: Math.round(durationMs * 100) / 100,
				startedAt,
				finishedAt,
				error: lastError?.message ?? 'Unknown error'
			};
		}

		if (callbacks?.onPageComplete) {
			await callbacks.onPageComplete(result, index, total);
		}

		this.logger.info(`Completed page ${index + 1}/${total}`, {
			pageId: pageEntry.id,
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
}): Promise<void> {
	const { close, label, timeoutMs, logger, meta } = opts;

	const closePromise = close().catch((err: unknown) => {
		logger.warn(`Failed to close ${label} gracefully`, {
			...meta,
			error: err instanceof Error ? err.message : String(err)
		});
	});

	let timeoutHandle!: ReturnType<typeof setTimeout>;
	const timeoutPromise = new Promise<'timeout'>((resolve) => {
		timeoutHandle = setTimeout(() => {
			resolve('timeout');
		}, timeoutMs);
	});

	const winner = await Promise.race([closePromise.then(() => 'closed' as const), timeoutPromise]);

	clearTimeout(timeoutHandle);

	if (winner === 'timeout') {
		logger.warn(`Timed out closing ${label}`, {
			...meta,
			timeoutMs
		});
	}
}

function normalizeUrl(url: string): string {
	const trimmed = url.trim();
	if (!trimmed) {
		return trimmed;
	}

	if (/^https?:\/\//i.test(trimmed)) {
		return trimmed;
	}

	return `https://${trimmed}`;
}

function parseScanUrls(raw: string): string[] {
	let parsed: unknown;
	try {
		parsed = JSON.parse(raw);
	} catch (err) {
		const message = err instanceof Error ? err.message : String(err);
		throw new Error(`Failed to create provenance from SCAN_URLS: ${message}`, { cause: err });
	}

	if (!Array.isArray(parsed) || parsed.some((value) => typeof value !== 'string')) {
		throw new Error('SCAN_URLS must be a JSON array of URLs (strings)');
	}

	return parsed as string[];
}

/**
 * Attaches a Provenance.auth block from PROVENANCE_AUTH_JSON when present.
 *
 * The orchestrator emits this env var with the canonical auth shape (form
 * recipe with from_env references, or storage_state with an artifact_key)
 * derived from JobConfig.Auth. Resolved credential values are never present in
 * this env var; only the recipe shape is.
 *
 * Behavior is byte-identical to today when PROVENANCE_AUTH_JSON is unset: no
 * auth block is attached and the synthesized Provenance matches the pre-auth
 * shape on disk.
 */
function attachAuthFromEnv(provenance: Provenance, logger: ScannerLogger): void {
	const raw = process.env.PROVENANCE_AUTH_JSON;
	if (!raw) {
		return;
	}

	let parsed: unknown;
	try {
		parsed = JSON.parse(raw);
	} catch (err) {
		logger.warn('PROVENANCE_AUTH_JSON is not valid JSON; ignoring auth env var', {
			error: err instanceof Error ? err.message : String(err)
		});
		return;
	}

	if (!isProvenanceAuth(parsed)) {
		logger.warn('PROVENANCE_AUTH_JSON did not match Provenance.auth shape; ignoring', {
			parsed: typeof parsed === 'object' && parsed !== null ? Object.keys(parsed) : typeof parsed
		});
		return;
	}

	provenance.auth = parsed;
	logger.info('Attached auth block from PROVENANCE_AUTH_JSON', {
		mode: parsed.mode
	});
}

function isProvenanceAuth(value: unknown): value is ProvenanceAuth {
	if (typeof value !== 'object' || value === null) {
		return false;
	}

	const v = value as Record<string, unknown>;
	if (v.mode === 'storage_state') {
		return typeof v.artifact_key === 'string' && v.artifact_key.length > 0;
	}

	if (v.mode === 'form') {
		return (
			typeof v.login_url === 'string' &&
			Array.isArray(v.steps) &&
			typeof v.success === 'object' &&
			v.success !== null
		);
	}

	return false;
}
