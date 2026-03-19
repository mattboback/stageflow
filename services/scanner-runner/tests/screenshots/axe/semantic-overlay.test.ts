/**
 * Semantic Overlay Tests
 */

import type { Page } from "playwright";

import { beforeEach, describe, expect, it, vi } from "vitest";

import type { AxeScreenshotConfig } from "../../../src/screenshots/axe/types";

import { captureSemanticOverlayScreenshot } from "../../../src/screenshots/axe/semantic-overlay";

vi.mock("node:fs", () => ({
  existsSync: vi.fn().mockReturnValue(true),
  mkdirSync: vi.fn(),
}));

vi.mock("uuid", () => ({
  v4: vi.fn().mockReturnValue("test-uuid-1234"),
}));

vi.mock("../../../src/screenshots/axe/image", () => ({
  saveScreenshot: vi.fn().mockResolvedValue(undefined),
  generateThumbnail: vi.fn().mockResolvedValue("thumb-test-uuid.png"),
}));

vi.mock("../../../src/screenshots/axe/targets", () => ({
  extractAxeViolationTargets: vi.fn().mockReturnValue({
    targets: ["button.bad", "a.empty"],
    selector: "button.bad",
  }),
}));

const DEFAULT_CONFIG: AxeScreenshotConfig = {
  screenshotsEnabled: true,
  injectHighlight: true,
  shotScale: 1,
  shotMinWidth: 200,
  shotMinHeight: 150,
  deviceScaleFactor: 1,
  highlightStyle: "dashed",
  mergeTargets: true,
  unionMaxTargets: 10,
  clipPadding: 20,
  contextRatio: 0.5,
  elementContextMultiplier: 2.5,
  elementContextMinSize: 100,
  scrollTimeout: 2000,
  thumbnailSize: 200,
  generateThumbnails: true,
  outputFormat: "png",
  webpQuality: 80,
  overlayMethod: "css-injection",
  semanticOverlayEnabled: true,
  semanticOverlayMaxHeadings: 20,
  semanticOverlayMaxLabelLength: 40,
  semanticOverlayLegendEnabled: true,
};

const createConfig = (
  overrides: Partial<AxeScreenshotConfig> = {},
): AxeScreenshotConfig => ({
  ...DEFAULT_CONFIG,
  ...overrides,
  elementContextMultiplier:
    overrides.elementContextMultiplier ?? DEFAULT_CONFIG.elementContextMultiplier,
  elementContextMinSize:
    overrides.elementContextMinSize ?? DEFAULT_CONFIG.elementContextMinSize,
});

const createMockPage = (evaluateResult = 5): Page => {
  return {
    evaluate: vi.fn().mockResolvedValue(evaluateResult),
    waitForTimeout: vi.fn().mockResolvedValue(undefined),
    screenshot: vi.fn().mockResolvedValue(Buffer.from("fake-png")),
  } as unknown as Page;
};

