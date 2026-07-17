import type { Page } from 'playwright';

import { createHash } from 'node:crypto';
import { existsSync, mkdirSync } from 'node:fs';
import { join } from 'node:path';
import sharp from 'sharp';

import type { IssueSeverity } from '../../core/types';
import type {
	AxeScreenshotConfig,
	PageOverviewDiagnostics,
	PageOverviewElement,
	PageOverviewResult,
	PageOverviewViolation
} from './types';
import { getEnvBool } from '../../utils/env';
import { createLogger } from '../../utils/logger';
import { normalizeSeverity } from '../../utils/severity';
import { loadPageOverviewConfig, type PageOverviewConfigInput } from './page-overview-config';
import {
	clampPercent,
	clipPageOverviewBounds,
	computeScreenshotScaleFactor,
	roundPercent,
	type BoundingBox
} from './page-overview-geometry';
import {
	expandScrollableContainersForOverview,
	preparePageForOverview,
	stabilizePageForScreenshot,
	waitForStablePaint,
	waitInPage
} from './page-preparation';

/* Sibling modules carry the extracted stages; re-export the shared surface so
   existing importers (and test mocks of this module) keep working. */
export { loadPageOverviewConfig } from './page-overview-config';
export type { PageOverviewConfig, PageOverviewConfigInput } from './page-overview-config';
export { clipPageOverviewBounds, computeScreenshotScaleFactor } from './page-overview-geometry';
export type { BoundingBox } from './page-overview-geometry';

const DEBUG_SCREENSHOTS = getEnvBool('A11Y_DEBUG_SCREENSHOTS', false);
const log = createLogger('PageOverview');

function debugLog(message: string, data?: unknown): void {
	if (!DEBUG_SCREENSHOTS) {
		return;
	}
	if (data !== undefined) {
		log.info(message, { data });
	} else {
		log.info(message);
	}
}

/**
 * Generate the same issue fingerprint used by WebServerFormatter.
 * This ensures overlay element IDs match the final report issue IDs.
 */
function generateIssueFingerprint(
	scanner: string,
	ruleId: string,
	pageId: string,
	selector: string
): string {
	const data = `${scanner}|${ruleId}|${pageId}|${selector}`.toLowerCase();
	return createHash('sha256').update(data).digest('hex').slice(0, 12);
}

export function collectPageOverviewTargets(
	violations: PageOverviewViolation[],
	maxElements: number,
	options?: { pageId?: string; scanner?: string }
): {
	issueId: string;
	ruleId: string;
	severity: IssueSeverity;
	selector: string;
	nodeIndex: number;
}[] {
	const { pageId = 'page', scanner = 'axe' } = options ?? {};
	const targets: {
		issueId: string;
		ruleId: string;
		severity: IssueSeverity;
		selector: string;
		nodeIndex: number;
	}[] = [];
	if (violations.length === 0 || maxElements <= 0) {
		return targets;
	}

	const maxNodesPerViolation = 5; // Keep in sync with WebServerFormatter occurrences slice.
	let elementCount = 0;

	for (const violation of violations) {
		const ruleId = violation.id || 'unknown';
		const severity = normalizeSeverity(violation.impact, 'minor');
		const nodes = violation.nodes ?? [];

		const primarySelectorRaw = nodes[0]?.target?.[0];
		const primarySelector = typeof primarySelectorRaw === 'string' ? primarySelectorRaw : '';

		// WebServerFormatter generates issue IDs using the FIRST node selector/target.
		// Keep all overlay elements for the same rule mapped to that single issue.
		const issueId = generateIssueFingerprint(scanner, ruleId, pageId, primarySelector);

		for (
			let nodeIndex = 0;
			nodeIndex < nodes.length && nodeIndex < maxNodesPerViolation;
			nodeIndex++
		) {
			if (elementCount >= maxElements) {
				return targets;
			}

			const node = nodes[nodeIndex];
			const selector = (node?.target ?? [])
				.map((t) => (typeof t === 'string' ? t.trim() : String(t).trim()))
				.find((t) => t.length > 0);

			if (!selector) {
				continue;
			}

			targets.push({
				issueId,
				ruleId,
				severity,
				selector,
				nodeIndex
			});
			elementCount += 1;
		}
	}

	return targets;
}

