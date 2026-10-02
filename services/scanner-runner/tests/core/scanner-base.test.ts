import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { PageIteratorCallbacks, PageScanCallback } from '../../src/core/page-iterator';
import type { PageScanResult, Provenance, ScanContext, ScannerConfig } from '../../src/core/types';

import { ScannerBase, type ScannerMetadata } from '../../src/core/scanner-base';

const fsHelperMocks = vi.hoisted(() => ({
	ensureDir: vi.fn(),
	pathExists: vi.fn(),
	readJson: vi.fn(),
	writeJson: vi.fn()
}));

const fsPromiseMocks = vi.hoisted(() => ({
	readdir: vi.fn(),
	stat: vi.fn(),
	writeFile: vi.fn()
}));

vi.mock('../../src/utils/fs', () => fsHelperMocks);
vi.mock('node:fs/promises', async () => {
	const actual = await vi.importActual<typeof import('node:fs/promises')>('node:fs/promises');
	return {
		...actual,
		readdir: fsPromiseMocks.readdir,
		stat: fsPromiseMocks.stat,
		writeFile: fsPromiseMocks.writeFile
	};
});

const mocks = vi.hoisted(() => {
	const mockBrowserManagerInstance = {
		createContext: vi.fn().mockResolvedValue({
			newPage: vi.fn().mockResolvedValue({
				goto: vi.fn(),
				evaluate: vi.fn(),
				waitForLoadState: vi.fn(),
				waitForTimeout: vi.fn(),
				pdf: vi.fn()
			}),
			close: vi.fn()
		}),
		close: vi.fn()
	};

	const mockPageIteratorInstance = {
		loadProvenance: vi.fn().mockResolvedValue({
			version: '1.0.0',
			job_id: 'test-job',
			base_url: 'http://localhost:8080',
			pages: [
				{
					id: 'page-1',
					url: 'http://localhost:8080/index.html',
					path: '/index.html'
				},
				{
					id: 'page-2',
					url: 'http://localhost:8080/about.html',
					path: '/about.html'
				}
			]
		}),
		iteratePages: vi
			.fn()
			.mockImplementation(
				async (
					provenance: Provenance,
					scannerFn: PageScanCallback,
					hooks: PageIteratorCallbacks = {}
				) => {
					const results: PageScanResult[] = [];
					const totalPages = provenance.pages.length;
					for (let i = 0; i < totalPages; i++) {
						const page = provenance.pages[i]!;
						const scanContext = {
							pageEntry: page
						} as unknown as ScanContext;

						if (hooks.onPageStart) {
							await hooks.onPageStart(page, i, totalPages);
						}

						const result = await scannerFn(scanContext);
						results.push(result);

						if (hooks.onPageComplete) {
							await hooks.onPageComplete(result, i, totalPages);
						}
					}
					return results;
				}
			)
	};

	const mockEventPublisherInstance = {
		connect: vi.fn().mockResolvedValue(undefined),
		publishPageCompleted: vi.fn().mockResolvedValue(undefined),
		publishScanCompleted: vi.fn().mockResolvedValue(undefined),
		publishScanFailed: vi.fn().mockResolvedValue(undefined),
		close: vi.fn().mockResolvedValue(undefined)
	};

	return {
		mockBrowserManagerInstance,
		mockPageIteratorInstance,
		mockEventPublisherInstance
	};
});

vi.mock('../../src/core/browser-manager', async () => {
	const actual = await vi.importActual<typeof import('../../src/core/browser-manager')>(
		'../../src/core/browser-manager'
	);
	return {
		...actual,
		BrowserManager: vi.fn().mockImplementation(function BrowserManagerMock() {
			return mocks.mockBrowserManagerInstance;
		})
	};
});

vi.mock('../../src/core/page-iterator', async () => {
	const actual = await vi.importActual<typeof import('../../src/core/page-iterator')>(
		'../../src/core/page-iterator'
	);
	return {
		...actual,
		PageIterator: vi.fn().mockImplementation(function PageIteratorMock() {
			return mocks.mockPageIteratorInstance;
		})
	};
});

vi.mock('../../src/core/output-budget', () => ({
	enforceOutputBudget: vi.fn().mockResolvedValue(undefined)
}));

vi.mock('../../src/core/scanner-files', async () => {
	const actual = await vi.importActual<typeof import('../../src/core/scanner-files')>(
		'../../src/core/scanner-files'
	);
	return {
		...actual,
		ScannerFiles: vi.fn().mockImplementation(function NatsPublisherMock() {
			return mocks.mockEventPublisherInstance;
		})
	};
});

