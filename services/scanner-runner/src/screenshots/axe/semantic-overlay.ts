import type { Page } from "playwright";

import { existsSync, mkdirSync } from "node:fs";
import { join } from "node:path";
import { v4 as uuidv4 } from "uuid";

import type {
  AxeScreenshotConfig,
  AxeViolation,
  EnhancedScreenshotResult,
} from "./types";

import { generateThumbnail, saveScreenshot } from "./image";
import { extractAxeViolationTargets } from "./targets";

const HEADING_OVERLAY_RULES = new Set(["heading-order", "empty-heading"]);
const TARGET_OVERLAY_RULES = new Set([
  "link-name",
  "button-name",
  "label",
  "form-field-multiple-labels",
]);

export async function captureSemanticOverlayScreenshot(input: {
  page: Page;
  violation: AxeViolation;
  resultsDir: string;
  cfg: AxeScreenshotConfig;
}): Promise<EnhancedScreenshotResult | null> {
  if (!input.cfg.screenshotsEnabled || !input.cfg.semanticOverlayEnabled) {
    return null;
  }

  const ruleId = input.violation.id ?? "";
  const overlayId = `sf-semantic-${uuidv4()}`;

  let overlayCount = 0;
  if (HEADING_OVERLAY_RULES.has(ruleId)) {
    overlayCount = await injectHeadingOverlay(input.page, {
      overlayId,
      maxHeadings: input.cfg.semanticOverlayMaxHeadings,
      maxLabelLength: input.cfg.semanticOverlayMaxLabelLength,
      legendEnabled: input.cfg.semanticOverlayLegendEnabled,
    });
  } else if (TARGET_OVERLAY_RULES.has(ruleId)) {
    const { targets } = extractAxeViolationTargets(input.violation);
    overlayCount = await injectTargetOverlay(input.page, {
      overlayId,
      targets,
      maxTargets: input.cfg.semanticOverlayMaxHeadings,
      maxLabelLength: input.cfg.semanticOverlayMaxLabelLength,
      legendEnabled: input.cfg.semanticOverlayLegendEnabled,
      ruleLabel: toRuleLabel(ruleId),
    });
  } else {
    return null;
  }

  if (!existsSync(input.resultsDir)) {
    mkdirSync(input.resultsDir, { recursive: true });
  }

  const ext = input.cfg.outputFormat === "webp" ? ".webp" : ".png";
  const screenshotFilename = `semantic-${ruleId || "unknown"}-${uuidv4()}${ext}`;
  const screenshotPath = join(input.resultsDir, screenshotFilename);

  try {
    if (overlayCount === 0) {
      return null;
    }

    await input.page.waitForTimeout(80);
    const buffer = await input.page.screenshot({ fullPage: true });
    await saveScreenshot(Buffer.from(buffer), screenshotPath, input.cfg);
  } catch {
    return null;
  } finally {
    await removeSemanticOverlay(input.page, overlayId);
  }

  const thumbnail = await generateThumbnail(screenshotPath, input.resultsDir, input.cfg);
  return { screenshot: screenshotFilename, thumbnail };
}

function buildHeadingOverlayCss(overlayId: string): string {
  return `
    [data-sf-heading-overlay="${overlayId}"][data-sf-heading] {
      position: relative !important;
      outline: 2px dashed rgba(124, 58, 237, 0.75) !important;
      outline-offset: 2px !important;
    }
    [data-sf-heading-overlay="${overlayId}"][data-sf-heading]::before {
      content: attr(data-sf-heading);
      position: absolute;
      top: -0.75rem;
      left: -0.75rem;
      font: 600 10px/1.2 "Inter", system-ui, sans-serif;
      color: #fff;
      background: rgba(124, 58, 237, 0.95);
      padding: 2px 6px;
      border-radius: 999px;
      z-index: 2147483647;
      pointer-events: none;
      white-space: nowrap;
    }
    [data-sf-heading-overlay="${overlayId}"].sf-heading-legend {
      position: fixed;
      top: 12px;
      left: 12px;
      max-width: 320px;
      max-height: 40vh;
      overflow: hidden;
      background: rgba(15, 23, 42, 0.9);
      color: #fff;
      font: 500 11px/1.4 "Inter", system-ui, sans-serif;
      padding: 8px 10px;
      border-radius: 10px;
      box-shadow: 0 12px 30px rgba(15, 23, 42, 0.35);
      z-index: 2147483647;
      pointer-events: none;
    }
    .sf-heading-legend-row {
      display: flex;
      gap: 6px;
      align-items: baseline;
      margin-bottom: 4px;
    }
    .sf-heading-legend-row:last-child {
      margin-bottom: 0;
    }
    .sf-heading-legend-pill {
      font-size: 10px;
      font-weight: 700;
      text-transform: uppercase;
      letter-spacing: 0.04em;
      background: rgba(124, 58, 237, 0.9);
      padding: 2px 6px;
      border-radius: 999px;
      white-space: nowrap;
    }
  `;
}

