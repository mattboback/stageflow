import { type Browser, chromium } from 'playwright';
import sharp from 'sharp';
import { afterAll, beforeAll, describe, expect, it } from 'vitest';

import { createScreenshotService } from '../../src/core/screenshots';

async function decodePngToRawRgba(buffer: Buffer): Promise<{
	data: Buffer;
	width: number;
	height: number;
}> {
	const { data, info } = await sharp(buffer)
		.ensureAlpha()
		.raw()
		.toBuffer({ resolveWithObject: true });

	return { data, width: info.width, height: info.height };
}

function getPixel(
	data: Buffer,
	width: number,
	x: number,
	y: number
): { r: number; g: number; b: number; a: number } {
	const idx = (y * width + x) * 4;
	return {
		r: data[idx] ?? 0,
		g: data[idx + 1] ?? 0,
		b: data[idx + 2] ?? 0,
		a: data[idx + 3] ?? 0
	};
}

function isRedish(pixel: { r: number; g: number; b: number; a: number }): boolean {
	return pixel.a > 200 && pixel.r > 180 && pixel.g < 120 && pixel.b < 120;
}

function anyRedPixelInRect(
	data: Buffer,
	width: number,
	height: number,
	rect: { x: number; y: number; width: number; height: number }
): boolean {
	const x0 = Math.max(0, rect.x);
	const y0 = Math.max(0, rect.y);
	const x1 = Math.min(width - 1, rect.x + rect.width);
	const y1 = Math.min(height - 1, rect.y + rect.height);

	for (let y = y0; y <= y1; y++) {
		for (let x = x0; x <= x1; x++) {
			if (isRedish(getPixel(data, width, x, y))) {
				return true;
			}
		}
	}

	return false;
}

describe('ScreenshotService (integration)', () => {
	let browser: Browser;

	beforeAll(async () => {
		browser = await chromium.launch({
			headless: true,
			args: ['--no-sandbox', '--disable-setuid-sandbox']
		});
	});

	afterAll(async () => {
		await browser.close();
	});

	it('captureWithHighlights produces a real outline near the element bounds and cleans up styles', async () => {
		const context = await browser.newContext({
			viewport: { width: 400, height: 300 },
			deviceScaleFactor: 1
		});
		const page = await context.newPage();

		try {
			await page.setContent(`
          <!doctype html>
          <html>
            <head>
              <style>
                html, body { margin: 0; padding: 0; width: 400px; height: 300px; background: #ffffff; }
                #box { position: absolute; left: 120px; top: 80px; width: 100px; height: 60px; background: #00ff00; }
              </style>
            </head>
            <body>
              <div id="box"></div>
            </body>
          </html>
        `);

			const service = createScreenshotService();
			const borderWidthCss = 10;

			const result = await service.captureWithHighlights(page, [{ selector: '#box' }], {
				format: 'png',
				defaultStyle: {
					borderColor: '#ff0000',
					borderWidth: borderWidthCss,
					borderStyle: 'solid',
					backgroundColor: 'rgba(255,0,0,0.1)',
					opacity: 1
				}
			});

			expect(result.highlightedElements).toHaveLength(1);
			expect(result.highlightedElements[0]?.selector).toBe('#box');
			expect(result.highlightedElements[0]?.visible).toBe(true);

			const bounds = result.highlightedElements[0]!.bounds;
			expect(Math.abs(bounds.x - 120)).toBeLessThanOrEqual(1);
			expect(Math.abs(bounds.y - 80)).toBeLessThanOrEqual(1);
			expect(Math.abs(bounds.width - 100)).toBeLessThanOrEqual(1);
			expect(Math.abs(bounds.height - 60)).toBeLessThanOrEqual(1);

			const { data, width, height } = await decodePngToRawRgba(result.buffer);
			expect(width).toBeGreaterThan(0);
			expect(height).toBeGreaterThan(0);

			const dpr = width / 400;
			expect(dpr).toBeCloseTo(1, 3);

			const expectedLeft = Math.round(bounds.x * dpr);
			const expectedMidY = Math.round((bounds.y + bounds.height / 2) * dpr);
			const borderPx = Math.max(2, Math.round(borderWidthCss * dpr));

			// Scan just outside the left edge where the outline should appear (over white background).
			const foundRed = anyRedPixelInRect(data, width, height, {
				x: expectedLeft - borderPx,
				y: expectedMidY - 3,
				width: borderPx - 1,
				height: 6
			});
			expect(foundRed).toBe(true);

			const cleanup = await page.evaluate(() => {
				const el = document.querySelector('#box');
				if (!el || !(el instanceof HTMLElement)) {
					return null;
				}
				const hasHighlightClass = Array.from(el.classList).some((c) =>
					c.startsWith('stageflow-highlight-')
				);
				return {
					hasHighlightClass,
					outline: el.style.outline,
					backgroundColor: el.style.backgroundColor,
					opacity: el.style.opacity
				};
			});

			expect(cleanup).toEqual({
				hasHighlightClass: false,
				outline: '',
				backgroundColor: '',
				opacity: ''
			});
		} finally {
			await context.close();
		}
	}, 30_000);

	it('captureWithHighlights produces an outline at DPR=2 and returns scaled image dimensions', async () => {
		const context = await browser.newContext({
			viewport: { width: 400, height: 300 },
			deviceScaleFactor: 2
		});
		const page = await context.newPage();

		try {
			await page.setContent(`
          <!doctype html>
          <html>
            <head>
              <style>
                html, body { margin: 0; padding: 0; width: 400px; height: 300px; background: #ffffff; }
                #box { position: absolute; left: 120px; top: 80px; width: 100px; height: 60px; background: #00ff00; }
              </style>
            </head>
            <body>
              <div id="box"></div>
            </body>
          </html>
        `);

			const service = createScreenshotService();
			const borderWidthCss = 10;

			const result = await service.captureWithHighlights(page, [{ selector: '#box' }], {
				format: 'png',
				defaultStyle: {
					borderColor: '#ff0000',
					borderWidth: borderWidthCss,
					borderStyle: 'solid',
					backgroundColor: 'rgba(255,0,0,0.1)',
					opacity: 1
				}
			});

			const { data, width, height } = await decodePngToRawRgba(result.buffer);
			expect(width).toBe(800);
			expect(height).toBe(600);

			const dpr = width / 400;
			expect(dpr).toBeCloseTo(2, 3);

			const bounds = result.highlightedElements[0]!.bounds;
			const expectedLeft = Math.round(bounds.x * dpr);
			const expectedMidY = Math.round((bounds.y + bounds.height / 2) * dpr);
			const borderPx = Math.max(2, Math.round(borderWidthCss * dpr));

			const foundRed = anyRedPixelInRect(data, width, height, {
				x: expectedLeft - borderPx,
				y: expectedMidY - 3,
				width: borderPx - 1,
				height: 6
			});
			expect(foundRed).toBe(true);
		} finally {
			await context.close();
		}
	}, 30_000);
});
