import type { Page } from "playwright";

import { existsSync, mkdirSync } from "node:fs";
import { join } from "node:path";
import { v4 as uuidv4 } from "uuid";

import type {
  AxeScreenshotConfig,
  AxeViolation,
  ElementBounds,
  EnhancedScreenshotResult,
  FriendlyNodeInfo,
  PageLocationInfo,
} from "./types";

import { getScreenshotPolicy } from "../../config/rule-behaviors";
import { computeElementClip, computeSingleTargetClip, computeUnionClip } from "./clip";
import { buildFriendlyNodeInfo } from "./friendly-node";
import { injectHighlightCSS, removeHighlightCSS } from "./highlight-css";
import { compositeOverlay, generateThumbnail, saveScreenshot } from "./image";
import { captureLocationInfo } from "./location";
import { captureSemanticOverlayScreenshot } from "./semantic-overlay";
import { extractAxeViolationTargets } from "./targets";

export async function captureViolationScreenshot(input: {
  page: Page;
  violation: AxeViolation;
  resultsDir: string;
  cfg: AxeScreenshotConfig;
}): Promise<EnhancedScreenshotResult | null> {
  if (!input.cfg.screenshotsEnabled) {
    return null;
  }

  const policy = getScreenshotPolicy(input.violation.id);
  if (policy === "semantic") {
    return captureSemanticOverlayScreenshot(input);
  }

  if (policy === "never") {
    return null;
  }

  const { targets, selector } = extractAxeViolationTargets(input.violation);

  const ext = input.cfg.outputFormat === "webp" ? ".webp" : ".png";
  const screenshotFilename = `violation-${input.violation.id ?? "unknown"}-${uuidv4()}${ext}`;
  const screenshotPath = join(input.resultsDir, screenshotFilename);

  if (!existsSync(input.resultsDir)) {
    mkdirSync(input.resultsDir, { recursive: true });
  }

  const viewport = input.page.viewportSize() ?? { width: 1280, height: 720 };
  const vw = viewport.width;
  const vh = viewport.height;

  const useCSSInjection = input.cfg.overlayMethod === "css-injection";

  try {
    if (useCSSInjection && input.cfg.injectHighlight && targets.length > 0) {
      await injectHighlightCSS(input.page, targets, input.cfg);
    }

    const locationInfo = await captureLocationInfo(input.page);

    const friendlyNode = selector
      ? await buildFriendlyNodeInfo(input.page, selector)
      : undefined;

    if (input.cfg.mergeTargets && targets.length > 0) {
      const result = await tryUnionBoundingBoxScreenshot({
        page: input.page,
        targets,
        vw,
        vh,
        screenshotPath,
        cfg: input.cfg,
        returnBuffer: !useCSSInjection,
      });
      if (result.success) {
        await applyOverlayIfNeeded({
          useCSSInjection,
          buffer: result.buffer,
          elementBounds: result.elementBounds,
          clipWidth: result.clipWidth,
          screenshotPath,
          cfg: input.cfg,
        });

        return await finalizeViolationCapture({
          screenshotFilename,
          screenshotPath,
          resultsDir: input.resultsDir,
          cfg: input.cfg,
          locationInfo,
          friendlyNode,
          elementBounds: result.elementBounds,
        });
      }
    }

    if (selector) {
      const result = await trySingleTargetViewportScreenshot({
        page: input.page,
        selector,
        vw,
        vh,
        screenshotPath,
        cfg: input.cfg,
        returnBuffer: !useCSSInjection && input.cfg.injectHighlight,
      });
      if (result.success) {
        await applyOverlayIfNeeded({
          useCSSInjection,
          buffer: result.buffer,
          elementBounds: result.elementBounds,
          clipWidth: result.clipWidth,
          screenshotPath,
          cfg: input.cfg,
        });

        return await finalizeViolationCapture({
          screenshotFilename,
          screenshotPath,
          resultsDir: input.resultsDir,
          cfg: input.cfg,
          locationInfo,
          friendlyNode,
          elementBounds: result.elementBounds,
        });
      }

      const elementResult = await tryElementScreenshot({
        page: input.page,
        selector,
        vw,
        vh,
        screenshotPath,
        cfg: input.cfg,
        returnBuffer: !useCSSInjection && input.cfg.injectHighlight,
      });
      if (elementResult.success) {
        await applyOverlayIfNeeded({
          useCSSInjection,
          buffer: elementResult.buffer,
          elementBounds: elementResult.elementBounds,
          clipWidth: elementResult.clipWidth,
          screenshotPath,
          cfg: input.cfg,
        });

        return await finalizeViolationCapture({
          screenshotFilename,
          screenshotPath,
          resultsDir: input.resultsDir,
          cfg: input.cfg,
          locationInfo,
          friendlyNode,
          elementBounds: elementResult.elementBounds,
        });
      }
    }

    const buffer = await input.page.screenshot({ fullPage: false });
    await saveScreenshot(Buffer.from(buffer), screenshotPath, input.cfg);

    return await finalizeViolationCapture({
      screenshotFilename,
      screenshotPath,
      resultsDir: input.resultsDir,
      cfg: input.cfg,
      locationInfo,
      friendlyNode,
    });
  } catch {
    return null;
  } finally {
    if (useCSSInjection) {
      await removeHighlightCSS(input.page);
    }
  }
}

