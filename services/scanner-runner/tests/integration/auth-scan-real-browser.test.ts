/**
 * End-to-end authenticated-scan integration test (plan task 10, full
 * real-browser version).
 *
 * Spins up a Node http fixture login app on 127.0.0.1, drives the full
 * scanner-runner pipeline against it for both Provenance.auth modes
 * (`form` with `from_env` references, and `storage_state` captured by
 * first replaying the form flow), and asserts:
 *
 *   - Inside the scan callback, @axe-core/playwright runs against the
 *     post-login DOM and the captured violation lives inside the
 *     `<main id="post-login">` marker, not on the /login redirect.
 *   - When E2E_LIGHTHOUSE=1, real Lighthouse is run against the
 *     post-login URL via chrome-launcher and the audited `requestedUrl`
 *     is the post-login URL, not /login.
 *   - The redaction grep extends across stored Provenance, the
 *     synthesized unified report, the scan stage log, captured stage
 *     uploads, captured NATS payloads, and audit events: no canary
 *     credential value (USER, PASSWORD, or session cookie value)
 *     appears anywhere in any of them.
 *
 * The canary credentials are reused from
 * tests/core/auth-pipeline-redaction.test.ts so both the unit-level
 * redaction backstop and this real-browser run share a single
 * build-failure-grade source of truth.
 */

import type { AddressInfo } from 'node:net';

import AxeBuilder from '@axe-core/playwright';
import { copyFile, mkdir, mkdtemp, readFile, rm, writeFile } from 'node:fs/promises';
import http, { type Server } from 'node:http';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import { type BrowserContext, type Page, chromium } from 'playwright';
import { afterAll, beforeAll, describe, expect, it } from 'vitest';

import type {
	PageScanResult,
	Provenance,
	ScanEventPublisher,
	ScanResults,
	ScanTiming,
	ScannerConfig,
	StorageProvider
} from '../../src/core/types';

import { BrowserManager } from '../../src/core/browser-manager';
import {
	PageIterator,
	type PageIteratorAuditEvent,
	type PageScanCallback
} from '../../src/core/page-iterator';
import { ScanStageLogger } from '../../src/core/scan-stage-logger';

// Reused verbatim from tests/core/auth-pipeline-redaction.test.ts so the
// build-failure backstop is unified across the redaction guard and this
// real-browser end-to-end test.
const SECRET_USER = 'leak-canary-user-r4ndom-7c2x';
const SECRET_PASSWORD = 'leak-canary-password-q9p1z';

const SESSION_COOKIE_NAME = 'sf-e2e-session';
const SESSION_COOKIE_VALUE = 'leak-canary-session-2k8m9';

// All canary values that must not leak to any stored/published surface.
const ALL_CANARIES = [SECRET_USER, SECRET_PASSWORD, SESSION_COOKIE_VALUE] as const;

// Real chromium launches make this test family slow; bump the per-test
// timeout to a value that comfortably covers boot + axe + assertions on a
// loaded CI runner.
const REAL_BROWSER_TIMEOUT_MS = 90_000;
const LIGHTHOUSE_TIMEOUT_MS = 180_000;

interface FixtureApp {
	url: string;
	loginUrl: string;
	profileUrl: string;
	close: () => Promise<void>;
}