export async function capturePageOverviewRaw(
	page: Page,
	violations: PageOverviewViolation[],
	resultsDir: string,
	pageId: string,
	scannerId: string,
	screenshotCfg: AxeScreenshotConfig,
	overviewCfg: PageOverviewConfigInput
): Promise<(PageOverviewResult & { buffer: Buffer }) | null> {
	const resolvedOverviewCfg = loadPageOverviewConfig(overviewCfg);
	if (!resolvedOverviewCfg.enabled || !screenshotCfg.screenshotsEnabled) {
		return null;
	}

	const elementTargets = collectPageOverviewTargets(violations, resolvedOverviewCfg.maxElements, {
		pageId,
		scanner: scannerId
	});

	if (!existsSync(resultsDir)) {
		mkdirSync(resultsDir, { recursive: true });
	}

	const ext = screenshotCfg.outputFormat === 'webp' ? '.webp' : '.png';
	const screenshotFilename = `page-overview-${pageId}${ext}`;
	const screenshotPath = join(resultsDir, screenshotFilename);

	// Initialize diagnostics collection
	const diagnostics: PageOverviewDiagnostics = {
		animationFinishedCount: 0,
		animationPausedCount: 0,
		assetWaitStatus: 'skipped',
		contentVisibilityElementCount: 0,
		contentVisibilityForced: false,
		cssPageWidth: 0,
		cssPageHeight: 0,
		fontWaitStatus: 'not_supported',
		lazyMediaCount: 0,
		screenshotWidth: 0,
		screenshotHeight: 0,
		scaleX: 1,
		scaleY: 1,
		devicePixelRatio: 1,
		captureHeight: 0,
		elementCount: elementTargets.length,
		preScrollAfterHeight: 0,
		preScrollBeforeHeight: 0,
		preScrollCompleted: false,
		preScrollSteps: 0,
		wasClipped: false,
		elements: []
	};

	try {
		// Step 1: Realize lazy/revealed page content before measurements and capture.
		const preparation = await preparePageForOverview(page, resolvedOverviewCfg);
		diagnostics.assetWaitStatus = preparation.assetWaitStatus;
		diagnostics.contentVisibilityElementCount = preparation.contentVisibilityElementCount;
		diagnostics.contentVisibilityForced = preparation.contentVisibilityForced;
		diagnostics.fontWaitStatus = preparation.fontWaitStatus;
		diagnostics.lazyMediaCount = preparation.lazyMediaCount;
		diagnostics.preScrollAfterHeight = preparation.preScrollAfterHeight;
		diagnostics.preScrollBeforeHeight = preparation.preScrollBeforeHeight;
		diagnostics.preScrollCompleted = preparation.preScrollCompleted;
		diagnostics.preScrollSteps = preparation.preScrollSteps;

		const scrollExpansion = await expandScrollableContainersForOverview(page);
		debugLog('Expanded scrollable containers before page overview capture', scrollExpansion);

		// Step 2: Stabilize moving parts without forcing reveal animations back to hidden states.
		const stabilization = await stabilizePageForScreenshot(page, resolvedOverviewCfg);
		diagnostics.animationFinishedCount = stabilization.animationFinishedCount;
		diagnostics.animationPausedCount = stabilization.animationPausedCount;
		await waitInPage(page, 50);
		await waitForStablePaint(page);

		// Step 3: Get page dimensions and devicePixelRatio
		const pageInfo = await page.evaluate(() => {
			const doc = document.documentElement;
			const body = document.body;
			return {
				width: Math.max(doc.scrollWidth, doc.clientWidth, body.scrollWidth, body.clientWidth),
				height: Math.max(doc.scrollHeight, doc.clientHeight, body.scrollHeight, body.clientHeight),
				devicePixelRatio: window.devicePixelRatio || 1
			};
		});

		const captureHeight =
			resolvedOverviewCfg.maxHeight > 0
				? Math.min(pageInfo.height, resolvedOverviewCfg.maxHeight)
				: pageInfo.height;
		const wasClipped = resolvedOverviewCfg.maxHeight > 0 && captureHeight < pageInfo.height;

		diagnostics.cssPageWidth = pageInfo.width;
		diagnostics.cssPageHeight = pageInfo.height;
		diagnostics.captureHeight = captureHeight;
		diagnostics.devicePixelRatio = pageInfo.devicePixelRatio;
		diagnostics.wasClipped = wasClipped;

		debugLog('Page dimensions', {
			cssWidth: pageInfo.width,
			cssHeight: pageInfo.height,
			captureHeight,
			devicePixelRatio: pageInfo.devicePixelRatio,
			wasClipped
		});

		// Step 4: CRITICAL - Collect ALL bounding boxes BEFORE taking screenshot
		// This prevents layout drift between screenshot capture and coordinate collection
		const rawBoundingBoxes = new Map<number, BoundingBox | null>();

		for (const [i, target] of elementTargets.entries()) {
			try {
				const locator = page.locator(target.selector).first();
				const box = await locator.boundingBox();
				rawBoundingBoxes.set(i, box);

				if (DEBUG_SCREENSHOTS) {
					diagnostics.elements.push({
						selector: target.selector,
						rawBox: box ? { x: box.x, y: box.y, width: box.width, height: box.height } : null,
						scaledBox: null,
						percentBounds: null,
						skipped: false
					});
				}
			} catch {
				rawBoundingBoxes.set(i, null);
				if (DEBUG_SCREENSHOTS) {
					diagnostics.elements.push({
						selector: target.selector,
						rawBox: null,
						scaledBox: null,
						percentBounds: null,
						skipped: true,
						skipReason: 'locator_error'
					});
				}
			}
		}

		debugLog(`Collected ${rawBoundingBoxes.size} bounding boxes BEFORE screenshot`);

		// Step 5: Take the screenshot (after bounding boxes are captured)
		const clip = wasClipped
			? { x: 0, y: 0, width: pageInfo.width, height: captureHeight }
			: undefined;
		const screenshotBuffer = await page.screenshot({
			fullPage: true,
			...(clip !== undefined ? { clip } : {})
		});

		// Step 6: Get screenshot actual dimensions
		const metadata = await sharp(screenshotBuffer).metadata();
		const screenshotWidth = metadata.width || pageInfo.width;
		const screenshotHeight = metadata.height || captureHeight;

		const scaleX = computeScreenshotScaleFactor(screenshotWidth, pageInfo.width);
		const scaleY = computeScreenshotScaleFactor(screenshotHeight, captureHeight);

		diagnostics.screenshotWidth = screenshotWidth;
		diagnostics.screenshotHeight = screenshotHeight;
		diagnostics.scaleX = scaleX;
		diagnostics.scaleY = scaleY;

		debugLog('Screenshot captured', {
			screenshotWidth,
			screenshotHeight,
			scaleX,
			scaleY,
			expectedScaleFromDPR: pageInfo.devicePixelRatio
		});

		// Step 7: Convert pre-captured bounding boxes to screenshot coordinates
		const elementsWithBounds: PageOverviewElement[] = [];

		for (const [i, target] of elementTargets.entries()) {
			const box = rawBoundingBoxes.get(i);

			if (!box) {
				const diagnosticElement = diagnostics.elements[i];
				if (DEBUG_SCREENSHOTS && diagnosticElement) {
					diagnosticElement.skipped = true;
					diagnosticElement.skipReason = diagnosticElement.skipReason ?? 'no_box';
				}
				continue;
			}

			if (box.y > captureHeight) {
				const diagnosticElement = diagnostics.elements[i];
				if (DEBUG_SCREENSHOTS && diagnosticElement) {
					diagnosticElement.skipped = true;
					diagnosticElement.skipReason = 'below_capture_area';
				}
				continue;
			}

			// Convert CSS pixel bounding boxes to screenshot pixel coordinates
			const scaledBox = {
				x: box.x * scaleX,
				y: box.y * scaleY,
				width: box.width * scaleX,
				height: box.height * scaleY
			};

			const clippedBox = clipPageOverviewBounds(scaledBox, screenshotWidth, screenshotHeight);

			if (!clippedBox) {
				const diagnosticElement = diagnostics.elements[i];
				if (DEBUG_SCREENSHOTS && diagnosticElement) {
					diagnosticElement.scaledBox = scaledBox;
					diagnosticElement.skipped = true;
					diagnosticElement.skipReason = 'clipped_out';
				}
				continue;
			}

			const xPercent = clampPercent((clippedBox.x / screenshotWidth) * 100);
			const yPercent = clampPercent((clippedBox.y / screenshotHeight) * 100);
			const widthPercent = clampPercent((clippedBox.width / screenshotWidth) * 100);
			const heightPercent = clampPercent((clippedBox.height / screenshotHeight) * 100);

			{
				const diagnosticElement = diagnostics.elements[i];
				if (DEBUG_SCREENSHOTS && diagnosticElement) {
					diagnosticElement.scaledBox = clippedBox;
					diagnosticElement.percentBounds = {
						x: roundPercent(xPercent),
						y: roundPercent(yPercent),
						width: roundPercent(widthPercent),
						height: roundPercent(heightPercent)
					};
				}
			}

			elementsWithBounds.push({
				issueId: target.issueId,
				ruleId: target.ruleId,
				severity: target.severity,
				selector: target.selector,
				nodeIndex: target.nodeIndex,
				xPercent: roundPercent(xPercent),
				yPercent: roundPercent(yPercent),
				widthPercent: roundPercent(widthPercent),
				heightPercent: roundPercent(heightPercent),
				x: Math.max(0, Math.round(clippedBox.x)),
				y: Math.max(0, Math.round(clippedBox.y)),
				width: Math.max(0, Math.round(clippedBox.width)),
				height: Math.max(0, Math.round(clippedBox.height))
			});
		}

		debugLog(`Final elements with bounds: ${elementsWithBounds.length}/${elementTargets.length}`);

		if (DEBUG_SCREENSHOTS) {
			debugLog('Full diagnostics', diagnostics);
		}

		return {
			screenshotPath,
			screenshotFilename,
			pageWidth: screenshotWidth,
			pageHeight: screenshotHeight,
			elements: elementsWithBounds,
			buffer: Buffer.from(screenshotBuffer)
		};
	} catch (error) {
		debugLog('Capture failed', { error: String(error) });
		return null;
	}
}
