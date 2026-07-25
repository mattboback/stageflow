import type { BrowserContext, Page } from 'playwright';

import { mkdtemp, rm } from 'node:fs/promises';
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

vi.mock('../../../src/screenshots/axe-screenshot-service', () => ({
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
		resultsDir = await mkdtemp(path.join(os.tmpdir(), 'stageflow-axe-test-'));
		capturePageOverviewMock.mockResolvedValue(null);
		axeAnalyzeMock.mockResolvedValue({
			violations: [],
			passes: [],
			incomplete: [],
			inapplicable: []
		});
	});

	afterEach(async () => {
		await rm(resultsDir, { force: true, recursive: true });
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
					// axe reports color-contrast at `serious` on incomplete results too.
					// This fixture used to omit `impact`, so the assertion below passed
					// only because the severity fell through to the `moderate` default —
					// production emitted `serious`. Kept realistic so the mapper's
					// "unverified is not failing" rule is what the test actually proves.
					impact: 'serious',
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
					impact: 'serious',
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
					impact: 'serious',
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

	it('forwards axe contrast check data on incomplete nodes', async () => {
		const { AxeScanner } = await import('../../../src/scanners/axe');
		axeAnalyzeMock.mockResolvedValue({
			violations: [],
			passes: [],
			inapplicable: [],
			incomplete: [
				{
					id: 'color-contrast',
					impact: 'serious',
					tags: ['cat.color', 'wcag2aa', 'wcag143'],
					nodes: [
						{
							target: ['.hero-help'],
							html: '<p class="hero-help">Help text</p>',
							failureSummary: 'Fix any of the following: ...',
							any: [
								{
									id: 'color-contrast',
									data: {
										fgColor: '#666666',
										bgColor: null,
										contrastRatio: 0,
										expectedContrastRatio: '4.5:1',
										fontSize: '10.0pt (13.3333px)',
										fontWeight: 'normal',
										messageKey: 'bgImage',
										shadowColor: 'drop-me'
									}
								}
							]
						}
					]
				}
			]
		});

		const scanner = new AxeScanner();
		const result = await scanner.scanPage(createMockContext(resultsDir));

		expect(result.success).toBe(true);
		expect(result.issues).toHaveLength(1);
		expect(result.issues[0]?.metadata?.contrastData).toEqual({
			fgColor: '#666666',
			contrastRatio: 0,
			expectedContrastRatio: '4.5:1',
			fontSize: '10.0pt (13.3333px)',
			fontWeight: 'normal',
			messageKey: 'bgImage'
		});
	});

	it('omits contrastData when axe provides no check data', async () => {
		const { AxeScanner } = await import('../../../src/scanners/axe');
		axeAnalyzeMock.mockResolvedValue({
			violations: [],
			passes: [],
			inapplicable: [],
			incomplete: [
				{
					id: 'color-contrast',
					nodes: [{ target: ['.plain'], html: '<span class="plain">text</span>' }]
				}
			]
		});

		const scanner = new AxeScanner();
		const result = await scanner.scanPage(createMockContext(resultsDir));

		expect(result.issues).toHaveLength(1);
		expect(result.issues[0]?.metadata?.contrastData).toBeUndefined();
	});

	it('attaches first-node contrast data to color-contrast violations', async () => {
		const { AxeScanner } = await import('../../../src/scanners/axe');
		axeAnalyzeMock.mockResolvedValue({
			violations: [
				{
					id: 'color-contrast',
					impact: 'serious',
					help: 'Elements must meet minimum color contrast ratio thresholds',
					tags: ['cat.color', 'wcag2aa', 'wcag143'],
					nodes: [
						{
							target: ['.low-contrast'],
							html: '<p class="low-contrast">Faint</p>',
							failureSummary: 'Fix any of the following: ...',
							any: [
								{
									id: 'color-contrast',
									data: {
										fgColor: '#999999',
										bgColor: '#ffffff',
										contrastRatio: 2.85,
										expectedContrastRatio: '4.5:1'
									}
								}
							]
						},
						{
							target: ['.other'],
							html: '<p class="other">Other</p>',
							any: [{ id: 'color-contrast', data: { fgColor: '#000000', bgColor: '#111111' } }]
						}
					]
				},
				{
					id: 'image-alt',
					impact: 'critical',
					help: 'Images must have alternate text',
					tags: ['cat.text-alternatives', 'wcag2a'],
					nodes: [
						{
							target: ['img.logo'],
							html: '<img class="logo" src="logo.png">',
							any: [{ id: 'has-alt', data: { someField: 'value' } }]
						}
					]
				}
			],
			passes: [],
			inapplicable: [],
			incomplete: []
		});

		const scanner = new AxeScanner();
		const result = await scanner.scanPage(createMockContext(resultsDir));

		expect(result.success).toBe(true);
		const contrastIssue = result.issues.find((issue) => issue.id === 'color-contrast');
		expect(contrastIssue?.metadata?.contrastData).toEqual({
			fgColor: '#999999',
			bgColor: '#ffffff',
			contrastRatio: 2.85,
			expectedContrastRatio: '4.5:1'
		});

		const altIssue = result.issues.find((issue) => issue.id === 'image-alt');
		expect(altIssue?.metadata?.contrastData).toBeUndefined();
	});

	it('does not promote contrast incompletes that have no text to verify', async () => {
		const { AxeScanner } = await import('../../../src/scanners/axe');
		axeAnalyzeMock.mockResolvedValue({
			violations: [],
			passes: [],
			inapplicable: [],
			incomplete: [
				{
					id: 'color-contrast',
					impact: 'serious',
					tags: ['cat.color', 'wcag2aa', 'wcag143'],
					nodes: [
						{
							// axe's `nonBmp`: the element's content is only non-text
							// characters, so there is no contrast a reviewer could assess.
							// Scanning StageFlow's own landing page produced nine of these
							// on aria-hidden icon glyphs alone.
							target: ['.btn > span[aria-hidden="true"]'],
							html: '<span aria-hidden="true">→</span>',
							any: [{ id: 'color-contrast', data: { messageKey: 'nonBmp' } }]
						}
					]
				}
			]
		});

		const scanner = new AxeScanner();
		const result = await scanner.scanPage(createMockContext(resultsDir));

		expect(result.success).toBe(true);
		expect(result.issues).toEqual([]);
	});

	it('keeps the verifiable nodes when one result mixes both kinds', async () => {
		const { AxeScanner } = await import('../../../src/scanners/axe');
		axeAnalyzeMock.mockResolvedValue({
			violations: [],
			passes: [],
			inapplicable: [],
			incomplete: [
				{
					id: 'color-contrast',
					impact: 'serious',
					tags: ['cat.color', 'wcag2aa', 'wcag143'],
					nodes: [
						{
							target: ['.icon'],
							html: '<span aria-hidden="true">→</span>',
							any: [{ id: 'color-contrast', data: { messageKey: 'nonBmp' } }]
						},
						{
							// `bgOverlap` is genuinely ambiguous: there is real text, and a
							// human has to determine what is behind it.
							target: ['.gauge__val'],
							html: '<span class="gauge__val">88</span>',
							any: [{ id: 'color-contrast', data: { messageKey: 'bgOverlap' } }]
						}
					]
				}
			]
		});

		const scanner = new AxeScanner();
		const result = await scanner.scanPage(createMockContext(resultsDir));

		expect(result.success).toBe(true);
		expect(result.issues).toHaveLength(1);
		expect(result.issues[0]).toMatchObject({
			id: 'color-contrast',
			severity: 'moderate',
			location: { selector: '.gauge__val' }
		});
		// Filtering is per node, so the surviving node must still be indexed from the
		// filtered list rather than carrying its original offset.
		expect(result.issues[0]?.metadata).toMatchObject({ incompleteNodeIndex: 0 });
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