async function finalizeViolationCapture(input: {
  screenshotFilename: string;
  screenshotPath: string;
  resultsDir: string;
  cfg: AxeScreenshotConfig;
  locationInfo?: PageLocationInfo;
  friendlyNode?: FriendlyNodeInfo;
  elementBounds?: ElementBounds[];
}): Promise<EnhancedScreenshotResult> {
  const thumbnail = await generateThumbnail(
    input.screenshotPath,
    input.resultsDir,
    input.cfg,
  );

  return {
    screenshot: input.screenshotFilename,
    thumbnail,
    locationInfo: input.locationInfo,
    friendlyNode: input.friendlyNode,
    elementBounds: input.elementBounds,
  };
}

async function applyOverlayIfNeeded(input: {
  useCSSInjection: boolean;
  buffer?: Buffer;
  elementBounds?: ElementBounds[];
  clipWidth?: number;
  screenshotPath: string;
  cfg: AxeScreenshotConfig;
}): Promise<void> {
  if (
    input.useCSSInjection ||
    !input.buffer ||
    !input.elementBounds ||
    !input.clipWidth
  ) {
    return;
  }

  const overlaidBuffer = await compositeOverlay(
    input.buffer,
    input.elementBounds,
    input.cfg,
    input.clipWidth,
  );
  await saveScreenshot(overlaidBuffer, input.screenshotPath, input.cfg);
}

async function tryUnionBoundingBoxScreenshot(input: {
  page: Page;
  targets: string[];
  vw: number;
  vh: number;
  screenshotPath: string;
  cfg: AxeScreenshotConfig;
  returnBuffer: boolean;
}): Promise<{
  success: boolean;
  elementBounds?: ElementBounds[];
  buffer?: Buffer;
  clipWidth?: number;
}> {
  const targetSelectors = input.targets.slice(0, input.cfg.unionMaxTargets);

  // First pass: scroll to the first *workable* target to position the page.
  // Some pages (carousels, portals) include hidden "cloned" elements that match selectors but have no box.
  for (const candidate of targetSelectors) {
    try {
      const locator = input.page.locator(candidate).first();
      await locator.scrollIntoViewIfNeeded({ timeout: input.cfg.scrollTimeout });
      break;
    } catch {
      // try next
    }
  }

  // Small delay to let scroll settle
  await input.page.waitForTimeout(50);

  // Second pass: capture all bounding boxes at the current scroll position
  const capturedBoxes: {
    box: { x: number; y: number; width: number; height: number };
    selector: string;
  }[] = [];

  for (const t of targetSelectors) {
    try {
      const locator = input.page.locator(t).first();
      const box = await locator.boundingBox();
      if (!box) {
        continue;
      }

      // Only include elements that are at least partially visible in the viewport
      if (box.y + box.height < 0 || box.y > input.vh) {
        continue;
      }

      capturedBoxes.push({ box, selector: t });
    } catch {
      // ignore per-target failures
    }
  }

  const computed = computeUnionClip(
    capturedBoxes,
    { width: input.vw, height: input.vh },
    input.cfg,
  );
  if (!computed) {
    return { success: false };
  }

  const screenshotBuffer = await input.page.screenshot({
    clip: computed.clip,
  });
  const bufferData = Buffer.from(screenshotBuffer);

  if (input.returnBuffer) {
    return {
      success: true,
      elementBounds: computed.elementBounds,
      buffer: bufferData,
      clipWidth: computed.clip.width,
    };
  }

  await saveScreenshot(bufferData, input.screenshotPath, input.cfg);
  return {
    success: true,
    elementBounds: computed.elementBounds,
    clipWidth: computed.clip.width,
  };
}

