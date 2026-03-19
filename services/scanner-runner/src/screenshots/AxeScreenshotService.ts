import type { Page } from "playwright";

import type {
	AxeScreenshotConfig,
	AxeViolation,
	PageOverviewResult,
	PageOverviewViolation,
	ViolationScreenshotCaptureResult,
} from "./axe/types";

import { loadAxeScreenshotConfig } from "./axe/config";
import { saveScreenshot } from "./axe/image";
import {
	capturePageOverviewRaw,
	loadPageOverviewConfig,
} from "./axe/page-overview";
import { captureViolationScreenshot } from "./axe/violation-capture";

export type {
	AxeViolation,
	EnhancedScreenshotResult,
	PageOverviewViolation,
	ViolationCaptureFailure,
	ViolationScreenshotCaptureResult,
} from "./axe/types";

/**
 * AxeScreenshotService encapsulates the screenshot strategy used for
 * accessibility violations:
 *
 * - Injects temporary highlight CSS around violation targets.
 * - Attempts a union-of-bounding-boxes crop over multiple targets.
 * - Falls back to a clipped viewport around the first target.
 * - Falls back to an element-only screenshot.
 * - Finally falls back to a small viewport screenshot.
 */
export class AxeScreenshotService {
	private readonly cfg: AxeScreenshotConfig;

	constructor(config?: Partial<AxeScreenshotConfig>) {
		this.cfg = loadAxeScreenshotConfig(config);
	}

	/**
	 * Capture a contextual screenshot highlighting the violation targets.
	 *
	 * Returns a typed capture outcome: captured, skipped, or failed with reason.
	 */
	async captureViolationScreenshot(
		page: Page,
		violation: AxeViolation,
		resultsDir: string,
	): Promise<ViolationScreenshotCaptureResult> {
		return captureViolationScreenshot({
			page,
			violation,
			resultsDir,
			cfg: this.cfg,
		});
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
		options?: { scannerId?: string },
	): Promise<PageOverviewResult | null> {
		const scannerId = options?.scannerId ?? "axe";
		const overviewCfg = loadPageOverviewConfig();
		const captured = await capturePageOverviewRaw(
			page,
			violations,
			resultsDir,
			pageId,
			scannerId,
			this.cfg,
			overviewCfg,
		);

		if (!captured) {
			return null;
		}

		await saveScreenshot(captured.buffer, captured.screenshotPath, this.cfg);

		return {
			screenshotPath: captured.screenshotPath,
			screenshotFilename: captured.screenshotFilename,
			pageWidth: captured.pageWidth,
			pageHeight: captured.pageHeight,
			elements: captured.elements,
		};
	}
}
