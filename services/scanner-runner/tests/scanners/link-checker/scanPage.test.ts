/**
 * Link Checker scanPage Integration Tests
 *
 * Tests for the scanPage method with mocked Playwright Page.
 */

import type { BrowserContext, Page } from 'playwright';

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { PageEntry, ScanContext, ScannerConfig, ScannerLogger } from '../../../src/core/types';

import { LinkCheckerScanner } from '../../../src/scanners/link-checker';

// Helper to create mock logger
const createMockLogger = (): ScannerLogger => ({
	info: vi.fn(),
	warn: vi.fn(),
	error: vi.fn(),
	debug: vi.fn()
});

// Helper to create mock page
const createMockPage = (overrides: Partial<Page> = {}): Page => {
	const mockPage = {
		goto: vi.fn().mockResolvedValue(null),
		evaluate: vi.fn(),
		url: vi.fn().mockReturnValue('https://example.com'),
		...overrides
	} as unknown as Page;
	return mockPage;
};

// Helper to create mock context
const createMockContext = (overrides: Partial<ScanContext> = {}): ScanContext => {
	const pageEntry: PageEntry = {
		id: 'test-page-1',
		url: 'https://example.com',
		path: '/'
	};

	const config: ScannerConfig = {
		jobId: 'test-job',
		provenancePath: '/tmp/provenance.json',
		resultsDir: '/tmp/results',
		scannerName: 'link-checker',
		concurrency: 1,
		maxRetries: 0,
		browser: {
			headless: true,
			args: [],
			defaultViewport: { width: 1280, height: 720 },
			deviceScaleFactor: 1,
			defaultTimeout: 30000,
			pageLoadTimeout: 30000
		},
		storage: {
			endpoint: 'localhost:9000',
			accessKey: 'test',
			secretKey: 'test',
			useSSL: false,
			bucket: 'test'
		},
		messaging: {
			url: 'nats://localhost:4222',
			subjects: {
				pageCompleted: 'scan.page.completed',
				scanCompleted: 'scan.completed',
				scanFailed: 'scan.failed'
			}
		}
	};

	return {
		page: createMockPage(),
		context: {} as BrowserContext,
		pageEntry,
		resultsDir: '/tmp/results',
		config,
		logger: createMockLogger(),
		targetValidationPolicy: { allowedOrigins: ['https://example.com'] },
		...overrides
	};
};

