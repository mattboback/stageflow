/**
 * AxeScreenshotService Tests
 */

import type { Page } from 'playwright';

import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { PageOverviewViolation } from '../../src/screenshots/axe/types';

import { AxeScreenshotService } from '../../src/screenshots/axe-screenshot-service';

vi.mock('../../src/screenshots/axe/config', () => ({
	loadAxeScreenshotConfig: vi.fn((overrides?: Record<string, unknown>) => ({
		screenshotsEnabled: true,
		outputFormat: 'png',
		webpQuality: 80,
		...(overrides ?? {})
	}))
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
	saveScreenshot: vi.fn().mockResolvedValue(undefined)
}));

const createMockPage = (): Page => {
	return {
		viewportSize: vi.fn().mockReturnValue({ width: 1280, height: 720 }),
		screenshot: vi.fn().mockResolvedValue(Buffer.from('fake-png')),
		evaluate: vi.fn().mockResolvedValue({})
	} as unknown as Page;
};

describe('AxeScreenshotService', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	describe('constructor', () => {
		it('creates service with default config', () => {
			expect(new AxeScreenshotService()).toBeDefined();
		});

		it('creates service with config overrides', () => {
			expect(new AxeScreenshotService({ screenshotsEnabled: false })).toBeDefined();
		});
	});

	describe('capturePageOverview', () => {
		it('captures page overview with all violations', async () => {
			const { saveScreenshot } = await import('../../src/screenshots/axe/image');
			const service = new AxeScreenshotService();
			const mockPage = createMockPage();

			const violations: PageOverviewViolation[] = [
				{ id: 'button-name', impact: 'critical', nodes: [{ target: ['button.bad'] }] },
				{ id: 'color-contrast', impact: 'serious', nodes: [{ target: ['p.low'] }] }
			];

			const result = await service.capturePageOverview(
				mockPage,
				violations,
				'/tmp/results',
				'page-1'
			);

			expect(saveScreenshot).toHaveBeenCalledWith(
				Buffer.from('fake-png'),
				'/tmp/results/overview.png',
				expect.objectContaining({ outputFormat: 'png', webpQuality: 80 })
			);
			expect(result).toEqual({
				screenshotPath: '/tmp/results/overview.png',
				screenshotFilename: 'overview.png',
				pageWidth: 1280,
				pageHeight: 2000,
				elements: []
			});
		});

		it('returns null when capture fails', async () => {
			const { capturePageOverviewRaw } = await import('../../src/screenshots/axe/page-overview');
			(capturePageOverviewRaw as ReturnType<typeof vi.fn>).mockResolvedValueOnce(null);

			const result = await new AxeScreenshotService().capturePageOverview(
				createMockPage(),
				[],
				'/tmp/results',
				'page-1'
			);

			expect(result).toBeNull();
		});

		it('returns null when saving the overview screenshot fails', async () => {
			const { saveScreenshot } = await import('../../src/screenshots/axe/image');
			(saveScreenshot as ReturnType<typeof vi.fn>).mockRejectedValueOnce(
				new Error('Processed image is too large for the WebP format')
			);

			const result = await new AxeScreenshotService().capturePageOverview(
				createMockPage(),
				[],
				'/tmp/results',
				'page-large'
			);

			expect(result).toBeNull();
		});

		it('passes the page id, scanner id and config to the capture', async () => {
			const { capturePageOverviewRaw } = await import('../../src/screenshots/axe/page-overview');
			const service = new AxeScreenshotService({ outputFormat: 'webp' });
			const mockPage = createMockPage();

			await service.capturePageOverview(mockPage, [], '/tmp/results', 'unique-page-id', {
				scannerId: 'lighthouse'
			});

			expect(capturePageOverviewRaw).toHaveBeenCalledWith(
				mockPage,
				[],
				'/tmp/results',
				'unique-page-id',
				'lighthouse',
				expect.objectContaining({ outputFormat: 'webp' }),
				expect.objectContaining({ enabled: true, maxElements: 100 })
			);
		});
	});
});
