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

import { getEnvBool, getEnvInt } from '../../utils/env';
import { createLogger } from '../../utils/logger';

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
import { normalizeSeverity } from '../../utils/severity';

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

export function loadPageOverviewConfig(
	overrides?: Partial<{
		enabled: boolean;
		finishAnimations: boolean;
		forceContentVisibility: boolean;
		maxElements: number;
		maxHeight: number;
		maxScrollSteps: number;
		preScroll: boolean;
		scrollPauseMs: number;
		scrollSettleMs: number;
		scrollStepPx: number;
	}>
): {
	enabled: boolean;
	finishAnimations: boolean;
	forceContentVisibility: boolean;
	maxElements: number;
	maxHeight: number;
	maxScrollSteps: number;
	preScroll: boolean;
	scrollPauseMs: number;
	scrollSettleMs: number;
	scrollStepPx: number;
} {
	return {
		enabled: getEnvBool('A11Y_PAGE_OVERVIEW_ENABLED', true),
		finishAnimations: getEnvBool('A11Y_PAGE_OVERVIEW_FINISH_ANIMATIONS', true),
		forceContentVisibility: getEnvBool('A11Y_PAGE_OVERVIEW_FORCE_CONTENT_VISIBILITY', true),
		maxElements: Math.max(0, getEnvInt('A11Y_PAGE_OVERVIEW_MAX_ELEMENTS', 50)),
		maxHeight: Math.max(0, getEnvInt('A11Y_PAGE_OVERVIEW_MAX_HEIGHT', 0)),
		maxScrollSteps: Math.max(1, getEnvInt('A11Y_PAGE_OVERVIEW_MAX_SCROLL_STEPS', 80)),
		preScroll: getEnvBool('A11Y_PAGE_OVERVIEW_PRE_SCROLL', true),
		scrollPauseMs: Math.max(0, getEnvInt('A11Y_PAGE_OVERVIEW_SCROLL_PAUSE_MS', 75)),
		scrollSettleMs: Math.max(0, getEnvInt('A11Y_PAGE_OVERVIEW_SCROLL_SETTLE_MS', 150)),
		scrollStepPx: Math.max(1, getEnvInt('A11Y_PAGE_OVERVIEW_SCROLL_STEP_PX', 800)),
		...overrides
	};
}

interface PageOverviewConfig {
	enabled: boolean;
	finishAnimations: boolean;
	forceContentVisibility: boolean;
	maxElements: number;
	maxHeight: number;
	maxScrollSteps: number;
	preScroll: boolean;
	scrollPauseMs: number;
	scrollSettleMs: number;
	scrollStepPx: number;
	skipLargeElements?: boolean;
	largeElementWidthRatio?: number;
	largeElementHeightRatio?: number;
}

type PageOverviewConfigInput = Partial<PageOverviewConfig> &
	Pick<PageOverviewConfig, 'enabled' | 'maxElements' | 'maxHeight'>;

interface PagePreparationResult {
	assetWaitStatus: 'completed' | 'failed' | 'skipped' | 'timed_out';
	contentVisibilityForced: boolean;
	contentVisibilityElementCount: number;
	fontWaitStatus: 'completed' | 'failed' | 'not_supported' | 'timed_out';
	lazyMediaCount: number;
	preScrollAfterHeight: number;
	preScrollBeforeHeight: number;
	preScrollCompleted: boolean;
	preScrollSteps: number;
}

interface ScrollContainerExpansionResult {
	expandedCount: number;
	maxScrollHeight: number;
}

interface AnimationStabilizationResult {
	animationFinishedCount: number;
	animationPausedCount: number;
}

async function waitInPage(page: Page, timeoutMs: number): Promise<void> {
	if (timeoutMs <= 0) {
		return;
	}
	await page.waitForTimeout(timeoutMs);
}

