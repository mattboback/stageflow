import sharp from "sharp";

import type { IssueSeverity } from "../../core/types";
import type { PageOverviewElement } from "./types";

export const SEVERITY_COLORS: Record<
	IssueSeverity,
	{ stroke: string; fill: string }
> = {
	critical: { stroke: "#ef4444", fill: "rgba(239, 68, 68, 0.15)" },
	serious: { stroke: "#f97316", fill: "rgba(249, 115, 22, 0.15)" },
	moderate: { stroke: "#3b82f6", fill: "rgba(59, 130, 246, 0.15)" },
	minor: { stroke: "#6b7280", fill: "rgba(107, 114, 128, 0.15)" },
	info: { stroke: "#94a3b8", fill: "rgba(148, 163, 184, 0.12)" },
};

export interface OverlayOptions {
	showLabels?: boolean;
	labelFontSize?: number;
}

export function buildPageOverviewOverlaySvg(
	elements: PageOverviewElement[],
	width: number,
	height: number,
	options: OverlayOptions = {},
): Buffer {
	const { showLabels = true, labelFontSize = 32 } = options;
	const strokeWidth = 4;
	const badgeSize = Math.max(44, labelFontSize + 14);
	const badgeRadius = badgeSize / 2;

	const rects = elements
		.map((el, index) => {
			const colors = SEVERITY_COLORS[el.severity];
			const rect = `
          <rect
            x="${el.x}"
            y="${el.y}"
            width="${el.width}"
            height="${el.height}"
            fill="${colors.fill}"
            stroke="${colors.stroke}"
            stroke-width="${strokeWidth}"
          />`;

			if (!showLabels) {
				return rect;
			}

			// Position badge at top-left corner of the element
			const badgeX = Math.max(badgeRadius, el.x);
			const badgeY = Math.max(badgeRadius, el.y);
			const labelNum = index + 1;

			const badge = `
          <circle
            cx="${badgeX}"
            cy="${badgeY}"
            r="${badgeRadius}"
            fill="${colors.stroke}"
          />
          <text
            x="${badgeX}"
            y="${badgeY}"
            text-anchor="middle"
            dominant-baseline="central"
            fill="white"
            font-family="Arial, sans-serif"
            font-size="${labelFontSize}"
            font-weight="bold"
          >${labelNum}</text>`;

			return rect + badge;
		})
		.join("\n");

	return Buffer.from(`
      <svg width="${width}" height="${height}" xmlns="http://www.w3.org/2000/svg">
        ${rects}
      </svg>
    `);
}

export async function compositePageOverviewOverlay(
	screenshotBuffer: Buffer,
	elements: PageOverviewElement[],
	width: number,
	height: number,
): Promise<Buffer> {
	if (elements.length === 0) {
		return screenshotBuffer;
	}

	const svgOverlay = buildPageOverviewOverlaySvg(elements, width, height);
	return sharp(screenshotBuffer)
		.composite([{ input: svgOverlay, top: 0, left: 0 }])
		.toBuffer();
}