describe('LinkCheckerScanner.scanPage', () => {
	const originalFetch = globalThis.fetch;

	beforeEach(() => {
		vi.useFakeTimers();
	});

	afterEach(() => {
		globalThis.fetch = originalFetch;
		vi.useRealTimers();
	});

	describe('success path', () => {
		it('returns a valid PageScanResult on successful scan', async () => {
			globalThis.fetch = vi.fn().mockResolvedValue({
				status: 200,
				redirected: false,
				url: 'https://example.com/page'
			});

			const mockPage = createMockPage({
				evaluate: vi
					.fn()
					.mockResolvedValueOnce([
						{
							href: 'https://example.com/about',
							text: 'About',
							isInternal: true,
							element: 'a'
						}
					])
					.mockResolvedValueOnce([])
					.mockResolvedValueOnce([])
			});

			const context = createMockContext({ page: mockPage });
			const scanner = new LinkCheckerScanner();

			const resultPromise = scanner.scanPage(context);
			await vi.advanceTimersByTimeAsync(200);
			const result = await resultPromise;

			expect(result.success).toBe(true);
			expect(result.pageId).toBe('test-page-1');
			expect(result.url).toBe('https://example.com');
			expect(result.path).toBe('/');
			expect(result.durationMs).toBeGreaterThanOrEqual(0);
			expect(result.startedAt).toBeDefined();
			expect(result.finishedAt).toBeDefined();
		});

		it('calculates correct rawResults statistics', async () => {
			globalThis.fetch = vi.fn().mockResolvedValue({
				status: 200,
				redirected: false,
				url: 'https://example.com'
			});

			const mockPage = createMockPage({
				evaluate: vi
					.fn()
					.mockResolvedValueOnce([
						{
							href: 'https://example.com/page1',
							text: 'Page 1',
							isInternal: true,
							element: 'a'
						},
						{
							href: 'https://example.com/page2',
							text: 'Page 2',
							isInternal: true,
							element: 'a'
						},
						{
							href: 'https://external.com',
							text: 'External',
							isInternal: false,
							element: 'a'
						}
					])
					.mockResolvedValueOnce([])
					.mockResolvedValueOnce([])
			});

			const context = createMockContext({ page: mockPage });
			const scanner = new LinkCheckerScanner();

			const resultPromise = scanner.scanPage(context);
			await vi.advanceTimersByTimeAsync(500);
			const result = await resultPromise;

			expect(result.rawResults).toEqual({
				totalLinks: 3,
				internalLinks: 2,
				externalLinks: 1,
				brokenCount: 0,
				redirectChainCount: 0,
				averageResponseTime: expect.any(Number)
			});
		});

		it('logs extraction and completion info', async () => {
			globalThis.fetch = vi.fn().mockResolvedValue({
				status: 200,
				redirected: false,
				url: 'https://example.com'
			});

			const mockLogger = createMockLogger();
			const mockPage = createMockPage({
				evaluate: vi
					.fn()
					.mockResolvedValueOnce([
						{
							href: 'https://example.com/test',
							text: 'Test',
							isInternal: true,
							element: 'a'
						}
					])
					.mockResolvedValueOnce([])
					.mockResolvedValueOnce([])
			});

			const context = createMockContext({ page: mockPage, logger: mockLogger });
			const scanner = new LinkCheckerScanner();

			const resultPromise = scanner.scanPage(context);
			await vi.advanceTimersByTimeAsync(200);
			await resultPromise;

			expect(mockLogger.info).toHaveBeenCalledWith(
				'Extracted links',
				expect.objectContaining({
					count: 1,
					url: 'https://example.com'
				})
			);
			expect(mockLogger.info).toHaveBeenCalledWith(
				'Link check complete',
				expect.objectContaining({
					url: 'https://example.com',
					totalLinks: 1
				})
			);
		});
	});

	describe('issue detection', () => {
		it('detects broken links (404 status)', async () => {
			globalThis.fetch = vi.fn().mockResolvedValue({
				status: 404,
				redirected: false,
				url: 'https://example.com/missing'
			});

			const mockPage = createMockPage({
				evaluate: vi
					.fn()
					.mockResolvedValueOnce([
						{
							href: 'https://example.com/missing',
							text: 'Missing',
							isInternal: true,
							element: 'a'
						}
					])
					.mockResolvedValueOnce([])
					.mockResolvedValueOnce([])
			});

			const context = createMockContext({ page: mockPage });
			const scanner = new LinkCheckerScanner();

			const resultPromise = scanner.scanPage(context);
			await vi.advanceTimersByTimeAsync(200);
			const result = await resultPromise;

			expect(result.issues).toHaveLength(1);
			expect(result.issues[0]).toMatchObject({
				id: 'link-checker-broken-404',
				scanner: 'link-checker',
				severity: 'serious',
				category: 'links',
				title: expect.stringContaining('404')
			});
		});

		it('detects server errors (5xx status) as critical', async () => {
			globalThis.fetch = vi.fn().mockResolvedValue({
				status: 500,
				redirected: false,
				url: 'https://example.com/error'
			});

			const mockPage = createMockPage({
				evaluate: vi
					.fn()
					.mockResolvedValueOnce([
						{
							href: 'https://example.com/error',
							text: 'Error',
							isInternal: true,
							element: 'a'
						}
					])
					.mockResolvedValueOnce([])
					.mockResolvedValueOnce([])
			});

			const context = createMockContext({ page: mockPage });
			const scanner = new LinkCheckerScanner();

			const resultPromise = scanner.scanPage(context);
			await vi.advanceTimersByTimeAsync(200);
			const result = await resultPromise;

			expect(result.issues).toHaveLength(1);
			expect(result.issues[0]).toMatchObject({
				severity: 'critical',
				title: expect.stringContaining('500')
			});
		});

		it('detects connection errors as serious', async () => {
			globalThis.fetch = vi.fn().mockRejectedValue(new Error('ECONNREFUSED'));

			const mockPage = createMockPage({
				evaluate: vi
					.fn()
					.mockResolvedValueOnce([
						{
							href: 'https://unreachable.example',
							text: 'Unreachable',
							isInternal: false,
							element: 'a'
						}
					])
					.mockResolvedValueOnce([])
					.mockResolvedValueOnce([])
			});

			const context = createMockContext({ page: mockPage });
			const scanner = new LinkCheckerScanner();

			const resultPromise = scanner.scanPage(context);
			await vi.advanceTimersByTimeAsync(200);
			const result = await resultPromise;

			expect(result.issues).toHaveLength(1);
			expect(result.issues[0]).toMatchObject({
				id: 'link-checker-broken-0',
				severity: 'serious',
				title: expect.stringContaining('Connection Error')
			});
		});

		it('detects empty/placeholder links', async () => {
			globalThis.fetch = vi.fn().mockResolvedValue({
				status: 200,
				redirected: false,
				url: 'https://example.com'
			});

			const mockPage = createMockPage({
				evaluate: vi
					.fn()
					.mockResolvedValueOnce([])
					.mockResolvedValueOnce([
						'<a href="#">Click here</a>',
						'<a href="javascript:void(0)">Do nothing</a>'
					])
					.mockResolvedValueOnce([])
			});

			const context = createMockContext({ page: mockPage });
			const scanner = new LinkCheckerScanner();

			const resultPromise = scanner.scanPage(context);
			await vi.advanceTimersByTimeAsync(200);
			const result = await resultPromise;

			expect(result.issues).toHaveLength(1);
			expect(result.issues[0]).toMatchObject({
				id: 'link-checker-empty-links',
				severity: 'moderate',
				category: 'links',
				title: 'Empty or Placeholder Links',
				description: expect.stringContaining('2 link(s)')
			});
		});

		it('detects links without accessible text', async () => {
			globalThis.fetch = vi.fn().mockResolvedValue({
				status: 200,
				redirected: false,
				url: 'https://example.com'
			});

			const mockPage = createMockPage({
				evaluate: vi
					.fn()
					.mockResolvedValueOnce([])
					.mockResolvedValueOnce([])
					.mockResolvedValueOnce([
						'<a href="/page"><span class="icon"></span></a>',
						'<a href="/other"></a>'
					])
			});

			const context = createMockContext({ page: mockPage });
			const scanner = new LinkCheckerScanner();

			const resultPromise = scanner.scanPage(context);
			await vi.advanceTimersByTimeAsync(200);
			const result = await resultPromise;

			expect(result.issues).toHaveLength(1);
			expect(result.issues[0]).toMatchObject({
				id: 'link-checker-no-text-links',
				severity: 'serious',
				category: 'accessibility',
				title: 'Links Without Accessible Text'
			});
			expect(result.issues[0]?.helpUrl).toContain('WCAG');
		});
	});

	describe('error handling', () => {
		it('returns success:false when link extraction fails', async () => {
			const mockPage = createMockPage({
				evaluate: vi.fn().mockRejectedValue(new Error('Evaluation timeout'))
			});

			const mockLogger = createMockLogger();
			const context = createMockContext({ page: mockPage, logger: mockLogger });
			const scanner = new LinkCheckerScanner();

			const resultPromise = scanner.scanPage(context);
			await vi.advanceTimersByTimeAsync(100);
			const result = await resultPromise;

			expect(result.success).toBe(false);
			expect(result.error).toBe('Evaluation timeout');
			expect(result.issues).toHaveLength(0);
			expect(mockLogger.error).toHaveBeenCalledWith(
				'Link check failed',
				expect.objectContaining({
					error: 'Evaluation timeout'
				})
			);
		});

		it('returns valid PageScanResult structure even on error', async () => {
			const mockPage = createMockPage({
				evaluate: vi.fn().mockRejectedValue(new Error('Network error'))
			});

			const context = createMockContext({ page: mockPage });
			const scanner = new LinkCheckerScanner();

			const resultPromise = scanner.scanPage(context);
			await vi.advanceTimersByTimeAsync(100);
			const result = await resultPromise;

			expect(result).toMatchObject({
				pageId: 'test-page-1',
				url: 'https://example.com',
				path: '/',
				success: false,
				issues: [],
				durationMs: expect.any(Number),
				startedAt: expect.any(String),
				finishedAt: expect.any(String),
				error: 'Network error'
			});
		});

		it('handles non-Error thrown values', async () => {
			const mockPage = createMockPage({
				evaluate: vi.fn().mockRejectedValue('String error')
			});

			const context = createMockContext({ page: mockPage });
			const scanner = new LinkCheckerScanner();

			const resultPromise = scanner.scanPage(context);
			await vi.advanceTimersByTimeAsync(100);
			const result = await resultPromise;

			expect(result.success).toBe(false);
			expect(result.error).toBe('String error');
		});
	});

	describe('link extraction behavior', () => {
		it('handles page with no links gracefully', async () => {
			const mockPage = createMockPage({
				evaluate: vi
					.fn()
					.mockResolvedValueOnce([])
					.mockResolvedValueOnce([])
					.mockResolvedValueOnce([])
			});

			const context = createMockContext({ page: mockPage });
			const scanner = new LinkCheckerScanner();

			const resultPromise = scanner.scanPage(context);
			await vi.advanceTimersByTimeAsync(100);
			const result = await resultPromise;

			expect(result.success).toBe(true);
			expect(result.issues).toHaveLength(0);
			expect(result.rawResults).toMatchObject({
				totalLinks: 0,
				internalLinks: 0,
				externalLinks: 0,
				averageResponseTime: 0
			});
		});

		it('calculates correct average response time', async () => {
			let callCount = 0;
			globalThis.fetch = vi.fn().mockImplementation(() => {
				callCount++;
				return Promise.resolve({
					status: 200,
					redirected: false,
					url: `https://example.com/page${callCount}`
				});
			});

			const mockPage = createMockPage({
				evaluate: vi
					.fn()
					.mockResolvedValueOnce([
						{
							href: 'https://example.com/page1',
							text: 'Page 1',
							isInternal: true,
							element: 'a'
						},
						{
							href: 'https://example.com/page2',
							text: 'Page 2',
							isInternal: true,
							element: 'a'
						}
					])
					.mockResolvedValueOnce([])
					.mockResolvedValueOnce([])
			});

			const context = createMockContext({ page: mockPage });
			const scanner = new LinkCheckerScanner();

			const resultPromise = scanner.scanPage(context);
			await vi.advanceTimersByTimeAsync(500);
			const result = await resultPromise;

			expect(result.rawResults).toHaveProperty('averageResponseTime');
			const rawResults = result.rawResults as { averageResponseTime: number };
			expect(rawResults.averageResponseTime).toBeGreaterThanOrEqual(0);
		});
	});

	describe('issue metadata', () => {
		it('includes link URLs in broken link issue metadata', async () => {
			globalThis.fetch = vi.fn().mockResolvedValue({
				status: 404,
				redirected: false,
				url: 'https://example.com/broken'
			});

			const mockPage = createMockPage({
				evaluate: vi
					.fn()
					.mockResolvedValueOnce([
						{
							href: 'https://example.com/broken',
							text: 'Broken',
							isInternal: true,
							element: 'a'
						}
					])
					.mockResolvedValueOnce([])
					.mockResolvedValueOnce([])
			});

			const context = createMockContext({ page: mockPage });
			const scanner = new LinkCheckerScanner();

			const resultPromise = scanner.scanPage(context);
			await vi.advanceTimersByTimeAsync(200);
			const result = await resultPromise;

			const metadata = result.issues[0]?.metadata as {
				links: { url: string }[];
				totalCount: number;
			};
			expect(metadata).toMatchObject({
				links: expect.arrayContaining([
					expect.objectContaining({ url: 'https://example.com/broken' })
				]),
				totalCount: 1
			});
		});

		it('limits links in metadata to 10 items', async () => {
			globalThis.fetch = vi.fn().mockResolvedValue({
				status: 404,
				redirected: false,
				url: 'https://example.com/broken'
			});

			const brokenLinks = Array.from({ length: 15 }, (_, i) => ({
				href: `https://example.com/broken${i}`,
				text: `Broken ${i}`,
				isInternal: true,
				element: 'a'
			}));

			const mockPage = createMockPage({
				evaluate: vi
					.fn()
					.mockResolvedValueOnce(brokenLinks)
					.mockResolvedValueOnce([])
					.mockResolvedValueOnce([])
			});

			const context = createMockContext({ page: mockPage });
			const scanner = new LinkCheckerScanner();

			const resultPromise = scanner.scanPage(context);
			await vi.advanceTimersByTimeAsync(1000);
			const result = await resultPromise;

			const metadata = result.issues[0]?.metadata as {
				links: unknown[];
				totalCount: number;
			};
			expect(metadata.links).toHaveLength(10);
			expect(metadata.totalCount).toBe(15);
		});
	});
});