function buildTargetOverlayCss(overlayId: string): string {
  return `
    [data-sf-violation-overlay="${overlayId}"] {
      outline: 2px dashed rgba(14, 165, 233, 0.8) !important;
      outline-offset: 2px !important;
    }
    [data-sf-violation-overlay="${overlayId}"].sf-violation-legend {
      position: fixed;
      top: 12px;
      right: 12px;
      max-width: 360px;
      max-height: 45vh;
      overflow: hidden;
      background: rgba(15, 23, 42, 0.9);
      color: #fff;
      font: 500 11px/1.4 "Inter", system-ui, sans-serif;
      padding: 10px 12px;
      border-radius: 12px;
      box-shadow: 0 12px 30px rgba(15, 23, 42, 0.35);
      z-index: 2147483647;
      pointer-events: none;
    }
    .sf-violation-legend-title {
      font-size: 11px;
      font-weight: 700;
      text-transform: uppercase;
      letter-spacing: 0.04em;
      color: rgba(255, 255, 255, 0.8);
      margin-bottom: 6px;
    }
    .sf-violation-legend-row {
      display: flex;
      gap: 6px;
      align-items: baseline;
      margin-bottom: 4px;
    }
    .sf-violation-legend-row:last-child {
      margin-bottom: 0;
    }
    .sf-violation-legend-pill {
      font-size: 10px;
      font-weight: 700;
      text-transform: uppercase;
      letter-spacing: 0.04em;
      background: rgba(14, 165, 233, 0.9);
      padding: 2px 6px;
      border-radius: 999px;
      white-space: nowrap;
    }
  `;
}

async function injectHeadingOverlay(
  page: Page,
  options: {
    overlayId: string;
    maxHeadings: number;
    maxLabelLength: number;
    legendEnabled: boolean;
  },
): Promise<number> {
  const css = buildHeadingOverlayCss(options.overlayId);
  return page.evaluate(
    ({ overlayId, cssText, maxHeadings, maxLabelLength, legendEnabled }) => {
      const overlayAttr = "data-sf-heading-overlay";
      const labelAttr = "data-sf-heading";
      const styleId = `${overlayId}-style`;

      const styleEl = document.createElement("style");
      styleEl.id = styleId;
      styleEl.textContent = cssText;
      document.head.appendChild(styleEl);

      const headings = Array.from(document.querySelectorAll("h1, h2, h3, h4, h5, h6"));
      const limited = headings.slice(0, Math.max(0, maxHeadings));

      const normalizeText = (value: string): string => value.replace(/\\s+/g, " ").trim();

      limited.forEach((heading, index) => {
        const level = heading.tagName.toUpperCase();
        const rawText = normalizeText(heading.textContent || "");
        const clipped =
          maxLabelLength > 0 && rawText.length > maxLabelLength
            ? `${rawText.slice(0, maxLabelLength)}...`
            : rawText;
        const label = `${level} #${index + 1}${clipped ? `: ${clipped}` : ""}`;
        heading.setAttribute(labelAttr, label);
        heading.setAttribute(overlayAttr, overlayId);
      });

      if (legendEnabled && limited.length > 0) {
        const legend = document.createElement("div");
        legend.className = "sf-heading-legend";
        legend.setAttribute(overlayAttr, overlayId);

        limited.forEach((heading, index) => {
          const row = document.createElement("div");
          row.className = "sf-heading-legend-row";

          const pill = document.createElement("span");
          pill.className = "sf-heading-legend-pill";
          pill.textContent = `${heading.tagName.toUpperCase()} #${index + 1}`;

          const text = document.createElement("span");
          const content = normalizeText(heading.textContent || "");
          text.textContent = content.length > 0 ? content : "(empty)";

          row.appendChild(pill);
          row.appendChild(text);
          legend.appendChild(row);
        });

        document.body.appendChild(legend);
      }

      if (limited.length === 0) {
        styleEl.remove();
      }

      return limited.length;
    },
    {
      overlayId: options.overlayId,
      cssText: css,
      maxHeadings: options.maxHeadings,
      maxLabelLength: options.maxLabelLength,
      legendEnabled: options.legendEnabled,
    },
  );
}

