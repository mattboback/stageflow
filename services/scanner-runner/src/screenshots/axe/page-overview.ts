import type { Page } from 'playwright';

import { issueFingerprint } from '../../core/finding-identity';
import { existsSync, mkdirSync } from 'node:fs';
import { join } from 'node:path';
import sharp from 'sharp';

import type { IssueSeverity } from '../../core/types';
import type {
	AxeScreenshotConfig,
	PageOverviewElement,
	PageOverviewResult,
	PageOverviewViolation
} from './types';
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
   existing importers (and test mocks of this module) keep working. Only what is
   actually imported through this module is re-exported — PageOverviewConfig and
   BoundingBox are consumed from their own modules, and re-exporting them here as
   well left two exports that nothing resolved. */
export { loadPageOverviewConfig } from './page-overview-config';
export type { PageOverviewConfigInput } from './page-overview-config';

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

	const maxNodesPerViolation = 5; // Rich screenshot evidence stays bounded independently of occurrence IDs.

	// Every finding gets its first located node before any finding gets a
	// second one, so a cap never leaves a finding without an element to crop.
	const locatedPerViolation = violations.map((violation) => {
		const ruleId = violation.id || 'unknown';
		const severity = normalizeSeverity(violation.impact, 'minor');
		const nodes = (violation.nodes ?? []).slice(0, maxNodesPerViolation);

		const primarySelectorRaw = nodes[0]?.target?.[0];
		const primarySelector = typeof primarySelectorRaw === 'string' ? primarySelectorRaw : '';

		// WebServerFormatter generates issue IDs using the FIRST node selector/target.
		// Keep all overlay elements for the same rule mapped to that single issue.
		const issueId = issueFingerprint(scanner, ruleId, pageId, primarySelector);

		return nodes.flatMap((node, nodeIndex) => {
			const selector = (node.target ?? [])
				.map((t) => (typeof t === 'string' ? t.trim() : String(t).trim()))
				.find((t) => t.length > 0);

			return selector ? [{ issueId, ruleId, severity, selector, nodeIndex }] : [];
		});
	});

	for (let round = 0; round < maxNodesPerViolation; round++) {
		for (const located of locatedPerViolation) {
			const target = located[round];
			if (!target) {
				continue;
			}

			if (targets.length >= maxElements) {
				return targets;
			}

			targets.push(target);
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

	try {
		// Step 1: Realize lazy/revealed page content before measurements and capture.
		await preparePageForOverview(page, resolvedOverviewCfg);
		await expandScrollableContainersForOverview(page);

		// Step 2: Stabilize moving parts without forcing reveal animations back to hidden states.
		await stabilizePageForScreenshot(page, resolvedOverviewCfg);
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

		// Step 4: CRITICAL - Collect ALL bounding boxes BEFORE taking screenshot
		// This prevents layout drift between screenshot capture and coordinate collection
		const rawBoundingBoxes = new Map<number, BoundingBox | null>();

		for (const [i, target] of elementTargets.entries()) {
			try {
				const locator = page.locator(target.selector).first();
				rawBoundingBoxes.set(i, await locator.boundingBox());
			} catch {
				rawBoundingBoxes.set(i, null);
			}
		}

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

		// Step 7: Convert pre-captured bounding boxes to screenshot coordinates
		const elementsWithBounds: PageOverviewElement[] = [];

		for (const [i, target] of elementTargets.entries()) {
			const box = rawBoundingBoxes.get(i);

			if (!box || box.y > captureHeight) {
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
				continue;
			}

			const xPercent = clampPercent((clippedBox.x / screenshotWidth) * 100);
			const yPercent = clampPercent((clippedBox.y / screenshotHeight) * 100);
			const widthPercent = clampPercent((clippedBox.width / screenshotWidth) * 100);
			const heightPercent = clampPercent((clippedBox.height / screenshotHeight) * 100);

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

		return {
			screenshotPath,
			screenshotFilename,
			pageWidth: screenshotWidth,
			pageHeight: screenshotHeight,
			elements: elementsWithBounds,
			buffer: Buffer.from(screenshotBuffer)
		};
	} catch {
		return null;
	}
}
