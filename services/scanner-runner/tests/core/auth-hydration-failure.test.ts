/**
 * AuthHydrationError path test.
 *
 * Asserts the contract from the design note: when auth hydration fails the
 * iterator does not run page scans, every page surfaces an
 * `auth-hydration-failed` issue at severity `critical`, the scanner audit log
 * sees an `auth_hydration_failed` event, and the post-login URL captured at
 * failure is included in the issue metadata.
 */

import type { BrowserContext, Page } from 'playwright';

import { describe, expect, it, vi } from 'vitest';

import type { BrowserManager } from '../../src/core/browser-manager';
import type {
	PageScanResult,
	Provenance,
	ScannerConfig,
	ScannerLogger,
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
	pathExists: ReturnType<typeof vi.fn>;
	readJson: ReturnType<typeof vi.fn>;
	writeJson: ReturnType<typeof vi.fn>;
	ensureDir: ReturnType<typeof vi.fn>;
	chmod: ReturnType<typeof vi.fn>;
} = { ...fsHelperMocks, chmod: chmodMock };

const baseScannerConfig: ScannerConfig = {
	jobId: 'test-job',
	provenancePath: '/tmp/provenance.json',
	resultsDir: '/tmp/results',
	scannerName: 'axe',
	concurrency: 2,
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
		bucket: 'test-bucket'
	},
	messaging: {
		url: '',
		subjects: {
			pageCompleted: 'a',
			scanCompleted: 'b',
			scanFailed: 'c'
		}
	}
};

const SECRET_USER = 'user+tag@example.com';
const SECRET_PASSWORD = 'p@ss word+1';

function makeLogger(): ScannerLogger & {
	info: ReturnType<typeof vi.fn>;
	warn: ReturnType<typeof vi.fn>;
	error: ReturnType<typeof vi.fn>;
	debug: ReturnType<typeof vi.fn>;
} {
	return {
		info: vi.fn(),
		warn: vi.fn(),
		error: vi.fn(),
		debug: vi.fn()
	};
}

function makeBrowserManager(
	overrides?: Partial<BrowserManager>,
	pageOverrides?: Partial<Record<keyof Page, unknown>>
): BrowserManager {
	const mockPage = {
		goto: vi.fn().mockResolvedValue(undefined),
		close: vi.fn().mockResolvedValue(undefined),
		setViewportSize: vi.fn().mockResolvedValue(undefined),
		waitForLoadState: vi.fn().mockResolvedValue(undefined),
		waitForSelector: vi.fn().mockResolvedValue(undefined),
		waitForTimeout: vi.fn().mockResolvedValue(undefined),
		evaluate: vi.fn().mockResolvedValue(true),
		url: vi.fn().mockReturnValue('https://app.example.com/login'),
		...pageOverrides
	} as unknown as Page;

	const mockContext = {
		newPage: vi.fn().mockResolvedValue(mockPage),
		close: vi.fn().mockResolvedValue(undefined)
	} as unknown as BrowserContext;

	return {
		createContext: vi.fn().mockResolvedValue(mockContext),
		navigateToPage: vi.fn().mockResolvedValue(undefined),
		executePreScanActions: vi.fn().mockResolvedValue(undefined),
		...overrides
	} as unknown as BrowserManager;
}

