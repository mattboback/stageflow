/**
 * Hydrate-then-iterate test (mocked Playwright surface).
 *
 * Asserts the design contract: hydrateAuth runs once per scanner before page
 * iteration, the secrets resolver replaces from_env references at execution
 * time, and an auth_hydrated audit event is emitted with the post-login URL
 * only (never credentials).
 */

import type { BrowserContext, Page } from 'playwright';

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { BrowserManager } from '../../src/core/browser-manager';
import type {
	PageScanResult,
	Provenance,
	ScannerConfig,
	StorageProvider
} from '../../src/core/types';

import {
	PageIterator,
	type PageIteratorAuditEvent,
	type PageScanCallback
} from '../../src/core/page-iterator';

const fsHelperMocks = vi.hoisted(() => ({
	pathExists: vi.fn().mockResolvedValue(false),
	readJson: vi.fn(),
	writeJson: vi.fn().mockResolvedValue(undefined),
	ensureDir: vi.fn().mockResolvedValue(undefined)
}));

const chmodMock = vi.hoisted(() => vi.fn().mockResolvedValue(undefined));

vi.mock('../../src/utils/fs', () => fsHelperMocks);
vi.mock('node:fs/promises', async () => {
	const actual = await vi.importActual<typeof import('node:fs/promises')>('node:fs/promises');
	return {
		...actual,
		chmod: chmodMock
	};
});

const fsMock: {
	ensureDir: ReturnType<typeof vi.fn>;
	chmod: ReturnType<typeof vi.fn>;
} = { ensureDir: fsHelperMocks.ensureDir, chmod: chmodMock };

const SECRET_USER = 'hydrate-test-user-9j2n3kl12';
const SECRET_PASSWORD = 'hydrate-test-pw-uw7gx48p2j';

const baseConfig: ScannerConfig = {
	jobId: 'job-hydrate',
	provenancePath: '/tmp/provenance.json',
	resultsDir: '/tmp/results',
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
	},
	storage: {
		endpoint: 'localhost:9000',
		accessKey: 'k',
		secretKey: 's',
		useSSL: false,
		bucket: 'job-hydrate'
	},
	messaging: {
		url: '',
		subjects: { pageCompleted: 'a', scanCompleted: 'b', scanFailed: 'c' }
	}
};

interface MockHarness {
	pages: Page[];
	context: BrowserContext;
	browserManager: BrowserManager;
}

function createHarness(): MockHarness {
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

	const browserManager: BrowserManager = {
		createContext: vi.fn().mockResolvedValue(context),
		navigateToPage: vi.fn().mockImplementation((page: Page, url: string) => {
			(page as Page & { _setUrl: (u: string) => void })._setUrl(
				url === 'https://app.example.com/login' ? 'https://app.example.com/profile' : url
			);
			return Promise.resolve();
		}),
		executePreScanActions: vi.fn().mockResolvedValue(undefined)
	} as unknown as BrowserManager;

	return { pages, context, browserManager };
}

