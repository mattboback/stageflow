import type { Page } from 'playwright';

import { existsSync, mkdirSync } from 'node:fs';
import { join } from 'node:path';
import { v4 as uuidv4 } from 'uuid';

import type {
	AxeScreenshotConfig,
	AxeViolation,
	ElementBounds,
	EnhancedScreenshotResult,
	FriendlyNodeInfo,
	PageLocationInfo,
	ViolationCaptureFailure,
	ViolationCaptureFailureReason,
	ViolationCaptureStep,
	ViolationScreenshotCaptureResult
} from './types';

import { getScreenshotPolicy } from '../../config/rule-behaviors';
import { computeElementClip, computeSingleTargetClip, computeUnionClip } from './clip';
import { buildFriendlyNodeInfo } from './friendly-node';
import { injectHighlightCSS, removeHighlightCSS } from './highlight-css';
import { compositeOverlay, generateThumbnail, saveScreenshot } from './image';
import { captureLocationInfo } from './location';
import { captureSemanticOverlayScreenshot } from './semantic-overlay';
import { extractAxeViolationTargets } from './targets';

export async function captureViolationScreenshot(input: {
	page: Page;
	violation: AxeViolation;
	resultsDir: string;
	cfg: AxeScreenshotConfig;
}): Promise<ViolationScreenshotCaptureResult> {
	if (!input.cfg.screenshotsEnabled) {
		return { status: 'skipped', reason: 'disabled' };
	}

	const policy = getScreenshotPolicy(input.violation.id);
	if (policy === 'semantic') {
		try {
			const semanticResult = await captureSemanticOverlayScreenshot(input);
			if (!semanticResult) {
				return {
					status: 'failed',
					failure: buildFailure(
						'semantic',
						'screenshot_failed',
						'semantic overlay strategy returned no screenshot'
					),
					fallbacks: []
				};
			}

			return {
				status: 'captured',
				screenshot: semanticResult,
				fallbacks: []
			};
		} catch (error) {
			return {
				status: 'failed',
				failure: buildFailure('semantic', 'screenshot_failed', error),
				fallbacks: []
			};
		}
	}

	if (policy === 'never') {
		return { status: 'skipped', reason: 'policy_never' };
	}

	const { targets, selector } = extractAxeViolationTargets(input.violation);

	const ext = input.cfg.outputFormat === 'webp' ? '.webp' : '.png';
	const screenshotFilename = `violation-${input.violation.id ?? 'unknown'}-${uuidv4()}${ext}`;
	const screenshotPath = join(input.resultsDir, screenshotFilename);

	if (!existsSync(input.resultsDir)) {
		mkdirSync(input.resultsDir, { recursive: true });
	}

	const viewport = input.page.viewportSize() ?? { width: 1280, height: 720 };
	const vw = viewport.width;
	const vh = viewport.height;

	const useCSSInjection = input.cfg.overlayMethod === 'css-injection';
	const fallbacks: ViolationCaptureFailure[] = [];

	let locationInfo: PageLocationInfo | undefined;
	let friendlyNode: FriendlyNodeInfo | undefined;
	const buildCaptureContext = () => ({
		screenshotPath,
		screenshotFilename,
		resultsDir: input.resultsDir,
		cfg: input.cfg,
		...(locationInfo !== undefined ? { locationInfo } : {}),
		...(friendlyNode !== undefined ? { friendlyNode } : {})
	});

	try {
		if (useCSSInjection && input.cfg.injectHighlight && targets.length > 0) {
			try {
				await injectHighlightCSS(input.page, targets, input.cfg);
			} catch (error) {
				fallbacks.push(buildFailure('union', 'overlay_failed', error));
			}
		}

		try {
			locationInfo = await captureLocationInfo(input.page);
		} catch {
			locationInfo = undefined;
		}

		if (selector) {
			try {
				friendlyNode = await buildFriendlyNodeInfo(input.page, selector);
			} catch {
				friendlyNode = undefined;
			}
		}

		if (input.cfg.mergeTargets && targets.length > 0) {
			const unionAttempt = await tryUnionBoundingBoxScreenshot({
				page: input.page,
				targets,
				vw,
				vh,
				screenshotPath,
				cfg: input.cfg,
				returnBuffer: !useCSSInjection
			});
			const unionResult = await finalizeStrategyCapture({
				step: 'union',
				useCSSInjection,
				attempt: unionAttempt,
				...buildCaptureContext()
			});
			if (unionResult.status === 'captured') {
				return {
					status: 'captured',
					screenshot: unionResult.screenshot,
					fallbacks
				};
			}

			fallbacks.push(unionResult.failure);
		}

		if (selector) {
			const singleAttempt = await trySingleTargetViewportScreenshot({
				page: input.page,
				selector,
				vw,
				vh,
				screenshotPath,
				cfg: input.cfg,
				returnBuffer: !useCSSInjection && input.cfg.injectHighlight
			});
			const singleResult = await finalizeStrategyCapture({
				step: 'single-target',
				useCSSInjection,
				attempt: singleAttempt,
				...buildCaptureContext()
			});
			if (singleResult.status === 'captured') {
				return {
					status: 'captured',
					screenshot: singleResult.screenshot,
					fallbacks
				};
			}

			fallbacks.push(singleResult.failure);

			const elementAttempt = await tryElementScreenshot({
				page: input.page,
				selector,
				vw,
				vh,
				screenshotPath,
				cfg: input.cfg,
				returnBuffer: !useCSSInjection && input.cfg.injectHighlight
			});
			const elementResult = await finalizeStrategyCapture({
				step: 'element',
				useCSSInjection,
				attempt: elementAttempt,
				...buildCaptureContext()
			});
			if (elementResult.status === 'captured') {
				return {
					status: 'captured',
					screenshot: elementResult.screenshot,
					fallbacks
				};
			}

			fallbacks.push(elementResult.failure);
		}

		const viewportResult = await tryViewportScreenshot({
			page: input.page,
			screenshotPath,
			cfg: input.cfg
		});
		if (!viewportResult.success) {
			fallbacks.push(viewportResult.failure);

			return {
				status: 'failed',
				failure: viewportResult.failure,
				fallbacks
			};
		}

		const finalized = await finalizeViolationCapture({
			screenshotFilename,
			screenshotPath,
			resultsDir: input.resultsDir,
			cfg: input.cfg,
			...(locationInfo !== undefined ? { locationInfo } : {}),
			...(friendlyNode !== undefined ? { friendlyNode } : {})
		});

		return {
			status: 'captured',
			screenshot: finalized,
			fallbacks
		};
	} catch (error) {
		const failure = buildFailure('viewport', 'screenshot_failed', error);
		fallbacks.push(failure);

		return {
			status: 'failed',
			failure,
			fallbacks
		};
	} finally {
		if (useCSSInjection) {
			try {
				await removeHighlightCSS(input.page);
			} catch {
				// Best-effort cleanup.
			}
		}
	}
}

