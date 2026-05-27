import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { type Browser, chromium } from 'playwright';
import sharp from 'sharp';
import { afterAll, beforeAll, describe, expect, it } from 'vitest';

import type { PageOverviewViolation } from '../../../src/screenshots/axe/types';

import { loadAxeScreenshotConfig } from '../../../src/screenshots/axe/config';
import { capturePageOverviewRaw } from '../../../src/screenshots/axe/page-overview';

function createTempDir(prefix: string): string {
	return fs.mkdtempSync(path.join(os.tmpdir(), prefix));
}

function approxEqual(actual: number, expected: number, tolerance: number): boolean {
	return Math.abs(actual - expected) <= tolerance;
}

async function readPixel(
	buffer: Buffer,
	x: number,
	y: number
): Promise<{ r: number; g: number; b: number; a: number }> {
	const { data, info } = await sharp(buffer)
		.ensureAlpha()
		.raw()
		.toBuffer({ resolveWithObject: true });
	const clampedX = Math.max(0, Math.min(info.width - 1, Math.round(x)));
	const clampedY = Math.max(0, Math.min(info.height - 1, Math.round(y)));
	const offset = (clampedY * info.width + clampedX) * info.channels;
	return {
		r: data[offset] ?? 0,
		g: data[offset + 1] ?? 0,
		b: data[offset + 2] ?? 0,
		a: data[offset + 3] ?? 0
	};
}