function startFixtureLoginApp(): Promise<FixtureApp> {
	const handler = (req: http.IncomingMessage, res: http.ServerResponse): void => {
		const cookieHeader = req.headers.cookie ?? '';
		const isAuthenticated = cookieHeader.includes(`${SESSION_COOKIE_NAME}=${SESSION_COOKIE_VALUE}`);

		const url = req.url ?? '/';

		if (req.method === 'GET' && (url === '/' || url.startsWith('/?'))) {
			res.statusCode = 302;
			res.setHeader('Location', isAuthenticated ? '/profile' : '/login');
			res.end();
			return;
		}

		if (req.method === 'GET' && url.startsWith('/login')) {
			res.statusCode = 200;
			res.setHeader('Content-Type', 'text/html; charset=utf-8');
			res.end(loginPageHtml());
			return;
		}

		if (req.method === 'POST' && url === '/login') {
			let body = '';
			req.setEncoding('utf8');
			req.on('data', (chunk: string) => {
				body += chunk;
			});
			req.on('end', () => {
				const params = new URLSearchParams(body);
				if (params.get('email') === SECRET_USER && params.get('password') === SECRET_PASSWORD) {
					res.statusCode = 302;
					res.setHeader(
						'Set-Cookie',
						`${SESSION_COOKIE_NAME}=${SESSION_COOKIE_VALUE}; Path=/; HttpOnly`
					);
					res.setHeader('Location', '/profile');
					res.end();
				} else {
					res.statusCode = 401;
					res.setHeader('Content-Type', 'text/html; charset=utf-8');
					res.end('<!doctype html><html><body><h1>Login failed</h1></body></html>');
				}
			});
			return;
		}

		if (req.method === 'GET' && url.startsWith('/profile')) {
			if (!isAuthenticated) {
				res.statusCode = 302;
				res.setHeader('Location', '/login');
				res.end();
				return;
			}
			res.statusCode = 200;
			res.setHeader('Content-Type', 'text/html; charset=utf-8');
			res.end(profilePageHtml());
			return;
		}

		res.statusCode = 404;
		res.end('not found');
	};

	const server: Server = http.createServer(handler);

	return new Promise((resolve, reject) => {
		server.once('error', reject);
		server.listen(0, '127.0.0.1', () => {
			const addr = server.address() as AddressInfo;
			const baseUrl = `http://127.0.0.1:${addr.port}`;
			resolve({
				url: baseUrl,
				loginUrl: `${baseUrl}/login`,
				profileUrl: `${baseUrl}/profile`,
				close: () =>
					new Promise<void>((resolveClose) => {
						server.close(() => {
							resolveClose();
						});
					})
			});
		});
	});
}

function loginPageHtml(): string {
	// `novalidate` lets the form post even when the canary email
	// (`leak-canary-user-...`) wouldn't pass HTML5 validation; this
	// matches the common production pattern of forms that delegate
	// validation to the server.
	return `<!doctype html>
<html lang="en">
<head>
	<meta charset="utf-8" />
	<title>Sign in</title>
</head>
<body>
	<main id="login-page">
		<h1>Sign in</h1>
		<form method="POST" action="/login" novalidate>
			<label for="email">Email</label>
			<input type="email" id="email" name="email" required />
			<label for="password">Password</label>
			<input type="password" id="password" name="password" required />
			<button type="submit">Sign in</button>
		</form>
	</main>
</body>
</html>`;
}

function profilePageHtml(): string {
	// Deliberate axe violation: <img src="x"> with no alt attribute.
	// This violation only exists on the post-login DOM, so a scan that
	// audits /login (a redirect target) cannot produce it.
	return `<!doctype html>
<html lang="en">
<head>
	<meta charset="utf-8" />
	<title>Profile</title>
</head>
<body>
	<main id="post-login">
		<h1>Welcome back</h1>
		<p data-testid="post-login-marker">post-login content</p>
		<img src="x" />
	</main>
</body>
</html>`;
}

class CapturingPublisher implements ScanEventPublisher {
	pageCompleted: PageScanResult[] = [];
	scanCompleted: { results: ScanResults; timing?: ScanTiming }[] = [];
	scanFailed: { error: string; details?: string }[] = [];

	publishPageCompleted = (result: PageScanResult): Promise<void> => {
		this.pageCompleted.push(JSON.parse(JSON.stringify(result)) as PageScanResult);
		return Promise.resolve();
	};

	publishScanCompleted = (results: ScanResults, timing?: ScanTiming): Promise<void> => {
		this.scanCompleted.push({
			results: JSON.parse(JSON.stringify(results)) as ScanResults,
			...(timing !== undefined ? { timing: JSON.parse(JSON.stringify(timing)) as ScanTiming } : {})
		});
		return Promise.resolve();
	};

	publishScanFailed = (error: string, details?: string): Promise<void> => {
		this.scanFailed.push({ error, ...(details !== undefined ? { details } : {}) });
		return Promise.resolve();
	};

