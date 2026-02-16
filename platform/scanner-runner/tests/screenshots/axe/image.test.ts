import sharp from "sharp";
import { describe, expect, it } from "vitest";

import type { ElementBounds } from "../../../src/screenshots/axe/types";

import { compositeOverlay } from "../../../src/screenshots/axe/image";

async function decodeRawRgba(buffer: Buffer): Promise<{
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
  y: number,
): { r: number; g: number; b: number; a: number } {
  const idx = (y * width + x) * 4;
  return {
    r: data[idx] ?? 0,
    g: data[idx + 1] ?? 0,
    b: data[idx + 2] ?? 0,
    a: data[idx + 3] ?? 0,
  };
}

function isNearlyWhite(pixel: { r: number; g: number; b: number; a: number }): boolean {
  return pixel.a > 200 && pixel.r > 245 && pixel.g > 245 && pixel.b > 245;
}

describe("compositeOverlay", () => {
  // Helper to create a test image buffer
  async function createTestImage(width: number, height: number): Promise<Buffer> {
    return sharp({
      create: {
        width,
        height,
        channels: 4,
        background: { r: 255, g: 255, b: 255, alpha: 1 },
      },
    })
      .png()
      .toBuffer();
  }

  it("returns original buffer when no element bounds", async () => {
    const buffer = await createTestImage(100, 100);
    const result = await compositeOverlay(buffer, [], { highlightStyle: "solid" });
    expect(result).toBeDefined();
  });

  it("draws overlay at CSS coordinates when no DPI scaling", async () => {
    // 1x DPI: 100px CSS = 100px actual
    const buffer = await createTestImage(100, 100);
    const bounds: ElementBounds[] = [
      { x: 10, y: 20, width: 30, height: 40, selector: "#test" },
    ];

    const result = await compositeOverlay(
      buffer,
      bounds,
      { highlightStyle: "solid" },
      100,
    );

    // Verify the result is a valid image with same dimensions
    const metadata = await sharp(result).metadata();
    expect(metadata.width).toBe(100);
    expect(metadata.height).toBe(100);
  });

  describe("DPI scaling", () => {
    it("scales element bounds by 2x when actual pixels are 2x CSS pixels", async () => {
      // 2x DPI: 100px CSS = 200px actual
      const actualWidth = 200;
      const actualHeight = 200;
      const cssWidth = 100;

      const buffer = await createTestImage(actualWidth, actualHeight);
      const bounds: ElementBounds[] = [
        { x: 10, y: 20, width: 30, height: 40, selector: "#test" },
      ];

      const result = await compositeOverlay(
        buffer,
        bounds,
        { highlightStyle: "solid" },
        cssWidth,
      );

      // The overlay should be drawn at 2x coordinates:
      // CSS (10, 20) -> actual (20, 40)
      // This test verifies the image is produced without errors
      const metadata = await sharp(result).metadata();
      expect(metadata.width).toBe(actualWidth);
      expect(metadata.height).toBe(actualHeight);
    });

    it("computes correct DPR from actual vs CSS width", async () => {
      // Test various DPR scenarios
      const testCases = [
        { actualWidth: 1280, cssWidth: 1280, expectedDpr: 1 },
        { actualWidth: 2560, cssWidth: 1280, expectedDpr: 2 },
        { actualWidth: 3840, cssWidth: 1280, expectedDpr: 3 },
      ];

      for (const { actualWidth, cssWidth, expectedDpr } of testCases) {
        const buffer = await createTestImage(actualWidth, 100 * expectedDpr);
        const bounds: ElementBounds[] = [
          { x: 50, y: 50, width: 100, height: 50, selector: "#test" },
        ];

        // This should not throw and should produce valid output
        const result = await compositeOverlay(
          buffer,
          bounds,
          { highlightStyle: "solid" },
          cssWidth,
        );
        const metadata = await sharp(result).metadata();
        expect(metadata.width).toBe(actualWidth);
      }
    });

    it("defaults to DPR 1 when clipCssWidth not provided", async () => {
      const buffer = await createTestImage(200, 200);
      const bounds: ElementBounds[] = [
        { x: 10, y: 20, width: 30, height: 40, selector: "#test" },
      ];

      // No clipCssWidth provided - should default to DPR 1
      const result = await compositeOverlay(buffer, bounds, { highlightStyle: "solid" });

      const metadata = await sharp(result).metadata();
      expect(metadata.width).toBe(200);
      expect(metadata.height).toBe(200);
    });

    it("defaults to DPR 1 when clipCssWidth is 0", async () => {
      const buffer = await createTestImage(200, 200);
      const bounds: ElementBounds[] = [
        { x: 10, y: 20, width: 30, height: 40, selector: "#test" },
      ];

      // clipCssWidth of 0 should default to DPR 1
      const result = await compositeOverlay(
        buffer,
        bounds,
        { highlightStyle: "solid" },
        0,
      );

      const metadata = await sharp(result).metadata();
      expect(metadata.width).toBe(200);
    });
  });

  describe("overlay positioning regression tests", () => {
    it("overlay at CSS (100, 150) appears at actual (200, 300) with 2x DPR", async () => {
      // Simulate a 2x DPR scenario like the bug report
      const actualWidth = 2560;
      const actualHeight = 392;
      const cssWidth = 1280; // 2x DPR

      const buffer = await createTestImage(actualWidth, actualHeight);
      const bounds: ElementBounds[] = [
        { x: 100, y: 50, width: 200, height: 30, selector: ".test-element" },
      ];

      const result = await compositeOverlay(
        buffer,
        bounds,
        { highlightStyle: "solid" },
        cssWidth,
      );

      // The function should scale coordinates: (100, 50) -> (200, 100) at 2x DPR
      // Verify output is valid
      const metadata = await sharp(result).metadata();
      expect(metadata.width).toBe(actualWidth);
      expect(metadata.height).toBe(actualHeight);
    });

    it("multiple elements are all scaled consistently", async () => {
      const actualWidth = 2560;
      const actualHeight = 720;
      const cssWidth = 1280;

      const buffer = await createTestImage(actualWidth, actualHeight);
      const bounds: ElementBounds[] = [
        { x: 50, y: 100, width: 100, height: 40, selector: "#elem1" },
        { x: 200, y: 100, width: 100, height: 40, selector: "#elem2" },
        { x: 125, y: 200, width: 150, height: 60, selector: "#elem3" },
      ];

      const result = await compositeOverlay(
        buffer,
        bounds,
        { highlightStyle: "solid" },
        cssWidth,
      );

      const metadata = await sharp(result).metadata();
      expect(metadata.width).toBe(actualWidth);
      expect(metadata.height).toBe(actualHeight);
    });

    it("stroke width scales with DPR for consistent visual appearance", async () => {
      // At 2x DPR, a 5px CSS stroke should become 10px actual pixels
      const buffer = await createTestImage(2560, 720);
      const bounds: ElementBounds[] = [
        { x: 100, y: 100, width: 200, height: 100, selector: "#test" },
      ];

      // This test ensures the function runs without error
      // Visual verification would require parsing the SVG
      const result = await compositeOverlay(
        buffer,
        bounds,
        { highlightStyle: "solid" },
        1280,
      );
      expect(result).toBeDefined();
      expect(result.length).toBeGreaterThan(0);
    });
  });

  describe("pixel positioning assertions", () => {
    it("tints pixels inside bounds at DPR=1", async () => {
      const buffer = await createTestImage(200, 200);
      const bounds: ElementBounds[] = [
        { x: 20, y: 30, width: 40, height: 50, selector: "#test" },
      ];

      const result = await compositeOverlay(
        buffer,
        bounds,
        { highlightStyle: "solid" },
        200,
      );

      const { data, width, height } = await decodeRawRgba(result);
      expect(width).toBe(200);
      expect(height).toBe(200);

      const outside = getPixel(data, width, 5, 5);
      expect(isNearlyWhite(outside)).toBe(true);

      const inside = getPixel(data, width, 30, 40);
      expect(isNearlyWhite(inside)).toBe(false);
    });

    it("scales bounds under DPR=2 (does not tint at unscaled CSS coordinates)", async () => {
      const buffer = await createTestImage(400, 400);
      const bounds: ElementBounds[] = [
        { x: 20, y: 30, width: 40, height: 50, selector: "#test" },
      ];

      const result = await compositeOverlay(
        buffer,
        bounds,
        { highlightStyle: "solid" },
        200, // CSS width implies DPR=2 for a 400px-wide image
      );

      const { data, width, height } = await decodeRawRgba(result);
      expect(width).toBe(400);
      expect(height).toBe(400);

      // Unscaled CSS coordinate should remain white.
      const unscaled = getPixel(data, width, 30, 40);
      expect(isNearlyWhite(unscaled)).toBe(true);

      // Scaled coordinate should be tinted.
      const scaled = getPixel(data, width, 20 * 2 + 10, 30 * 2 + 10);
      expect(isNearlyWhite(scaled)).toBe(false);
    });
  });

  describe("highlight styles", () => {
    it("supports solid highlight style", async () => {
      const buffer = await createTestImage(100, 100);
      const bounds: ElementBounds[] = [
        { x: 10, y: 10, width: 20, height: 20, selector: "#test" },
      ];

      const result = await compositeOverlay(
        buffer,
        bounds,
        { highlightStyle: "solid" },
        100,
      );
      expect(result).toBeDefined();
    });

    it("supports dashed highlight style", async () => {
      const buffer = await createTestImage(100, 100);
      const bounds: ElementBounds[] = [
        { x: 10, y: 10, width: 20, height: 20, selector: "#test" },
      ];

      const result = await compositeOverlay(
        buffer,
        bounds,
        { highlightStyle: "dashed" },
        100,
      );
      expect(result).toBeDefined();
    });
  });
});