async function finalizeStrategyCapture(input: {
	step: ViolationCaptureStep;
	useCSSInjection: boolean;
	attempt: StrategyAttemptResult;
	screenshotPath: string;
	screenshotFilename: string;
	resultsDir: string;
	cfg: AxeScreenshotConfig;
	locationInfo?: PageLocationInfo;
	friendlyNode?: FriendlyNodeInfo;
}): Promise<
	| { status: 'captured'; screenshot: EnhancedScreenshotResult }
	| { status: 'failed'; failure: ViolationCaptureFailure }
> {
	if (!input.attempt.success) {
		return { status: 'failed', failure: input.attempt.failure };
	}

	const overlayResult = await applyOverlayIfNeeded({
		step: input.step,
		useCSSInjection: input.useCSSInjection,
		screenshotPath: input.screenshotPath,
		cfg: input.cfg,
		...(input.attempt.buffer !== undefined ? { buffer: input.attempt.buffer } : {}),
		...(input.attempt.elementBounds !== undefined
			? { elementBounds: input.attempt.elementBounds }
			: {}),
		...(input.attempt.clipWidth !== undefined ? { clipWidth: input.attempt.clipWidth } : {})
	});
	if (!overlayResult.success) {
		return { status: 'failed', failure: overlayResult.failure };
	}

	try {
		const screenshot = await finalizeViolationCapture({
			screenshotFilename: input.screenshotFilename,
			screenshotPath: input.screenshotPath,
			resultsDir: input.resultsDir,
			cfg: input.cfg,
			...(input.locationInfo !== undefined ? { locationInfo: input.locationInfo } : {}),
			...(input.friendlyNode !== undefined ? { friendlyNode: input.friendlyNode } : {}),
			...(input.attempt.elementBounds !== undefined
				? { elementBounds: input.attempt.elementBounds }
				: {})
		});

		return { status: 'captured', screenshot };
	} catch (error) {
		return {
			status: 'failed',
			failure: buildFailure(input.step, 'screenshot_failed', error)
		};
	}
}