	close = (): Promise<void> => Promise.resolve();
}

class CapturingStorageProvider implements StorageProvider {
	uploads: { bucket: string; key: string; body: string }[] = [];
	downloadFromPath: string | undefined;

	ensureBucket = (): Promise<void> => Promise.resolve();

	upload = async (bucket: string, key: string, filePath: string): Promise<void> => {
		const body = await readFile(filePath, 'utf8');
		this.uploads.push({ bucket, key, body });
	};

	uploadBuffer = (bucket: string, key: string, data: Buffer): Promise<void> => {
		this.uploads.push({ bucket, key, body: data.toString('utf8') });
		return Promise.resolve();
	};

	uploadDirectory = (): Promise<number> => Promise.resolve(0);

	download = async (_bucket: string, _key: string, dest: string): Promise<void> => {
		if (this.downloadFromPath === undefined) {
			throw new Error('CapturingStorageProvider.download called without downloadFromPath set');
		}
		await mkdir(dirname(dest), { recursive: true });
		await copyFile(this.downloadFromPath, dest);
	};

	exists = (): Promise<boolean> => Promise.resolve(true);
}

function makeConfig(jobId: string, resultsDir: string): ScannerConfig {
	return {
		jobId,
		provenancePath: join(resultsDir, 'provenance.json'),
		resultsDir,
		scannerName: 'axe',
		concurrency: 1,
		maxRetries: 1,
		browser: {
			headless: true,
			args: [
				'--no-sandbox',
				'--disable-setuid-sandbox',
				'--disable-dev-shm-usage',
				'--disable-gpu'
			],
			defaultViewport: { width: 1280, height: 720 },
			deviceScaleFactor: 1,
			defaultTimeout: 30_000,
			pageLoadTimeout: 30_000
		},
		storage: {
			endpoint: 'localhost:9000',
			accessKey: 'k',
			secretKey: 's',
			useSSL: false,
			bucket: jobId
		},
		messaging: {
			url: '',
			subjects: { pageCompleted: 'a', scanCompleted: 'b', scanFailed: 'c' }
		}
	};
}

/**
 * Asserts no canary value appears in `haystack`. Throws a labelled error
 * naming the surface that leaked, the canary that leaked, and a short
 * excerpt of the surrounding bytes for debugging.
 */
function assertNoCanaryLeak(label: string, haystack: string): void {
	for (const canary of ALL_CANARIES) {
		const idx = haystack.indexOf(canary);
		if (idx >= 0) {
			const start = Math.max(0, idx - 40);
			const end = Math.min(haystack.length, idx + canary.length + 40);
			throw new Error(
				`canary leak detected in "${label}": "${canary}" appears at offset ${String(idx)}\n` +
					`...${haystack.slice(start, end)}...`
			);
		}
	}
}

interface PipelineRun {
	results: PageScanResult[];
	auditEvents: PageIteratorAuditEvent[];
	publisher: CapturingPublisher;
	stageStorage: CapturingStorageProvider;
	stageLogPath: string;
	recipePath: string;
	provenancePath: string;
	axeViolations: AxeViolationCapture[];
}

interface AxeViolationCapture {
	pageUrl: string;
	ruleId: string;
	targets: string[];
	html: string;
	ancestorPath: string | undefined;
}

interface RunPipelineOptions {
	jobId: string;
	provenance: Provenance;
	resultsDir: string;
	browserManager: BrowserManager;
	storageProvider?: StorageProvider;
	auxStorageStateFile?: string;
}

