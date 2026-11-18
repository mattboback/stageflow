import { fireEvent, render, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { ContrastSampler } from './ContrastSampler';

class MockImage {
	naturalWidth = 200;
	naturalHeight = 300;
	crossOrigin = '';
	onload: (() => void) | null = null;
	onerror: (() => void) | null = null;

	set src(_value: string) {
		queueMicrotask(() => this.onload?.());
	}
}

class MockDOMPoint {
	x: number;
	y: number;

	constructor(x: number, y: number) {
		this.x = x;
		this.y = y;
	}

	matrixTransform() {
		return this;
	}
}

afterEach(() => {
	vi.restoreAllMocks();
	vi.unstubAllGlobals();
});

describe('ContrastSampler', () => {
	it('samples the clicked point instead of the previous cursor position', async () => {
		vi.stubGlobal('Image', MockImage);
		vi.stubGlobal('DOMPoint', MockDOMPoint);

		const getImageData = vi.fn((x: number, y: number) => ({
			data: [x, y, 0, 255]
		}));
		vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockImplementation(function (
			this: HTMLCanvasElement
		) {
			return {
				canvas: this,
				drawImage: vi.fn(),
				getImageData,
				fillRect: vi.fn(),
				strokeRect: vi.fn(),
				imageSmoothingEnabled: false,
				fillStyle: '',
				strokeStyle: ''
			} as unknown as CanvasRenderingContext2D;
		});

		const onPick = vi.fn();
		const { container } = render(
			<ContrastSampler
				imageUrl="https://example.com/screenshot.png"
				pageWidth={100}
				pageHeight={100}
				viewBox={{ x: 0, y: 0, width: 100, height: 100 }}
				onPick={onPick}
			/>
		);

		await waitFor(() => expect(getImageData).toHaveBeenCalled());
		const svg = container.querySelector('svg')!;
		Object.defineProperty(svg, 'getScreenCTM', {
			value: () => ({ inverse: () => ({}) })
		});

		getImageData.mockClear();
		fireEvent.click(svg, { clientX: 25, clientY: 25 });

		expect(getImageData).toHaveBeenLastCalledWith(50, 75, 1, 1);
		expect(onPick).toHaveBeenCalledWith('fg', '#324b00');
	});
});
