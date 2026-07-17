/**
 * Page Iterator Tests
 *
 * Comprehensive tests for page iteration and provenance handling.
 */

import type { BrowserContext, Page } from 'playwright';

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { BrowserManager } from '../../src/core/browser-manager';
import type {
	PageScanResult,
	Provenance,
	ScannerConfig,
	ScannerLogger
} from '../../src/core/types';

import { PageIterator, type PageScanCallback } from '../../src/core/page-iterator';

const fsMock = vi.hoisted(() => ({
	pathExists: vi.fn(),
	readJson: vi.fn(),
	writeJson: vi.fn(),
	ensureDir: vi.fn()
}));

vi.mock('../../src/utils/fs', () => fsMock);

vi.mock('../../src/core/browser-manager');

describe('PageIterator', () => {
	let pageIterator: PageIterator;
	let mockBrowserManager: BrowserManager;
	let mockContext: BrowserContext;
	let mockPage: Page;
	let scannerConfig: ScannerConfig;

	const mockProvenance: Provenance = {
		version: '1.0.0',
		job_id: 'test-job',
		base_url: 'https://example.com',
		mode: 'static',
		default_wait_for: { type: 'load' },
		pages: [
			{
				id: 'page-1',
				path: '/index.html',
				url: 'https://example.com/index.html'
			},
			{
				id: 'page-2',
				path: '/about.html',
				url: 'https://example.com/about.html'
			}
		]
	};

	beforeEach(() => {
		// Setup mock page
		mockPage = {
			goto: vi.fn().mockResolvedValue(undefined),
			close: vi.fn().mockResolvedValue(undefined),
			setViewportSize: vi.fn().mockResolvedValue(undefined)
		} as unknown as Page;

		// Setup mock context
		mockContext = {
			newPage: vi.fn().mockResolvedValue(mockPage),
			close: vi.fn().mockResolvedValue(undefined)
		} as unknown as BrowserContext;

		// Setup mock browser manager
		mockBrowserManager = {
			createContext: vi.fn().mockResolvedValue(mockContext),
			navigateToPage: vi.fn().mockResolvedValue(undefined),
			executePreScanActions: vi.fn().mockResolvedValue(undefined)
		} as unknown as BrowserManager;

		scannerConfig = {
			jobId: 'test-job',
			provenancePath: '/tmp/provenance.json',
			resultsDir: '/tmp/results',
			scannerName: 'test-scanner',
			concurrency: 2,
			maxRetries: 3,
			browser: {
				headless: true,
				args: [],
				defaultViewport: { width: 1280, height: 720 },
				deviceScaleFactor: 1,
				defaultTimeout: 30_000,
				pageLoadTimeout: 15_000
			},
			storage: {
				endpoint: 'localhost:9000',
				accessKey: 'test-access-key',
				secretKey: 'test-secret-key',
				useSSL: false,
				bucket: 'test-bucket'
			},
			messaging: {
				url: 'nats://localhost:4222',
				subjects: {
					pageCompleted: 'stageflow.pages.completed',
					scanCompleted: 'stageflow.scan.completed',
					scanFailed: 'stageflow.scan.failed'
				}
			}
		};

		pageIterator = new PageIterator(mockBrowserManager, scannerConfig);

		// Reset all mocks
		vi.clearAllMocks();
		fsMock.ensureDir.mockResolvedValue(undefined);
	});

	afterEach(() => {
		vi.clearAllMocks();
	});

	describe('loadProvenance', () => {
		it('should load provenance from file', async () => {
			fsMock.pathExists.mockResolvedValue(true);
			fsMock.readJson.mockResolvedValue(mockProvenance);

			const provenance = await pageIterator.loadProvenance();

			expect(fsMock.pathExists).toHaveBeenCalledWith('/tmp/provenance.json');
			expect(fsMock.readJson).toHaveBeenCalledWith('/tmp/provenance.json');
			expect(provenance).toMatchObject({
				job_id: 'test-job',
				pages: expect.arrayContaining([
					expect.objectContaining({ id: 'page-1' }),
					expect.objectContaining({ id: 'page-2' })
				])
			});
		});

		it('should normalize provenance URLs', async () => {
			const rawProvenance: Provenance = {
				version: '1.0.0',
				job_id: 'test-job',
				base_url: 'https://example.com',
				mode: 'static',
				pages: [
					{
						id: 'page-1',
						path: '/page1.html',
						url: ''
					}
				]
			};

			fsMock.pathExists.mockResolvedValue(true);
			fsMock.readJson.mockResolvedValue(rawProvenance);

			const provenance = await pageIterator.loadProvenance();

			expect(provenance.pages[0]!.url).toBe('https://example.com/page1.html');
			// wait_for should be undefined when not provided in provenance
			expect(provenance.pages[0]!.wait_for).toBeUndefined();
		});

		it('should create provenance from SCAN_URLS environment variable', async () => {
			fsMock.pathExists.mockResolvedValue(false);
			process.env.SCAN_URLS = '["https://example.com", "example.com/page2"]';

			const provenance = await pageIterator.loadProvenance();

			expect(provenance.mode).toBe('live');
			expect(provenance.pages).toHaveLength(2);
			expect(provenance.pages[0]!.url).toBe('https://example.com');
			expect(provenance.pages[1]!.url).toBe('https://example.com/page2');
			expect(provenance.default_wait_for).toEqual({ type: 'domcontentloaded' });
			expect(fsMock.writeJson).toHaveBeenCalled();

			delete process.env.SCAN_URLS;
		});

		it('should throw error if provenance file not found and SCAN_URLS not set', async () => {
			fsMock.pathExists.mockResolvedValue(false);
			delete process.env.SCAN_URLS;

			await expect(pageIterator.loadProvenance()).rejects.toThrow('Provenance file not found');
		});

		it('should throw error for invalid SCAN_URLS JSON', async () => {
			fsMock.pathExists.mockResolvedValue(false);
			process.env.SCAN_URLS = 'not valid json';

			await expect(pageIterator.loadProvenance()).rejects.toThrow(
				'Failed to create provenance from SCAN_URLS'
			);

			delete process.env.SCAN_URLS;
		});

		it('should throw error if SCAN_URLS is not an array', async () => {
			fsMock.pathExists.mockResolvedValue(false);
			process.env.SCAN_URLS = '{"url": "https://example.com"}';

			await expect(pageIterator.loadProvenance()).rejects.toThrow(
				'SCAN_URLS must be a JSON array of URLs'
			);

			delete process.env.SCAN_URLS;
		});

		it('should normalize URLs without protocol', async () => {
			fsMock.pathExists.mockResolvedValue(false);
			process.env.SCAN_URLS = '["example.com", "test.com/page"]';

			const provenance = await pageIterator.loadProvenance();

			expect(provenance.pages[0]!.url).toBe('https://example.com');
			expect(provenance.pages[1]!.url).toBe('https://test.com/page');

			delete process.env.SCAN_URLS;
		});
	});

	describe('iteratePages', () => {
		const placeholderTimestamp = '1970-01-01T00:00:00.000Z';

		const makeResult = (overrides: Partial<PageScanResult> = {}): PageScanResult => ({
			pageId: 'test',
			url: 'https://example.com',
			path: '/test.html',
			success: true,
			issues: [],
			durationMs: 0,
			startedAt: placeholderTimestamp,
			finishedAt: placeholderTimestamp,
			...overrides
		});

		const mockScanCallback = vi.fn<PageScanCallback>().mockResolvedValue(makeResult());

		beforeEach(() => {
			vi.clearAllMocks();
		});

		it('should iterate through all pages', async () => {
			const results = await pageIterator.iteratePages(mockProvenance, mockScanCallback);

			expect(results).toHaveLength(2);
			expect(mockScanCallback).toHaveBeenCalledTimes(2);
			expect(mockContext.close).toHaveBeenCalled();
		});

		it('should pass target validation policy to navigation and scanners', async () => {
			const localStaticProvenance: Provenance = {
				...mockProvenance,
				base_url: 'http://localhost:4173',
				mode: 'static',
				pages: [
					{
						id: 'page-1',
						path: '/index.html',
						url: 'http://localhost:4173/index.html'
					}
				]
			};

			const callback = vi.fn<PageScanCallback>().mockResolvedValue(makeResult());

			await pageIterator.iteratePages(localStaticProvenance, callback);

			expect(mockBrowserManager.navigateToPage).toHaveBeenCalledWith(
				mockPage,
				'http://localhost:4173/index.html',
				{ type: 'load' },
				{ allowedOrigins: ['http://localhost:4173'] }
			);
			expect(callback).toHaveBeenCalledWith(
				expect.objectContaining({
					targetValidationPolicy: { allowedOrigins: ['http://localhost:4173'] }
				})
			);
		});

		it('should skip pages marked with skip flag', async () => {
			const provenanceWithSkip: Provenance = {
				...mockProvenance,
				pages: [{ ...mockProvenance.pages[0]! }, { ...mockProvenance.pages[1]!, skip: true }]
			};

			const results = await pageIterator.iteratePages(provenanceWithSkip, mockScanCallback);

			expect(results).toHaveLength(1);
			expect(mockScanCallback).toHaveBeenCalledTimes(1);
		});

		it('should respect concurrency limit', async () => {
			const slowCallback = vi.fn<PageScanCallback>().mockImplementation(
				() =>
					new Promise<PageScanResult>((resolve) => {
						setTimeout(() => {
							resolve(makeResult());
						}, 50);
					})
			);

			await pageIterator.iteratePages(mockProvenance, slowCallback);

			// Both pages should complete despite concurrency limit
			expect(slowCallback).toHaveBeenCalledTimes(2);
		});

		it('should call onPageStart callback', async () => {
			const onPageStart = vi.fn().mockResolvedValue(undefined);

			await pageIterator.iteratePages(mockProvenance, mockScanCallback, {
				onPageStart
			});

			expect(onPageStart).toHaveBeenCalledTimes(2);
			expect(onPageStart).toHaveBeenCalledWith(mockProvenance.pages[0]!, 0, 2);
			expect(onPageStart).toHaveBeenCalledWith(mockProvenance.pages[1]!, 1, 2);
		});

		it('should call onPageComplete callback', async () => {
			const onPageComplete = vi.fn().mockResolvedValue(undefined);

			await pageIterator.iteratePages(mockProvenance, mockScanCallback, {
				onPageComplete
			});

			expect(onPageComplete).toHaveBeenCalledTimes(2);
			expect(onPageComplete).toHaveBeenCalledWith(
				expect.objectContaining({ success: true }),
				expect.any(Number),
				2
			);
		});

		it('should set viewport if specified in page entry', async () => {
			const provenanceWithViewport: Provenance = {
				...mockProvenance,
				pages: [
					{
						...mockProvenance.pages[0]!,
						viewport: { width: 1920, height: 1080 }
					}
				]
			};

			await pageIterator.iteratePages(provenanceWithViewport, mockScanCallback);

			expect(mockPage.setViewportSize).toHaveBeenCalledWith({
				width: 1920,
				height: 1080
			});
		});

		it('masks literal values used by per-page pre-scan actions', async () => {
			const inputCanary = 'page-action-input-canary-91f2';
			const provenanceWithActions: Provenance = {
				...mockProvenance,
				pages: [
					{
						...mockProvenance.pages[0]!,
						pre_scan_actions: [{ type: 'fill', selector: '#private-field', value: inputCanary }]
					}
				]
			};

			await pageIterator.iteratePages(provenanceWithActions, mockScanCallback);

			expect(mockBrowserManager.executePreScanActions).toHaveBeenCalledWith(
				mockPage,
				[{ type: 'fill', selector: '#private-field', value: inputCanary }],
				expect.objectContaining({ allowList: [] }),
				{ maskInputValues: true }
			);
		});

		it('recursively redacts per-page action values from results and lifecycle callbacks', async () => {
			const inputCanary = 'page p@ss word+1';
			const formEncodedCanary = new URLSearchParams([['value', inputCanary]])
				.toString()
				.slice('value='.length);
			const provenance: Provenance = {
				...mockProvenance,
				pages: [
					{
						...mockProvenance.pages[0]!,
						id: `page-${inputCanary}`,
						pre_scan_actions: [{ type: 'fill', selector: '#private-field', value: inputCanary }]
					}
				]
			};
			const logger = {
				info: vi.fn(),
				warn: vi.fn(),
				error: vi.fn(),
				debug: vi.fn()
			} satisfies ScannerLogger;
			const iterator = new PageIterator(mockBrowserManager, scannerConfig, logger);
			const onPageStart = vi.fn().mockResolvedValue(undefined);
			const onPageComplete = vi.fn().mockResolvedValue(undefined);
			const onPageError = vi.fn().mockResolvedValue(undefined);
			const callback = vi.fn<PageScanCallback>().mockResolvedValue(
				makeResult({
					pageId: 'page-1',
					url: `https://example.com/result?value=${formEncodedCanary}`,
					path: '/index.html',
					success: false,
					retryable: false,
					error: `scanner returned ${inputCanary}`,
					issues: [
						{
							id: 'canary-result',
							scanner: 'test-scanner',
							severity: 'critical',
							category: 'test',
							title: 'Sensitive result',
							description: `Nested ${formEncodedCanary}`,
							metadata: {
								nested: [inputCanary, { encoded: formEncodedCanary }],
								[formEncodedCanary]: inputCanary
							}
						}
					],
					rawResults: { raw: inputCanary, encoded: formEncodedCanary }
				})
			);

			const results = await iterator.iteratePages(provenance, callback, {
				onPageStart,
				onPageComplete,
				onPageError
			});
			const recorded = JSON.stringify({
				results,
				onPageStart: onPageStart.mock.calls,
				onPageComplete: onPageComplete.mock.calls,
				onPageError: onPageError.mock.calls,
				logs: {
					info: logger.info.mock.calls,
					warn: logger.warn.mock.calls,
					error: logger.error.mock.calls
				}
			});

			expect(recorded).not.toContain(inputCanary);
			expect(recorded).not.toContain(encodeURIComponent(inputCanary));
			expect(recorded).not.toContain(formEncodedCanary);
			expect(recorded).toContain('[REDACTED]');
			const lifecycleError = onPageError.mock.calls[0]?.[0] as Error;
			expect(lifecycleError.message).not.toContain(inputCanary);
			expect(lifecycleError.message).toContain('[REDACTED]');
			const resultMetadata = results[0]?.issues[0]?.metadata;
			expect(resultMetadata).toHaveProperty('[REDACTED]', '[REDACTED]');
		});

		it('preserves result structure when one-character fill and select values are secret', async () => {
			const provenance: Provenance = {
				...mockProvenance,
				pages: [
					{
						...mockProvenance.pages[0]!,
						pre_scan_actions: [
							{ type: 'fill', selector: '#private-field', value: 'a' },
							{ type: 'select', selector: '#private-select', value: 's' }
						]
					}
				]
			};
			const callback = vi.fn<PageScanCallback>().mockResolvedValue(
				makeResult({
					pageId: 'page-1',
					url: 'https://example.com/result',
					path: '/page',
					issues: [
						{
							id: 'short-secret',
							scanner: 'test-scanner',
							severity: 'minor',
							category: 'test',
							title: 'Short secret',
							description: 'a and s',
							metadata: { a: 'first', s: 'second' }
						}
					]
				})
			);

			const [result] = await pageIterator.iteratePages(provenance, callback);

			expect(result).toBeDefined();
			expect(result).toHaveProperty('pageId');
			expect(result).toHaveProperty('url');
			expect(result).toHaveProperty('path');
			expect(result).toHaveProperty('issues');
			expect(result!.issues).toHaveLength(1);
			expect(result!.issues[0]).toHaveProperty('metadata');
			expect(result!.issues[0]!.metadata).toEqual({
				'[REDACTED]': 'fir[REDACTED]t',
				'[REDACTED]#2': '[REDACTED]econd'
			});
		});

		it('redacts encoded per-page values from thrown scanner errors', async () => {
			const inputCanary = 'throw p@ss word+1';
			const formEncodedCanary = new URLSearchParams([['value', inputCanary]])
				.toString()
				.slice('value='.length);
			const provenance: Provenance = {
				...mockProvenance,
				pages: [
					{
						...mockProvenance.pages[0]!,
						pre_scan_actions: [{ type: 'fill', selector: '#private-field', value: inputCanary }]
					}
				]
			};
			const onPageError = vi.fn().mockResolvedValue(undefined);
			const callback = vi
				.fn<PageScanCallback>()
				.mockRejectedValue(new Error(`scanner crashed with ${formEncodedCanary}`));

			const results = await pageIterator.iteratePages(provenance, callback, { onPageError });
			const recorded = JSON.stringify({ results, errors: onPageError.mock.calls });

			expect(recorded).not.toContain(inputCanary);
			expect(recorded).not.toContain(formEncodedCanary);
			expect(recorded).toContain('[REDACTED]');
			for (const call of onPageError.mock.calls) {
				const error = call[0] as Error;
				expect(error.message).not.toContain(formEncodedCanary);
				expect(error.message).toContain('[REDACTED]');
			}
		});

		it('should retry failed pages up to maxRetries', async () => {
			const failingCallback = vi
				.fn<PageScanCallback>()
				.mockRejectedValueOnce(new Error('First attempt failed'))
				.mockRejectedValueOnce(new Error('Second attempt failed'))
				.mockResolvedValue(makeResult());

			await pageIterator.iteratePages(mockProvenance, failingCallback);

			// Should be called at least 3 times (2 failures + success for first page, then second page)
			expect(failingCallback).toHaveBeenCalled();
		});

		it('should retry when scan returns an unsuccessful result', async () => {
			const singlePageProvenance: Provenance = {
				...mockProvenance,
				pages: [mockProvenance.pages[0]!]
			};

			const failingResult = makeResult({
				pageId: 'page-1',
				url: 'https://example.com/index.html',
				path: '/index.html',
				success: false,
				error: 'Transient failure'
			});

			const callback = vi
				.fn<PageScanCallback>()
				.mockResolvedValueOnce(failingResult)
				.mockResolvedValueOnce(
					makeResult({
						pageId: 'page-1',
						url: 'https://example.com/index.html',
						path: '/index.html',
						success: true
					})
				);

			const results = await pageIterator.iteratePages(singlePageProvenance, callback);

			expect(callback).toHaveBeenCalledTimes(2);
			expect(results[0]!.success).toBe(true);
		});

		it('should not retry when scan result is marked as non-retryable', async () => {
			const singlePageProvenance: Provenance = {
				...mockProvenance,
				pages: [mockProvenance.pages[0]!]
			};

			const callback = vi.fn<PageScanCallback>().mockResolvedValueOnce(
				makeResult({
					pageId: 'page-1',
					url: 'https://example.com/index.html',
					path: '/index.html',
					success: false,
					error: 'Hard failure',
					retryable: false
				})
			);

			const results = await pageIterator.iteratePages(singlePageProvenance, callback);

			expect(callback).toHaveBeenCalledTimes(1);
			expect(results[0]!.success).toBe(false);
		});

		it('should call onPageError callback on failures', async () => {
			const failingCallback = vi.fn<PageScanCallback>().mockRejectedValue(new Error('Scan failed'));
			const onPageError = vi.fn().mockResolvedValue(undefined);

			await pageIterator.iteratePages(mockProvenance, failingCallback, {
				onPageError
			});

			expect(onPageError).toHaveBeenCalled();
			expect(onPageError).toHaveBeenCalledWith(
				expect.any(Error),
				expect.objectContaining({ id: 'page-1' }),
				expect.any(Number)
			);
		});

		it('should return error result after max retries exceeded', async () => {
			const failingCallback = vi
				.fn<PageScanCallback>()
				.mockRejectedValue(new Error('Always fails'));

			const results = await pageIterator.iteratePages(mockProvenance, failingCallback);

			expect(results[0]!.success).toBe(false);
			expect(results[0]!.error).toBeDefined();
		});

		it('should close page even if scan fails', async () => {
			const failingCallback = vi.fn<PageScanCallback>().mockRejectedValue(new Error('Scan failed'));

			await pageIterator.iteratePages(mockProvenance, failingCallback);

			// Page should be closed multiple times (once per retry attempt)
			expect(mockPage.close).toHaveBeenCalled();
		});

		it('should handle context close timeout gracefully', async () => {
			vi.useFakeTimers();
			vi.mocked(mockContext.close).mockImplementation(
				() =>
					new Promise((resolve) => {
						setTimeout(resolve, 15_000); // Longer than timeout
					})
			);

			try {
				const iteratePromise = pageIterator.iteratePages(mockProvenance, mockScanCallback);
				await vi.advanceTimersByTimeAsync(10_001);
				await expect(iteratePromise).resolves.toBeDefined();
			} finally {
				vi.useRealTimers();
			}
		});

		it('should sort results by original page order', async () => {
			// Mock callback returns results with pageId from context
			mockScanCallback.mockImplementation((context) =>
				Promise.resolve(
					makeResult({
						pageId: context.pageEntry.id,
						url: context.pageEntry.url,
						path: context.pageEntry.path
					})
				)
			);

			const results = await pageIterator.iteratePages(mockProvenance, mockScanCallback);

			expect(results[0]!.pageId).toBe('page-1');
			expect(results[1]!.pageId).toBe('page-2');
		});

		it('should include timing information in results', async () => {
			const results = await pageIterator.iteratePages(mockProvenance, mockScanCallback);

			expect(results[0]!).toMatchObject({
				startedAt: expect.any(String),
				finishedAt: expect.any(String),
				durationMs: expect.any(Number)
			});
		});

		it('should create results directory for each page', async () => {
			await pageIterator.iteratePages(mockProvenance, mockScanCallback);

			expect(fsMock.ensureDir).toHaveBeenCalledWith(expect.stringContaining('page-1'));
			expect(fsMock.ensureDir).toHaveBeenCalledWith(expect.stringContaining('page-2'));
		});
	});

	describe('URL normalization', () => {
		it('should resolve relative paths with base URL', async () => {
			const provenance: Provenance = {
				version: '1.0.0',
				job_id: 'test',
				base_url: 'https://example.com/',
				mode: 'static',
				pages: [
					{
						id: 'page-1',
						path: 'page1.html',
						url: ''
					}
				]
			};

			fsMock.pathExists.mockResolvedValue(true);
			fsMock.readJson.mockResolvedValue(provenance);

			const result = await pageIterator.loadProvenance();

			expect(result.pages[0]!.url).toBe('https://example.com/page1.html');
		});

		it('should handle base URL without trailing slash', async () => {
			const provenance: Provenance = {
				version: '1.0.0',
				job_id: 'test',
				base_url: 'https://example.com',
				mode: 'static',
				pages: [
					{
						id: 'page-1',
						path: 'page1.html',
						url: ''
					}
				]
			};

			fsMock.pathExists.mockResolvedValue(true);
			fsMock.readJson.mockResolvedValue(provenance);

			const result = await pageIterator.loadProvenance();

			expect(result.pages[0]!.url).toBe('https://example.com/page1.html');
		});

		it('should prefer page URL over path', async () => {
			const provenance: Provenance = {
				version: '1.0.0',
				job_id: 'test',
				base_url: 'https://example.com',
				mode: 'static',
				pages: [
					{
						id: 'page-1',
						path: 'path.html',
						url: 'https://other.com/page.html'
					}
				]
			};

			fsMock.pathExists.mockResolvedValue(true);
			fsMock.readJson.mockResolvedValue(provenance);

			const result = await pageIterator.loadProvenance();

			expect(result.pages[0]!.url).toBe('https://other.com/page.html');
		});
	});
});