async function runRealBrowserPipeline(opts: RunPipelineOptions): Promise<PipelineRun> {
	const { jobId, provenance, resultsDir, browserManager, storageProvider } = opts;
	const config = makeConfig(jobId, resultsDir);

	// Persist Provenance to disk first so PageIterator.loadProvenance reads
	// the file (matches the canonical flow where the orchestrator writes
	// the document into the scanner-runner workspace before it starts).
	await mkdir(resultsDir, { recursive: true });
	await writeFile(config.provenancePath, JSON.stringify(provenance, null, 2), 'utf8');

	const stageStorage = new CapturingStorageProvider();
	const stageLogger = new ScanStageLogger(config, stageStorage);
	const { recipePath } = await stageLogger.start();

	const iterator = new PageIterator(
		browserManager,
		config,
		undefined,
		storageProvider !== undefined ? { storageProvider } : undefined
	);

	const loadedProvenance = await iterator.loadProvenance();

	const publisher = new CapturingPublisher();
	const auditEvents: PageIteratorAuditEvent[] = [];
	const axeViolations: AxeViolationCapture[] = [];

	const scanCallback: PageScanCallback = async (ctx) => {
		const startedAt = new Date().toISOString();
		const t0 = Date.now();

		const axe = new AxeBuilder({ page: ctx.page });
		const axeResult = await axe.analyze();

		for (const violation of axeResult.violations) {
			for (const node of violation.nodes) {
				axeViolations.push({
					pageUrl: ctx.page.url(),
					ruleId: violation.id,
					targets: node.target.map((t) => String(t)),
					html: node.html,
					ancestorPath: ancestorPathOf(node.html)
				});
			}
		}

		const finishedAt = new Date().toISOString();
		return {
			pageId: ctx.pageEntry.id,
			url: ctx.page.url(),
			path: ctx.pageEntry.path,
			success: true,
			issues: [],
			durationMs: Date.now() - t0,
			startedAt,
			finishedAt
		};
	};

	const results = await iterator.iteratePages(loadedProvenance, scanCallback, {
		onPageComplete: async (r, i, total) => {
			await publisher.publishPageCompleted(r, i, total);
		},
		onAuditEvent: (event) => {
			auditEvents.push(event);
			stageLogger.recordEvent(event.type, event.details ?? {});
		}
	});

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
		startedAt: new Date().toISOString(),
		completedAt: new Date().toISOString(),
		durationMs: 1
	};

	stageLogger.setMetrics({
		pages_total: results.length,
		pages_scanned: results.length,
		total_issues: 0
	});
	stageLogger.setArtifacts({
		results_key: `${config.jobId}/${config.scannerName}/results.json`
	});
	const { stageLogPath } = await stageLogger.finalizeSuccess();

	await publisher.publishScanCompleted(aggregate);

	return {
		results,
		auditEvents,
		publisher,
		stageStorage,
		stageLogPath,
		recipePath,
		provenancePath: config.provenancePath,
		axeViolations
	};
}

function ancestorPathOf(html: string): string | undefined {
	// Best-effort: the axe report emits the failing element's outerHTML, not
	// its DOM path. We use this only to make sure the violation came from the
	// post-login DOM. The post-login fixture wraps everything in
	// <main id="post-login">; the login form sits in <main id="login-page">.
	if (html.includes('id="post-login"') || html.includes("id='post-login'")) {
		return 'main#post-login';
	}
	return undefined;
}

async function captureStorageStateViaForm(opts: {
	fixture: FixtureApp;
	destPath: string;
}): Promise<void> {
	// Drives a fresh, ephemeral chromium to exercise the form login and dump
	// the resulting Playwright storage-state JSON. Mirrors the on-disk shape
	// `stageflow auth capture` produces locally, without going through the
	// CLI surface, so the runtime path under test is the production hydrator
	// rather than the CLI.
	const browser = await chromium.launch({
		headless: true,
		args: ['--no-sandbox', '--disable-setuid-sandbox', '--disable-dev-shm-usage', '--disable-gpu'],
		chromiumSandbox: false
	});
	try {
		const context: BrowserContext = await browser.newContext();
		const page: Page = await context.newPage();
		await page.goto(opts.fixture.loginUrl, { waitUntil: 'load' });
		await page.fill('input[name=email]', SECRET_USER);
		await page.fill('input[name=password]', SECRET_PASSWORD);
		await Promise.all([
			page.waitForURL(opts.fixture.profileUrl, { waitUntil: 'load' }),
			page.click('button[type=submit]')
		]);
		await mkdir(dirname(opts.destPath), { recursive: true });
		await context.storageState({ path: opts.destPath });
		await page.close();
		await context.close();
	} finally {
		await browser.close();
	}
}