describe("captureSemanticOverlayScreenshot", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("config checks", () => {
    it("returns null when screenshotsEnabled is false", async () => {
      const mockPage = createMockPage();
      const config = createConfig({ screenshotsEnabled: false });

      const result = await captureSemanticOverlayScreenshot({
        page: mockPage,
        violation: { id: "heading-order" },
        resultsDir: "/tmp/results",
        cfg: config,
      });

      expect(result).toBeNull();
      expect(mockPage.evaluate).not.toHaveBeenCalled();
    });

    it("returns null when semanticOverlayEnabled is false", async () => {
      const mockPage = createMockPage();
      const config = createConfig({ semanticOverlayEnabled: false });

      const result = await captureSemanticOverlayScreenshot({
        page: mockPage,
        violation: { id: "heading-order" },
        resultsDir: "/tmp/results",
        cfg: config,
      });

      expect(result).toBeNull();
    });
  });

  describe("rule matching", () => {
    it("injects heading overlay for heading-order rule", async () => {
      const mockPage = createMockPage(3);
      const config = createConfig();

      const result = await captureSemanticOverlayScreenshot({
        page: mockPage,
        violation: { id: "heading-order" },
        resultsDir: "/tmp/results",
        cfg: config,
      });

      expect(mockPage.evaluate).toHaveBeenCalled();
      expect(result).toBeDefined();
      expect(result?.screenshot).toContain("semantic-heading-order");
    });

    it("injects heading overlay for empty-heading rule", async () => {
      const mockPage = createMockPage(2);
      const config = createConfig();

      const result = await captureSemanticOverlayScreenshot({
        page: mockPage,
        violation: { id: "empty-heading" },
        resultsDir: "/tmp/results",
        cfg: config,
      });

      expect(result).toBeDefined();
      expect(result?.screenshot).toContain("semantic-empty-heading");
    });

    it("injects target overlay for link-name rule", async () => {
      const mockPage = createMockPage(2);
      const config = createConfig();

      const result = await captureSemanticOverlayScreenshot({
        page: mockPage,
        violation: { id: "link-name", nodes: [{ target: ["a.empty"] }] },
        resultsDir: "/tmp/results",
        cfg: config,
      });

      expect(result).toBeDefined();
      expect(result?.screenshot).toContain("semantic-link-name");
    });

    it("injects target overlay for button-name rule", async () => {
      const mockPage = createMockPage(1);
      const config = createConfig();

      const result = await captureSemanticOverlayScreenshot({
        page: mockPage,
        violation: { id: "button-name", nodes: [{ target: ["button.bad"] }] },
        resultsDir: "/tmp/results",
        cfg: config,
      });

      expect(result).toBeDefined();
      expect(result?.screenshot).toContain("semantic-button-name");
    });

    it("injects target overlay for label rule", async () => {
      const mockPage = createMockPage(1);
      const config = createConfig();

      const result = await captureSemanticOverlayScreenshot({
        page: mockPage,
        violation: { id: "label", nodes: [{ target: ["input#email"] }] },
        resultsDir: "/tmp/results",
        cfg: config,
      });

      expect(result).toBeDefined();
      expect(result?.screenshot).toContain("semantic-label");
    });

    it("returns null for unsupported rules", async () => {
      const mockPage = createMockPage();
      const config = createConfig();

      const result = await captureSemanticOverlayScreenshot({
        page: mockPage,
        violation: { id: "color-contrast" },
        resultsDir: "/tmp/results",
        cfg: config,
      });

      expect(result).toBeNull();
    });

    it("returns null for unknown rule IDs", async () => {
      const mockPage = createMockPage();
      const config = createConfig();

      const result = await captureSemanticOverlayScreenshot({
        page: mockPage,
        violation: { id: "some-unknown-rule" },
        resultsDir: "/tmp/results",
        cfg: config,
      });

      expect(result).toBeNull();
    });
  });

  describe("directory handling", () => {
    it("creates results directory if it does not exist", async () => {
      const { existsSync, mkdirSync } = await import("node:fs");
      (existsSync as ReturnType<typeof vi.fn>).mockReturnValueOnce(false);

      const mockPage = createMockPage(2);
      const config = createConfig();

      await captureSemanticOverlayScreenshot({
        page: mockPage,
        violation: { id: "heading-order" },
        resultsDir: "/tmp/new-results",
        cfg: config,
      });

      expect(mkdirSync).toHaveBeenCalledWith("/tmp/new-results", { recursive: true });
    });

    it("does not create directory if it already exists", async () => {
      const { existsSync, mkdirSync } = await import("node:fs");
      (existsSync as ReturnType<typeof vi.fn>).mockReturnValueOnce(true);

      const mockPage = createMockPage(2);
      const config = createConfig();

      await captureSemanticOverlayScreenshot({
        page: mockPage,
        violation: { id: "heading-order" },
        resultsDir: "/tmp/existing-results",
        cfg: config,
      });

      expect(mkdirSync).not.toHaveBeenCalled();
    });
  });

  describe("overlay count handling", () => {
    it("returns null when overlay count is 0", async () => {
      const mockPage = createMockPage(0);
      const config = createConfig();

      const result = await captureSemanticOverlayScreenshot({
        page: mockPage,
        violation: { id: "heading-order" },
        resultsDir: "/tmp/results",
        cfg: config,
      });

      expect(result).toBeNull();
    });

    it("proceeds when overlay count is positive", async () => {
      const mockPage = createMockPage(5);
      const config = createConfig();

      const result = await captureSemanticOverlayScreenshot({
        page: mockPage,
        violation: { id: "heading-order" },
        resultsDir: "/tmp/results",
        cfg: config,
      });

      expect(result).toBeDefined();
      expect(mockPage.screenshot).toHaveBeenCalled();
    });
  });

  describe("screenshot capture", () => {
    it("takes full page screenshot after injection", async () => {
      const mockPage = createMockPage(3);
      const config = createConfig();

      await captureSemanticOverlayScreenshot({
        page: mockPage,
        violation: { id: "heading-order" },
        resultsDir: "/tmp/results",
        cfg: config,
      });

      expect(mockPage.screenshot).toHaveBeenCalledWith({ fullPage: true });
    });

    it("waits before taking screenshot", async () => {
      const mockPage = createMockPage(3);
      const config = createConfig();

      await captureSemanticOverlayScreenshot({
        page: mockPage,
        violation: { id: "heading-order" },
        resultsDir: "/tmp/results",
        cfg: config,
      });

      expect(mockPage.waitForTimeout).toHaveBeenCalledWith(80);
    });

    it("saves screenshot with correct format (png)", async () => {
      const { saveScreenshot } = await import("../../../src/screenshots/axe/image");
      const mockPage = createMockPage(2);
      const config = createConfig({ outputFormat: "png" });

      await captureSemanticOverlayScreenshot({
        page: mockPage,
        violation: { id: "heading-order" },
        resultsDir: "/tmp/results",
        cfg: config,
      });

      expect(saveScreenshot).toHaveBeenCalledWith(
        expect.any(Buffer),
        expect.stringContaining(".png"),
        expect.any(Object),
      );
    });

    it("saves screenshot with webp format when configured", async () => {
      const { saveScreenshot } = await import("../../../src/screenshots/axe/image");
      const mockPage = createMockPage(2);
      const config = createConfig({ outputFormat: "webp" });

      await captureSemanticOverlayScreenshot({
        page: mockPage,
        violation: { id: "heading-order" },
        resultsDir: "/tmp/results",
        cfg: config,
      });

      expect(saveScreenshot).toHaveBeenCalledWith(
        expect.any(Buffer),
        expect.stringContaining(".webp"),
        expect.any(Object),
      );
    });
  });

  describe("cleanup", () => {
    it("removes overlay after capture (success)", async () => {
      const mockPage = createMockPage(3);
      const config = createConfig();

      await captureSemanticOverlayScreenshot({
        page: mockPage,
        violation: { id: "heading-order" },
        resultsDir: "/tmp/results",
        cfg: config,
      });

      expect(mockPage.evaluate).toHaveBeenCalledTimes(2);
    });

    it("removes overlay even on screenshot error", async () => {
      const mockPage = {
        evaluate: vi.fn().mockResolvedValue(3),
        waitForTimeout: vi.fn().mockResolvedValue(undefined),
        screenshot: vi.fn().mockRejectedValue(new Error("Screenshot failed")),
      } as unknown as Page;
      const config = createConfig();

      const result = await captureSemanticOverlayScreenshot({
        page: mockPage,
        violation: { id: "heading-order" },
        resultsDir: "/tmp/results",
        cfg: config,
      });

      expect(mockPage.evaluate).toHaveBeenCalledTimes(2);
      expect(result).toBeNull();
    });
  });

  describe("result structure", () => {
    it("returns screenshot filename and thumbnail", async () => {
      const mockPage = createMockPage(3);
      const config = createConfig();

      const result = await captureSemanticOverlayScreenshot({
        page: mockPage,
        violation: { id: "heading-order" },
        resultsDir: "/tmp/results",
        cfg: config,
      });

      expect(result).toMatchObject({
        screenshot: expect.stringContaining("semantic-heading-order"),
        thumbnail: expect.any(String),
      });
    });

    it("includes uuid in filename for uniqueness", async () => {
      const mockPage = createMockPage(2);
      const config = createConfig();

      const result = await captureSemanticOverlayScreenshot({
        page: mockPage,
        violation: { id: "heading-order" },
        resultsDir: "/tmp/results",
        cfg: config,
      });

      expect(result?.screenshot).toContain("test-uuid-1234");
    });
  });

  describe("error handling", () => {
    it("propagates injection error", async () => {
      const mockPage = {
        evaluate: vi.fn().mockRejectedValue(new Error("Injection failed")),
        waitForTimeout: vi.fn(),
        screenshot: vi.fn(),
      } as unknown as Page;
      const config = createConfig();

      await expect(
        captureSemanticOverlayScreenshot({
          page: mockPage,
          violation: { id: "heading-order" },
          resultsDir: "/tmp/results",
          cfg: config,
        }),
      ).rejects.toThrow("Injection failed");
    });
  });

  describe("config options", () => {
    it("respects semanticOverlayMaxHeadings config", async () => {
      const mockPage = createMockPage(5);
      const config = createConfig({ semanticOverlayMaxHeadings: 3 });

      await captureSemanticOverlayScreenshot({
        page: mockPage,
        violation: { id: "heading-order" },
        resultsDir: "/tmp/results",
        cfg: config,
      });

      expect(mockPage.evaluate).toHaveBeenCalledWith(
        expect.any(Function),
        expect.objectContaining({
          maxHeadings: 3,
        }),
      );
    });

    it("respects semanticOverlayMaxLabelLength config", async () => {
      const mockPage = createMockPage(5);
      const config = createConfig({ semanticOverlayMaxLabelLength: 20 });

      await captureSemanticOverlayScreenshot({
        page: mockPage,
        violation: { id: "heading-order" },
        resultsDir: "/tmp/results",
        cfg: config,
      });

      expect(mockPage.evaluate).toHaveBeenCalledWith(
        expect.any(Function),
        expect.objectContaining({
          maxLabelLength: 20,
        }),
      );
    });

    it("respects semanticOverlayLegendEnabled config", async () => {
      const mockPage = createMockPage(5);
      const config = createConfig({ semanticOverlayLegendEnabled: false });

      await captureSemanticOverlayScreenshot({
        page: mockPage,
        violation: { id: "heading-order" },
        resultsDir: "/tmp/results",
        cfg: config,
      });

      expect(mockPage.evaluate).toHaveBeenCalledWith(
        expect.any(Function),
        expect.objectContaining({
          legendEnabled: false,
        }),
      );
    });
  });
});
