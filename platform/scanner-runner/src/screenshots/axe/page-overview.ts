import type { Page } from "playwright";

import { createHash } from "node:crypto";
import { existsSync, mkdirSync } from "node:fs";
import { join } from "node:path";
import sharp from "sharp";

import type { IssueSeverity } from "../../core/types";
import type {
  AxeScreenshotConfig,
  PageOverviewDiagnostics,
  PageOverviewElement,
  PageOverviewResult,
  PageOverviewViolation,
} from "./types";

import { getEnvBool, getEnvInt } from "../../utils/env";

const DEBUG_SCREENSHOTS = getEnvBool("A11Y_DEBUG_SCREENSHOTS", false);

function debugLog(message: string, data?: unknown): void {
  if (!DEBUG_SCREENSHOTS) {
    return;
  }
  if (data !== undefined) {
    console.log(`[PageOverview] ${message}`, JSON.stringify(data, null, 2));
  } else {
    console.log(`[PageOverview] ${message}`);
  }
}
import { normalizeSeverity } from "../../utils/severity";

/**
 * Generate the same issue fingerprint used by WebServerFormatter.
 * This ensures overlay element IDs match the final report issue IDs.
 */
function generateIssueFingerprint(
  scanner: string,
  ruleId: string,
  pageId: string,
  selector: string,
): string {
  const data = `${scanner}|${ruleId}|${pageId}|${selector}`.toLowerCase();
  return createHash("sha256").update(data).digest("hex").slice(0, 12);
}

export function loadPageOverviewConfig(
  overrides?: Partial<{
    enabled: boolean;
    maxElements: number;
    maxHeight: number;
  }>,
): {
  enabled: boolean;
  maxElements: number;
  maxHeight: number;
} {
  return {
    enabled: getEnvBool("A11Y_PAGE_OVERVIEW_ENABLED", true),
    maxElements: Math.max(0, getEnvInt("A11Y_PAGE_OVERVIEW_MAX_ELEMENTS", 50)),
    maxHeight: Math.max(0, getEnvInt("A11Y_PAGE_OVERVIEW_MAX_HEIGHT", 5000)),
    ...overrides,
  };
}

async function freezePageForScreenshot(page: Page): Promise<void> {
  // Best-effort: stabilize layouts by pausing animations/transitions. Many sites use carousels
  // (e.g. CSS transforms) that can move between the screenshot capture and bounding-box lookup.
  try {
    await page.addStyleTag({
      content: `
        *,
        *::before,
        *::after {
          transition: none !important;
          animation: none !important;
          scroll-behavior: auto !important;
        }
      `,
    });
  } catch {
    // Ignore CSP/style injection failures.
  }

  try {
    await page.evaluate(() => {
      try {
        for (const anim of document.getAnimations()) {
          anim.pause();
        }
      } catch {
        // ignore
      }
    });
  } catch {
    // Ignore evaluate failures (cross-origin frames, detached page, etc).
  }
}

export function computeScreenshotScaleFactor(
  actualPixels: number,
  cssPixels: number,
): number {
  if (actualPixels <= 0 || cssPixels <= 0) {
    return 1;
  }

  const scale = actualPixels / cssPixels;
  if (!Number.isFinite(scale) || scale <= 0) {
    return 1;
  }

  return scale;
}

interface BoundingBox { x: number; y: number; width: number; height: number }

export function clipPageOverviewBounds(
  bounds: BoundingBox,
  maxWidth: number,
  maxHeight: number,
): BoundingBox | null {
  if (maxWidth <= 0 || maxHeight <= 0) {
    return null;
  }

  if (
    !Number.isFinite(bounds.x) ||
    !Number.isFinite(bounds.y) ||
    !Number.isFinite(bounds.width) ||
    !Number.isFinite(bounds.height)
  ) {
    return null;
  }

  if (bounds.width <= 0 || bounds.height <= 0) {
    return null;
  }

  const right = bounds.x + bounds.width;
  const bottom = bounds.y + bounds.height;

  if (right <= 0 || bottom <= 0) {
    return null;
  }

  if (bounds.x >= maxWidth || bounds.y >= maxHeight) {
    return null;
  }

  const clampedX = Math.max(0, bounds.x);
  const clampedY = Math.max(0, bounds.y);
  const clampedRight = Math.min(maxWidth, right);
  const clampedBottom = Math.min(maxHeight, bottom);

  const width = clampedRight - clampedX;
  const height = clampedBottom - clampedY;

  if (width <= 0 || height <= 0) {
    return null;
  }

  return { x: clampedX, y: clampedY, width, height };
}

function clampPercent(value: number): number {
  if (!Number.isFinite(value)) {
    return 0;
  }

  return Math.min(100, Math.max(0, value));
}

function roundPercent(value: number): number {
  return Math.round(value * 100) / 100;
}