interface LighthouseRunResult {
	requestedUrl: string;
	finalUrl: string;
}

async function runLighthouseAgainstPostLogin(opts: {
	url: string;
	cookieHeader: string;
}): Promise<LighthouseRunResult> {
	// Dynamic import to match the production lighthouse scanner's loader and
	// keep this test file safe to import-evaluate even when chrome-launcher
	// or lighthouse can't initialize on a particular host.
	const chromeLauncher = (await import('chrome-launcher')) as unknown as {
		launch: (opts: Record<string, unknown>) => Promise<{
			port: number;
			pid?: number;
			kill: () => Promise<void>;
		}>;
	};
	const lighthouseModule = (await import('lighthouse')) as unknown as {
		default: (
			url: string,
			flags: Record<string, unknown>,
			config: Record<string, unknown>
		) => Promise<{ lhr: { requestedUrl: string; finalUrl: string } } | undefined>;
	};

	const chromePath = chromium.executablePath();
	const chrome = await chromeLauncher.launch({
		chromePath,
		chromeFlags: [
			'--headless=new',
			'--no-sandbox',
			'--disable-setuid-sandbox',
			'--disable-dev-shm-usage',
			'--disable-gpu'
		]
	});

	try {
		const flags: Record<string, unknown> = {
			port: chrome.port,
			output: 'json',
			onlyCategories: ['accessibility'],
			extraHeaders: { Cookie: opts.cookieHeader },
			throttling: { cpuSlowdownMultiplier: 1, rttMs: 0, throughputKbps: 0 },
			disableStorageReset: false,
			formFactor: 'desktop',
			screenEmulation: {
				mobile: false,
				width: 1280,
				height: 720,
				deviceScaleFactor: 1,
				disabled: false
			}
		};
		const config: Record<string, unknown> = {
			extends: 'lighthouse:default',
			settings: {
				onlyCategories: ['accessibility'],
				formFactor: 'desktop',
				throttling: { cpuSlowdownMultiplier: 1 },
				screenEmulation: {
					mobile: false,
					width: 1280,
					height: 720,
					deviceScaleFactor: 1,
					disabled: false
				}
			}
		};

		const runnerResult = await lighthouseModule.default(opts.url, flags, config);
		if (!runnerResult?.lhr) {
			throw new Error('Lighthouse did not return an lhr');
		}

		return {
			requestedUrl: runnerResult.lhr.requestedUrl,
			finalUrl: runnerResult.lhr.finalUrl
		};
	} finally {
		try {
			await chrome.kill();
		} catch {
			// ignore
		}
	}
}

