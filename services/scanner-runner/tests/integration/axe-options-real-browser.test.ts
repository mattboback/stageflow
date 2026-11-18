import { type Browser, chromium } from 'playwright';
import { mkdtemp, rm } from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import { afterAll, beforeAll, describe, expect, it, vi } from 'vitest';

import type { ScanContext, ScannerConfig } from '../../src/core';
import { AxeScanner } from '../../src/scanners/axe';

// Filter coverage uses real axe analysis; screenshot rendering has its own suites.
vi.mock('../../src/screenshots/axe-screenshot-service', () => ({
	AxeScreenshotService: class {
		capturePageOverview = vi.fn().mockResolvedValue(null);
	}
}));

class ConfiguredAxeScanner extends AxeScanner {
	async configure(config: ScannerConfig): Promise<void> {
		this.config = config;
		await this.initialize();
	}
}

describe('axe options with real analysis', () => {
	let browser: Browser;
	let resultsDir: string;

	beforeAll(async () => {
		resultsDir = await mkdtemp(path.join(os.tmpdir(), 'stageflow-axe-options-'));
		browser = await chromium.launch({ headless: true, args: ['--no-sandbox'] });
	});

	afterAll(async () => {
		await browser.close();
		await rm(resultsDir, { recursive: true, force: true });
	});

	it('retains unrestricted findings and applies disabled rules and tag restrictions', async () => {
		const cases = [
			{ name: 'default', options: {} },
			{ name: 'disabled', options: { disabledRules: ['image-alt'] } },
			{ name: 'tagged', options: { runOnlyTags: ['wcag2a'] } }
		];
		const rulesByCase = new Map<string, string[]>();
		for (const { name, options } of cases) {
			const context = await browser.newContext();
			try {
				const page = await context.newPage();
				await page.setContent(
					'<!doctype html><html><head></head><body><main><h2>Section</h2><h4>Skipped level</h4><img src="data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7"><a href="/docs"></a></main></body></html>'
				);
				const scanDir = path.join(resultsDir, name);
				const config: ScannerConfig = {
					jobId: 'axe-options-fixture',
					provenancePath: path.join(resultsDir, 'provenance.json'),
					resultsDir: scanDir,
					scannerName: 'axe',
					concurrency: 1,
					maxRetries: 0,
					browser: {
						headless: true,
						args: ['--no-sandbox'],
						defaultViewport: { width: 1280, height: 720 },
						deviceScaleFactor: 1,
						defaultTimeout: 30_000,
						pageLoadTimeout: 30_000
					},
					options
				};
				const scanContext: ScanContext = {
					page,
					context,
					pageEntry: { id: 'fixture-page', url: 'https://example.com/', path: '/' },
					resultsDir: scanDir,
					config,
					logger: { info: vi.fn(), warn: vi.fn(), error: vi.fn(), debug: vi.fn() }
				};
				const scanner = new ConfiguredAxeScanner();
				await scanner.configure(config);
				const result = await scanner.scanPage(scanContext);
				expect(result).toMatchObject({ success: true });
				rulesByCase.set(
					name,
					result.issues.map(({ id }) => id)
				);
			} finally {
				await context.close();
			}
		}

		expect(rulesByCase.get('default')).toEqual(
			expect.arrayContaining([
				'document-title',
				'html-has-lang',
				'image-alt',
				'heading-order',
				'link-name'
			])
		);
		expect(rulesByCase.get('disabled')).not.toContain('image-alt');
		expect(rulesByCase.get('disabled')).toEqual(
			expect.arrayContaining(['document-title', 'html-has-lang', 'heading-order', 'link-name'])
		);
		expect(rulesByCase.get('tagged')).not.toContain('heading-order');
		expect(rulesByCase.get('tagged')).toEqual(
			expect.arrayContaining(['document-title', 'html-has-lang', 'image-alt', 'link-name'])
		);
	}, 30_000);
});
