/**
 * AxeScreenshotService Tests
 */

import type { Page } from 'playwright';

import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { AxeViolation, PageOverviewViolation } from '../../src/screenshots/axe/types';

import { AxeScreenshotService } from '../../src/screenshots/AxeScreenshotService';

vi.mock('../../src/screenshots/axe/config', () => ({
	loadAxeScreenshotConfig: vi.fn((overrides?: Record<string, unknown>) => ({
		screenshotsEnabled: true,
		injectHighlight: true,
		shotScale: 1,
		shotMinWidth: 200,
		shotMinHeight: 150,
		deviceScaleFactor: 1,
		highlightStyle: 'dashed',
		mergeTargets: true,
		unionMaxTargets: 10,
		clipPadding: 20,
		contextRatio: 0.5,
		scrollTimeout: 2000,
		thumbnailSize: 200,
		generateThumbnails: true,
		outputFormat: 'png',
		webpQuality: 80,
		overlayMethod: 'css-injection',
		semanticOverlayEnabled: true,
		semanticOverlayMaxHeadings: 20,
		semanticOverlayMaxLabelLength: 40,
		semanticOverlayLegendEnabled: true,
		...(overrides ?? {})
	}))
}));

vi.mock('../../src/screenshots/axe/violation-capture', () => ({
	captureViolationScreenshot: vi.fn().mockResolvedValue({
		status: 'captured',
		screenshot: {
			screenshot: 'violation-test-uuid.png',
			thumbnail: 'violation-test-uuid-thumb.png',
			locationInfo: {
				scrollY: 0,
				viewportHeight: 720,
				docHeight: 1500,
				position: 0.24
			},
			friendlyNode: { label: 'Button', tagName: 'button' }
		},
		fallbacks: []
	})
}));

vi.mock('../../src/screenshots/axe/page-overview', () => ({
	loadPageOverviewConfig: vi.fn(() => ({
		enabled: true,
		maxElements: 100
	})),
	capturePageOverviewRaw: vi.fn().mockResolvedValue({
		buffer: Buffer.from('fake-png'),
		screenshotPath: '/tmp/results/overview.png',
		screenshotFilename: 'overview.png',
		pageWidth: 1280,
		pageHeight: 2000,
		elements: []
	})
}));

vi.mock('../../src/screenshots/axe/image', () => ({
	saveScreenshot: vi.fn().mockResolvedValue(undefined),
	generateThumbnail: vi.fn().mockResolvedValue('overview-thumb.png'),
	compositeOverlay: vi.fn().mockResolvedValue(Buffer.from('overlaid-png'))
}));

const createMockPage = (): Page => {
	return {
		viewportSize: vi.fn().mockReturnValue({ width: 1280, height: 720 }),
		screenshot: vi.fn().mockResolvedValue(Buffer.from('fake-png')),
		evaluate: vi.fn().mockResolvedValue({}),
		locator: vi.fn().mockReturnValue({
			first: vi.fn().mockReturnThis(),
			scrollIntoViewIfNeeded: vi.fn().mockResolvedValue(undefined),
			boundingBox: vi.fn().mockResolvedValue({ x: 100, y: 100, width: 50, height: 30 }),
			evaluate: vi.fn().mockResolvedValue({})
		}),
		waitForTimeout: vi.fn().mockResolvedValue(undefined)
	} as unknown as Page;
};

