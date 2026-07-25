export interface ViewBox {
	x: number;
	y: number;
	width: number;
	height: number;
}

export interface Rect {
	x: number;
	y: number;
	width: number;
	height: number;
}

interface CropOptions {
	minWidth?: number;
	minHeight?: number;
	padding?: number;
	maxAspect?: number;
}

/** Constrains `value` to [min, max]. Shared with screenshot-sampling. */
export function clamp(value: number, min: number, max: number): number {
	return Math.min(max, Math.max(min, value));
}

/*
 * Crop the page-overview screenshot tightly around one element. Context
 * scales with the element instead of a fixed window, so a 12px icon isn't
 * drowned in half a page and a full-width banner still fits.
 */
export function getCroppedViewBox(
	pageWidth: number,
	pageHeight: number,
	element: Rect,
	options: CropOptions = {}
): ViewBox | null {
	if (
		!Number.isFinite(pageWidth) ||
		!Number.isFinite(pageHeight) ||
		pageWidth <= 0 ||
		pageHeight <= 0
	) {
		return null;
	}

	const padding = options.padding ?? clamp(0.5 * Math.max(element.width, element.height), 16, 64);
	const minWidth = options.minWidth ?? 200;
	const minHeight = options.minHeight ?? 120;
	const maxAspect = options.maxAspect ?? 3;

	let width = Math.max(minWidth, element.width + padding * 2);
	let height = Math.max(minHeight, element.height + padding * 2);

	// Expand the short axis so extreme strips (skip links, full-width bars)
	// don't produce an unreadably thin crop.
	if (width / height > maxAspect) {
		height = width / maxAspect;
	} else if (height / width > maxAspect) {
		width = height / maxAspect;
	}

	width = Math.min(pageWidth, width);
	height = Math.min(pageHeight, height);

	const centerX = element.x + element.width / 2;
	const centerY = element.y + element.height / 2;

	const x = clamp(centerX - width / 2, 0, Math.max(0, pageWidth - width));
	const y = clamp(centerY - height / 2, 0, Math.max(0, pageHeight - height));

	return { x, y, width, height };
}