describe('capturePageOverviewRaw (integration)', () => {
	let browser: Browser | null = null;

	function getBrowser(): Browser {
		if (!browser) {
			throw new Error('browser was not initialized');
		}
		return browser;
	}

	beforeAll(async () => {
		browser = await chromium.launch({
			headless: true,
			args: ['--no-sandbox', '--disable-setuid-sandbox']
		});
	});

	afterAll(async () => {
		await browser?.close();
		browser = null;
	});

	it('maps element bounds at DPR=1 with no clipping', async () => {
		const resultsDir = createTempDir('stageflow-page-overview-');
		const context = await getBrowser().newContext({
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
                html, body { margin: 0; padding: 0; }
                #spacer { height: 600px; width: 1px; }
                #inside { position: absolute; left: 50px; top: 100px; width: 120px; height: 60px; background: #00ff00; }
              </style>
            </head>
            <body>
              <div id="spacer"></div>
              <div id="inside"></div>
            </body>
          </html>
        `);

			const violations: PageOverviewViolation[] = [
				{
					id: 'rule-1',
					impact: 'critical',
					nodes: [{ target: ['#inside'] }]
				}
			];

			const screenshotCfg = loadAxeScreenshotConfig({
				screenshotsEnabled: true,
				outputFormat: 'png'
			});

			const captured = await capturePageOverviewRaw(
				page,
				violations,
				resultsDir,
				'page1',
				'axe',
				screenshotCfg,
				{ enabled: true, maxElements: 50, maxHeight: 5000 }
			);

			expect(captured).not.toBeNull();
			if (!captured) {
				throw new Error('expected capture result');
			}

			const metadata = await sharp(captured.buffer).metadata();
			expect(captured.pageWidth).toBe(metadata.width);
			expect(captured.pageHeight).toBe(metadata.height);

			const inside = captured.elements.find((el) => el.selector === '#inside');
			expect(inside).toBeDefined();

			const expectedBox = await page.locator('#inside').boundingBox();
			expect(expectedBox).not.toBeNull();
			if (!expectedBox || !inside) {
				throw new Error('expected bounding box and element');
			}

			const dpr = captured.pageWidth / 400;
			expect(dpr).toBeCloseTo(1, 3);

			expect(approxEqual(inside.x, Math.round(expectedBox.x * dpr), 1)).toBe(true);
			expect(approxEqual(inside.y, Math.round(expectedBox.y * dpr), 1)).toBe(true);
			expect(approxEqual(inside.width, Math.round(expectedBox.width * dpr), 1)).toBe(true);
			expect(approxEqual(inside.height, Math.round(expectedBox.height * dpr), 1)).toBe(true);
		} finally {
			await context.close();
			fs.rmSync(resultsDir, { recursive: true, force: true });
		}
	}, 30_000);

	it('captures a clean page overview screenshot even when there are no violations', async () => {
		const resultsDir = createTempDir('stageflow-page-overview-clean-');
		const context = await getBrowser().newContext({
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
                main { display: grid; place-items: center; width: 100%; height: 100%; color: #111827; }
              </style>
            </head>
            <body>
              <main>Clean page</main>
            </body>
          </html>
        `);

			const screenshotCfg = loadAxeScreenshotConfig({
				screenshotsEnabled: true,
				outputFormat: 'png'
			});

			const captured = await capturePageOverviewRaw(
				page,
				[],
				resultsDir,
				'clean-page',
				'axe',
				screenshotCfg,
				{ enabled: true, maxElements: 50, maxHeight: 5000 }
			);

			expect(captured).not.toBeNull();
			if (!captured) {
				throw new Error('expected clean page overview result');
			}

			expect(captured.screenshotFilename).toBe('page-overview-clean-page.png');
			expect(captured.elements).toEqual([]);
			expect(captured.pageWidth).toBeGreaterThan(0);
			expect(captured.pageHeight).toBeGreaterThan(0);
		} finally {
			await context.close();
			fs.rmSync(resultsDir, { recursive: true, force: true });
		}
	}, 30_000);

	it('pre-scrolls the page so IntersectionObserver reveal content is visible before capture', async () => {
		const resultsDir = createTempDir('stageflow-page-overview-reveal-');
		const context = await getBrowser().newContext({
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
                html, body { margin: 0; padding: 0; background: #ffffff; }
                #spacer { height: 1500px; width: 1px; }
                #revealed {
                  position: absolute;
                  left: 60px;
                  top: 1120px;
                  width: 180px;
                  height: 100px;
                  background: rgb(238, 32, 48);
                  opacity: 0;
                  transform: translateY(40px);
                  transition: opacity 120ms linear, transform 120ms linear;
                }
                #revealed.visible {
                  opacity: 1;
                  transform: translateY(0);
                }
              </style>
            </head>
            <body>
              <div id="spacer"></div>
              <div id="revealed"></div>
              <script>
                const target = document.querySelector('#revealed');
                const observer = new IntersectionObserver((entries) => {
                  if (entries.some((entry) => entry.isIntersecting)) {
                    target.classList.add('visible');
                  }
                });
                observer.observe(target);
              </script>
            </body>
          </html>
        `);

			const violations: PageOverviewViolation[] = [
				{
					id: 'revealed-rule',
					impact: 'serious',
					nodes: [{ target: ['#revealed'] }]
				}
			];

			const screenshotCfg = loadAxeScreenshotConfig({
				screenshotsEnabled: true,
				outputFormat: 'png'
			});

			const captured = await capturePageOverviewRaw(
				page,
				violations,
				resultsDir,
				'reveal-page',
				'axe',
				screenshotCfg,
				{
					enabled: true,
					maxElements: 50,
					maxHeight: 0,
					scrollPauseMs: 30,
					scrollSettleMs: 150,
					scrollStepPx: 250
				}
			);

			expect(captured).not.toBeNull();
			if (!captured) {
				throw new Error('expected reveal capture result');
			}

			const revealed = captured.elements.find((el) => el.selector === '#revealed');
			expect(revealed).toBeDefined();
			if (!revealed) {
				throw new Error('expected revealed element');
			}

			const pixel = await readPixel(
				captured.buffer,
				revealed.x + revealed.width / 2,
				revealed.y + revealed.height / 2
			);
			expect(pixel.r).toBeGreaterThan(200);
			expect(pixel.g).toBeLessThan(80);
			expect(pixel.b).toBeLessThan(90);
		} finally {
			await context.close();
			fs.rmSync(resultsDir, { recursive: true, force: true });
		}
	}, 30_000);

	it('expands internal scroll containers so app-shell pages are captured beyond the viewport', async () => {
		const resultsDir = createTempDir('stageflow-page-overview-inner-scroll-');
		const context = await getBrowser().newContext({
			viewport: { width: 500, height: 320 },
			deviceScaleFactor: 1
		});
		const page = await context.newPage();

		try {
			await page.setContent(`
          <!doctype html>
          <html>
            <head>
              <style>
                html, body {
                  margin: 0;
                  padding: 0;
                  width: 500px;
                  height: 320px;
                  overflow: hidden;
                  background: #ffffff;
                }
                .app-shell {
                  height: 320px;
                  overflow: hidden;
                  display: flex;
                }
                .sidebar {
                  width: 80px;
                  background: #f3f4f6;
                }
                .content-pane {
                  flex: 1;
                  height: 320px;
                  overflow-y: auto;
                  position: relative;
                  background: #ffffff;
                }
                .content-inner {
                  position: relative;
                  min-height: 1280px;
                }
                #deep-issue {
                  position: absolute;
                  left: 120px;
                  top: 980px;
                  width: 180px;
                  height: 80px;
                  background: rgb(220, 38, 38);
                }
              </style>
            </head>
            <body>
              <div class="app-shell">
                <aside class="sidebar">Nav</aside>
                <main class="content-pane">
                  <div class="content-inner">
                    <div id="deep-issue"></div>
                  </div>
                </main>
              </div>
            </body>
          </html>
        `);

			const violations: PageOverviewViolation[] = [
				{
					id: 'deep-in-scroll-container',
					impact: 'serious',
					nodes: [{ target: ['#deep-issue'] }]
				}
			];

			const screenshotCfg = loadAxeScreenshotConfig({
				screenshotsEnabled: true,
				outputFormat: 'png'
			});

			const captured = await capturePageOverviewRaw(
				page,
				violations,
				resultsDir,
				'inner-scroll-page',
				'axe',
				screenshotCfg,
				{ enabled: true, maxElements: 50, maxHeight: 0, preScroll: false }
			);

			expect(captured).not.toBeNull();
			if (!captured) {
				throw new Error('expected inner-scroll capture result');
			}

			expect(captured.pageHeight).toBeGreaterThan(1000);
			const deepIssue = captured.elements.find((el) => el.selector === '#deep-issue');
			expect(deepIssue).toBeDefined();
			if (!deepIssue) {
				throw new Error('expected deep issue element');
			}

			expect(deepIssue.y).toBeGreaterThan(900);
			const pixel = await readPixel(
				captured.buffer,
				deepIssue.x + deepIssue.width / 2,
				deepIssue.y + deepIssue.height / 2
			);
			expect(pixel.r).toBeGreaterThan(180);
			expect(pixel.g).toBeLessThan(80);
			expect(pixel.b).toBeLessThan(80);
		} finally {
			await context.close();
			fs.rmSync(resultsDir, { recursive: true, force: true });
		}
	}, 30_000);

	it('forces content-visibility auto content to paint in the overview screenshot', async () => {
		const resultsDir = createTempDir('stageflow-page-overview-content-visibility-');
		const context = await getBrowser().newContext({
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
                html, body { margin: 0; padding: 0; background: #ffffff; }
                #spacer { height: 1300px; width: 1px; }
                #auto-section {
                  content-visibility: auto;
                  contain-intrinsic-size: 220px;
                  position: absolute;
                  left: 40px;
                  top: 920px;
                  width: 260px;
                  height: 180px;
                }
                #inside-auto {
                  width: 180px;
                  height: 90px;
                  background: rgb(22, 163, 74);
                }
              </style>
            </head>
            <body>
              <div id="spacer"></div>
              <section id="auto-section">
                <div id="inside-auto"></div>
              </section>
            </body>
          </html>
        `);

			const violations: PageOverviewViolation[] = [
				{
					id: 'content-visibility-rule',
					impact: 'moderate',
					nodes: [{ target: ['#inside-auto'] }]
				}
			];

			const screenshotCfg = loadAxeScreenshotConfig({
				screenshotsEnabled: true,
				outputFormat: 'png'
			});

			const captured = await capturePageOverviewRaw(
				page,
				violations,
				resultsDir,
				'content-visibility-page',
				'axe',
				screenshotCfg,
				{ enabled: true, maxElements: 50, maxHeight: 0 }
			);

			expect(captured).not.toBeNull();
			if (!captured) {
				throw new Error('expected content visibility capture result');
			}

			const insideAuto = captured.elements.find((el) => el.selector === '#inside-auto');
			expect(insideAuto).toBeDefined();
			if (!insideAuto) {
				throw new Error('expected content-visibility element');
			}

			const pixel = await readPixel(
				captured.buffer,
				insideAuto.x + insideAuto.width / 2,
				insideAuto.y + insideAuto.height / 2
			);
			expect(pixel.r).toBeLessThan(80);
			expect(pixel.g).toBeGreaterThan(120);
			expect(pixel.b).toBeLessThan(110);
		} finally {
			await context.close();
			fs.rmSync(resultsDir, { recursive: true, force: true });
		}
	}, 30_000);

	it('finishes finite opacity animations instead of freezing them hidden', async () => {
		const resultsDir = createTempDir('stageflow-page-overview-animation-');
		const context = await getBrowser().newContext({
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
                html, body { margin: 0; padding: 0; background: #ffffff; }
                @keyframes fade-in {
                  from { opacity: 0; }
                  to { opacity: 1; }
                }
                #animated {
                  position: absolute;
                  left: 70px;
                  top: 80px;
                  width: 160px;
                  height: 90px;
                  background: rgb(37, 99, 235);
                  opacity: 0;
                  animation: fade-in 10s linear forwards;
                }
              </style>
            </head>
            <body>
              <div id="animated"></div>
            </body>
          </html>
        `);

			const violations: PageOverviewViolation[] = [
				{
					id: 'animated-rule',
					impact: 'minor',
					nodes: [{ target: ['#animated'] }]
				}
			];

			const screenshotCfg = loadAxeScreenshotConfig({
				screenshotsEnabled: true,
				outputFormat: 'png'
			});

			const captured = await capturePageOverviewRaw(
				page,
				violations,
				resultsDir,
				'animation-page',
				'axe',
				screenshotCfg,
				{ enabled: true, maxElements: 50, maxHeight: 0 }
			);

			expect(captured).not.toBeNull();
			if (!captured) {
				throw new Error('expected animated capture result');
			}

			const animated = captured.elements.find((el) => el.selector === '#animated');
			expect(animated).toBeDefined();
			if (!animated) {
				throw new Error('expected animated element');
			}

			const pixel = await readPixel(
				captured.buffer,
				animated.x + animated.width / 2,
				animated.y + animated.height / 2
			);
			expect(pixel.r).toBeLessThan(90);
			expect(pixel.g).toBeLessThan(130);
			expect(pixel.b).toBeGreaterThan(180);
		} finally {
			await context.close();
			fs.rmSync(resultsDir, { recursive: true, force: true });
		}
	}, 30_000);

	it('clips partially-visible elements and skips elements below an explicit capture height', async () => {
		const resultsDir = createTempDir('stageflow-page-overview-');
		const context = await getBrowser().newContext({
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
                html, body { margin: 0; padding: 0; }
                #spacer { height: 2000px; width: 1px; }
                #inside { position: absolute; left: 50px; top: 100px; width: 120px; height: 60px; background: #00ff00; }
                #partial { position: absolute; left: 20px; top: 950px; width: 200px; height: 200px; background: #00ff00; }
                #below { position: absolute; left: 30px; top: 1200px; width: 140px; height: 40px; background: #00ff00; }
              </style>
            </head>
            <body>
              <div id="spacer"></div>
              <div id="inside"></div>
              <div id="partial"></div>
              <div id="below"></div>
            </body>
          </html>
        `);

			const violations: PageOverviewViolation[] = [
				{
					id: 'rule-1',
					impact: 'serious',
					nodes: [{ target: ['#inside'] }, { target: ['#partial'] }, { target: ['#below'] }]
				}
			];

			const screenshotCfg = loadAxeScreenshotConfig({
				screenshotsEnabled: true,
				outputFormat: 'png'
			});

			const captured = await capturePageOverviewRaw(
				page,
				violations,
				resultsDir,
				'page2',
				'axe',
				screenshotCfg,
				{ enabled: true, maxElements: 50, maxHeight: 1000 }
			);

			expect(captured).not.toBeNull();
			if (!captured) {
				throw new Error('expected capture result');
			}

			const below = captured.elements.find((el) => el.selector === '#below');
			expect(below).toBeUndefined();

			const partial = captured.elements.find((el) => el.selector === '#partial');
			expect(partial).toBeDefined();
			if (!partial) {
				throw new Error('expected partial element');
			}

			// `#partial` starts at y=950 and the captureHeight is 1000 CSS px.
			// It should be included, but clipped to a height of ~50px in the screenshot.
			expect(approxEqual(partial.y, 950, 1)).toBe(true);
			expect(approxEqual(partial.height, 50, 2)).toBe(true);
		} finally {
			await context.close();
			fs.rmSync(resultsDir, { recursive: true, force: true });
		}
	}, 30_000);

	it('maps element bounds under DPR=2 with clipping', async () => {
		const resultsDir = createTempDir('stageflow-page-overview-');
		const context = await getBrowser().newContext({
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
                html, body { margin: 0; padding: 0; }
                #spacer { height: 2000px; width: 1px; }
                #inside { position: absolute; left: 50px; top: 100px; width: 120px; height: 60px; background: #00ff00; }
                #partial { position: absolute; left: 20px; top: 950px; width: 200px; height: 200px; background: #00ff00; }
              </style>
            </head>
            <body>
              <div id="spacer"></div>
              <div id="inside"></div>
              <div id="partial"></div>
            </body>
          </html>
        `);

			const violations: PageOverviewViolation[] = [
				{
					id: 'rule-1',
					impact: 'serious',
					nodes: [{ target: ['#inside'] }, { target: ['#partial'] }]
				}
			];

			const screenshotCfg = loadAxeScreenshotConfig({
				screenshotsEnabled: true,
				outputFormat: 'png'
			});

			const captured = await capturePageOverviewRaw(
				page,
				violations,
				resultsDir,
				'page3',
				'axe',
				screenshotCfg,
				{ enabled: true, maxElements: 50, maxHeight: 1000 }
			);

			expect(captured).not.toBeNull();
			if (!captured) {
				throw new Error('expected capture result');
			}

			const dpr = captured.pageWidth / 400;
			expect(dpr).toBeCloseTo(2, 3);

			const inside = captured.elements.find((el) => el.selector === '#inside');
			expect(inside).toBeDefined();
			if (!inside) {
				throw new Error('expected inside element');
			}

			const insideBox = await page.locator('#inside').boundingBox();
			expect(insideBox).not.toBeNull();
			if (!insideBox) {
				throw new Error('expected inside bounding box');
			}

			expect(approxEqual(inside.x, Math.round(insideBox.x * dpr), 2)).toBe(true);
			expect(approxEqual(inside.y, Math.round(insideBox.y * dpr), 2)).toBe(true);
			expect(approxEqual(inside.width, Math.round(insideBox.width * dpr), 2)).toBe(true);
			expect(approxEqual(inside.height, Math.round(insideBox.height * dpr), 2)).toBe(true);

			const partial = captured.elements.find((el) => el.selector === '#partial');
			expect(partial).toBeDefined();
			if (!partial) {
				throw new Error('expected partial element');
			}

			// Under DPR=2, y and height are scaled. The clipped height should be ~50 * 2 = 100px.
			expect(approxEqual(partial.y, Math.round(950 * dpr), 4)).toBe(true);
			expect(approxEqual(partial.height, Math.round(50 * dpr), 4)).toBe(true);
		} finally {
			await context.close();
			fs.rmSync(resultsDir, { recursive: true, force: true });
		}
	}, 30_000);
});