async function trySingleTargetViewportScreenshot(input: {
  page: Page;
  selector: string;
  vw: number;
  vh: number;
  screenshotPath: string;
  cfg: AxeScreenshotConfig;
  returnBuffer: boolean;
}): Promise<{
  success: boolean;
  elementBounds?: ElementBounds[];
  buffer?: Buffer;
  clipWidth?: number;
}> {
  try {
    const locator = input.page.locator(input.selector).first();
    try {
      await locator.scrollIntoViewIfNeeded({ timeout: input.cfg.scrollTimeout });
    } catch {
      // ignore scroll failures
    }

    const box = await locator.boundingBox();
    if (!box) {
      return { success: false };
    }

    const clip = computeSingleTargetClip(
      box,
      { width: input.vw, height: input.vh },
      input.cfg,
    );
    if (!clip) {
      return { success: false };
    }

    const buffer = await input.page.screenshot({
      clip,
    });

    const bufferData = Buffer.from(buffer);
    const elementBounds: ElementBounds[] = [
      {
        x: Math.max(0, Math.round(box.x - clip.x)),
        y: Math.max(0, Math.round(box.y - clip.y)),
        width: Math.round(box.width),
        height: Math.round(box.height),
        selector: input.selector,
      },
    ];

    if (input.returnBuffer) {
      return { success: true, elementBounds, buffer: bufferData, clipWidth: clip.width };
    }

    await saveScreenshot(bufferData, input.screenshotPath, input.cfg);
    return { success: true, elementBounds, clipWidth: clip.width };
  } catch {
    return { success: false };
  }
}

async function tryElementScreenshot(input: {
  page: Page;
  selector: string;
  vw: number;
  vh: number;
  screenshotPath: string;
  cfg: AxeScreenshotConfig;
  returnBuffer: boolean;
}): Promise<{
  success: boolean;
  elementBounds?: ElementBounds[];
  buffer?: Buffer;
  clipWidth?: number;
}> {
  try {
    const locator = input.page.locator(input.selector).first();
    const box = await locator.boundingBox();
    if (!box) {
      return { success: false };
    }

    const clip = computeElementClip(
      box,
      { width: input.vw, height: input.vh },
      input.cfg,
    );
    if (!clip) {
      return { success: false };
    }

    const buffer = await input.page.screenshot({ clip });

    const bufferData = Buffer.from(buffer);
    const elementBounds: ElementBounds[] = [
      {
        x: Math.max(0, Math.round(box.x - clip.x)),
        y: Math.max(0, Math.round(box.y - clip.y)),
        width: Math.round(box.width),
        height: Math.round(box.height),
        selector: input.selector,
      },
    ];

    if (input.returnBuffer) {
      return { success: true, elementBounds, buffer: bufferData, clipWidth: clip.width };
    }

    await saveScreenshot(bufferData, input.screenshotPath, input.cfg);
    return { success: true, elementBounds, clipWidth: clip.width };
  } catch {
    return { success: false };
  }
}
