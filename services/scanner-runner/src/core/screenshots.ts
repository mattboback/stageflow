/**
 * Screenshot Service
 *
 * Minimal screenshot utilities used by scanners (full-page + highlighted capture).
 */

import type { Page } from 'playwright';

import sharp from 'sharp';

import type { ScannerLogger } from './types';

import { createLogger } from '../utils/logger';

type ImageFormat = 'png' | 'jpeg' | 'webp';

interface ScreenshotResult {
	buffer: Buffer;
	width: number;
	height: number;
	format: ImageFormat;
}

interface BoundingBox {
	x: number;
	y: number;
	width: number;
	height: number;
}

interface HighlightedElement {
	selector: string;
	bounds: BoundingBox;
	visible: boolean;
}

interface HighlightedScreenshotResult extends ScreenshotResult {
	highlightedElements: HighlightedElement[];
}

interface FullPageCaptureOptions {
	format?: ImageFormat;
	quality?: number;
	scale?: number;
	timeout?: number;
}

interface HighlightStyle {
	borderColor?: string;
	borderWidth?: number;
	borderStyle?: 'solid' | 'dashed' | 'dotted';
	backgroundColor?: string;
	opacity?: number;
}

interface HighlightTarget {
	selector: string;
	style?: HighlightStyle;
}

interface HighlightCaptureOptions extends FullPageCaptureOptions {
	defaultStyle?: HighlightStyle;
	maxTargets?: number;
}

export interface ScreenshotService {
	captureFullPage(
		page: Page,
		options?: {
			format?: 'png' | 'jpeg' | 'webp';
			quality?: number;
			scale?: number;
			timeout?: number;
		}
	): Promise<{
		buffer: Buffer;
		width: number;
		height: number;
		format: 'png' | 'jpeg' | 'webp';
	}>;
	captureWithHighlights(
		page: Page,
		targets: {
			selector: string;
			style?: {
				borderColor?: string;
				borderWidth?: number;
				borderStyle?: 'solid' | 'dashed' | 'dotted';
				backgroundColor?: string;
				opacity?: number;
			};
		}[],
		options?: {
			format?: 'png' | 'jpeg' | 'webp';
			quality?: number;
			scale?: number;
			timeout?: number;
			defaultStyle?: {
				borderColor?: string;
				borderWidth?: number;
				borderStyle?: 'solid' | 'dashed' | 'dotted';
				backgroundColor?: string;
				opacity?: number;
			};
			maxTargets?: number;
		}
	): Promise<{
		buffer: Buffer;
		width: number;
		height: number;
		format: 'png' | 'jpeg' | 'webp';
		highlightedElements: {
			selector: string;
			bounds: { x: number; y: number; width: number; height: number };
			visible: boolean;
		}[];
	}>;
}

const DEFAULT_HIGHLIGHT_STYLE: HighlightStyle = {
	borderColor: '#ff0000',
	borderWidth: 3,
	borderStyle: 'solid',
	backgroundColor: 'rgba(255, 0, 0, 0.1)',
	opacity: 1
};

class DefaultScreenshotService implements ScreenshotService {
	private readonly logger: ScannerLogger;

	constructor(logger?: ScannerLogger) {
		this.logger = logger ?? createLogger('ScreenshotService');
	}

	async captureFullPage(
		page: Page,
		options: FullPageCaptureOptions = {}
	): Promise<ScreenshotResult> {
		const { format = 'png', quality, scale = 1, timeout = 30_000 } = options;

		await this.settleDynamicContent(page, Math.min(timeout, 10_000));

		try {
			const screenshotOptions = {
				fullPage: true,
				type: format === 'webp' ? 'png' : format,
				timeout,
				scale: scale === 1 ? 'device' : 'css',
				...(format === 'jpeg' && quality !== undefined ? { quality } : {})
			} as const;
			const buffer = await page.screenshot({
				...screenshotOptions
			});

			const finalBuffer =
				format === 'webp'
					? await sharp(buffer)
							.webp({ quality: quality ?? 80 })
							.toBuffer()
					: buffer;

			const metadata = await sharp(finalBuffer).metadata();

			return {
				buffer: finalBuffer,
				width: metadata.width,
				height: metadata.height,
				format
			};
		} catch (err) {
			this.logger.error('Failed to capture full page screenshot', {
				error: err instanceof Error ? err.message : String(err)
			});
			throw err;
		}
	}

