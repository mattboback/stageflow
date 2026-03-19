import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import type { PageOverviewViolation } from "../../../src/screenshots/axe/types";

import {
	clipPageOverviewBounds,
	collectPageOverviewTargets,
	computeScreenshotScaleFactor,
	loadPageOverviewConfig,
} from "../../../src/screenshots/axe/page-overview";

describe("collectPageOverviewTargets", () => {
	it("respects maxElements and skips missing selectors", () => {
		const violations: PageOverviewViolation[] = [
			{
				id: "r1",
				impact: "critical",
				nodes: [{ target: ["  "] }, { target: ["#a"] }],
			},
			{
				id: "r2",
				impact: "serious",
				nodes: [{ target: ["#b"] }, { target: ["#c"] }],
			},
		];

		const targets = collectPageOverviewTargets(violations, 2);
		expect(targets).toHaveLength(2);
		expect(targets[0]?.ruleId).toBe("r1");
		expect(targets[0]?.selector).toBe("#a");
		expect(targets[1]?.ruleId).toBe("r2");
		expect(targets[1]?.selector).toBe("#b");
	});

	it("returns empty when maxElements is 0", () => {
		const violations: PageOverviewViolation[] = [
			{ id: "r1", impact: "minor", nodes: [{ target: ["#a"] }] },
		];
		expect(collectPageOverviewTargets(violations, 0)).toEqual([]);
	});
});

describe("computeScreenshotScaleFactor", () => {
	it("returns 1 for invalid inputs", () => {
		expect(computeScreenshotScaleFactor(0, 1280)).toBe(1);
		expect(computeScreenshotScaleFactor(2560, 0)).toBe(1);
		expect(computeScreenshotScaleFactor(-1, 1280)).toBe(1);
		expect(computeScreenshotScaleFactor(2560, -1)).toBe(1);
	});

	it("computes DPR scale from actual vs CSS pixels", () => {
		expect(computeScreenshotScaleFactor(2560, 1280)).toBe(2);
		expect(computeScreenshotScaleFactor(3030, 1515)).toBe(2);
	});

	it("regression: percent math stays consistent under DPR scaling", () => {
		// Screenshot is 2x the CSS dimensions (deviceScaleFactor = 2)
		const screenshotWidth = 2560;
		const screenshotHeight = 3030;
		const cssWidth = 1280;
		const cssHeight = 1515;

		const scaleX = computeScreenshotScaleFactor(screenshotWidth, cssWidth);
		const scaleY = computeScreenshotScaleFactor(screenshotHeight, cssHeight);

		const cssBox = { x: 100, y: 200, width: 50, height: 20 };
		const scaledBox = {
			x: cssBox.x * scaleX,
			y: cssBox.y * scaleY,
			width: cssBox.width * scaleX,
			height: cssBox.height * scaleY,
		};

		const xPercent =
			Math.round((scaledBox.x / screenshotWidth) * 100 * 100) / 100;
		const yPercent =
			Math.round((scaledBox.y / screenshotHeight) * 100 * 100) / 100;
		const widthPercent =
			Math.round((scaledBox.width / screenshotWidth) * 100 * 100) / 100;
		const heightPercent =
			Math.round((scaledBox.height / screenshotHeight) * 100 * 100) / 100;

		// Expected percentages match the original CSS ratios: x=100/1280, y=200/1515, etc.
		expect(xPercent).toBe(7.81);
		expect(yPercent).toBe(13.2);
		expect(widthPercent).toBe(3.91);
		expect(heightPercent).toBe(1.32);
	});
});