interface StrategyAttemptSuccess {
	success: true;
	elementBounds?: ElementBounds[];
	buffer?: Buffer;
	clipWidth?: number;
}

interface StrategyAttemptFailure {
	success: false;
	failure: ViolationCaptureFailure;
}

type StrategyAttemptResult = StrategyAttemptSuccess | StrategyAttemptFailure;

function buildFailure(
	step: ViolationCaptureStep,
	reason: ViolationCaptureFailureReason,
	error?: unknown
): ViolationCaptureFailure {
	const message = errorToMessage(error);
	return {
		step,
		reason,
		...(message !== undefined ? { message } : {})
	};
}

function errorToMessage(error: unknown): string | undefined {
	if (typeof error === 'string') {
		return error;
	}

	if (error instanceof Error) {
		return error.message;
	}

	return undefined;
}

async function tryViewportScreenshot(input: {
	page: Page;
	screenshotPath: string;
	cfg: AxeScreenshotConfig;
}): Promise<StrategyAttemptResult> {
	try {
		const buffer = await input.page.screenshot({ fullPage: false });
		await saveScreenshot(Buffer.from(buffer), input.screenshotPath, input.cfg);

		return { success: true };
	} catch (error) {
		return {
			success: false,
			failure: buildFailure('viewport', 'screenshot_failed', error)
		};
	}
}

async function finalizeViolationCapture(input: {
	screenshotFilename: string;
	screenshotPath: string;
	resultsDir: string;
	cfg: AxeScreenshotConfig;
	locationInfo?: PageLocationInfo;
	friendlyNode?: FriendlyNodeInfo;
	elementBounds?: ElementBounds[];
}): Promise<EnhancedScreenshotResult> {
	const thumbnail = await generateThumbnail(input.screenshotPath, input.resultsDir, input.cfg);

	return {
		screenshot: input.screenshotFilename,
		thumbnail,
		...(input.locationInfo !== undefined ? { locationInfo: input.locationInfo } : {}),
		...(input.friendlyNode !== undefined ? { friendlyNode: input.friendlyNode } : {}),
		...(input.elementBounds !== undefined ? { elementBounds: input.elementBounds } : {})
	};
}

