import type { Page } from 'playwright';

import type { PageOverviewConfig } from './page-overview-config';

export interface PagePreparationResult {
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

export interface ScrollContainerExpansionResult {
	expandedCount: number;
	maxScrollHeight: number;
}

export interface AnimationStabilizationResult {
	animationFinishedCount: number;
	animationPausedCount: number;
}

export async function waitInPage(page: Page, timeoutMs: number): Promise<void> {
	if (timeoutMs <= 0) {
		return;
	}
	await page.waitForTimeout(timeoutMs);
}

export async function preparePageForOverview(
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

export async function expandScrollableContainersForOverview(
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

export async function stabilizePageForScreenshot(
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

export async function waitForStablePaint(page: Page): Promise<void> {
	try {
		await page.evaluate(async () => {
			const waitForAnimationFrame = (): Promise<void> =>
				new Promise((resolve) => {
					window.requestAnimationFrame(() => {
						resolve();
					});
				});

			if ('fonts' in document) {
				try {
					await Promise.race([
						document.fonts.ready,
						new Promise<void>((resolve) => {
							window.setTimeout(resolve, 500);
						})
					]);
				} catch {
					// Font readiness is best-effort; the frame waits below still flush layout.
				}
			}

			await waitForAnimationFrame();
			await waitForAnimationFrame();
		});
	} catch {
		// Ignore pages that navigate/detach while screenshots are being prepared.
	}
}
