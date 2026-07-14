import { describe, expect, it } from 'vitest';

import { getCroppedViewBox } from './screenshot-crop';

const PAGE_W = 1280;
const PAGE_H = 4000;

describe('getCroppedViewBox', () => {
	it('returns null for invalid page dimensions', () => {
		const el = { x: 10, y: 10, width: 50, height: 20 };
		expect(getCroppedViewBox(0, PAGE_H, el)).toBeNull();
		expect(getCroppedViewBox(PAGE_W, Number.NaN, el)).toBeNull();
	});

	it('keeps the crop tight around a tiny element', () => {
		const el = { x: 600, y: 2000, width: 12, height: 12 };
		const box = getCroppedViewBox(PAGE_W, PAGE_H, el)!;

		// Floored at the legibility minimum, nowhere near the old 520px window.
		expect(box.width).toBe(200);
		expect(box.height).toBe(120);
		// Element stays centered.
		expect(box.x + box.width / 2).toBeCloseTo(606);
		expect(box.y + box.height / 2).toBeCloseTo(2006);
	});

	it('scales context with a mid-sized element instead of a fixed window', () => {
		const el = { x: 100, y: 500, width: 300, height: 80 };
		const box = getCroppedViewBox(PAGE_W, PAGE_H, el)!;

		// padding = clamp(0.5 * 300, 16, 64) = 64
		expect(box.width).toBe(300 + 128);
		expect(box.height).toBe(80 + 128);
		expect(box.x).toBeLessThanOrEqual(el.x);
		expect(box.x + box.width).toBeGreaterThanOrEqual(el.x + el.width);
	});

	it('caps extreme aspect ratios by expanding the short axis', () => {
		const strip = { x: 0, y: 100, width: 1200, height: 4 };
		const box = getCroppedViewBox(PAGE_W, PAGE_H, strip)!;
		expect(box.width / box.height).toBeLessThanOrEqual(3.001);
	});

	it('contains a full-page element within page bounds', () => {
		const el = { x: 0, y: 0, width: PAGE_W, height: PAGE_H };
		const box = getCroppedViewBox(PAGE_W, PAGE_H, el)!;
		expect(box).toEqual({ x: 0, y: 0, width: PAGE_W, height: PAGE_H });
	});

	it('clamps crops at page edges without spilling outside', () => {
		const el = { x: 1260, y: 3990, width: 30, height: 20 };
		const box = getCroppedViewBox(PAGE_W, PAGE_H, el)!;
		expect(box.x).toBeGreaterThanOrEqual(0);
		expect(box.y).toBeGreaterThanOrEqual(0);
		expect(box.x + box.width).toBeLessThanOrEqual(PAGE_W);
		expect(box.y + box.height).toBeLessThanOrEqual(PAGE_H);
	});
});