	async captureWithHighlights(
		page: Page,
		targets: HighlightTarget[],
		options: HighlightCaptureOptions = {}
	): Promise<HighlightedScreenshotResult> {
		const {
			format = 'png',
			quality,
			timeout = 30_000,
			defaultStyle = DEFAULT_HIGHLIGHT_STYLE,
			maxTargets = 50
		} = options;

		const limitedTargets = targets.slice(0, maxTargets);
		const highlightedElements: HighlightedElement[] = [];
		const styleId = `stageflow-highlight-${Date.now()}`;

		await this.settleDynamicContent(page, Math.min(timeout, 10_000));

		try {
			await page.addStyleTag({
				content: this.generateHighlightCSS(styleId, defaultStyle)
			});

			for (const target of limitedTargets) {
				try {
					const element = await page.$(target.selector);
					if (!element) {
						continue;
					}

					const box = await element.boundingBox();
					const style = { ...defaultStyle, ...target.style };

					await element.evaluate(
						(el, params) => {
							const { styleId, style } = params;

							el.classList.add(styleId);
							if (style.borderColor) {
								(el as HTMLElement).style.outline =
									`${style.borderWidth}px ${style.borderStyle} ${style.borderColor}`;
							}
							if (style.backgroundColor) {
								(el as HTMLElement).style.backgroundColor = style.backgroundColor;
							}
							if (typeof style.opacity === 'number') {
								(el as HTMLElement).style.opacity = String(style.opacity);
							}
						},
						{ styleId, style }
					);

					highlightedElements.push({
						selector: target.selector,
						bounds: box ?? { x: 0, y: 0, width: 0, height: 0 },
						visible: Boolean(box)
					});
				} catch {
					// Ignore elements that vanish or error during highlight pass.
				}
			}

			const screenshotOptions = {
				fullPage: true,
				type: format === 'webp' ? 'png' : format,
				timeout,
				...(format === 'jpeg' && quality !== undefined ? { quality } : {})
			} as const;
			const buffer = await page.screenshot(screenshotOptions);

			const finalBuffer =
				format === 'webp'
					? await sharp(buffer)
							.webp({ quality: quality ?? 80 })
							.toBuffer()
					: buffer;

			const metadata = await sharp(finalBuffer).metadata();

			return {
				buffer: finalBuffer,
				width: metadata.width,
				height: metadata.height,
				format,
				highlightedElements
			};
		} catch (err) {
			this.logger.error('Failed to capture highlighted screenshot', {
				targetCount: targets.length,
				error: err instanceof Error ? err.message : String(err)
			});
			throw err;
		} finally {
			await page
				.evaluate((cleanupId) => {
					for (const el of document.querySelectorAll(`.${cleanupId}`)) {
						el.classList.remove(cleanupId);
						(el as HTMLElement).style.outline = '';
						(el as HTMLElement).style.backgroundColor = '';
						(el as HTMLElement).style.opacity = '';
					}
				}, styleId)
				.catch(() => undefined);
		}
	}

	/**
	 * Scroll through the page before a full-page capture so IntersectionObserver
	 * reveal animations fire and lazy-loaded images start fetching; otherwise
	 * below-the-fold content is captured at its hidden (opacity: 0) state.
	 * Best-effort: any failure falls through and the capture proceeds as-is.
	 */
	private async settleDynamicContent(page: Page, budgetMs: number): Promise<void> {
		try {
			await page.evaluate(async (maxMs) => {
				const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));
				const deadline = Date.now() + maxMs;
				const root = document.scrollingElement ?? document.documentElement;
				const step = Math.max(window.innerHeight, 1);

				for (let y = 0; y <= root.scrollHeight && Date.now() < deadline; y += step) {
					window.scrollTo(0, y);
					await sleep(120);
				}
				window.scrollTo(0, root.scrollHeight);
				await sleep(200);

				const pendingImages = Array.from(document.images)
					.filter((img) => !img.complete)
					.map((img) => img.decode().catch(() => undefined));
				await Promise.race([
					Promise.allSettled(pendingImages),
					sleep(Math.max(deadline - Date.now(), 0))
				]);

				window.scrollTo(0, 0);
				await sleep(400);
			}, budgetMs);
		} catch (err) {
			this.logger.warn?.('Pre-capture scroll pass failed; capturing current state', {
				error: err instanceof Error ? err.message : String(err)
			});
		}
	}

	private generateHighlightCSS(styleId: string, style: HighlightStyle): string {
		return `
      .${styleId} {
        outline: ${style.borderWidth}px ${style.borderStyle} ${style.borderColor} !important;
        background-color: ${style.backgroundColor} !important;
        opacity: ${style.opacity} !important;
      }
    `;
	}
}

export function createScreenshotService(logger?: ScannerLogger): ScreenshotService {
	return new DefaultScreenshotService(logger);
}