async function preparePageForOverview(
	page: Page,
	config: PageOverviewConfig
): Promise<PagePreparationResult> {
	const result: PagePreparationResult = {
		assetWaitStatus: 'skipped',
		contentVisibilityForced: false,
		contentVisibilityElementCount: 0,
		fontWaitStatus: 'not_supported',
		lazyMediaCount: 0,
		preScrollAfterHeight: 0,
		preScrollBeforeHeight: 0,
		preScrollCompleted: false,
		preScrollSteps: 0
	};

	try {
		await page.emulateMedia({ reducedMotion: 'reduce' });
	} catch {
		// Best effort only; older browser contexts may reject media overrides.
	}

	try {
		await page.addStyleTag({
			content: `
				html,
				body,
				* {
					scroll-behavior: auto !important;
				}
			`
		});
	} catch {
		// Ignore CSP/style injection failures.
	}

	try {
		const mediaResult = await page.evaluate(() => {
			let lazyMediaCount = 0;
			for (const element of Array.from(
				document.querySelectorAll('img[loading], iframe[loading]')
			)) {
				const loading = element.getAttribute('loading');
				if (loading?.toLowerCase() === 'lazy') {
					lazyMediaCount += 1;
				}
				element.setAttribute('loading', 'eager');
			}

			return { lazyMediaCount };
		});
		result.lazyMediaCount = mediaResult.lazyMediaCount;
	} catch {
		// Ignore evaluate failures (detached page, navigation races, etc).
	}

	if (config.forceContentVisibility) {
		try {
			result.contentVisibilityElementCount = await page.evaluate(() => {
				let count = 0;
				for (const element of Array.from(document.querySelectorAll('*'))) {
					const style = window.getComputedStyle(element);
					if (style.contentVisibility === 'auto' || style.contentVisibility === 'hidden') {
						count += 1;
					}
				}
				return count;
			});

			await page.addStyleTag({
				content: `
					* {
						content-visibility: visible !important;
					}
				`
			});
			result.contentVisibilityForced = true;
		} catch {
			result.contentVisibilityForced = false;
		}
	}

	try {
		result.fontWaitStatus = await page.evaluate(async () => {
			if (!('fonts' in document)) {
				return 'not_supported' as const;
			}

			try {
				await Promise.race([
					document.fonts.ready,
					new Promise<'timed_out'>((resolve) => {
						window.setTimeout(() => {
							resolve('timed_out');
						}, 2_000);
					})
				]);
				return document.fonts.status === 'loaded' ? 'completed' : 'timed_out';
			} catch {
				return 'failed' as const;
			}
		});
	} catch {
		result.fontWaitStatus = 'failed';
	}

	try {
		result.assetWaitStatus = await page.evaluate(async () => {
			const images = Array.from(document.images);
			if (images.length === 0) {
				return 'skipped' as const;
			}

			const waitForImage = (image: HTMLImageElement): Promise<void> => {
				if (image.complete) {
					return Promise.resolve();
				}
				return new Promise<void>((resolve) => {
					const done = () => {
						resolve();
					};
					image.addEventListener('load', done, { once: true });
					image.addEventListener('error', done, { once: true });
				});
			};

			try {
				const status = await Promise.race([
					Promise.all(images.map(waitForImage)).then(() => 'completed' as const),
					new Promise<'timed_out'>((resolve) => {
						window.setTimeout(() => {
							resolve('timed_out');
						}, 2_000);
					})
				]);
				return status;
			} catch {
				return 'failed' as const;
			}
		});
	} catch {
		result.assetWaitStatus = 'failed';
	}

	const pageHeight = async (): Promise<number> =>
		page.evaluate(() => {
			const doc = document.documentElement;
			const body = document.body;
			return Math.max(doc.scrollHeight, doc.clientHeight, body.scrollHeight, body.clientHeight);
		});

	try {
		result.preScrollBeforeHeight = await pageHeight();
	} catch {
		result.preScrollBeforeHeight = 0;
	}

	if (config.preScroll) {
		try {
			await page.evaluate(() => {
				window.scrollTo(0, 0);
			});
			await waitInPage(page, config.scrollPauseMs);

			let y = 0;
			for (let step = 0; step < config.maxScrollSteps; step++) {
				const height = await pageHeight();
				const viewportHeight = await page.evaluate(() => window.innerHeight || 0);
				const maxY = Math.max(0, height - viewportHeight);
				const nextY = Math.min(maxY, y + config.scrollStepPx);

				await page.evaluate((scrollY) => {
					window.scrollTo(0, scrollY);
				}, nextY);
				result.preScrollSteps += 1;
				await waitInPage(page, config.scrollPauseMs);

				if (nextY >= maxY) {
					const grownHeight = await pageHeight();
					if (grownHeight <= height || nextY >= Math.max(0, grownHeight - viewportHeight)) {
						break;
					}
				}

				if (nextY === y && maxY === 0) {
					break;
				}
				y = nextY;
			}
			result.preScrollCompleted = true;
		} catch {
			result.preScrollCompleted = false;
		}
	}

	try {
		await page.evaluate(() => {
			window.scrollTo(0, 0);
		});
		await waitInPage(page, config.scrollSettleMs);
		result.preScrollAfterHeight = await pageHeight();
	} catch {
		result.preScrollAfterHeight = result.preScrollBeforeHeight;
	}

	return result;
}

