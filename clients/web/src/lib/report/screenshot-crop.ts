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
}

function clamp(value: number, min: number, max: number): number {
	return Math.min(max, Math.max(min, value));
}

export function getCroppedViewBox(
	pageWidth: number,
	pageHeight: number,
	element: Rect,
	options: CropOptions = {}
): ViewBox | null {
	if (!Number.isFinite(pageWidth) || !Number.isFinite(pageHeight) || pageWidth <= 0 || pageHeight <= 0) {
		return null;
	}

	const minWidth = options.minWidth ?? 520;
	const minHeight = options.minHeight ?? 300;
	const padding = options.padding ?? 80;

	const desiredWidth = Math.max(minWidth, element.width + padding * 2);
	const desiredHeight = Math.max(minHeight, element.height + padding * 2);

	const width = Math.min(pageWidth, desiredWidth);
	const height = Math.min(pageHeight, desiredHeight);

	const centerX = element.x + element.width / 2;
	const centerY = element.y + element.height / 2;

	const x = clamp(centerX - width / 2, 0, Math.max(0, pageWidth - width));
	const y = clamp(centerY - height / 2, 0, Math.max(0, pageHeight - height));

	return { x, y, width, height };
}