describe('hydrate-then-iterate', () => {
	const originalUser = process.env.STAGEFLOW_AUTH_USER;
	const originalPassword = process.env.STAGEFLOW_AUTH_PASSWORD;

	beforeEach(() => {
		process.env.STAGEFLOW_AUTH_USER = SECRET_USER;
		process.env.STAGEFLOW_AUTH_PASSWORD = SECRET_PASSWORD;
	});

	afterEach(() => {
		if (originalUser === undefined) {
			delete process.env.STAGEFLOW_AUTH_USER;
		} else {
			process.env.STAGEFLOW_AUTH_USER = originalUser;
		}
		if (originalPassword === undefined) {
			delete process.env.STAGEFLOW_AUTH_PASSWORD;
		} else {
			process.env.STAGEFLOW_AUTH_PASSWORD = originalPassword;
		}
		vi.clearAllMocks();
	});

	it('runs form hydration once before iterating pages and resolves from_env', async () => {
		const harness = createHarness();
		const provenance: Provenance = {
			version: '1.0.0',
			job_id: 'job-hydrate',
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
						value: { from_env: 'STAGEFLOW_AUTH_USER' }
					},
					{
						type: 'fill',
						selector: 'input[name=password]',
						value: { from_env: 'STAGEFLOW_AUTH_PASSWORD' }
					},
					{ type: 'click', selector: 'button[type=submit]' }
				],
				success: { type: 'load' }
			}
		};

		const auditEvents: PageIteratorAuditEvent[] = [];
		const onAuditEvent = (event: PageIteratorAuditEvent): void => {
			auditEvents.push(event);
		};

		const iterator = new PageIterator(harness.browserManager, baseConfig);
		const scanCallback = vi.fn<PageScanCallback>().mockImplementation((ctx) =>
			Promise.resolve<PageScanResult>({
				pageId: ctx.pageEntry.id,
				url: ctx.pageEntry.url,
				path: ctx.pageEntry.path,
				success: true,
				issues: [],
				durationMs: 0,
				startedAt: '1970-01-01T00:00:00.000Z',
				finishedAt: '1970-01-01T00:00:00.000Z'
			})
		);

		const results = await iterator.iteratePages(provenance, scanCallback, { onAuditEvent });

		// hydration ran exactly once before any page navigation
		expect(harness.browserManager.navigateToPage).toHaveBeenCalledTimes(3);
		const navUrls = (
			harness.browserManager.navigateToPage as ReturnType<typeof vi.fn>
		).mock.calls.map((args) => args[1] as string);
		expect(navUrls[0]).toBe('https://app.example.com/login');
		expect(navUrls.slice(1)).toEqual([
			'https://app.example.com/profile',
			'https://app.example.com/settings'
		]);

		// hydration replayed steps via executePreScanActions with secrets resolver
		expect(harness.browserManager.executePreScanActions).toHaveBeenCalledTimes(1);
		const [, , resolver] = (
			harness.browserManager.executePreScanActions as ReturnType<typeof vi.fn>
		).mock.calls[0]!;
		expect(typeof resolver).toBe('object');
		expect(resolver).not.toBeNull();
		const r = resolver as {
			resolve: (ref: { from_env: string }) => string;
			allowList: readonly string[];
		};
		expect(r.allowList).toEqual(['STAGEFLOW_AUTH_PASSWORD', 'STAGEFLOW_AUTH_USER']);
		expect(r.resolve({ from_env: 'STAGEFLOW_AUTH_USER' })).toBe(SECRET_USER);

		// audit event has post-login URL but no credentials
		expect(auditEvents).toContainEqual(
			expect.objectContaining({
				type: 'auth_hydrated',
				details: expect.objectContaining({
					mode: 'form',
					login_url: 'https://app.example.com/login',
					post_login_url: 'https://app.example.com/profile'
				})
			})
		);
		const audit = JSON.stringify(auditEvents);
		expect(audit).not.toContain(SECRET_USER);
		expect(audit).not.toContain(SECRET_PASSWORD);

		// page scans succeeded
		expect(results).toHaveLength(2);
		expect(results.every((res) => res.success)).toBe(true);
	});

	it('runs storage_state hydration once before context creation', async () => {
		const harness = createHarness();

		const storageProvider: StorageProvider = {
			ensureBucket: vi.fn(),
			upload: vi.fn(),
			uploadBuffer: vi.fn(),
			uploadDirectory: vi.fn().mockResolvedValue(0),
			download: vi.fn().mockResolvedValue(undefined),
			exists: vi.fn().mockResolvedValue(true)
		};

		const provenance: Provenance = {
			version: '1.0.0',
			job_id: 'job-hydrate',
			base_url: 'https://app.example.com',
			pages: [{ id: 'profile', path: '/profile', url: 'https://app.example.com/profile' }],
			auth: { mode: 'storage_state', artifact_key: 'job-hydrate/auth/storage-state.json' }
		};

		const auditEvents: PageIteratorAuditEvent[] = [];
		const iterator = new PageIterator(harness.browserManager, baseConfig, undefined, {
			storageProvider
		});
		const scanCallback = vi.fn<PageScanCallback>().mockImplementation((ctx) =>
			Promise.resolve<PageScanResult>({
				pageId: ctx.pageEntry.id,
				url: ctx.pageEntry.url,
				path: ctx.pageEntry.path,
				success: true,
				issues: [],
				durationMs: 0,
				startedAt: '1970-01-01T00:00:00.000Z',
				finishedAt: '1970-01-01T00:00:00.000Z'
			})
		);

		const results = await iterator.iteratePages(provenance, scanCallback, {
			onAuditEvent: (event) => auditEvents.push(event)
		});

		expect(storageProvider.download).toHaveBeenCalledWith(
			'job-hydrate',
			'job-hydrate/auth/storage-state.json',
			expect.stringContaining('storage-state.json')
		);
		expect(fsMock.chmod).toHaveBeenCalledWith(expect.any(String), 0o600);

		expect(harness.browserManager.createContext).toHaveBeenCalledWith(
			undefined,
			expect.objectContaining({
				storageStatePath: expect.stringContaining('storage-state.json')
			})
		);

		expect(auditEvents).toContainEqual(
			expect.objectContaining({
				type: 'auth_hydrated',
				details: expect.objectContaining({
					mode: 'storage_state',
					artifact_key: 'job-hydrate/auth/storage-state.json'
				})
			})
		);

		// no form steps were replayed (storage_state mode)
		expect(harness.browserManager.executePreScanActions).not.toHaveBeenCalled();
		expect(results).toHaveLength(1);
		expect(results[0]!.success).toBe(true);
	});
});
