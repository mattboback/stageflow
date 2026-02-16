import sharp from "sharp";
import { describe, expect, it, vi } from "vitest";

import type { ScannerLogger } from "../../src/core/types";

import { createScreenshotService } from "../../src/core/screenshots";

type Page = import("playwright").Page;

async function createTestPng(width: number, height: number): Promise<Buffer> {
  return sharp({
    create: {
      width,
      height,
      channels: 4,
      background: { r: 10, g: 20, b: 30, alpha: 1 },
    },
  })
    .png()
    .toBuffer();
}

function createLogger(): ScannerLogger {
  return {
    info: vi.fn(),
    warn: vi.fn(),
    error: vi.fn(),
    debug: vi.fn(),
  };
}

describe("ScreenshotService", () => {
  it("captures full page screenshots (png + webp)", async () => {
    const logger = createLogger();
    const service = createScreenshotService(logger);

    const png = await createTestPng(64, 32);
    const screenshot = vi.fn().mockResolvedValue(png);

    const page = { screenshot } as unknown as Page;

    const pngResult = await service.captureFullPage(page);
    expect(pngResult.format).toBe("png");
    expect(pngResult.width).toBe(64);
    expect(pngResult.height).toBe(32);

    const webpResult = await service.captureFullPage(page, {
      format: "webp",
      quality: 75,
      scale: 2,
    });
    expect(webpResult.format).toBe("webp");
    expect(webpResult.width).toBe(64);
    expect(webpResult.height).toBe(32);

    expect(screenshot).toHaveBeenCalledWith(
      expect.objectContaining({
        fullPage: true,
        type: "png",
        scale: "css",
      }),
    );
  });

  it("logs and rethrows when full page capture fails", async () => {
    const logger = createLogger();
    const service = createScreenshotService(logger);

    const page = {
      screenshot: vi.fn().mockRejectedValue(new Error("nope")),
    } as unknown as Page;

    await expect(service.captureFullPage(page)).rejects.toThrow("nope");
    expect(logger.error).toHaveBeenCalledWith(
      "Failed to capture full page screenshot",
      expect.objectContaining({ error: "nope" }),
    );
  });

  it("captures highlights, enforces maxTargets, and always runs cleanup", async () => {
    const logger = createLogger();
    const service = createScreenshotService(logger);

    const nowSpy = vi.spyOn(Date, "now").mockReturnValue(123);
    try {
      const png = await createTestPng(80, 40);

      const addStyleTag = vi.fn().mockResolvedValue(undefined);
      const cleanupEvaluate = vi.fn().mockResolvedValue(undefined);
      const screenshot = vi.fn().mockResolvedValue(png);

      const elementOk = {
        boundingBox: vi.fn().mockResolvedValue({ x: 1, y: 2, width: 3, height: 4 }),
        evaluate: vi.fn().mockResolvedValue(undefined),
      };

      const elementNoBox = {
        boundingBox: vi.fn().mockResolvedValue(null),
        evaluate: vi.fn().mockResolvedValue(undefined),
      };

      const elementThrows = {
        boundingBox: vi.fn().mockImplementation(() => {
          throw new Error("vanished");
        }),
        evaluate: vi.fn(),
      };

      const select = vi
        .fn()
        .mockResolvedValueOnce(elementThrows)
        .mockResolvedValueOnce(elementOk)
        .mockResolvedValueOnce(elementNoBox);

      const page = {
        addStyleTag,
        evaluate: cleanupEvaluate,
        screenshot,
        $: select,
      } as unknown as Page;

      const result = await service.captureWithHighlights(
        page,
        [
          { selector: "#bad" },
          { selector: "#ok" },
          { selector: "#nobox" },
        ],
        { maxTargets: 2 },
      );

      expect(select).toHaveBeenCalledTimes(2);
      expect(addStyleTag).toHaveBeenCalledWith(
        expect.objectContaining({
          content: expect.stringContaining(".stageflow-highlight-123"),
        }),
      );

      expect(result.highlightedElements).toHaveLength(1);
      expect(result.highlightedElements[0]!.selector).toBe("#ok");
      expect(result.highlightedElements[0]!.visible).toBe(true);

      expect(cleanupEvaluate).toHaveBeenCalledTimes(1);
      expect(cleanupEvaluate).toHaveBeenCalledWith(expect.any(Function), "stageflow-highlight-123");
    } finally {
      nowSpy.mockRestore();
    }
  });

  it("runs cleanup even if screenshot capture throws", async () => {
    const logger = createLogger();
    const service = createScreenshotService(logger);

    const nowSpy = vi.spyOn(Date, "now").mockReturnValue(999);
    try {
      const addStyleTag = vi.fn().mockResolvedValue(undefined);
      const cleanupEvaluate = vi.fn().mockResolvedValue(undefined);
      const screenshot = vi.fn().mockRejectedValue(new Error("boom"));

      const element = {
        boundingBox: vi.fn().mockResolvedValue({ x: 1, y: 2, width: 3, height: 4 }),
        evaluate: vi.fn().mockResolvedValue(undefined),
      };

      const page = {
        addStyleTag,
        evaluate: cleanupEvaluate,
        screenshot,
        $: vi.fn().mockResolvedValue(element),
      } as unknown as Page;

      await expect(service.captureWithHighlights(page, [{ selector: "#ok" }])).rejects.toThrow(
        "boom",
      );
      expect(cleanupEvaluate).toHaveBeenCalledTimes(1);
    } finally {
      nowSpy.mockRestore();
    }
  });
});