describe('AxeScreenshotService', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	describe('constructor', () => {
		it('creates service with default config', () => {
			const service = new AxeScreenshotService();
			expect(service).toBeDefined();
		});

		it('creates service with partial config overrides', () => {
			const service = new AxeScreenshotService({
				screenshotsEnabled: false,
				thumbnailSize: 300
			});
			expect(service).toBeDefined();
		});

		it('creates service with all config options', () => {
			const service = new AxeScreenshotService({
				screenshotsEnabled: true,
				injectHighlight: false,
				shotScale: 2,
				shotMinWidth: 400,
				shotMinHeight: 300,
				deviceScaleFactor: 2,
				highlightStyle: 'solid',
				mergeTargets: false,
				unionMaxTargets: 5,
				clipPadding: 30,
				contextRatio: 0.7,
				scrollTimeout: 3000,
				thumbnailSize: 150,
				generateThumbnails: false,
				outputFormat: 'webp',
				webpQuality: 90,
				overlayMethod: 'sharp-composite',
				semanticOverlayEnabled: false,
				semanticOverlayMaxHeadings: 10,
				semanticOverlayMaxLabelLength: 30,
				semanticOverlayLegendEnabled: false
			});
			expect(service).toBeDefined();
		});
	});

	describe('captureViolationScreenshot', () => {
		it('delegates to captureViolationScreenshot function', async () => {
			const { captureViolationScreenshot: mockCapture } =
				await import('../../src/screenshots/axe/violation-capture');
			const service = new AxeScreenshotService();
			const mockPage = createMockPage();

			const violation: AxeViolation = {
				id: 'button-name',
				nodes: [{ target: ['button.submit'] }]
			};

			const result = await service.captureViolationScreenshot(mockPage, violation, '/tmp/results');

			expect(mockCapture).toHaveBeenCalledWith({
				page: mockPage,
				violation,
				resultsDir: '/tmp/results',
				cfg: expect.any(Object)
			});
			expect(result.status).toBe('captured');
			if (result.status !== 'captured') {
				throw new Error('expected captured result');
			}
			expect(result.screenshot.screenshot).toBe('violation-test-uuid.png');
		});

		it('returns result with screenshot and thumbnail', async () => {
			const service = new AxeScreenshotService();
			const mockPage = createMockPage();

			const violation: AxeViolation = {
				id: 'color-contrast',
				nodes: [{ target: ['p.low-contrast'] }]
			};

			const result = await service.captureViolationScreenshot(mockPage, violation, '/tmp/results');

			expect(result.status).toBe('captured');
			if (result.status !== 'captured') {
				throw new Error('expected captured result');
			}
			expect(result.screenshot).toMatchObject({
				screenshot: expect.any(String),
				thumbnail: expect.any(String)
			});
		});

		it('returns result with location info', async () => {
			const service = new AxeScreenshotService();
			const mockPage = createMockPage();

			const violation: AxeViolation = {
				id: 'image-alt',
				nodes: [{ target: ['img.hero'] }]
			};

			const result = await service.captureViolationScreenshot(mockPage, violation, '/tmp/results');

			expect(result.status).toBe('captured');
			if (result.status !== 'captured') {
				throw new Error('expected captured result');
			}
			expect(result.screenshot.locationInfo).toMatchObject({
				scrollY: expect.any(Number),
				viewportHeight: expect.any(Number),
				docHeight: expect.any(Number),
				position: expect.any(Number)
			});
		});

		it('returns result with friendly node info', async () => {
			const service = new AxeScreenshotService();
			const mockPage = createMockPage();

			const violation: AxeViolation = {
				id: 'link-name',
				nodes: [{ target: ['a.empty'] }]
			};

			const result = await service.captureViolationScreenshot(mockPage, violation, '/tmp/results');

			expect(result.status).toBe('captured');
			if (result.status !== 'captured') {
				throw new Error('expected captured result');
			}
			expect(result.screenshot.friendlyNode).toBeDefined();
		});

		it('handles violation with no nodes', async () => {
			const service = new AxeScreenshotService();
			const mockPage = createMockPage();

			const violation: AxeViolation = {
				id: 'some-rule',
				nodes: []
			};

			const result = await service.captureViolationScreenshot(mockPage, violation, '/tmp/results');

			expect(result.status).toBe('captured');
		});

		it('handles violation with undefined nodes', async () => {
			const service = new AxeScreenshotService();
			const mockPage = createMockPage();

			const violation: AxeViolation = {
				id: 'some-rule'
			};

			const result = await service.captureViolationScreenshot(mockPage, violation, '/tmp/results');

			expect(result.status).toBe('captured');
		});
	});

	describe('capturePageOverview', () => {
		it('captures page overview with all violations', async () => {
			const { saveScreenshot } = await import('../../src/screenshots/axe/image');
			const service = new AxeScreenshotService();
			const mockPage = createMockPage();

			const violations: PageOverviewViolation[] = [
				{
					id: 'button-name',
					impact: 'critical',
					nodes: [{ target: ['button.bad'] }]
				},
				{
					id: 'color-contrast',
					impact: 'serious',
					nodes: [{ target: ['p.low'] }]
				}
			];

			const result = await service.capturePageOverview(
				mockPage,
				violations,
				'/tmp/results',
				'page-1'
			);

			expect(saveScreenshot).toHaveBeenCalled();
			expect(result).toBeDefined();
			expect(result?.screenshotPath).toBeDefined();
			expect(result?.screenshotFilename).toBeDefined();
		});

		it('returns page dimensions in result', async () => {
			const service = new AxeScreenshotService();
			const mockPage = createMockPage();

			const violations: PageOverviewViolation[] = [{ id: 'test', impact: 'minor', nodes: [] }];

			const result = await service.capturePageOverview(
				mockPage,
				violations,
				'/tmp/results',
				'page-1'
			);

			expect(result?.pageWidth).toBeDefined();
			expect(result?.pageHeight).toBeDefined();
		});

		it('returns elements array in result', async () => {
			const service = new AxeScreenshotService();
			const mockPage = createMockPage();

			const violations: PageOverviewViolation[] = [
				{
					id: 'heading-order',
					impact: 'moderate',
					nodes: [{ target: ['h2.skip'] }]
				}
			];

			const result = await service.capturePageOverview(
				mockPage,
				violations,
				'/tmp/results',
				'page-1'
			);

			expect(result?.elements).toBeDefined();
			expect(Array.isArray(result?.elements)).toBe(true);
		});

		it('returns null when capture fails', async () => {
			const { capturePageOverviewRaw } = await import('../../src/screenshots/axe/page-overview');
			(capturePageOverviewRaw as ReturnType<typeof vi.fn>).mockResolvedValueOnce(null);

			const service = new AxeScreenshotService();
			const mockPage = createMockPage();

			const result = await service.capturePageOverview(mockPage, [], '/tmp/results', 'page-1');

			expect(result).toBeNull();
		});

		it('handles empty violations array', async () => {
			const service = new AxeScreenshotService();
			const mockPage = createMockPage();

			const result = await service.capturePageOverview(mockPage, [], '/tmp/results', 'page-empty');

			expect(result).toBeDefined();
		});

		it('uses pageId in filename', async () => {
			const { capturePageOverviewRaw } = await import('../../src/screenshots/axe/page-overview');
			const service = new AxeScreenshotService();
			const mockPage = createMockPage();

			await service.capturePageOverview(mockPage, [], '/tmp/results', 'unique-page-id');

			expect(capturePageOverviewRaw).toHaveBeenCalledWith(
				mockPage,
				[],
				'/tmp/results',
				'unique-page-id',
				'axe',
				expect.any(Object),
				expect.any(Object)
			);
		});
	});

	describe('config propagation', () => {
		it('passes config to violation capture', async () => {
			const { captureViolationScreenshot: mockCapture } =
				await import('../../src/screenshots/axe/violation-capture');
			const service = new AxeScreenshotService({
				thumbnailSize: 500
			});
			const mockPage = createMockPage();

			await service.captureViolationScreenshot(mockPage, { id: 'test' }, '/tmp/results');

			expect(mockCapture).toHaveBeenCalledWith(
				expect.objectContaining({
					cfg: expect.objectContaining({
						thumbnailSize: 500
					})
				})
			);
		});

		it('passes config to page overview capture', async () => {
			const { capturePageOverviewRaw } = await import('../../src/screenshots/axe/page-overview');
			const service = new AxeScreenshotService({
				outputFormat: 'webp'
			});
			const mockPage = createMockPage();

			await service.capturePageOverview(mockPage, [], '/tmp/results', 'page-1');

			expect(capturePageOverviewRaw).toHaveBeenCalledWith(
				expect.anything(),
				expect.anything(),
				expect.anything(),
				expect.anything(),
				'axe',
				expect.objectContaining({
					outputFormat: 'webp'
				}),
				expect.anything()
			);
		});
	});
});