async function applyOverlayIfNeeded(input: {
	step: ViolationCaptureStep;
	useCSSInjection: boolean;
	buffer?: Buffer;
	elementBounds?: ElementBounds[];
	clipWidth?: number;
	screenshotPath: string;
	cfg: AxeScreenshotConfig;
}): Promise<{ success: true } | { success: false; failure: ViolationCaptureFailure }> {
	if (input.useCSSInjection || !input.buffer || !input.elementBounds || !input.clipWidth) {
		return { success: true };
	}

	try {
		const overlaidBuffer = await compositeOverlay(
			input.buffer,
			input.elementBounds,
			input.cfg,
			input.clipWidth
		);
		await saveScreenshot(overlaidBuffer, input.screenshotPath, input.cfg);

		return { success: true };
	} catch (error) {
		return {
			success: false,
			failure: buildFailure(input.step, 'overlay_failed', error)
		};
	}
}

async function tryUnionBoundingBoxScreenshot(input: {
	page: Page;
	targets: string[];
	vw: number;
	vh: number;
	screenshotPath: string;
	cfg: AxeScreenshotConfig;
	returnBuffer: boolean;
}): Promise<StrategyAttemptResult> {
	const targetSelectors = input.targets.slice(0, input.cfg.unionMaxTargets);
	if (targetSelectors.length === 0) {
		return {
			success: false,
			failure: buildFailure('union', 'clip_invalid', 'no target selectors available')
		};
	}

	let hadScrollError = false;

	// Adaptive strategy: try multiple anchor points and keep the candidate with
	// highest target coverage in the viewport.
	let bestCapturedBoxes: {
		box: { x: number; y: number; width: number; height: number };
		selector: string;
	}[] = [];

	for (const anchorSelector of targetSelectors) {
		try {
			const locator = input.page.locator(anchorSelector).first();
			await locator.scrollIntoViewIfNeeded({
				timeout: input.cfg.scrollTimeout
			});
		} catch {
			hadScrollError = true;
		}

		try {
			// Allow scroll position/layout to settle before reading bounds.
			await input.page.waitForTimeout(50);
		} catch (error) {
			return {
				success: false,
				failure: buildFailure('union', 'screenshot_failed', error)
			};
		}

		const candidateBoxes = await collectVisibleTargetBoxes({
			page: input.page,
			targetSelectors,
			vh: input.vh
		});
		if (candidateBoxes.length > bestCapturedBoxes.length) {
			bestCapturedBoxes = candidateBoxes;
		}

		if (bestCapturedBoxes.length >= targetSelectors.length) {
			break;
		}
	}

	if (bestCapturedBoxes.length === 0) {
		return {
			success: false,
			failure: hadScrollError
				? buildFailure('union', 'scroll_failed', 'failed to scroll to any target')
				: buildFailure('union', 'bounding_box_missing', 'no visible target bounding boxes')
		};
	}

	const computed = computeUnionClip(
		bestCapturedBoxes,
		{ width: input.vw, height: input.vh },
		input.cfg
	);
	if (!computed) {
		return {
			success: false,
			failure: buildFailure('union', 'clip_invalid', 'unable to compute union clip')
		};
	}

	try {
		const screenshotBuffer = await input.page.screenshot({
			clip: computed.clip
		});
		const bufferData = Buffer.from(screenshotBuffer);

		if (input.returnBuffer) {
			return {
				success: true,
				elementBounds: computed.elementBounds,
				buffer: bufferData,
				clipWidth: computed.clip.width
			};
		}

		await saveScreenshot(bufferData, input.screenshotPath, input.cfg);

		return {
			success: true,
			elementBounds: computed.elementBounds,
			clipWidth: computed.clip.width
		};
	} catch (error) {
		return {
			success: false,
			failure: buildFailure('union', 'screenshot_failed', error)
		};
	}
}

async function collectVisibleTargetBoxes(input: {
	page: Page;
	targetSelectors: string[];
	vh: number;
}): Promise<
	{
		box: { x: number; y: number; width: number; height: number };
		selector: string;
	}[]