async function expandScrollableContainersForOverview(
	page: Page
): Promise<ScrollContainerExpansionResult> {
	try {
		return await page.evaluate(() => {
			const MIN_OVERFLOW_PX = 24;
			const MIN_CONTAINER_HEIGHT_PX = 80;
			const elements = Array.from(document.querySelectorAll<HTMLElement>('body *'));
			let expandedCount = 0;
			let maxScrollHeight = 0;

			const relaxContainer = (element: HTMLElement, minHeight?: number): void => {
				element.style.setProperty('max-height', 'none', 'important');
				element.style.setProperty('overflow', 'visible', 'important');
				element.style.setProperty('overflow-y', 'visible', 'important');
				if (minHeight !== undefined && minHeight > 0) {
					element.style.setProperty('min-height', `${Math.ceil(minHeight)}px`, 'important');
				}
			};

			document.documentElement.style.setProperty('height', 'auto', 'important');
			relaxContainer(document.documentElement);
			document.body.style.setProperty('height', 'auto', 'important');
			relaxContainer(document.body);

			for (const element of elements) {
				const rect = element.getBoundingClientRect();
				if (rect.width <= 0 || rect.height <= 0) {
					continue;
				}

				const overflowAmount = element.scrollHeight - element.clientHeight;
				if (
					overflowAmount < MIN_OVERFLOW_PX ||
					element.clientHeight < MIN_CONTAINER_HEIGHT_PX ||
					element.scrollHeight <= element.clientHeight
				) {
					continue;
				}

				const style = window.getComputedStyle(element);
				const overflowY = style.overflowY.toLowerCase();
				if (!['auto', 'scroll', 'hidden', 'clip'].includes(overflowY)) {
					continue;
				}

				element.dataset.stageflowOverviewExpanded = 'true';
				element.style.setProperty('height', `${element.scrollHeight}px`, 'important');
				relaxContainer(element, element.scrollHeight);

				let ancestor = element.parentElement;
				while (ancestor && ancestor !== document.body) {
					ancestor.style.setProperty('height', 'auto', 'important');
					relaxContainer(ancestor, Math.max(ancestor.scrollHeight, element.scrollHeight));
					ancestor = ancestor.parentElement;
				}

				maxScrollHeight = Math.max(maxScrollHeight, element.scrollHeight);
				expandedCount += 1;
			}

			return { expandedCount, maxScrollHeight };
		});
	} catch {
		return { expandedCount: 0, maxScrollHeight: 0 };
	}
}

async function stabilizePageForScreenshot(
	page: Page,
	config: PageOverviewConfig
): Promise<AnimationStabilizationResult> {
	const result: AnimationStabilizationResult = {
		animationFinishedCount: 0,
		animationPausedCount: 0
	};

	try {
		await page.addStyleTag({
			content: `
				*,
				*::before,
				*::after {
					transition-delay: 0s !important;
					transition-duration: 0s !important;
					scroll-behavior: auto !important;
				}
			`
		});
	} catch {
		// Ignore CSP/style injection failures.
	}

	if (!config.finishAnimations) {
		return result;
	}

	try {
		return await page.evaluate(() => {
			let animationFinishedCount = 0;
			let animationPausedCount = 0;

			try {
				for (const animation of document.getAnimations()) {
					const timing = animation.effect?.getTiming();
					const duration = typeof timing?.duration === 'number' ? timing.duration : Number.NaN;
					const iterations =
						typeof timing?.iterations === 'number' ? timing.iterations : Number.NaN;
					const isFiniteAnimation =
						Number.isFinite(duration) &&
						Number.isFinite(iterations) &&
						iterations > 0 &&
						duration >= 0;

					if (isFiniteAnimation) {
						try {
							animation.finish();
							animationFinishedCount += 1;
						} catch {
							try {
								animation.pause();
								animationPausedCount += 1;
							} catch {
								// ignore animations that cannot be controlled
							}
						}
					} else {
						try {
							animation.pause();
							animationPausedCount += 1;
						} catch {
							// ignore animations that cannot be controlled
						}
					}
				}
			} catch {
				// ignore unsupported animation APIs
			}

			return { animationFinishedCount, animationPausedCount };
		});
	} catch {
		return result;
	}
}

export function computeScreenshotScaleFactor(actualPixels: number, cssPixels: number): number {
	if (actualPixels <= 0 || cssPixels <= 0) {
		return 1;
	}

	const scale = actualPixels / cssPixels;
	if (!Number.isFinite(scale) || scale <= 0) {
		return 1;
	}

	return scale;
}

interface BoundingBox {
	x: number;
	y: number;
	width: number;
	height: number;
}

export function clipPageOverviewBounds(
	bounds: BoundingBox,
	maxWidth: number,
	maxHeight: number
): BoundingBox | null {
	if (maxWidth <= 0 || maxHeight <= 0) {
		return null;
	}

	if (
		!Number.isFinite(bounds.x) ||
		!Number.isFinite(bounds.y) ||
		!Number.isFinite(bounds.width) ||
		!Number.isFinite(bounds.height)
	) {
		return null;
	}

	if (bounds.width <= 0 || bounds.height <= 0) {
		return null;
	}

	const right = bounds.x + bounds.width;
	const bottom = bounds.y + bounds.height;

	if (right <= 0 || bottom <= 0) {
		return null;
	}

	if (bounds.x >= maxWidth || bounds.y >= maxHeight) {
		return null;
	}

	const clampedX = Math.max(0, bounds.x);
	const clampedY = Math.max(0, bounds.y);
	const clampedRight = Math.min(maxWidth, right);
	const clampedBottom = Math.min(maxHeight, bottom);

	const width = clampedRight - clampedX;
	const height = clampedBottom - clampedY;

	if (width <= 0 || height <= 0) {
		return null;
	}

	return { x: clampedX, y: clampedY, width, height };
}

function clampPercent(value: number): number {
	if (!Number.isFinite(value)) {
		return 0;
	}

	return Math.min(100, Math.max(0, value));
}

function roundPercent(value: number): number {
	return Math.round(value * 100) / 100;
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