describe('Authenticated scan, real browser end-to-end', () => {
	const originalAllowPrivate = process.env.ALLOW_PRIVATE_TARGETS;
	const originalUser = process.env.STAGEFLOW_AUTH_USER;
	const originalPassword = process.env.STAGEFLOW_AUTH_PASSWORD;

	let fixture: FixtureApp;
	let workspaceDir: string;
	let browserManager: BrowserManager;

	beforeAll(async () => {
		// Loopback fixture: 127.0.0.1 falls inside the runtime SSRF blocklist.
		// Allowlist private targets so PageIterator can navigate to it. The
		// production guard is unchanged: ALLOW_PRIVATE_TARGETS is the same
		// flag operators set when scanning a localhost service.
		process.env.ALLOW_PRIVATE_TARGETS = 'true';
		process.env.STAGEFLOW_AUTH_USER = SECRET_USER;
		process.env.STAGEFLOW_AUTH_PASSWORD = SECRET_PASSWORD;

		fixture = await startFixtureLoginApp();
		workspaceDir = await mkdtemp(join(tmpdir(), 'stageflow-e2e-auth-'));
		browserManager = new BrowserManager({
			headless: true,
			args: [
				'--no-sandbox',
				'--disable-setuid-sandbox',
				'--disable-dev-shm-usage',
				'--disable-gpu'
			],
			defaultViewport: { width: 1280, height: 720 },
			deviceScaleFactor: 1,
			defaultTimeout: 30_000,
			pageLoadTimeout: 30_000
		});
	}, REAL_BROWSER_TIMEOUT_MS);

	afterAll(async () => {
		try {
			await browserManager.close();
		} catch {
			// ignore
		}
		await fixture.close();
		await rm(workspaceDir, { recursive: true, force: true });

		if (originalAllowPrivate === undefined) {
			delete process.env.ALLOW_PRIVATE_TARGETS;
		} else {
			process.env.ALLOW_PRIVATE_TARGETS = originalAllowPrivate;
		}
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
	}, REAL_BROWSER_TIMEOUT_MS);

	it(
		'form mode: scans the post-login DOM, axe violation lives under #post-login, no canary leaks anywhere',
		async () => {
			const jobId = 'job-e2e-form';
			const resultsDir = join(workspaceDir, jobId);
			const provenance: Provenance = {
				version: '1.0.0',
				job_id: jobId,
				base_url: fixture.url,
				mode: 'live',
				default_wait_for: { type: 'load' },
				pages: [{ id: 'profile', path: '/profile', url: fixture.profileUrl }],
				auth: {
					mode: 'form',
					login_url: fixture.loginUrl,
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
					success: { type: 'selector', selector: '#post-login', timeout: 15_000 }
				}
			};

			const run = await runRealBrowserPipeline({
				jobId,
				provenance,
				resultsDir,
				browserManager
			});

			expect(run.results).toHaveLength(1);
			expect(run.results[0]?.success).toBe(true);
			expect(run.results[0]?.url).toBe(fixture.profileUrl);

			const authEvent = run.auditEvents.find((e) => e.type === 'auth_hydrated');
			expect(authEvent).toBeDefined();
			expect(authEvent?.details).toMatchObject({
				mode: 'form',
				login_url: fixture.loginUrl,
				post_login_url: fixture.profileUrl
			});

			// The deliberate axe violation lives only on the post-login DOM.
			// If the scanner had audited /login (a redirect target), no
			// image-alt violation would exist.
			const imageAltViolations = run.axeViolations.filter((v) => v.ruleId === 'image-alt');
			expect(imageAltViolations.length).toBeGreaterThan(0);
			for (const v of imageAltViolations) {
				expect(v.pageUrl).toBe(fixture.profileUrl);
			}
			// The captured violations sit inside <main id="post-login">. We
			// verify by grabbing the fixture's profile DOM through the same
			// authenticated context once more and asserting the marker
			// surrounds the violating <img>.
			const ctxForCheck = await browserManager.createContext();
			try {
				// Replay form auth manually for this isolated check.
				const checkPage = await ctxForCheck.newPage();
				await checkPage.goto(fixture.loginUrl, { waitUntil: 'load' });
				await checkPage.fill('input[name=email]', SECRET_USER);
				await checkPage.fill('input[name=password]', SECRET_PASSWORD);
				await Promise.all([
					checkPage.waitForURL(fixture.profileUrl),
					checkPage.click('button[type=submit]')
				]);
				const markerCount = await checkPage.locator('#post-login img:not([alt])').count();
				expect(markerCount).toBeGreaterThan(0);
				await checkPage.close();
			} finally {
				await ctxForCheck.close();
			}

			await assertNoCanariesAcrossSurfaces(run);
		},
		REAL_BROWSER_TIMEOUT_MS
	);

	it(
		'storage_state mode: scans the post-login DOM via a captured session, no canary leaks anywhere',
		async () => {
			const jobId = 'job-e2e-storage-state';
			const resultsDir = join(workspaceDir, jobId);

			// Capture a real Playwright storage-state JSON by replaying the
			// form once in a separate ephemeral browser, then feed the file
			// to PageIterator via a stub StorageProvider.
			const captureDir = join(workspaceDir, 'capture');
			const captureFile = join(captureDir, 'storage-state.json');
			await captureStorageStateViaForm({ fixture, destPath: captureFile });

			// Sanity: the captured file is a real Playwright storage-state
			// document with the fixture's session cookie. The cookie value
			// IS the canary; this is the file we pass to PageIterator.
			const captured = JSON.parse(await readFile(captureFile, 'utf8')) as {
				cookies: { name: string; value: string }[];
			};
			const sessionCookie = captured.cookies.find((c) => c.name === SESSION_COOKIE_NAME);
			expect(sessionCookie?.value).toBe(SESSION_COOKIE_VALUE);

			const storageProvider = new CapturingStorageProvider();
			storageProvider.downloadFromPath = captureFile;

			const provenance: Provenance = {
				version: '1.0.0',
				job_id: jobId,
				base_url: fixture.url,
				mode: 'live',
				default_wait_for: { type: 'load' },
				pages: [{ id: 'profile', path: '/profile', url: fixture.profileUrl }],
				auth: {
					mode: 'storage_state',
					artifact_key: `${jobId}/auth/storage-state.json`
				}
			};

			const run = await runRealBrowserPipeline({
				jobId,
				provenance,
				resultsDir,
				browserManager,
				storageProvider
			});

			expect(run.results).toHaveLength(1);
			expect(run.results[0]?.success).toBe(true);
			expect(run.results[0]?.url).toBe(fixture.profileUrl);

			const authEvent = run.auditEvents.find((e) => e.type === 'auth_hydrated');
			expect(authEvent?.details).toMatchObject({
				mode: 'storage_state',
				artifact_key: `${jobId}/auth/storage-state.json`
			});

			const imageAltViolations = run.axeViolations.filter((v) => v.ruleId === 'image-alt');
			expect(imageAltViolations.length).toBeGreaterThan(0);
			for (const v of imageAltViolations) {
				expect(v.pageUrl).toBe(fixture.profileUrl);
			}

			await assertNoCanariesAcrossSurfaces(run);
		},
		REAL_BROWSER_TIMEOUT_MS
	);

	it.runIf(process.env.E2E_LIGHTHOUSE === '1')(
		'lighthouse audits the post-login URL (gated by E2E_LIGHTHOUSE=1)',
		async () => {
			const cookieHeader = `${SESSION_COOKIE_NAME}=${SESSION_COOKIE_VALUE}`;
			const result = await runLighthouseAgainstPostLogin({
				url: fixture.profileUrl,
				cookieHeader
			});
			expect(result.requestedUrl).toBe(fixture.profileUrl);
			// finalUrl can carry a trailing slash variation; assert it points at /profile.
			expect(result.finalUrl.replace(/\/$/, '')).toBe(fixture.profileUrl.replace(/\/$/, ''));
		},
		LIGHTHOUSE_TIMEOUT_MS
	);
});