describe("clipPageOverviewBounds", () => {
	it("returns null for invalid max dimensions", () => {
		expect(
			clipPageOverviewBounds({ x: 0, y: 0, width: 10, height: 10 }, 0, 100),
		).toBeNull();
		expect(
			clipPageOverviewBounds({ x: 0, y: 0, width: 10, height: 10 }, 100, 0),
		).toBeNull();
	});

	it("returns null for non-finite or non-positive bounds", () => {
		expect(
			clipPageOverviewBounds(
				{ x: Number.NaN, y: 0, width: 10, height: 10 },
				100,
				100,
			),
		).toBeNull();
		expect(
			clipPageOverviewBounds({ x: 0, y: 0, width: 0, height: 10 }, 100, 100),
		).toBeNull();
		expect(
			clipPageOverviewBounds({ x: 0, y: 0, width: 10, height: -1 }, 100, 100),
		).toBeNull();
	});

	it("returns null when bounds are entirely outside the image", () => {
		expect(
			clipPageOverviewBounds({ x: 200, y: 0, width: 10, height: 10 }, 100, 100),
		).toBeNull();
		expect(
			clipPageOverviewBounds({ x: 0, y: 200, width: 10, height: 10 }, 100, 100),
		).toBeNull();
		expect(
			clipPageOverviewBounds(
				{ x: -50, y: -50, width: 10, height: 10 },
				100,
				100,
			),
		).toBeNull();
	});

	it("clips bounds that extend beyond edges", () => {
		expect(
			clipPageOverviewBounds({ x: -10, y: 5, width: 50, height: 10 }, 100, 100),
		).toEqual({
			x: 0,
			y: 5,
			width: 40,
			height: 10,
		});

		expect(
			clipPageOverviewBounds({ x: 90, y: 90, width: 20, height: 20 }, 100, 100),
		).toEqual({
			x: 90,
			y: 90,
			width: 10,
			height: 10,
		});
	});
});

describe("loadPageOverviewConfig", () => {
	const originalEnv = process.env;

	beforeEach(() => {
		vi.resetModules();
		process.env = { ...originalEnv };
	});

	afterEach(() => {
		process.env = originalEnv;
	});

	it("returns default values when no environment variables are set", () => {
		process.env.A11Y_PAGE_OVERVIEW_ENABLED = undefined;
		process.env.A11Y_PAGE_OVERVIEW_MAX_ELEMENTS = undefined;
		process.env.A11Y_PAGE_OVERVIEW_MAX_HEIGHT = undefined;

		const config = loadPageOverviewConfig();

		expect(config.enabled).toBe(true);
		expect(config.maxElements).toBe(50);
		expect(config.maxHeight).toBe(5000);
	});

	it("reads enabled from A11Y_PAGE_OVERVIEW_ENABLED env var", () => {
		process.env.A11Y_PAGE_OVERVIEW_ENABLED = "false";
		expect(loadPageOverviewConfig().enabled).toBe(false);

		process.env.A11Y_PAGE_OVERVIEW_ENABLED = "true";
		expect(loadPageOverviewConfig().enabled).toBe(true);

		process.env.A11Y_PAGE_OVERVIEW_ENABLED = "0";
		expect(loadPageOverviewConfig().enabled).toBe(false);

		process.env.A11Y_PAGE_OVERVIEW_ENABLED = "1";
		expect(loadPageOverviewConfig().enabled).toBe(true);
	});

	it("reads maxElements from A11Y_PAGE_OVERVIEW_MAX_ELEMENTS env var", () => {
		process.env.A11Y_PAGE_OVERVIEW_MAX_ELEMENTS = "100";
		expect(loadPageOverviewConfig().maxElements).toBe(100);

		process.env.A11Y_PAGE_OVERVIEW_MAX_ELEMENTS = "25";
		expect(loadPageOverviewConfig().maxElements).toBe(25);
	});

	it("clamps maxElements to minimum of 0", () => {
		process.env.A11Y_PAGE_OVERVIEW_MAX_ELEMENTS = "-10";
		expect(loadPageOverviewConfig().maxElements).toBe(0);
	});

	it("reads maxHeight from A11Y_PAGE_OVERVIEW_MAX_HEIGHT env var", () => {
		process.env.A11Y_PAGE_OVERVIEW_MAX_HEIGHT = "10000";
		expect(loadPageOverviewConfig().maxHeight).toBe(10000);
	});

	it("allows overrides to take precedence over environment variables", () => {
		process.env.A11Y_PAGE_OVERVIEW_ENABLED = "true";
		process.env.A11Y_PAGE_OVERVIEW_MAX_ELEMENTS = "100";

		const config = loadPageOverviewConfig({
			enabled: false,
			maxElements: 25,
		});

		expect(config.enabled).toBe(false);
		expect(config.maxElements).toBe(25);
		// Non-overridden values should still come from env
		expect(config.maxHeight).toBe(5000); // default, not in env
	});

	it("handles partial overrides correctly", () => {
		const config = loadPageOverviewConfig({
			maxElements: 10,
		});

		expect(config.enabled).toBe(true); // default
		expect(config.maxElements).toBe(10); // override
		expect(config.maxHeight).toBe(5000); // default
	});
});
