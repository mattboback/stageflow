/**
 * Location Capture Tests
 */

import type { Page } from "playwright";

import { describe, expect, it, vi } from "vitest";

import { captureLocationInfo } from "../../../src/screenshots/axe/location";

const createMockPage = (evaluateResult: unknown): Page => {
	return {
		evaluate: vi.fn().mockResolvedValue(evaluateResult),
	} as unknown as Page;
};

describe("captureLocationInfo", () => {
	describe("successful capture", () => {
		it("captures scrollY, viewportHeight, and docHeight correctly", async () => {
			const mockPage = createMockPage({
				scrollY: 500,
				viewportHeight: 720,
				docHeight: 2000,
				position: 0.43,
			});

			const result = await captureLocationInfo(mockPage);

			expect(result).toEqual({
				scrollY: 500,
				viewportHeight: 720,
				docHeight: 2000,
				position: 0.43,
			});
		});

		it("captures page at top (scrollY = 0)", async () => {
			const mockPage = createMockPage({
				scrollY: 0,
				viewportHeight: 720,
				docHeight: 1500,
				position: 0.24,
			});

			const result = await captureLocationInfo(mockPage);

			expect(result?.scrollY).toBe(0);
			expect(result?.position).toBeGreaterThanOrEqual(0);
		});

		it("captures page at bottom (position near 1)", async () => {
			const mockPage = createMockPage({
				scrollY: 1280,
				viewportHeight: 720,
				docHeight: 2000,
				position: 0.82,
			});

			const result = await captureLocationInfo(mockPage);

			expect(result?.scrollY).toBe(1280);
			expect(result?.position).toBeLessThanOrEqual(1);
		});

		it("handles short pages (docHeight <= viewportHeight)", async () => {
			const mockPage = createMockPage({
				scrollY: 0,
				viewportHeight: 720,
				docHeight: 500,
				position: 0.72,
			});

			const result = await captureLocationInfo(mockPage);

			expect(result).toBeDefined();
			expect(result?.docHeight).toBe(500);
		});
	});

	describe("position calculation", () => {
		it("position is clamped between 0 and 1", async () => {
			const mockPage = createMockPage({
				scrollY: 2000,
				viewportHeight: 720,
				docHeight: 2000,
				position: 1,
			});

			const result = await captureLocationInfo(mockPage);

			expect(result?.position).toBeLessThanOrEqual(1);
			expect(result?.position).toBeGreaterThanOrEqual(0);
		});
	});

	describe("error handling", () => {
		it("returns undefined when page.evaluate throws", async () => {
			const mockPage = {
				evaluate: vi.fn().mockRejectedValue(new Error("Page crashed")),
			} as unknown as Page;

			const result = await captureLocationInfo(mockPage);

			expect(result).toBeUndefined();
		});

		it("handles navigation errors gracefully", async () => {
			const mockPage = {
				evaluate: vi
					.fn()
					.mockRejectedValue(new Error("Navigation interrupted")),
			} as unknown as Page;

			const result = await captureLocationInfo(mockPage);

			expect(result).toBeUndefined();
		});
	});

	describe("evaluate function call", () => {
		it("calls page.evaluate with a function", async () => {
			const mockEvaluate = vi.fn().mockResolvedValue({
				scrollY: 0,
				viewportHeight: 720,
				docHeight: 1000,
				position: 0.36,
			});

			const mockPage = { evaluate: mockEvaluate } as unknown as Page;

			await captureLocationInfo(mockPage);

			expect(mockEvaluate).toHaveBeenCalledTimes(1);
			expect(mockEvaluate).toHaveBeenCalledWith(expect.any(Function));
		});
	});

	describe("edge cases", () => {
		it("handles zero document height", async () => {
			const mockPage = createMockPage({
				scrollY: 0,
				viewportHeight: 720,
				docHeight: 0,
				position: 0,
			});

			const result = await captureLocationInfo(mockPage);

			expect(result).toBeDefined();
			expect(result?.docHeight).toBe(0);
		});

		it("handles fractional scroll values", async () => {
			const mockPage = createMockPage({
				scrollY: 123.5,
				viewportHeight: 720,
				docHeight: 1500.75,
				position: 0.322,
			});

			const result = await captureLocationInfo(mockPage);

			expect(result?.scrollY).toBe(123.5);
			expect(result?.docHeight).toBe(1500.75);
		});
	});
});
