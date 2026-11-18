/**
 * Exercise both auth modes through PageIterator and the files a scanner writes
 * for the orchestrator. Literal credentials must stay out of all of them.
 * Real-browser login coverage lives in integration/auth-scan-real-browser.test.ts.
 */

vi.mock('node:fs/promises', async () => {
	const actual = await vi.importActual<typeof import('node:fs/promises')>('node:fs/promises');
	return {
		...actual,
		copyFile: vi.fn(async (_src: string, dest: string) => {
			await actual.writeFile(dest, JSON.stringify({ cookies: [], origins: [] }));
		})
	};
});

import type { BrowserContext, Page } from 'playwright';

import { mkdtemp, readFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { BrowserManager } from '../../src/core/browser-manager';
import type { PageScanResult, Provenance, ScanResults, ScannerConfig } from '../../src/core/types';

import { PageIterator, type PageScanCallback } from '../../src/core/page-iterator';
import { ScannerFiles } from '../../src/core/scanner-files';

const SECRET_USER = 'leak-canary-user-r4ndom-7c2x';
const SECRET_PASSWORD = 'leak-canary-password-q9p1z';

interface BrowserHarness {
	pages: Page[];
	context: BrowserContext;
	browserManager: BrowserManager;
	storageStatePathSeen: string | undefined;
}

function createBrowserHarness(): BrowserHarness {
	const pages: Page[] = [];

	const context: BrowserContext = {
		newPage: vi.fn().mockImplementation(() => {
			let currentUrl = 'about:blank';
			const page = {
				url: () => currentUrl,
				close: vi.fn().mockResolvedValue(undefined),
				setViewportSize: vi.fn().mockResolvedValue(undefined),
				waitForLoadState: vi.fn().mockResolvedValue(undefined),
				waitForSelector: vi.fn().mockResolvedValue(undefined),
				waitForTimeout: vi.fn().mockResolvedValue(undefined),
				_setUrl(next: string) {
					currentUrl = next;
				}
			} as unknown as Page;
			pages.push(page);
			return Promise.resolve(page);
		}),
		close: vi.fn().mockResolvedValue(undefined)
	} as unknown as BrowserContext;

	const harness: BrowserHarness = {
		pages,
		context,
		browserManager: undefined as unknown as BrowserManager,
		storageStatePathSeen: undefined
	};

	const browserManager: BrowserManager = {
		assertPageHealthy: vi.fn(),
		closePage: vi.fn().mockImplementation((page: Page) => page.close()),
		createContext: vi
			.fn()
			.mockImplementation((_viewport: unknown, opts?: { storageStatePath?: string }) => {
				if (opts?.storageStatePath !== undefined) {
					harness.storageStatePathSeen = opts.storageStatePath;
				}
				return Promise.resolve(context);
			}),
		navigateToPage: vi.fn().mockImplementation((page: Page, url: string) => {
			(page as Page & { _setUrl: (u: string) => void })._setUrl(
				url === 'https://app.example.com/login' ? 'https://app.example.com/profile' : url
			);
			return Promise.resolve();
		}),
		executePreScanActions: vi.fn().mockResolvedValue(undefined)
	} as unknown as BrowserManager;

	harness.browserManager = browserManager;
	return harness;
}

function makeConfig(tmp: string): ScannerConfig {
	return {
		jobId: 'job-redact-pipeline',
		provenancePath: join(tmp, 'provenance.json'),
		resultsDir: tmp,
		scannerName: 'axe',
		concurrency: 1,
		maxRetries: 1,
		browser: {
			headless: true,
			args: [],
			defaultViewport: { width: 1280, height: 720 },
			deviceScaleFactor: 1,
			defaultTimeout: 30_000,
			pageLoadTimeout: 15_000
		}
	};
}

function expectNoSecrets(label: string, haystack: string): void {
	if (haystack.includes(SECRET_USER) || haystack.includes(SECRET_PASSWORD)) {
		throw new Error(
			`secret leak detected in ${label}: at least one of (USER, PASSWORD) appears in the captured payload`
		);
	}
}

async function runFullPipeline(
	tmp: string,
	provenance: Provenance
): Promise<{
	progress: string;
	outcome: string;
	results: PageScanResult[];
	harness: BrowserHarness;
}> {
	const harness = createBrowserHarness();

	const config = makeConfig(tmp);
	const iterator = new PageIterator(harness.browserManager, config);
	const files = new ScannerFiles(config.resultsDir);

	const scanCallback: PageScanCallback = (ctx) =>
		Promise.resolve<PageScanResult>({
			pageId: ctx.pageEntry.id,
			url: ctx.pageEntry.url,
			path: ctx.pageEntry.path,
			success: true,
			issues: [],
			durationMs: 1,
			startedAt: new Date(0).toISOString(),
			finishedAt: new Date(1).toISOString()
		});

	const results = await iterator.iteratePages(provenance, scanCallback, {
		onPageComplete: (result, index, total) => files.recordPage(result, index + 1, total)
	});
	await files.close();

	const aggregate: ScanResults = {
		jobId: config.jobId,
		scanner: config.scannerName,
		version: '1.0.0',
		totalPages: results.length,
		pages: results,
		summary: {
			totalIssues: 0,
			bySeverity: { critical: 0, serious: 0, moderate: 0, minor: 0, info: 0 },
			byCategory: {},
			pagesScanned: results.length,
			pagesFailed: 0,
			pagesWithIssues: 0,
			avgDurationMs: 0
		},
		startedAt: new Date(0).toISOString(),
		completedAt: new Date(1).toISOString(),
		durationMs: 1
	};
	await files.writeCompleted(aggregate);

	return {
		progress: await readFile(join(tmp, 'progress.jsonl'), 'utf8'),
		outcome: await readFile(join(tmp, 'outcome.json'), 'utf8'),
		results,
		harness
	};
}

describe('Auth pipeline redaction (form mode)', () => {
	const originalAuthEnv = process.env.PROVENANCE_AUTH_JSON;

	let tmp: string;

	beforeEach(async () => {
		tmp = await mkdtemp(join(tmpdir(), 'stageflow-pipeline-'));
	});

	afterEach(async () => {
		await rm(tmp, { recursive: true, force: true });

		if (originalAuthEnv === undefined) {
			delete process.env.PROVENANCE_AUTH_JSON;
		} else {
			process.env.PROVENANCE_AUTH_JSON = originalAuthEnv;
		}

		vi.clearAllMocks();
	});

	it('runs both auth modes end-to-end and keeps the scanner files free of credential values', async () => {
		// ---- Form mode ------------------------------------------------------
		const formProvenance: Provenance = {
			version: '1.0.0',
			job_id: 'job-redact-pipeline',
			base_url: 'https://app.example.com',
			pages: [
				{ id: 'profile', path: '/profile', url: 'https://app.example.com/profile' },
				{ id: 'settings', path: '/settings', url: 'https://app.example.com/settings' }
			],
			auth: {
				mode: 'form',
				login_url: 'https://app.example.com/login',
				steps: [
					{
						type: 'fill',
						selector: 'input[name=email]',
						value: SECRET_USER
					},
					{
						type: 'fill',
						selector: 'input[name=password]',
						value: SECRET_PASSWORD
					},
					{ type: 'click', selector: 'button[type=submit]' }
				],
				success: { type: 'load' }
			}
		};

		const formRun = await runFullPipeline(tmp, formProvenance);

		// The files the orchestrator reads, and the results the report is built from.
		expectNoSecrets('progress.jsonl', formRun.progress);
		expectNoSecrets('outcome.json', formRun.outcome);
		expectNoSecrets('page results', JSON.stringify(formRun.results));

		// Sanity: the form login ran and both pages were scanned after it.
		expect(formRun.harness.browserManager.executePreScanActions).toHaveBeenCalledTimes(1);
		expect(JSON.parse(formRun.outcome)).toMatchObject({ status: 'success', pages_scanned: 2 });
		expect(formRun.progress.trim().split('\n')).toHaveLength(2);

		// ---- Storage_state mode --------------------------------------------
		const storageStatePath = join(tmp, 'auth', 'storage-state.json');

		const ssProvenance: Provenance = {
			version: '1.0.0',
			job_id: 'job-redact-pipeline',
			base_url: 'https://app.example.com',
			pages: [{ id: 'profile', path: '/profile', url: 'https://app.example.com/profile' }],
			auth: {
				mode: 'storage_state'
			}
		};

		const ssRun = await runFullPipeline(tmp, ssProvenance);

		// Storage state path was threaded into createContext.
		expect(ssRun.harness.storageStatePathSeen).toBeDefined();
		expect(ssRun.harness.storageStatePathSeen).toContain(
			storageStatePath.split('auth/').pop() ?? ''
		);

		expectNoSecrets('ss progress.jsonl', ssRun.progress);
		expectNoSecrets('ss outcome.json', ssRun.outcome);
		expectNoSecrets('ss page results', JSON.stringify(ssRun.results));
		expect(JSON.parse(ssRun.outcome)).toMatchObject({ status: 'success', pages_scanned: 1 });
	});

	it.each([
		[`{"credential":"${SECRET_PASSWORD}"`, 'PROVENANCE_AUTH_JSON is not valid JSON'],
		['', 'PROVENANCE_AUTH_JSON is not valid JSON'],
		[
			JSON.stringify({ [SECRET_PASSWORD]: 'invalid' }),
			'PROVENANCE_AUTH_JSON did not match Provenance.auth shape'
		]
	])(
		'fails closed on invalid auth transport (%#) without exposing credentials',
		async (raw, message) => {
			const oldUrls = process.env.SCAN_URLS;
			process.env.SCAN_URLS = JSON.stringify(['https://app.example.com/profile']);
			process.env.PROVENANCE_AUTH_JSON = raw;
			try {
				const iterator = new PageIterator(createBrowserHarness().browserManager, makeConfig(tmp));
				await expect(iterator.loadProvenance()).rejects.toEqual(new Error(message));
			} finally {
				if (oldUrls === undefined) {
					delete process.env.SCAN_URLS;
				} else {
					process.env.SCAN_URLS = oldUrls;
				}
			}
		}
	);

	it('attaches Provenance.auth from PROVENANCE_AUTH_JSON when synthesizing Provenance from SCAN_URLS', async () => {
		// This is the orchestrator → scanner-runner glue. We don't run the full
		// iterator (no SCAN_URLS configured here); instead we exercise the
		// synthesis path directly by running loadProvenance().
		process.env.PROVENANCE_AUTH_JSON = JSON.stringify({
			mode: 'form',
			login_url: 'https://app.example.com/login',
			steps: [{ type: 'fill', selector: '#u', value: SECRET_USER }],
			success: { type: 'load' }
		});
		// SCAN_URLS forces the synthesize-from-env path.
		const originalScanUrls = process.env.SCAN_URLS;
		process.env.SCAN_URLS = JSON.stringify(['https://app.example.com/profile']);

		try {
			const harness = createBrowserHarness();
			const config = makeConfig(tmp);
			const iterator = new PageIterator(harness.browserManager, config);

			const provenance = await iterator.loadProvenance();

			expect(provenance.auth).toBeDefined();
			expect(provenance.auth?.mode).toBe('form');

			await expect(readFile(join(tmp, 'provenance.json'))).rejects.toThrow();
			expect(JSON.stringify(provenance.auth)).toContain(SECRET_USER);
		} finally {
			if (originalScanUrls === undefined) {
				delete process.env.SCAN_URLS;
			} else {
				process.env.SCAN_URLS = originalScanUrls;
			}
		}
	});
});