describe('PageIterator auth hydration failure path', () => {
	it('surfaces auth-hydration-failed issues for storage_state when the artifact is missing', async () => {
		const provenance: Provenance = {
			version: '1.0.0',
			job_id: 'test-job',
			base_url: 'https://app.example.com',
			pages: [
				{ id: 'profile', path: '/profile', url: 'https://app.example.com/profile' },
				{ id: 'settings', path: '/settings', url: 'https://app.example.com/settings' }
			],
			auth: { mode: 'storage_state', artifact_key: 'test-job/auth/storage-state.json' }
		};

		const storageProvider: StorageProvider = {
			ensureBucket: vi.fn(),
			upload: vi.fn(),
			uploadBuffer: vi.fn(),
			uploadDirectory: vi.fn().mockResolvedValue(0),
			download: vi.fn().mockRejectedValue(new Error('NoSuchKey: object missing')),
			exists: vi.fn().mockResolvedValue(false)
		};

		const browserManager = makeBrowserManager();
		const auditEvents: PageIteratorAuditEvent[] = [];
		const onAuditEvent = (event: PageIteratorAuditEvent): void => {
			auditEvents.push(event);
		};

		const iterator = new PageIterator(browserManager, baseScannerConfig, undefined, {
			storageProvider
		});
		const scanCallback = vi.fn<PageScanCallback>();

		const results = await iterator.iteratePages(provenance, scanCallback, { onAuditEvent });

		expect(scanCallback).not.toHaveBeenCalled();
		expect(browserManager.createContext).not.toHaveBeenCalled();

		expect(results).toHaveLength(2);
		for (const result of results) {
			expect(result.success).toBe(false);
			expect(result.retryable).toBe(false);
			expect(result.error).toMatch(/Failed to download storage state artifact/);
			expect(result.issues).toHaveLength(1);
			const issue = result.issues[0]!;
			expect(issue.id).toBe('auth-hydration-failed');
			expect(issue.severity).toBe('critical');
			expect(issue.category).toBe('auth');
			expect(issue.scanner).toBe('axe');
			expect(issue.metadata).toMatchObject({ mode: 'storage_state' });
		}

		expect(auditEvents).toHaveLength(1);
		expect(auditEvents[0]?.type).toBe('auth_hydration_failed');
		expect(auditEvents[0]?.details).toMatchObject({ mode: 'storage_state' });

		expect(fsMock.ensureDir).toHaveBeenCalled();
	});

	it('rejects storage_state when no StorageProvider is configured', async () => {
		const provenance: Provenance = {
			version: '1.0.0',
			job_id: 'test-job',
			base_url: 'https://app.example.com',
			pages: [{ id: 'profile', path: '/profile', url: 'https://app.example.com/profile' }],
			auth: { mode: 'storage_state', artifact_key: 'test-job/auth/storage-state.json' }
		};

		const browserManager = makeBrowserManager();
		const iterator = new PageIterator(browserManager, baseScannerConfig);
		const scanCallback = vi.fn<PageScanCallback>();

		const results: PageScanResult[] = await iterator.iteratePages(provenance, scanCallback);

		expect(results).toHaveLength(1);
		expect(results[0]!.success).toBe(false);
		expect(results[0]!.issues[0]!.id).toBe('auth-hydration-failed');
		expect(results[0]!.issues[0]!.description).toMatch(/StorageProvider not configured/);
		expect(browserManager.createContext).not.toHaveBeenCalled();
	});

	it('redacts raw and encoded credentials from form hydration failures', async () => {
		const formEncodedUser = new URLSearchParams([['username', SECRET_USER]])
			.toString()
			.slice('username='.length);
		const formEncodedPassword = new URLSearchParams([['password', SECRET_PASSWORD]])
			.toString()
			.slice('password='.length);
		const capturedUrl =
			`https://app.example.com/login?username=${formEncodedUser}` +
			`&password=${encodeURIComponent(SECRET_PASSWORD)}`;
		const provenance: Provenance = {
			version: '1.0.0',
			job_id: 'test-job',
			base_url: 'https://app.example.com',
			pages: [{ id: 'profile', path: '/profile', url: 'https://app.example.com/profile' }],
			auth: {
				mode: 'form',
				login_url: 'https://app.example.com/login',
				steps: [
					{ type: 'fill', selector: '#username', value: SECRET_USER },
					{ type: 'fill', selector: '#password', value: SECRET_PASSWORD },
					{ type: 'click', selector: 'button[type=submit]' }
				],
				success: { type: 'load' }
			}
		};
		const browserManager = makeBrowserManager(
			{
				executePreScanActions: vi
					.fn()
					.mockRejectedValue(new Error(`submit failed for ${formEncodedPassword}`))
			},
			{ url: vi.fn().mockReturnValue(capturedUrl) }
		);
		const auditEvents: PageIteratorAuditEvent[] = [];
		const logger = makeLogger();
		const iterator = new PageIterator(browserManager, baseScannerConfig, logger);

		const results = await iterator.iteratePages(provenance, vi.fn<PageScanCallback>(), {
			onAuditEvent: (event) => auditEvents.push(event)
		});

		expect(results[0]?.issues[0]?.metadata).toMatchObject({
			mode: 'form',
			postLoginUrl: 'https://app.example.com/login?username=[REDACTED]&password=[REDACTED]'
		});
		const recorded = JSON.stringify({
			results,
			auditEvents,
			logs: {
				info: logger.info.mock.calls,
				warn: logger.warn.mock.calls,
				error: logger.error.mock.calls,
				debug: logger.debug.mock.calls
			}
		});
		for (const canary of [
			SECRET_USER,
			SECRET_PASSWORD,
			encodeURIComponent(SECRET_USER),
			encodeURIComponent(SECRET_PASSWORD),
			formEncodedUser,
			formEncodedPassword
		]) {
			expect(recorded).not.toContain(canary);
		}
		expect(recorded).toContain('[REDACTED]');
	});

	it('fails form auth when success waiting leaves the browser on a visible login form', async () => {
		const provenance: Provenance = {
			version: '1.0.0',
			job_id: 'test-job',
			base_url: 'https://app.example.com',
			pages: [{ id: 'profile', path: '/profile', url: 'https://app.example.com/profile' }],
			auth: {
				mode: 'form',
				login_url: 'https://app.example.com/login',
				steps: [
					{ type: 'fill', selector: 'input[name=email]', value: 'demo@example.com' },
					{ type: 'fill', selector: 'input[name=password]', value: 'wrong-password' },
					{ type: 'click', selector: 'button[type=submit]' }
				],
				success: { type: 'load' }
			}
		};

		// Wrong credentials keep the browser on a visible login form, so the
		// post-login grace poll exhausts its window before declaring failure.
		// Drive that window with fake timers (the mocked `waitForTimeout` advances
		// the clock) so the test exercises the real timeout without sleeping for it.
		vi.useFakeTimers();
		const browserManager = makeBrowserManager(undefined, {
			waitForTimeout: vi.fn().mockImplementation((ms: number) => {
				vi.advanceTimersByTime(ms);
				return Promise.resolve();
			})
		});
		const auditEvents: PageIteratorAuditEvent[] = [];
		const iterator = new PageIterator(browserManager, baseScannerConfig);
		const scanCallback = vi.fn<PageScanCallback>();

		let results: PageScanResult[];
		try {
			results = await iterator.iteratePages(provenance, scanCallback, {
				onAuditEvent: (event) => auditEvents.push(event)
			});
		} finally {
			vi.useRealTimers();
		}

		expect(scanCallback).not.toHaveBeenCalled();
		expect(browserManager.executePreScanActions).toHaveBeenCalledOnce();
		expect(results).toHaveLength(1);
		expect(results[0]!.success).toBe(false);
		expect(results[0]!.error).toContain('did not leave the login page');
		expect(results[0]!.issues[0]!).toMatchObject({
			id: 'auth-hydration-failed',
			severity: 'critical',
			category: 'auth',
			metadata: {
				mode: 'form',
				loginUrl: 'https://app.example.com/login',
				postLoginUrl: 'https://app.example.com/login'
			}
		});

		expect(auditEvents).toContainEqual(
			expect.objectContaining({
				type: 'auth_hydration_failed',
				details: expect.objectContaining({
					mode: 'form',
					login_url: 'https://app.example.com/login',
					post_login_url: 'https://app.example.com/login'
				})
			})
		);
		expect(auditEvents.some((event) => event.type === 'auth_hydrated')).toBe(false);
	});
});