async function injectTargetOverlay(
  page: Page,
  options: {
    overlayId: string;
    targets: string[];
    maxTargets: number;
    maxLabelLength: number;
    legendEnabled: boolean;
    ruleLabel: string;
  },
): Promise<number> {
  const css = buildTargetOverlayCss(options.overlayId);
  return page.evaluate(
    ({
      overlayId,
      cssText,
      targets,
      maxTargets,
      maxLabelLength,
      legendEnabled,
      ruleLabel,
    }) => {
      const overlayAttr = "data-sf-violation-overlay";
      const styleId = `${overlayId}-style`;

      const styleEl = document.createElement("style");
      styleEl.id = styleId;
      styleEl.textContent = cssText;
      document.head.appendChild(styleEl);

      const normalizeText = (value: string): string => value.replace(/\s+/g, " ").trim();

      const getElementLabel = (el: Element): string => {
        const ariaLabel = el.getAttribute("aria-label");
        if (ariaLabel) {
          return normalizeText(ariaLabel);
        }

        const labelledBy = el.getAttribute("aria-labelledby");
        if (labelledBy) {
          const pieces = labelledBy
            .split(/\s+/)
            .map((id) => document.getElementById(id))
            .filter((node): node is HTMLElement => Boolean(node))
            .map((node) => normalizeText(node.textContent || ""))
            .filter((text) => text.length > 0);
          if (pieces.length > 0) {
            return pieces.join(" ");
          }
        }

        const alt = el.getAttribute("alt");
        if (alt) {
          return normalizeText(alt);
        }

        const title = el.getAttribute("title");
        if (title) {
          return normalizeText(title);
        }

        return normalizeText(el.textContent || "");
      };

      const unique = new Set<Element>();
      const elements: Element[] = [];
      for (const target of targets.slice(0, Math.max(0, maxTargets))) {
        let element: Element | null = null;
        try {
          element = document.querySelector(target);
        } catch {
          element = null;
        }
        if (!element) {
          continue;
        }
        const tagName = element.tagName.toUpperCase();
        if (tagName === "HTML" || tagName === "BODY") {
          continue;
        }
        if (unique.has(element)) {
          continue;
        }
        unique.add(element);
        element.setAttribute(overlayAttr, overlayId);
        elements.push(element);
      }

      if (legendEnabled && elements.length > 0) {
        const legend = document.createElement("div");
        legend.className = "sf-violation-legend";
        legend.setAttribute(overlayAttr, overlayId);

        const title = document.createElement("div");
        title.className = "sf-violation-legend-title";
        title.textContent = `Semantic overlay • ${ruleLabel}`;
        legend.appendChild(title);

        elements.forEach((el, index) => {
          const row = document.createElement("div");
          row.className = "sf-violation-legend-row";

          const pill = document.createElement("span");
          pill.className = "sf-violation-legend-pill";
          pill.textContent = `#${index + 1}`;

          const text = document.createElement("span");
          const content = getElementLabel(el);
          let clipped = content;
          if (maxLabelLength > 0 && clipped.length > maxLabelLength) {
            clipped = `${clipped.slice(0, maxLabelLength)}...`;
          }
          text.textContent = clipped.length > 0 ? clipped : "(no text)";

          row.appendChild(pill);
          row.appendChild(text);
          legend.appendChild(row);
        });

        document.body.appendChild(legend);
      }

      if (elements.length === 0) {
        styleEl.remove();
      }

      return elements.length;
    },
    {
      overlayId: options.overlayId,
      cssText: css,
      targets: options.targets,
      maxTargets: options.maxTargets,
      maxLabelLength: options.maxLabelLength,
      legendEnabled: options.legendEnabled,
      ruleLabel: options.ruleLabel,
    },
  );
}

async function removeSemanticOverlay(page: Page, overlayId: string): Promise<void> {
  await page.evaluate(
    ({ overlayId }) => {
      const overlayAttr = "data-sf-heading-overlay";
      const labelAttr = "data-sf-heading";
      const selector = `[${overlayAttr}="${overlayId}"]`;

      document.querySelectorAll(selector).forEach((el) => {
        el.removeAttribute(labelAttr);
        el.removeAttribute(overlayAttr);
        if (el.classList.contains("sf-heading-legend")) {
          el.remove();
        }
      });

      const targetOverlayAttr = "data-sf-violation-overlay";
      document.querySelectorAll(`[${targetOverlayAttr}="${overlayId}"]`).forEach((el) => {
        el.removeAttribute(targetOverlayAttr);
        if (el.classList.contains("sf-violation-legend")) {
          el.remove();
        }
      });

      const styleEl = document.getElementById(`${overlayId}-style`);
      if (styleEl) {
        styleEl.remove();
      }
    },
    {
      overlayId,
    },
  );
}

function toRuleLabel(ruleId: string): string {
  return ruleId.replace(/-/g, " ");
}
