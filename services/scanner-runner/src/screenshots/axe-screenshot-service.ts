import type { Page } from 'playwright';

import type { AxeScreenshotConfig, PageOverviewResult, PageOverviewViolation } from './axe/types';

import { createLogger } from '../utils/logger';
import { loadAxeScreenshotConfig } from './axe/config';
import { saveScreenshot } from './axe/image';
import { capturePageOverviewRaw, loadPageOverviewConfig } from './axe/page-overview';

export type { PageOverviewViolation } from './axe/types';

const log = createLogger('AxeScreenshotService');

/**
 * AxeScreenshotService captures one full-page overview per page, with every
 * finding's elements located by percentage bounds. The report crops that
 * image per finding, so there is no per-violation capture.
 */
export class AxeScreenshotService {
	private readonly cfg: AxeScreenshotConfig;

	constructor(config?: Partial<AxeScreenshotConfig>) {
		this.cfg = loadAxeScreenshotConfig(config);
	}

	/**
	 * Capture a full-page overview screenshot and return percentage-based bounds for frontend overlays.
	 *
	 * @param page - Playwright page instance
	 * @param violations - All violations for this page
	 * @param resultsDir - Directory to save the screenshot
	 * @param pageId - Unique page identifier for filename
	 * @returns PageOverviewResult or null if capture fails/is disabled
	 */
	async capturePageOverview(
		page: Page,
		violations: PageOverviewViolation[],
		resultsDir: string,
		pageId: string,
		options?: { scannerId?: string }
	): Promise<PageOverviewResult | null> {
		const scannerId = options?.scannerId ?? 'axe';
		const overviewCfg = loadPageOverviewConfig();
		const captured = await capturePageOverviewRaw(
			page,
			violations,
			resultsDir,
			pageId,
			scannerId,
			this.cfg,
			overviewCfg
		);

		if (!captured) {
			return null;
		}

		try {
			await saveScreenshot(captured.buffer, captured.screenshotPath, this.cfg);
		} catch (error) {
			log.warn('Failed to save page overview screenshot', {
				pageId,
				screenshotPath: captured.screenshotPath,
				error: error instanceof Error ? error.message : String(error)
			});

			return null;
		}

		return {
			screenshotPath: captured.screenshotPath,
			screenshotFilename: captured.screenshotFilename,
			pageWidth: captured.pageWidth,
			pageHeight: captured.pageHeight,
			elements: captured.elements
		};
	}
}
