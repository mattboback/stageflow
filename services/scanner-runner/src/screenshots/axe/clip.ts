import type { AxeScreenshotConfig, ElementBounds } from "./types";

export function computeUnionClip(
	capturedBoxes: {
		box: { x: number; y: number; width: number; height: number };
		selector: string;
	}[],
	viewport: { width: number; height: number },
	cfg: AxeScreenshotConfig,
): {
	clip: { x: number; y: number; width: number; height: number };
	elementBounds: ElementBounds[];
} | null {
	if (capturedBoxes.length === 0) {
		return null;
	}

	let minX = Number.POSITIVE_INFINITY;
	let minY = Number.POSITIVE_INFINITY;
	let maxX = Number.NEGATIVE_INFINITY;
	let maxY = Number.NEGATIVE_INFINITY;

	for (const { box } of capturedBoxes) {
		const bx1 = box.x;
		const by1 = box.y;
		const bx2 = box.x + box.width;
		const by2 = box.y + box.height;

		minX = Math.min(minX, bx1);
		minY = Math.min(minY, by1);
		maxX = Math.max(maxX, bx2);
		maxY = Math.max(maxY, by2);
	}

	if (
		!Number.isFinite(minX) ||
		!Number.isFinite(minY) ||
		!Number.isFinite(maxX) ||
		!Number.isFinite(maxY)
	) {
		return null;
	}

	// Account for CSS border visual space (outline + offset + shadow = ~15px).
	const borderPadding = 15;
	const pad = Math.max(0, cfg.clipPadding) + borderPadding;

	const unionWidth = maxX - minX;
	const unionHeight = maxY - minY;
	const contextX = (unionWidth * cfg.contextRatio) / 2;
	const contextY = (unionHeight * cfg.contextRatio) / 2;

	const clipX = Math.max(0, minX - pad - contextX);
	const clipY = Math.max(0, minY - pad - contextY);
	const clipW = Math.min(
		viewport.width - clipX,
		unionWidth + (pad + contextX) * 2,
	);
	const clipH = Math.min(
		viewport.height - clipY,
		unionHeight + (pad + contextY) * 2,
	);

	if (clipW <= 2 || clipH <= 2) {
		return null;
	}

	const elementBounds: ElementBounds[] = capturedBoxes.map(
		({ box, selector }) => ({
			x: Math.round(box.x - clipX),
			y: Math.round(box.y - clipY),
			width: Math.round(box.width),
			height: Math.round(box.height),
			selector,
		}),
	);

	return {
		clip: { x: clipX, y: clipY, width: clipW, height: clipH },
		elementBounds,
	};
}

export function computeSingleTargetClip(
	box: { x: number; y: number; width: number; height: number },
	viewport: { width: number; height: number },
	cfg: AxeScreenshotConfig,
): { x: number; y: number; width: number; height: number } | null {
	// Use most of the viewport for better context.
	const maxViewportRatio = 0.92;
	const borderPadding = 20;

	// Calculate desired size with generous context.
	const desiredW = Math.max(cfg.shotMinWidth, box.width * cfg.shotScale);
	const desiredH = Math.max(cfg.shotMinHeight, box.height * cfg.shotScale);
	const expandW = box.width * cfg.contextRatio + borderPadding * 2;
	const expandH = box.height * cfg.contextRatio + borderPadding * 2;

	// Clamp to a large fraction of the viewport to ensure page context is visible.
	const width = Math.min(desiredW + expandW, viewport.width * maxViewportRatio);
	const height = Math.min(
		desiredH + expandH,
		viewport.height * maxViewportRatio,
	);
	if (width <= 2 || height <= 2) {
		return null;
	}

	// Center the element in the screenshot.
	const cx = box.x + box.width / 2;
	const cy = box.y + box.height / 2;
	const x = Math.max(0, Math.min(cx - width / 2, viewport.width - width));
	const y = Math.max(0, Math.min(cy - height / 2, viewport.height - height));

	return { x, y, width, height };
}

export function computeElementClip(
	box: { x: number; y: number; width: number; height: number },
	viewport: { width: number; height: number },
	cfg: Pick<
		AxeScreenshotConfig,
		"elementContextMultiplier" | "elementContextMinSize"
	>,
	borderPadding = 20,
): { x: number; y: number; width: number; height: number } | null {
	// Keep a small border around the highlighted element.
	const pad = Math.max(0, borderPadding);

	// Cap the crop to most of the viewport so we still get page context.
	const maxViewportRatio = 0.92;
	const maxWidth = Math.max(0, Math.floor(viewport.width * maxViewportRatio));
	const maxHeight = Math.max(0, Math.floor(viewport.height * maxViewportRatio));

	const multiplier = Math.max(1, cfg.elementContextMultiplier);
	const minSize = Math.max(0, cfg.elementContextMinSize);

	// Size the crop relative to element size, with a sane minimum.
	const desiredWidth = Math.max(minSize, box.width * multiplier + pad * 2);
	const desiredHeight = Math.max(minSize, box.height * multiplier + pad * 2);

	const width = Math.round(Math.min(desiredWidth, maxWidth || viewport.width));
	const height = Math.round(
		Math.min(desiredHeight, maxHeight || viewport.height),
	);

	if (width <= 2 || height <= 2) {
		return null;
	}

	// Center the crop on the element.
	const cx = box.x + box.width / 2;
	const cy = box.y + box.height / 2;

	const x = Math.max(
		0,
		Math.min(Math.round(cx - width / 2), viewport.width - width),
	);
	const y = Math.max(
		0,
		Math.min(Math.round(cy - height / 2), viewport.height - height),
	);

	return { x, y, width, height };
}