export function collectPageOverviewTargets(
  violations: PageOverviewViolation[],
  maxElements: number,
  options?: { pageId?: string; scanner?: string },
): {
  issueId: string;
  ruleId: string;
  severity: IssueSeverity;
  selector: string;
  nodeIndex: number;
}[] {
  const { pageId = "page", scanner = "axe" } = options ?? {};
  const targets: {
    issueId: string;
    ruleId: string;
    severity: IssueSeverity;
    selector: string;
    nodeIndex: number;
  }[] = [];
  if (violations.length === 0 || maxElements <= 0) {
    return targets;
  }

  const maxNodesPerViolation = 5; // Keep in sync with WebServerFormatter occurrences slice.
  let elementCount = 0;

  for (const violation of violations) {
    const ruleId = violation.id || "unknown";
    const severity = normalizeSeverity(violation.impact, "minor");
    const nodes = violation.nodes ?? [];

    const primarySelectorRaw = nodes[0]?.target?.[0];
    const primarySelector = typeof primarySelectorRaw === "string" ? primarySelectorRaw : "";

    // WebServerFormatter generates issue IDs using the FIRST node selector/target.
    // Keep all overlay elements for the same rule mapped to that single issue.
    const issueId = generateIssueFingerprint(scanner, ruleId, pageId, primarySelector);

    for (
      let nodeIndex = 0;
      nodeIndex < nodes.length && nodeIndex < maxNodesPerViolation;
      nodeIndex++
    ) {
      if (elementCount >= maxElements) {
        return targets;
      }

      const node = nodes[nodeIndex];
      const selector = (node?.target ?? [])
        .map((t) => (typeof t === "string" ? t.trim() : String(t).trim()))
        .find((t) => t.length > 0);

      if (!selector) {
        continue;
      }

      targets.push({
        issueId,
        ruleId,
        severity,
        selector,
        nodeIndex,
      });
      elementCount += 1;
    }
  }

  return targets;
}

