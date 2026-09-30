import AxeBuilder from '@axe-core/playwright';
import { type Browser, type Page, chromium } from 'playwright';
import { readFile, writeFile } from 'node:fs/promises';
import { afterAll, beforeAll, describe, expect, it, vi } from 'vitest';

import { LinkCheckerScanner, type LinkInfo } from '../../src/scanners/link-checker';
import { SEOScanner } from '../../src/scanners/seo';
import { WebServerFormatter } from '../../src/core/web-server-formatter';
import type {
	PageScanResult,
	ScanContext,
	ScannerConfig,
	ScannerMetadata,
	ScanResults
} from '../../src/core/types';

vi.mock('../../src/core/page-settle', () => ({
	waitForPageSettled: vi.fn().mockResolvedValue(undefined)
}));
vi.mock('../../src/screenshots/page-overview-from-issues', () => ({
	capturePageOverviewFromIssues: vi.fn().mockResolvedValue(null)
}));
vi.mock('../../src/scanners/link-checker/validation', async (importOriginal) => ({
	...(await importOriginal<typeof import('../../src/scanners/link-checker/validation')>()),
	checkSingleLink: vi.fn((url: string) =>
		Promise.resolve({ url, status: 200, error: null, redirects: [], responseTime: 0 })
	)
}));

function scanContext(page: Page): ScanContext {
	return {
		page,
		context: page.context(),
		pageEntry: { id: 'fixture-page', url: 'https://example.com/', path: '/' },
		resultsDir: '/tmp/stageflow-coverage-fixture',
		config: { jobId: 'coverage-fixture' } as ScannerConfig,
		logger: { info: vi.fn(), warn: vi.fn(), error: vi.fn(), debug: vi.fn() }
	};
}

function storedReport(page: PageScanResult, metadata: ScannerMetadata) {
	const timestamp = '2026-09-30T00:00:00.000Z';
	const results: ScanResults = {
		jobId: 'coverage-fixture',
		scanner: metadata.name,
		version: metadata.version,
		totalPages: 1,
		pages: [{ ...page, durationMs: 0, startedAt: timestamp, finishedAt: timestamp }],
		summary: {
			totalIssues: page.issues.length,
			bySeverity: { critical: 0, serious: 0, moderate: 0, minor: 0, info: 0 },
			byCategory: {},
			pagesScanned: 1,
			pagesFailed: 0,
			pagesWithIssues: 1,
			avgDurationMs: 0
		},
		startedAt: timestamp,
		completedAt: timestamp,
		durationMs: 0
	};
	return new WebServerFormatter().format(
		{
			version: '1.0.0',
			job_id: 'coverage-fixture',
			base_url: 'https://example.com/',
			pages: [{ id: 'fixture-page', url: 'https://example.com/', path: '/' }]
		},
		results,
		metadata
	);
}

describe('specialist rule coverage', () => {
	let browser: Browser;

	beforeAll(async () => {
		browser = await chromium.launch({ headless: true, args: ['--no-sandbox'] });
	});

	afterAll(async () => {
		await browser.close();
	});

	it('axe detects the concerns delegated by SEO and link-checker', async () => {
		const context = await browser.newContext();
		const page = await context.newPage();
		try {
			await page.setContent(
				'<!doctype html><html><head></head><body><main><h2>Section</h2><h4>Skipped level</h4><img src="data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7"><a href="/docs"></a></main></body></html>'
			);
			const results = await new AxeBuilder({ page }).analyze();
			const rules = results.violations.map(({ id }) => id);
			expect(rules).toEqual(
				expect.arrayContaining([
					'document-title',
					'html-has-lang',
					'image-alt',
					'heading-order',
					'link-name'
				])
			);
			const seoScanner = new SEOScanner();
			const seoResult = await seoScanner.scanPage(scanContext(page));
			expect(seoResult.success).toBe(true);
			const seoReport = storedReport(seoResult, seoScanner.metadata);
			const failedChecks = seoReport.issues.map(({ ruleId }) => ruleId);
			expect(failedChecks).toEqual(
				expect.arrayContaining([
					'seo-missing-title',
					'seo-missing-language',
					'seo-images-missing-alt',
					'seo-heading-hierarchy'
				])
			);
			const linkScanner = new LinkCheckerScanner();
			const linkResult = await linkScanner.scanPage(scanContext(page));
			expect(linkResult.success).toBe(true);
			const linkReport = storedReport(linkResult, linkScanner.metadata);
			expect(linkReport.issues.map(({ ruleId }) => ruleId)).toContain('link-checker-no-text-links');
			const fixture = { seo: seoReport, 'link-checker': linkReport };
			const fixtureURL = new URL(
				'../../../orchestrator/internal/adapters/storage/testdata/specialist-scanner-output.json',
				import.meta.url
			);
			if (process.env.UPDATE_GOLDENS === '1') {
				await writeFile(fixtureURL, JSON.stringify(fixture, null, 2) + '\n');
			}
			expect(JSON.parse(await readFile(fixtureURL, 'utf8'))).toEqual(fixture);
		} finally {
			await context.close();
		}
	});

	it('SEO keeps checks whose semantics axe does not cover', async () => {
		const context = await browser.newContext();
		const page = await context.newPage();
		try {
			await page.setContent(
				'<!doctype html><html lang="en"><head><title>A useful title</title></head><body><main><p>This page has no primary heading or meta description.</p></main></body></html>'
			);
			const results = await new AxeBuilder({ page }).analyze();
			expect(results.violations.map(({ id }) => id)).not.toContain('meta-description');
			const seoScanner = new SEOScanner();
			const seoResult = await seoScanner.scanPage(scanContext(page));
			expect(seoResult.success).toBe(true);
			const seoReport = storedReport(seoResult, seoScanner.metadata);
			const failedChecks = seoReport.issues.map(({ ruleId }) => ruleId);
			expect(failedChecks).toEqual(
				expect.arrayContaining([
					'seo-missing-description',
					'seo-missing-h1',
					'seo-missing-viewport'
				])
			);
		} finally {
			await context.close();
		}
	});
	it('resolves document base URLs and avoids false accessible-name and empty-link findings', async () => {
		const context = await browser.newContext();
		const page = await context.newPage();
		try {
			await page.setContent(
				'<html lang="en"><head><title>Links</title><base href="https://example.com/docs/"></head><body><span id="label">Help</span><a id="first" href="about#one" aria-labelledby="label"></a><a id="second" href="about#two">About</a><a id="empty-href" href="">Reload</a><a id="no-href">Section</a><a id="placeholder" href="#">Placeholder</a><a id="script" href="JavaScript:void(0)">Action</a><a id="decorative" href="/image"><img alt="" src="data:,x"></a></body></html>'
			);
			const scanner = new LinkCheckerScanner() as unknown as {
				extractLinks: (page: Page, baseURL: string) => Promise<LinkInfo[]>;
				checkEmptyLinks: (page: Page) => Promise<{ selector: string }[]>;
				checkLinksWithoutText: (page: Page) => Promise<{ selector: string }[]>;
			};
			const links = await scanner.extractLinks(page, 'https://example.com/');
			expect(links.every(({ href }) => href.startsWith('https://'))).toBe(true);
			expect(links.filter(({ href }) => href === 'https://example.com/docs/about')).toHaveLength(2);
			expect((await scanner.checkEmptyLinks(page)).map(({ selector }) => selector)).toEqual([
				'#placeholder',
				'#script'
			]);
			expect((await scanner.checkLinksWithoutText(page)).map(({ selector }) => selector)).toEqual([
				'#decorative'
			]);
		} finally {
			await context.close();
		}
	});
});