class MockScanner extends ScannerBase {
	readonly metadata: ScannerMetadata = {
		name: 'mock-scanner',
		version: '1.0.0',
		description: 'Mock Scanner'
	};

	scanPage(context: ScanContext): Promise<PageScanResult> {
		return Promise.resolve({
			pageId: context.pageEntry.id,
			url: context.pageEntry.url,
			path: context.pageEntry.path,
			success: true,
			issues: [],
			durationMs: 100,
			startedAt: new Date().toISOString(),
			finishedAt: new Date().toISOString()
		});
	}

	validateProvenanceInput(provenance: Provenance): void {
		this.validateProvenance(provenance);
	}

	buildScanSummary(pageResults: PageScanResult[]) {
		return this.buildSummary(pageResults);
	}
}

describe('ScannerBase', () => {
	let scanner: MockScanner;
	let mockConfig: ScannerConfig;

	beforeEach(() => {
		vi.clearAllMocks();

		scanner = new MockScanner();

		mockConfig = {
			jobId: 'test-job',
			provenancePath: '/tmp/provenance.json',
			resultsDir: '/tmp/results',
			scannerName: 'mock-scanner',
			concurrency: 1,
			maxRetries: 0,
			browser: {
				headless: true,
				args: [],
				defaultViewport: { width: 1280, height: 720 },
				deviceScaleFactor: 1,
				defaultTimeout: 30000,
				pageLoadTimeout: 30000
			}
		};

		fsHelperMocks.ensureDir.mockResolvedValue(undefined);
		fsHelperMocks.writeJson.mockResolvedValue(undefined);
		fsPromiseMocks.writeFile.mockResolvedValue(undefined);
		fsHelperMocks.readJson.mockResolvedValue({});
		fsPromiseMocks.stat.mockResolvedValue({
			isDirectory: () => false,
			isFile: () => true
		});
		fsHelperMocks.pathExists.mockImplementation((path: string) => {
			if (path.includes('.stageflow-artifacts.json')) {
				return Promise.resolve(false);
			}
			return Promise.resolve(true);
		});
		fsPromiseMocks.readdir.mockResolvedValue([]);
	});

	it('should initialize and run a scan successfully', async () => {
		const results = await scanner.run(mockConfig);

		expect(fsHelperMocks.ensureDir).toHaveBeenCalledWith(mockConfig.resultsDir);

		expect(mocks.mockPageIteratorInstance.loadProvenance).toHaveBeenCalled();
		expect(mocks.mockPageIteratorInstance.iteratePages).toHaveBeenCalled();

		expect(results.jobId).toBe(mockConfig.jobId);
		expect(results.totalPages).toBe(2);
		expect(results.pages).toHaveLength(2);
		expect(results.summary.pagesScanned).toBe(2);

		expect(fsHelperMocks.writeJson).toHaveBeenCalledWith(
			expect.stringContaining('results.json'),
			expect.anything(),
			expect.anything()
		);

		expect(mocks.mockEventPublisherInstance.publishScanCompleted).toHaveBeenCalled();
		expect(mocks.mockEventPublisherInstance.publishPageCompleted).toHaveBeenCalledTimes(2);
	});

	it('emits scan.completed timing whose disjoint components do not exceed total_ms', async () => {
		await scanner.run(mockConfig);

		const call = mocks.mockEventPublisherInstance.publishScanCompleted.mock.calls[0];
		expect(call).toBeDefined();
		const timing = call![1] as {
			totalMs: number;
			pageIterationMs: number;
			writeResultsMs: number;
			publishCompletedMs: number;
			finalizationMs: number;
		};

		// The Go scan.completed validator treats the five phase fields as disjoint and
		// rejects the event when their sum exceeds total_ms. finalizationMs must therefore
		// be the unaccounted remainder, never a roll-up of write_results + upload_artifacts.
		const componentSum =
			timing.pageIterationMs +
			timing.writeResultsMs +
			timing.publishCompletedMs +
			timing.finalizationMs;
		expect(componentSum).toBeLessThanOrEqual(timing.totalMs);
		expect(timing.finalizationMs).toBeGreaterThanOrEqual(0);
	});

	it('should handle scan errors gracefully', async () => {
		vi.clearAllMocks();
		fsHelperMocks.ensureDir.mockResolvedValue(undefined);

		mocks.mockPageIteratorInstance.loadProvenance.mockRejectedValueOnce(
			new Error('Provenance missing')
		);

		await expect(scanner.run(mockConfig)).rejects.toThrow('Provenance missing');

		expect(mocks.mockEventPublisherInstance.publishScanFailed).toHaveBeenCalledWith(
			'Provenance missing',
			expect.anything(),
			{}
		);
	});

	it('validates provenance required fields and URL constraints', () => {
		const validProvenance: Provenance = {
			version: '1.0.0',
			job_id: 'job-1',
			base_url: 'http://localhost:8080',
			pages: [
				{
					id: 'page-1',
					path: '/index.html',
					url: 'http://localhost:8080/index.html'
				}
			]
		};

		expect(() => {
			scanner.validateProvenanceInput({ ...validProvenance, version: '' });
		}).toThrow('Provenance version is required');
		expect(() => {
			scanner.validateProvenanceInput({ ...validProvenance, job_id: '' });
		}).toThrow('Provenance job_id is required');
		expect(() => {
			scanner.validateProvenanceInput({ ...validProvenance, pages: [] });
		}).toThrow('Provenance does not contain any pages to scan');
		expect(() => {
			scanner.validateProvenanceInput({
				...validProvenance,
				base_url: '',
				pages: [{ id: 'page-1', path: '/index.html', url: '/index.html' }]
			});
		}).toThrow("Provenance base_url is required when pages don't have full URLs");

		expect(() => {
			scanner.validateProvenanceInput({
				...validProvenance,
				base_url: '',
				pages: [{ id: 'page-1', path: '/index.html', url: 'https://example.com' }]
			});
		}).not.toThrow();
	});

	it('builds summary with aggregated lighthouse categories', () => {
		const now = new Date().toISOString();
		const summary = scanner.buildScanSummary([
			{
				pageId: 'page-1',
				url: 'https://example.com',
				success: true,
				issues: [
					{
						id: 'issue-1',
						scanner: 'mock-scanner',
						severity: 'critical',
						category: 'accessibility',
						title: 'Issue 1',
						description: 'desc'
					}
				],
				durationMs: 100,
				startedAt: now,
				finishedAt: now,
				rawResults: {
					categories: {
						accessibility: {
							id: 'accessibility',
							title: 'Accessibility',
							score: 0.9
						},
						seo: { id: 'seo', title: 'SEO', score: null }
					}
				}
			},
			{
				pageId: 'page-2',
				url: 'https://example.com/about',
				success: true,
				issues: [
					{
						id: 'issue-2',
						scanner: 'mock-scanner',
						severity: 'minor',
						category: 'seo',
						title: 'Issue 2',
						description: 'desc'
					}
				],
				durationMs: 200,
				startedAt: now,
				finishedAt: now,
				rawResults: {
					categories: {
						accessibility: {
							id: 'accessibility',
							title: 'Accessibility',
							score: 0.7
						}
					}
				}
			}
		]);

		expect(summary.totalIssues).toBe(2);
		expect(summary.bySeverity.critical).toBe(1);
		expect(summary.bySeverity.minor).toBe(1);
		expect(summary.byCategory.accessibility).toBe(1);
		expect(summary.byCategory.seo).toBe(1);
		expect(summary.avgDurationMs).toBe(150);
		expect(summary.lighthouseCategories).toEqual([
			{ id: 'accessibility', title: 'Accessibility', avgScore: 0.8 }
		]);
	});

	it('wires lifecycle hooks for page callbacks and page errors', async () => {
		const onPageStart = vi.fn().mockResolvedValue(undefined);
		const onPageEnd = vi.fn().mockResolvedValue(undefined);
		const onError = vi.fn().mockResolvedValue(undefined);

		const scannerWithHooks = new MockScanner({
			onPageStart,
			onPageEnd,
			onError
		});

		mocks.mockPageIteratorInstance.iteratePages.mockImplementationOnce(
			async (
				provenance: Provenance,
				scannerFn: PageScanCallback,
				hooks: PageIteratorCallbacks = {}
			) => {
				const pageEntry = provenance.pages[0]!;
				await hooks.onPageStart?.(pageEntry, 0, 1);
				await hooks.onPageError?.(new Error('page failed'), pageEntry, 1);
				const result = await scannerFn({ pageEntry } as unknown as ScanContext);
				await hooks.onPageComplete?.(result, 0, 1);
				return [result];
			}
		);

		await scannerWithHooks.run(mockConfig);

		expect(onPageStart).toHaveBeenCalledTimes(1);
		expect(onPageEnd).toHaveBeenCalledTimes(1);
		expect(onError).toHaveBeenCalledWith(expect.any(Error), {
			pageEntry: expect.objectContaining({ id: 'page-1' })
		});
	});
});
