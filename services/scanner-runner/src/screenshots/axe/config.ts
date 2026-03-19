import type {
  AxeScreenshotConfig,
  HighlightStyle,
  OutputFormat,
  OverlayMethod,
} from "./types";

import { getEnvBool, getEnvInt, getEnvNumber, getEnvString } from "../../utils/env";

function parseHighlightStyle(raw: string): HighlightStyle {
  return raw.toLowerCase() === "dashed" ? "dashed" : "solid";
}

function parseOutputFormat(raw: string): OutputFormat {
  return raw.toLowerCase() === "png" ? "png" : "webp";
}

function parseOverlayMethod(raw: string): OverlayMethod {
  return raw.toLowerCase() === "css-injection" ? "css-injection" : "sharp-composite";
}

export function loadAxeScreenshotConfig(
  overrides?: Partial<AxeScreenshotConfig>,
): AxeScreenshotConfig {
  const highlightStyle = parseHighlightStyle(
    getEnvString("A11Y_HIGHLIGHT_STYLE", "solid"),
  );
  const outputFormat = parseOutputFormat(getEnvString("A11Y_SCREENSHOT_FORMAT", "webp"));
  const overlayMethod = parseOverlayMethod(
    getEnvString("A11Y_OVERLAY_METHOD", "sharp-composite"),
  );

  return {
    screenshotsEnabled: getEnvBool("A11Y_SHOT_ENABLED", true),
    injectHighlight: getEnvBool("A11Y_SHOT_HIGHLIGHT", true),
    shotScale: getEnvNumber("A11Y_SCREENSHOT_SCALE", 2),
    shotMinWidth: getEnvNumber("A11Y_SCREENSHOT_MIN_WIDTH", 900),
    shotMinHeight: getEnvNumber("A11Y_SCREENSHOT_MIN_HEIGHT", 600),
    deviceScaleFactor: getEnvNumber("A11Y_DEVICE_SCALE_FACTOR", 2),
    highlightStyle,
    mergeTargets: getEnvBool("A11Y_SHOT_MERGE_TARGETS", true),
    unionMaxTargets: Math.max(0, getEnvInt("A11Y_SHOT_MAX_TARGETS", 4)),
    clipPadding: getEnvNumber("A11Y_SHOT_PADDING", 80),
    contextRatio: Math.max(0, getEnvNumber("A11Y_SCREENSHOT_CONTEXT_RATIO", 1.0)),
    elementContextMultiplier: Math.max(
      1,
      getEnvNumber("A11Y_ELEMENT_CONTEXT_MULTIPLIER", 2.5),
    ),
    elementContextMinSize: Math.max(
      0,
      getEnvNumber("A11Y_ELEMENT_CONTEXT_MIN_SIZE", 400),
    ),
    scrollTimeout: Math.max(0, getEnvInt("A11Y_SCROLL_TIMEOUT", 300)),
    thumbnailSize: Math.max(0, getEnvInt("A11Y_THUMBNAIL_SIZE", 400)),
    generateThumbnails: getEnvBool("A11Y_GENERATE_THUMBNAILS", false),
    outputFormat,
    webpQuality: Math.min(100, Math.max(1, getEnvInt("A11Y_SCREENSHOT_QUALITY", 80))),
    overlayMethod,
    semanticOverlayEnabled: getEnvBool("A11Y_SEMANTIC_OVERLAY_ENABLED", true),
    semanticOverlayMaxHeadings: Math.max(
      0,
      getEnvInt("A11Y_SEMANTIC_OVERLAY_MAX_HEADINGS", 30),
    ),
    semanticOverlayMaxLabelLength: Math.max(
      0,
      getEnvInt("A11Y_SEMANTIC_OVERLAY_MAX_LABEL", 60),
    ),
    semanticOverlayLegendEnabled: getEnvBool("A11Y_SEMANTIC_OVERLAY_LEGEND", true),
    ...overrides,
  };
}