export async function capturePageOverviewRaw(
  page: Page,
  violations: PageOverviewViolation[],
  resultsDir: string,
  pageId: string,
  scannerId: string,
  screenshotCfg: AxeScreenshotConfig,
  overviewCfg: {
    enabled: boolean;
    maxElements: number;
    maxHeight: number;
    skipLargeElements?: boolean;
    largeElementWidthRatio?: number;
    largeElementHeightRatio?: number;
  },
): Promise<(PageOverviewResult & { buffer: Buffer }) | null> {
  if (!overviewCfg.enabled || !screenshotCfg.screenshotsEnabled) {
    return null;
  }

  if (violations.length === 0) {
    return null;
  }

  const elementTargets = collectPageOverviewTargets(violations, overviewCfg.maxElements, {
    pageId,
    scanner: scannerId,
  });

  if (!existsSync(resultsDir)) {
    mkdirSync(resultsDir, { recursive: true });
  }

  const ext = screenshotCfg.outputFormat === "webp" ? ".webp" : ".png";
  const screenshotFilename = `page-overview-${pageId}${ext}`;
  const screenshotPath = join(resultsDir, screenshotFilename);

  // Initialize diagnostics collection
  const diagnostics: PageOverviewDiagnostics = {
    cssPageWidth: 0,
    cssPageHeight: 0,
    screenshotWidth: 0,
    screenshotHeight: 0,
    scaleX: 1,
    scaleY: 1,
    devicePixelRatio: 1,
    captureHeight: 0,
    elementCount: elementTargets.length,
    elements: [],
  };

  try {
    // Step 1: Scroll to top and stabilize
    await page.evaluate(() => { window.scrollTo(0, 0); });
    await page.waitForTimeout(100);

    // Step 2: Freeze animations BEFORE collecting any measurements
    await freezePageForScreenshot(page);
    await page.waitForTimeout(50);

    // Step 3: Get page dimensions and devicePixelRatio
    const pageInfo = await page.evaluate(() => {
      const doc = document.documentElement;
      return {
        width: Math.max(doc.scrollWidth, doc.clientWidth),
        height: Math.max(doc.scrollHeight, doc.clientHeight),
        devicePixelRatio: window.devicePixelRatio || 1,
      };
    });

    const captureHeight = Math.min(pageInfo.height, overviewCfg.maxHeight);

    diagnostics.cssPageWidth = pageInfo.width;
    diagnostics.cssPageHeight = pageInfo.height;
    diagnostics.captureHeight = captureHeight;
    diagnostics.devicePixelRatio = pageInfo.devicePixelRatio;

    debugLog("Page dimensions", {
      cssWidth: pageInfo.width,
      cssHeight: pageInfo.height,
      captureHeight,
      devicePixelRatio: pageInfo.devicePixelRatio,
    });

    // Step 4: CRITICAL - Collect ALL bounding boxes BEFORE taking screenshot
    // This prevents layout drift between screenshot capture and coordinate collection
    const rawBoundingBoxes = new Map<number, BoundingBox | null>();

    for (const [i, target] of elementTargets.entries()) {
      try {
        const locator = page.locator(target.selector).first();
        const box = await locator.boundingBox();
        rawBoundingBoxes.set(i, box);

        if (DEBUG_SCREENSHOTS) {
          diagnostics.elements.push({
            selector: target.selector,
            rawBox: box ? { x: box.x, y: box.y, width: box.width, height: box.height } : null,
            scaledBox: null,
            percentBounds: null,
            skipped: false,
          });
        }
      } catch {
        rawBoundingBoxes.set(i, null);
        if (DEBUG_SCREENSHOTS) {
          diagnostics.elements.push({
            selector: target.selector,
            rawBox: null,
            scaledBox: null,
            percentBounds: null,
            skipped: true,
            skipReason: "locator_error",
          });
        }
      }
    }

    debugLog(`Collected ${rawBoundingBoxes.size} bounding boxes BEFORE screenshot`);

    // Step 5: Take the screenshot (after bounding boxes are captured)
    const screenshotBuffer = await page.screenshot({
      fullPage: true,
      clip:
        captureHeight < pageInfo.height
          ? { x: 0, y: 0, width: pageInfo.width, height: captureHeight }
          : undefined,
    });

    // Step 6: Get screenshot actual dimensions
    const metadata = await sharp(screenshotBuffer).metadata();
    const screenshotWidth = metadata.width || pageInfo.width;
    const screenshotHeight = metadata.height || captureHeight;

    const scaleX = computeScreenshotScaleFactor(screenshotWidth, pageInfo.width);
    const scaleY = computeScreenshotScaleFactor(screenshotHeight, captureHeight);

    diagnostics.screenshotWidth = screenshotWidth;
    diagnostics.screenshotHeight = screenshotHeight;
    diagnostics.scaleX = scaleX;
    diagnostics.scaleY = scaleY;

    debugLog("Screenshot captured", {
      screenshotWidth,
      screenshotHeight,
      scaleX,
      scaleY,
      expectedScaleFromDPR: pageInfo.devicePixelRatio,
    });

    // Step 7: Convert pre-captured bounding boxes to screenshot coordinates
    const elementsWithBounds: PageOverviewElement[] = [];

    for (const [i, target] of elementTargets.entries()) {
      const box = rawBoundingBoxes.get(i);

      if (!box) {
        const diagnosticElement = diagnostics.elements[i];
        if (DEBUG_SCREENSHOTS && diagnosticElement) {
          diagnosticElement.skipped = true;
          diagnosticElement.skipReason = diagnosticElement.skipReason ?? "no_box";
        }
        continue;
      }

      if (box.y > captureHeight) {
        const diagnosticElement = diagnostics.elements[i];
        if (DEBUG_SCREENSHOTS && diagnosticElement) {
          diagnosticElement.skipped = true;
          diagnosticElement.skipReason = "below_capture_area";
        }
        continue;
      }

      // Convert CSS pixel bounding boxes to screenshot pixel coordinates
      const scaledBox = {
        x: box.x * scaleX,
        y: box.y * scaleY,
        width: box.width * scaleX,
        height: box.height * scaleY,
      };

      const clippedBox = clipPageOverviewBounds(
        scaledBox,
        screenshotWidth,
        screenshotHeight,
      );

      if (!clippedBox) {
        const diagnosticElement = diagnostics.elements[i];
        if (DEBUG_SCREENSHOTS && diagnosticElement) {
          diagnosticElement.scaledBox = scaledBox;
          diagnosticElement.skipped = true;
          diagnosticElement.skipReason = "clipped_out";
        }
        continue;
      }

      const xPercent = clampPercent((clippedBox.x / screenshotWidth) * 100);
      const yPercent = clampPercent((clippedBox.y / screenshotHeight) * 100);
      const widthPercent = clampPercent((clippedBox.width / screenshotWidth) * 100);
      const heightPercent = clampPercent((clippedBox.height / screenshotHeight) * 100);

      {
        const diagnosticElement = diagnostics.elements[i];
        if (DEBUG_SCREENSHOTS && diagnosticElement) {
          diagnosticElement.scaledBox = clippedBox;
          diagnosticElement.percentBounds = {
          x: roundPercent(xPercent),
          y: roundPercent(yPercent),
          width: roundPercent(widthPercent),
          height: roundPercent(heightPercent),
          };
        }
      }

      elementsWithBounds.push({
        issueId: target.issueId,
        ruleId: target.ruleId,
        severity: target.severity,
        selector: target.selector,
        nodeIndex: target.nodeIndex,
        xPercent: roundPercent(xPercent),
        yPercent: roundPercent(yPercent),
        widthPercent: roundPercent(widthPercent),
        heightPercent: roundPercent(heightPercent),
        x: Math.max(0, Math.round(clippedBox.x)),
        y: Math.max(0, Math.round(clippedBox.y)),
        width: Math.max(0, Math.round(clippedBox.width)),
        height: Math.max(0, Math.round(clippedBox.height)),
      });
    }

    debugLog(`Final elements with bounds: ${elementsWithBounds.length}/${elementTargets.length}`);

    if (DEBUG_SCREENSHOTS) {
      debugLog("Full diagnostics", diagnostics);
    }

    return {
      screenshotPath,
      screenshotFilename,
      pageWidth: screenshotWidth,
      pageHeight: screenshotHeight,
      elements: elementsWithBounds,
      buffer: Buffer.from(screenshotBuffer),
    };
  } catch (error) {
    debugLog("Capture failed", { error: String(error) });
    return null;
  }
}
