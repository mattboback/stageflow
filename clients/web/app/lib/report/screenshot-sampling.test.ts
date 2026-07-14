import { describe, expect, it } from 'vitest';

import { clampSamplePoint, toCanvasPixel } from './screenshot-sampling';

describe('screenshot sampling coordinates', () => {
	it('uses independent horizontal and vertical image scales', () => {
		expect(toCanvasPixel({ x: 25, y: 25 }, 2, 3, 200, 300)).toEqual({
			x: 50,
			y: 75
		});
	});

	it('clamps canvas pixels and crop coordinates to their bounds', () => {
		expect(toCanvasPixel({ x: 500, y: -10 }, 2, 3, 200, 300)).toEqual({
			x: 199,
			y: 0
		});
		expect(clampSamplePoint({ x: 5, y: 500 }, { x: 10, y: 20, width: 100, height: 200 })).toEqual({
			x: 10,
			y: 220
		});
	});
});
