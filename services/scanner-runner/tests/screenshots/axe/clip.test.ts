import { describe, expect, it } from "vitest";

import {
	computeElementClip,
	computeSingleTargetClip,
	computeUnionClip,
} from "../../../src/screenshots/axe/clip";
import { loadAxeScreenshotConfig } from "../../../src/screenshots/axe/config";

describe("computeUnionClip", () => {
	it("returns null for empty input", () => {
		const cfg = loadAxeScreenshotConfig({ clipPadding: 0, contextRatio: 0 });
		expect(computeUnionClip([], { width: 200, height: 200 }, cfg)).toBeNull();
	});

	it("computes a padded clip and relative element bounds", () => {
		const cfg = loadAxeScreenshotConfig({ clipPadding: 0, contextRatio: 0 });
		const viewport = { width: 200, height: 200 };
		const boxes = [
			{ selector: "#a", box: { x: 10, y: 20, width: 50, height: 10 } },
			{ selector: "#b", box: { x: 70, y: 40, width: 20, height: 20 } },
		];

		const result = computeUnionClip(boxes, viewport, cfg);
		expect(result).not.toBeNull();
		const r = result as NonNullable<typeof result>;
		expect(r.clip).toEqual({ x: 0, y: 5, width: 110, height: 70 });
		expect(r.elementBounds).toEqual([
			{ selector: "#a", x: 10, y: 15, width: 50, height: 10 },
			{ selector: "#b", x: 70, y: 35, width: 20, height: 20 },
		]);
	});
});

describe("computeSingleTargetClip", () => {
	it("centers the element with context and clamps within the viewport", () => {
		const cfg = loadAxeScreenshotConfig({
			shotScale: 1,
			shotMinWidth: 0,
			shotMinHeight: 0,
			contextRatio: 0,
		});
		const viewport = { width: 200, height: 200 };
		const box = { x: 50, y: 60, width: 20, height: 10 };

		expect(computeSingleTargetClip(box, viewport, cfg)).toEqual({
			x: 30,
			y: 40,
			width: 60,
			height: 50,
		});
	});
});

describe("computeElementClip", () => {
	it("centers the crop with configurable context", () => {
		const viewport = { width: 300, height: 200 };
		const box = { x: 120, y: 60, width: 20, height: 10 };

		expect(
			computeElementClip(
				box,
				viewport,
				{ elementContextMultiplier: 2.5, elementContextMinSize: 100 },
				20,
			),
		).toEqual({
			x: 80,
			y: 15,
			width: 100,
			height: 100,
		});
	});

	it("clamps the crop within the viewport", () => {
		const viewport = { width: 80, height: 60 };
		const box = { x: 70, y: 50, width: 20, height: 10 };

		expect(
			computeElementClip(
				box,
				viewport,
				{ elementContextMultiplier: 3, elementContextMinSize: 200 },
				20,
			),
		).toEqual({
			x: 7,
			y: 5,
			width: 73,
			height: 55,
		});
	});
});

describe("elementBounds accuracy regression tests", () => {
	it("element bounds are positioned relative to clip origin", () => {
		const cfg = loadAxeScreenshotConfig({ clipPadding: 0, contextRatio: 0 });
		const viewport = { width: 1280, height: 720 };
		const boxes = [
			{ selector: "#target", box: { x: 100, y: 200, width: 50, height: 30 } },
		];

		const result = computeUnionClip(boxes, viewport, cfg);
		expect(result).not.toBeNull();
		const r = result as NonNullable<typeof result>;
		const box0 = boxes[0] as NonNullable<(typeof boxes)[0]>;
		const bounds0 = r.elementBounds[0] as NonNullable<
			(typeof r.elementBounds)[0]
		>;

		// The element bounds should be relative to the clip origin
		// If clip.x = 85 (100 - 15 border padding), then element.x should be 15
		const expectedRelativeX = box0.box.x - r.clip.x;
		const expectedRelativeY = box0.box.y - r.clip.y;

		expect(bounds0.x).toBe(expectedRelativeX);
		expect(bounds0.y).toBe(expectedRelativeY);
		expect(bounds0.width).toBe(box0.box.width);
		expect(bounds0.height).toBe(box0.box.height);
	});

	it("multiple elements maintain correct relative positions", () => {
		const cfg = loadAxeScreenshotConfig({ clipPadding: 10, contextRatio: 0.2 });
		const viewport = { width: 1280, height: 720 };
		const boxes = [
			{ selector: "#btn1", box: { x: 100, y: 100, width: 80, height: 40 } },
			{ selector: "#btn2", box: { x: 200, y: 100, width: 80, height: 40 } },
		];

		const result = computeUnionClip(boxes, viewport, cfg);
		expect(result).not.toBeNull();
		const r = result as NonNullable<typeof result>;

		// Both elements should maintain their relative distance
		const elem1 = r.elementBounds[0] as NonNullable<
			(typeof r.elementBounds)[0]
		>;
		const elem2 = r.elementBounds[1] as NonNullable<
			(typeof r.elementBounds)[0]
		>;

		// Original distance between elements was 100px (200 - 100)
		expect(elem2.x - elem1.x).toBe(100);
		// Both should be at same y
		expect(elem2.y).toBe(elem1.y);
	});

	it("element bounds x/y should never be negative within screenshot", () => {
		const cfg = loadAxeScreenshotConfig({ clipPadding: 20, contextRatio: 0.5 });
		const viewport = { width: 1280, height: 720 };
		const boxes = [
			{ selector: "#edge", box: { x: 10, y: 10, width: 100, height: 50 } },
		];

		const result = computeUnionClip(boxes, viewport, cfg);
		expect(result).not.toBeNull();
		const r = result as NonNullable<typeof result>;
		const bounds0 = r.elementBounds[0] as NonNullable<
			(typeof r.elementBounds)[0]
		>;

		// Element bounds should be non-negative (within the screenshot)
		expect(bounds0.x).toBeGreaterThanOrEqual(0);
		expect(bounds0.y).toBeGreaterThanOrEqual(0);
	});

	it("element should be fully contained within the clip region", () => {
		const cfg = loadAxeScreenshotConfig({ clipPadding: 15, contextRatio: 0.3 });
		const viewport = { width: 1280, height: 720 };
		const boxes = [
			{
				selector: "#contained",
				box: { x: 300, y: 200, width: 150, height: 80 },
			},
		];

		const result = computeUnionClip(boxes, viewport, cfg);
		expect(result).not.toBeNull();
		const r = result as NonNullable<typeof result>;

		const elem = r.elementBounds[0] as NonNullable<(typeof r.elementBounds)[0]>;
		const clip = r.clip;

		// Element right edge should be within clip width
		expect(elem.x + elem.width).toBeLessThanOrEqual(clip.width);
		// Element bottom edge should be within clip height
		expect(elem.y + elem.height).toBeLessThanOrEqual(clip.height);
	});
});