/**
 * Assert no canary credential value (USER, PASSWORD, session cookie value)
 * appears in any surface the scanner-runner persists or publishes during the
 * run: stored Provenance, scan recipe, scan stage log, captured stage
 * uploads, captured NATS payloads (page-completed, scan-completed,
 * scan-failed), the synthesized unified report aggregate, and the audit
 * event stream the iterator emits.
 *
 * This extends the per-surface backstop from
 * tests/core/auth-pipeline-redaction.test.ts with the bytes a real-browser
 * pipeline produces: real axe rawResults, a real cookie-bearing storage
 * state in the storage_state mode, and a real post-login URL recorded into
 * the auth_hydrated event.
 */
async function assertNoCanariesAcrossSurfaces(run: PipelineRun): Promise<void> {
	const provenanceText = await readFile(run.provenancePath, 'utf8');
	assertNoCanaryLeak('stored Provenance', provenanceText);

	for (const upload of run.stageStorage.uploads) {
		assertNoCanaryLeak(`stage upload ${upload.key}`, upload.body);
	}

	assertNoCanaryLeak(
		'NATS publishPageCompleted payloads',
		JSON.stringify(run.publisher.pageCompleted)
	);
	assertNoCanaryLeak(
		'NATS publishScanCompleted payloads (unified report aggregate)',
		JSON.stringify(run.publisher.scanCompleted)
	);
	assertNoCanaryLeak('NATS publishScanFailed payloads', JSON.stringify(run.publisher.scanFailed));
	assertNoCanaryLeak('PageIterator audit events', JSON.stringify(run.auditEvents));
	assertNoCanaryLeak('axe violation captures', JSON.stringify(run.axeViolations));
}
