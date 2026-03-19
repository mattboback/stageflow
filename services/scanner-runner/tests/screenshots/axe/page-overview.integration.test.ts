import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { type Browser, chromium } from "playwright";
import sharp from "sharp";
import { afterAll, beforeAll, describe, expect, it } from "vitest";

import type { PageOverviewViolation } from "../../../src/screenshots/axe/types";

import { loadAxeScreenshotConfig } from "../../../src/screenshots/axe/config";
import { capturePageOverviewRaw } from "../../../src/screenshots/axe/page-overview";

function createTempDir(prefix: string): string {
	return fs.mkdtempSync(path.join(os.tmpdir(), prefix));
}

function approxEqual(
	actual: number,
	expected: number,
	tolerance: number,
): boolean {
	return Math.abs(actual - expected) <= tolerance;
}

describe("capturePageOverviewRaw (integration)", () => {
	let browser: Browser;

	beforeAll(async () => {
		browser = await chromium.launch({
			headless: true,
			args: ["--no-sandbox", "--disable-setuid-sandbox"],
		});
	});

	afterAll(async () => {
		await browser.close();
	});

	it("maps element bounds at DPR=1 with no clipping", async () => {
		const resultsDir = createTempDir("stageflow-page-overview-");
		const context = await browser.newContext({
			viewport: { width: 400, height: 300 },
			deviceScaleFactor: 1,
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
					id: "rule-1",
					impact: "critical",
					nodes: [{ target: ["#inside"] }],
				},
			];

			const screenshotCfg = loadAxeScreenshotConfig({
				screenshotsEnabled: true,
				outputFormat: "png",
			});

			const captured = await capturePageOverviewRaw(
				page,
				violations,
				resultsDir,
				"page1",
				"axe",
				screenshotCfg,
				{ enabled: true, maxElements: 50, maxHeight: 5000 },
			);

			expect(captured).not.toBeNull();
			if (!captured) {
				throw new Error("expected capture result");
			}

			const metadata = await sharp(captured.buffer).metadata();
			expect(captured.pageWidth).toBe(metadata.width);
			expect(captured.pageHeight).toBe(metadata.height);

			const inside = captured.elements.find((el) => el.selector === "#inside");
			expect(inside).toBeDefined();

			const expectedBox = await page.locator("#inside").boundingBox();
			expect(expectedBox).not.toBeNull();
			if (!expectedBox || !inside) {
				throw new Error("expected bounding box and element");
			}

			const dpr = captured.pageWidth / 400;
			expect(dpr).toBeCloseTo(1, 3);

			expect(approxEqual(inside.x, Math.round(expectedBox.x * dpr), 1)).toBe(
				true,
			);
			expect(approxEqual(inside.y, Math.round(expectedBox.y * dpr), 1)).toBe(
				true,
			);
			expect(
				approxEqual(inside.width, Math.round(expectedBox.width * dpr), 1),
			).toBe(true);
			expect(
				approxEqual(inside.height, Math.round(expectedBox.height * dpr), 1),
			).toBe(true);
		} finally {
			await context.close();
			fs.rmSync(resultsDir, { recursive: true, force: true });
		}
	}, 30_000);

	it("captures a clean page overview screenshot even when there are no violations", async () => {
		const resultsDir = createTempDir("stageflow-page-overview-clean-");
		const context = await browser.newContext({
			viewport: { width: 400, height: 300 },
			deviceScaleFactor: 1,
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
				outputFormat: "png",
			});

			const captured = await capturePageOverviewRaw(
				page,
				[],
				resultsDir,
				"clean-page",
				"axe",
				screenshotCfg,
				{ enabled: true, maxElements: 50, maxHeight: 5000 },
			);

			expect(captured).not.toBeNull();
			if (!captured) {
				throw new Error("expected clean page overview result");
			}

			expect(captured.screenshotFilename).toBe("page-overview-clean-page.png");
			expect(captured.elements).toEqual([]);
			expect(captured.pageWidth).toBeGreaterThan(0);
			expect(captured.pageHeight).toBeGreaterThan(0);
		} finally {
			await context.close();
			fs.rmSync(resultsDir, { recursive: true, force: true });
		}
	}, 30_000);

	it("clips partially-visible elements and skips elements below the capture area", async () => {
		const resultsDir = createTempDir("stageflow-page-overview-");
		const context = await browser.newContext({
			viewport: { width: 400, height: 300 },
			deviceScaleFactor: 1,
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
					id: "rule-1",
					impact: "serious",
					nodes: [
						{ target: ["#inside"] },
						{ target: ["#partial"] },
						{ target: ["#below"] },
					],
				},
			];

			const screenshotCfg = loadAxeScreenshotConfig({
				screenshotsEnabled: true,
				outputFormat: "png",
			});

			const captured = await capturePageOverviewRaw(
				page,
				violations,
				resultsDir,
				"page2",
				"axe",
				screenshotCfg,
				{ enabled: true, maxElements: 50, maxHeight: 1000 },
			);

			expect(captured).not.toBeNull();
			if (!captured) {
				throw new Error("expected capture result");
			}

			const below = captured.elements.find((el) => el.selector === "#below");
			expect(below).toBeUndefined();

			const partial = captured.elements.find(
				(el) => el.selector === "#partial",
			);
			expect(partial).toBeDefined();
			if (!partial) {
				throw new Error("expected partial element");
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

	it("maps element bounds under DPR=2 with clipping", async () => {
		const resultsDir = createTempDir("stageflow-page-overview-");
		const context = await browser.newContext({
			viewport: { width: 400, height: 300 },
			deviceScaleFactor: 2,
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
					id: "rule-1",
					impact: "serious",
					nodes: [{ target: ["#inside"] }, { target: ["#partial"] }],
				},
			];

			const screenshotCfg = loadAxeScreenshotConfig({
				screenshotsEnabled: true,
				outputFormat: "png",
			});

			const captured = await capturePageOverviewRaw(
				page,
				violations,
				resultsDir,
				"page3",
				"axe",
				screenshotCfg,
				{ enabled: true, maxElements: 50, maxHeight: 1000 },
			);

			expect(captured).not.toBeNull();
			if (!captured) {
				throw new Error("expected capture result");
			}

			const dpr = captured.pageWidth / 400;
			expect(dpr).toBeCloseTo(2, 3);

			const inside = captured.elements.find((el) => el.selector === "#inside");
			expect(inside).toBeDefined();
			if (!inside) {
				throw new Error("expected inside element");
			}

			const insideBox = await page.locator("#inside").boundingBox();
			expect(insideBox).not.toBeNull();
			if (!insideBox) {
				throw new Error("expected inside bounding box");
			}

			expect(approxEqual(inside.x, Math.round(insideBox.x * dpr), 2)).toBe(
				true,
			);
			expect(approxEqual(inside.y, Math.round(insideBox.y * dpr), 2)).toBe(
				true,
			);
			expect(
				approxEqual(inside.width, Math.round(insideBox.width * dpr), 2),
			).toBe(true);
			expect(
				approxEqual(inside.height, Math.round(insideBox.height * dpr), 2),
			).toBe(true);

			const partial = captured.elements.find(
				(el) => el.selector === "#partial",
			);
			expect(partial).toBeDefined();
			if (!partial) {
				throw new Error("expected partial element");
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
