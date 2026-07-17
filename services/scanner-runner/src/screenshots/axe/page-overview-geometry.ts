export interface BoundingBox {
	x: number;
	y: number;
	width: number;
	height: number;
}

export function computeScreenshotScaleFactor(actualPixels: number, cssPixels: number): number {
	if (actualPixels <= 0 || cssPixels <= 0) {
		return 1;
	}

	const scale = actualPixels / cssPixels;
	if (!Number.isFinite(scale) || scale <= 0) {
		return 1;
	}

	return scale;
}

export function clipPageOverviewBounds(
	bounds: BoundingBox,
	maxWidth: number,
	maxHeight: number
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

export function clampPercent(value: number): number {
	if (!Number.isFinite(value)) {
		return 0;
	}

	return Math.min(100, Math.max(0, value));
}

export function roundPercent(value: number): number {
	return Math.round(value * 100) / 100;
}