> {
	const capturedBoxes: {
		box: { x: number; y: number; width: number; height: number };
		selector: string;
	}[] = [];

	for (const selector of input.targetSelectors) {
		try {
			const locator = input.page.locator(selector).first();
			const box = await locator.boundingBox();
			if (!box) {
				continue;
			}

			if (box.width <= 0 || box.height <= 0) {
				continue;
			}

			// Keep only elements at least partially visible in the viewport.
			if (box.y + box.height < 0 || box.y > input.vh) {
				continue;
			}

			capturedBoxes.push({ box, selector });
		} catch {
			// Ignore per-target failures to maximize best-effort capture.
		}
	}

	return capturedBoxes;
}

async function trySingleTargetViewportScreenshot(input: {
	page: Page;
	selector: string;
	vw: number;
	vh: number;
	screenshotPath: string;
	cfg: AxeScreenshotConfig;
	returnBuffer: boolean;
}): Promise<StrategyAttemptResult> {
	const locator = input.page.locator(input.selector).first();
	let scrollError: unknown;

	try {
		await locator.scrollIntoViewIfNeeded({ timeout: input.cfg.scrollTimeout });
	} catch (error) {
		scrollError = error;
	}

	try {
		const box = await locator.boundingBox();
		if (!box) {
			if (scrollError) {
				return {
					success: false,
					failure: buildFailure('single-target', 'scroll_failed', scrollError)
				};
			}

			return {
				success: false,
				failure: buildFailure(
					'single-target',
					'bounding_box_missing',
					'target element has no bounding box'
				)
			};
		}

		const clip = computeSingleTargetClip(box, { width: input.vw, height: input.vh }, input.cfg);
		if (!clip) {
			return {
				success: false,
				failure: buildFailure('single-target', 'clip_invalid')
			};
		}

		const buffer = await input.page.screenshot({
			clip
		});

		const bufferData = Buffer.from(buffer);
		const elementBounds: ElementBounds[] = [
			{
				x: Math.max(0, Math.round(box.x - clip.x)),
				y: Math.max(0, Math.round(box.y - clip.y)),
				width: Math.round(box.width),
				height: Math.round(box.height),
				selector: input.selector
			}
		];

		if (input.returnBuffer) {
			return {
				success: true,
				elementBounds,
				buffer: bufferData,
				clipWidth: clip.width
			};
		}

		await saveScreenshot(bufferData, input.screenshotPath, input.cfg);

		return { success: true, elementBounds, clipWidth: clip.width };
	} catch (error) {
		return {
			success: false,
			failure: buildFailure('single-target', 'screenshot_failed', error)
		};
	}
}

async function tryElementScreenshot(input: {
	page: Page;
	selector: string;
	vw: number;
	vh: number;
	screenshotPath: string;
	cfg: AxeScreenshotConfig;
	returnBuffer: boolean;
}): Promise<StrategyAttemptResult> {
	try {
		const locator = input.page.locator(input.selector).first();
		const box = await locator.boundingBox();
		if (!box) {
			return {
				success: false,
				failure: buildFailure('element', 'bounding_box_missing')
			};
		}

		const clip = computeElementClip(box, { width: input.vw, height: input.vh }, input.cfg);
		if (!clip) {
			return {
				success: false,
				failure: buildFailure('element', 'clip_invalid')
			};
		}

		const buffer = await input.page.screenshot({ clip });

		const bufferData = Buffer.from(buffer);
		const elementBounds: ElementBounds[] = [
			{
				x: Math.max(0, Math.round(box.x - clip.x)),
				y: Math.max(0, Math.round(box.y - clip.y)),
				width: Math.round(box.width),
				height: Math.round(box.height),
				selector: input.selector
			}
		];

		if (input.returnBuffer) {
			return {
				success: true,
				elementBounds,
				buffer: bufferData,
				clipWidth: clip.width
			};
		}

		await saveScreenshot(bufferData, input.screenshotPath, input.cfg);

		return { success: true, elementBounds, clipWidth: clip.width };
	} catch (error) {
		return {
			success: false,
			failure: buildFailure('element', 'screenshot_failed', error)
		};
	}
}
