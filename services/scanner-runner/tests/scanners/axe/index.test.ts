import type { BrowserContext, Page } from 'playwright';

import fs from 'fs-extra';
import os from 'node:os';
import path from 'node:path';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { PageEntry, ScanContext, ScannerConfig, ScannerLogger } from '../../../src/core';

const axeAnalyzeMock = vi.hoisted(() => vi.fn());
const capturePageOverviewMock = vi.hoisted(() => vi.fn());

vi.mock('@axe-core/playwright', () => {
	class MockAxeBuilder {
		disableRules = vi.fn(function (this: MockAxeBuilder) {
			return this;
		});

		withTags = vi.fn(function (this: MockAxeBuilder) {
			return this;
		});

		analyze = axeAnalyzeMock;
	}

	return { default: MockAxeBuilder };
});

vi.mock('../../../src/screenshots/AxeScreenshotService', () => ({
	AxeScreenshotService: class MockAxeScreenshotService {
		capturePageOverview = capturePageOverviewMock;

		captureViolationScreenshot = vi.fn().mockResolvedValue({ reason: 'test', status: 'skipped' });
	}
}));

const createMockLogger = (): ScannerLogger => ({
	info: vi.fn(),
	warn: vi.fn(),
	error: vi.fn(),
	debug: vi.fn()
});

const createScannerConfig = (): ScannerConfig => ({
	jobId: 'test-job',
	provenancePath: '/tmp/provenance.json',
	resultsDir: '/tmp/results',
	scannerName: 'axe',
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
});

const createMockPage = (): Page =>
	({
		waitForLoadState: vi.fn().mockResolvedValue(undefined),
		waitForTimeout: vi.fn().mockResolvedValue(undefined)
	}) as unknown as Page;

const createMockContext = (resultsDir: string): ScanContext => {
	const pageEntry: PageEntry = {
		id: 'page-1',
		path: '/',
		url: 'https://example.com'
	};

	return {
		page: createMockPage(),
		context: {} as BrowserContext,
		pageEntry,
		resultsDir,
		config: createScannerConfig(),
		logger: createMockLogger()
	};
};

describe('AxeScanner.scanPage', () => {
	let resultsDir: string;

	beforeEach(async () => {
		resultsDir = await fs.mkdtemp(path.join(os.tmpdir(), 'stageflow-axe-test-'));
		capturePageOverviewMock.mockResolvedValue(null);
		axeAnalyzeMock.mockResolvedValue({
			violations: [],
			passes: [],
			incomplete: [],
			inapplicable: []
		});
	});

	afterEach(async () => {
		await fs.remove(resultsDir);
		vi.clearAllMocks();
	});

	it('promotes incomplete color-contrast nodes to review issues', async () => {
		const { AxeScanner } = await import('../../../src/scanners/axe');
		axeAnalyzeMock.mockResolvedValue({
			violations: [],
			passes: [],
			inapplicable: [],
			incomplete: [
				{
					id: 'color-contrast',
					help: 'Elements must meet minimum color contrast ratio thresholds',
					helpUrl: 'https://dequeuniversity.com/rules/axe/4.11/color-contrast',
					description: 'Ensure the contrast between foreground and background colors meets WCAG.',
					tags: ['cat.color', 'wcag2aa', 'wcag143'],
					nodes: [
						{
							target: ['.hero-help'],
							html: '<p class="hero-help">Enter any public URL above and press Enter to run a live scan</p>',
							failureSummary:
								"Fix any of the following:\n  Element's background color could not be determined due to a pseudo element"
						},
						{
							target: ['.muted-link'],
							html: '<a class="muted-link">LinkedIn</a>',
							failureSummary:
								"Fix any of the following:\n  Element's background color could not be determined due to a background gradient"
						}
					]
				}
			]
		});

		const scanner = new AxeScanner();
		const result = await scanner.scanPage(createMockContext(resultsDir));

		expect(result.success).toBe(true);
		expect(result.issues).toHaveLength(2);
		expect(result.issues[0]).toMatchObject({
			id: 'color-contrast',
			scanner: 'axe',
			severity: 'moderate',
			category: 'wcag2aa',
			title: 'Color contrast needs manual verification',
			location: {
				selector: '.hero-help',
				html: '<p class="hero-help">Enter any public URL above and press Enter to run a live scan</p>'
			}
		});
		expect(result.issues[0]?.description).toContain('cannot treat it as a pass');
		expect(result.issues[0]?.metadata).toMatchObject({
			axeIncomplete: true,
			incompleteNodeIndex: 0,
			nodeCount: 1
		});
		expect(capturePageOverviewMock).toHaveBeenCalledWith(
			expect.anything(),
			[
				{
					id: 'color-contrast',
					nodes: [
						{
							target: ['.hero-help'],
							html: '<p class="hero-help">Enter any public URL above and press Enter to run a live scan</p>',
							failureSummary:
								"Fix any of the following:\n  Element's background color could not be determined due to a pseudo element"
						}
					]
				},
				{
					id: 'color-contrast',
					nodes: [
						{
							target: ['.muted-link'],
							html: '<a class="muted-link">LinkedIn</a>',
							failureSummary:
								"Fix any of the following:\n  Element's background color could not be determined due to a background gradient"
						}
					]
				}
			],
			expect.any(String),
			'page-1',
			{ scannerId: 'axe' }
		);
	});

	it('does not promote non-contrast incomplete results', async () => {
		const { AxeScanner } = await import('../../../src/scanners/axe');
		axeAnalyzeMock.mockResolvedValue({
			violations: [],
			passes: [],
			inapplicable: [],
			incomplete: [
				{
					id: 'aria-valid-attr',
					nodes: [{ target: ['.widget'], html: '<div class="widget"></div>' }]
				}
			]
		});

		const scanner = new AxeScanner();
		const result = await scanner.scanPage(createMockContext(resultsDir));

		expect(result.success).toBe(true);
		expect(result.issues).toEqual([]);
		expect(capturePageOverviewMock).toHaveBeenCalledWith(
			expect.anything(),
			[],
			expect.any(String),
			'page-1',
			{ scannerId: 'axe' }
		);
	});
});
