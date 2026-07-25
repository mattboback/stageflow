import type { ViewBox } from './screenshot-crop';
import { clamp } from './screenshot-crop';

export interface SamplePoint {
	x: number;
	y: number;
}

export function clampSamplePoint(point: SamplePoint, viewBox: ViewBox): SamplePoint {
	return {
		x: clamp(point.x, viewBox.x, viewBox.x + viewBox.width),
		y: clamp(point.y, viewBox.y, viewBox.y + viewBox.height)
	};
}

export function toCanvasPixel(
	point: SamplePoint,
	xScale: number,
	yScale: number,
	canvasWidth: number,
	canvasHeight: number
): SamplePoint {
	return {
		x: clamp(Math.round(point.x * xScale), 0, Math.max(0, canvasWidth - 1)),
		y: clamp(Math.round(point.y * yScale), 0, Math.max(0, canvasHeight - 1))
	};
}
