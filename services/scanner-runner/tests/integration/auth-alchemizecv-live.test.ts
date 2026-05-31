import { mkdtemp, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { afterEach, describe, expect, it } from 'vitest';

import type { PageScanResult, Provenance, ScannerConfig } from '../../src/core/types';

import { BrowserManager } from '../../src/core/browser-manager';
import { PageIterator, type PageScanCallback } from '../../src/core/page-iterator';

const ALCHEMIZECV_BASE_URL = 'https://alchemizecv.com';
const DEMO_EMAIL = process.env.QA_LOGIN_EMAIL ?? '';
const DEMO_PASSWORD = process.env.QA_LOGIN_PASS ?? '';
const RUN_LIVE_QA = process.env.STAGEFLOW_RUN_LIVE_ALCHEMIZECV_QA === '1';

const TEST_TIMEOUT_MS = 90_000;
const describeLive = RUN_LIVE_QA ? describe : describe.skip;

function makeConfig(resultsDir: string): ScannerConfig {
	return {
		jobId: 'alchemizecv-live-auth',
		provenancePath: join(resultsDir, 'provenance.json'),
		resultsDir,
		scannerName: 'auth-live',
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
			accessKey: 'unused',
			secretKey: 'unused',
			useSSL: false,
			bucket: 'unused'
		},
		messaging: {
			url: '',
			subjects: { pageCompleted: 'unused', scanCompleted: 'unused', scanFailed: 'unused' }
		}
	};
}

function makeProvenance(): Provenance {
	return {
		version: '1.0.0',
		job_id: 'alchemizecv-live-auth',
		base_url: ALCHEMIZECV_BASE_URL,
		mode: 'live',
		default_wait_for: { type: 'domcontentloaded' },
		pages: [
			{
				id: 'dashboard',
				path: '/dashboard',
				url: `${ALCHEMIZECV_BASE_URL}/dashboard`
			}
		],
		auth: {
			mode: 'form',
			login_url: `${ALCHEMIZECV_BASE_URL}/login`,
			steps: [
				{ type: 'fill', selector: 'input[type="email"]', value: DEMO_EMAIL },
				{ type: 'fill', selector: 'input[type="password"]', value: DEMO_PASSWORD },
				{ type: 'click', selector: 'button[type="submit"]' }
			],
			success: { type: 'selector', selector: 'a[href="/dashboard"]', timeout: 30_000 }
		}
	};
}

describeLive('live AlchemizeCV authenticated scanning', () => {
	let tmp: string | undefined;
	let browserManager: BrowserManager | undefined;

	afterEach(async () => {
		if (browserManager) {
			await browserManager.close();
			browserManager = undefined;
		}
		if (tmp) {
			await rm(tmp, { recursive: true, force: true });
			tmp = undefined;
		}
	});

	it(
		'logs into the live demo account and scans the dashboard as authenticated content',
		async () => {
			expect(DEMO_EMAIL, 'QA_LOGIN_EMAIL is required for live QA').not.toBe('');
			expect(DEMO_PASSWORD, 'QA_LOGIN_PASS is required for live QA').not.toBe('');

			tmp = await mkdtemp(join(tmpdir(), 'stageflow-alchemizecv-live-'));
			browserManager = new BrowserManager(makeConfig(tmp).browser);

			const iterator = new PageIterator(browserManager, makeConfig(tmp));
			const auditEvents: { type: string; details?: Record<string, unknown> }[] = [];

			const scanCallback: PageScanCallback = async (ctx) => {
				await ctx.page.waitForSelector('a[href="/dashboard"]', { timeout: 30_000 });
				const state = await ctx.page.evaluate(() => ({
					url: window.location.href,
					hasLoginPasswordField: document.querySelector('input[type="password"]') !== null,
					bodyText: document.body.innerText
				}));
				const me = await ctx.page.evaluate(async () => {
					const response = await fetch('/api/auth/me');
					return (await response.json()) as { email?: string };
				});

				expect(state.url).toBe(`${ALCHEMIZECV_BASE_URL}/dashboard`);
				expect(state.hasLoginPasswordField).toBe(false);
				expect(state.bodyText).toContain('Good');
				expect(me.email).toBe(DEMO_EMAIL);

				return {
					pageId: ctx.pageEntry.id,
					url: ctx.page.url(),
					path: ctx.pageEntry.path,
					success: true,
					issues: [],
					durationMs: 0,
					startedAt: new Date().toISOString(),
					finishedAt: new Date().toISOString()
				} satisfies PageScanResult;
			};

			const results = await iterator.iteratePages(makeProvenance(), scanCallback, {
				onAuditEvent: (event) => auditEvents.push(event)
			});

			expect(auditEvents).toContainEqual(
				expect.objectContaining({
					type: 'auth_hydrated',
					details: expect.objectContaining({
						mode: 'form',
						post_login_url: `${ALCHEMIZECV_BASE_URL}/dashboard`
					})
				})
			);
			expect(results).toHaveLength(1);
			expect(results[0]).toMatchObject({
				pageId: 'dashboard',
				success: true,
				url: `${ALCHEMIZECV_BASE_URL}/dashboard`
			});
			expect(results[0]!.issues).toEqual([]);
		},
		TEST_TIMEOUT_MS
	);
});
