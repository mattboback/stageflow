import {
	compositeOver,
	contrastRatio,
	contrastRatioFromStrings,
	describeMessageKey,
	formatRatio,
	isBoldWeight,
	isLargeText,
	parseAxeFontSize,
	parseColor,
	relativeLuminance,
	requiredLevel,
	requiredRatio,
	rgbToHex
} from '$lib/utils/contrast';
import { describe, expect, it } from 'vitest';

const WHITE = { r: 255, g: 255, b: 255, a: 1 };
const BLACK = { r: 0, g: 0, b: 0, a: 1 };

describe('contrast', () => {
	describe('parseColor', () => {
		it('parses 6-digit hex', () => {
			expect(parseColor('#1b5c5e')).toEqual({ r: 27, g: 92, b: 94, a: 1 });
		});

		it('parses 3-digit hex with expansion', () => {
			expect(parseColor('#abc')).toEqual({ r: 170, g: 187, b: 204, a: 1 });
		});

		it('parses 8-digit hex with alpha', () => {
			const color = parseColor('#00000080');
			expect(color).not.toBeNull();
			expect(color!.a).toBeCloseTo(128 / 255, 3);
		});

		it('parses hex without a leading hash', () => {
			expect(parseColor('ffffff')).toEqual(WHITE);
		});

		it('parses rgb() with commas', () => {
			expect(parseColor('rgb(1, 2, 3)')).toEqual({ r: 1, g: 2, b: 3, a: 1 });
		});

		it('parses rgba() with decimal alpha', () => {
			expect(parseColor('rgba(0, 0, 0, 0.5)')).toEqual({ r: 0, g: 0, b: 0, a: 0.5 });
		});

		it('parses space-separated rgb with slash alpha', () => {
			expect(parseColor('rgb(10 20 30 / 50%)')).toEqual({ r: 10, g: 20, b: 30, a: 0.5 });
		});

		it('returns null for unparseable input', () => {
			expect(parseColor('magenta')).toBeNull();
			expect(parseColor('')).toBeNull();
			expect(parseColor(null)).toBeNull();
			expect(parseColor(undefined)).toBeNull();
		});
	});

	describe('rgbToHex', () => {
		it('formats channels as lowercase hex', () => {
			expect(rgbToHex({ r: 27, g: 92, b: 94 })).toBe('#1b5c5e');
			expect(rgbToHex({ r: 255, g: 255, b: 255 })).toBe('#ffffff');
		});

		it('clamps out-of-range channels', () => {
			expect(rgbToHex({ r: 300, g: -5, b: 0 })).toBe('#ff0000');
		});
	});

	describe('relativeLuminance', () => {
		it('is 1 for white and 0 for black', () => {
			expect(relativeLuminance(WHITE)).toBeCloseTo(1, 5);
			expect(relativeLuminance(BLACK)).toBeCloseTo(0, 5);
		});
	});

	describe('contrastRatio', () => {
		it('is 21 for black on white', () => {
			expect(contrastRatio(BLACK, WHITE)).toBeCloseTo(21, 2);
		});

		it('is 1 for identical colors', () => {
			expect(contrastRatio(WHITE, WHITE)).toBeCloseTo(1, 5);
		});

		it('is symmetric', () => {
			const grey = parseColor('#777777')!;
			expect(contrastRatio(grey, WHITE)).toBeCloseTo(contrastRatio(WHITE, grey), 5);
		});

		it('matches the known #777 on white value', () => {
			expect(contrastRatio(parseColor('#777777')!, WHITE)).toBeCloseTo(4.48, 2);
		});

		it('matches the known pure red on white value', () => {
			expect(contrastRatio(parseColor('#ff0000')!, WHITE)).toBeCloseTo(4.0, 2);
		});

		it('composites translucent foregrounds over the background', () => {
			const halfBlack = { ...BLACK, a: 0.5 };
			const solidEquivalent = { r: 127.5, g: 127.5, b: 127.5, a: 1 };
			expect(contrastRatio(halfBlack, WHITE)).toBeCloseTo(contrastRatio(solidEquivalent, WHITE), 5);
		});
	});

	describe('compositeOver', () => {
		it('blends by alpha', () => {
			const result = compositeOver({ ...BLACK, a: 0.5 }, WHITE);
			expect(result.r).toBeCloseTo(127.5, 3);
			expect(result.a).toBe(1);
		});

		it('returns opaque colors unchanged', () => {
			expect(compositeOver(BLACK, WHITE)).toEqual(BLACK);
		});
	});

	describe('contrastRatioFromStrings', () => {
		it('parses and computes', () => {
			expect(contrastRatioFromStrings('#000000', '#ffffff')).toBeCloseTo(21, 2);
		});

		it('returns null when either color is unparseable', () => {
			expect(contrastRatioFromStrings('nope', '#ffffff')).toBeNull();
			expect(contrastRatioFromStrings('#000000', undefined)).toBeNull();
		});
	});

	describe('formatRatio', () => {
		it('rounds to two decimals', () => {
			expect(formatRatio(4.4782)).toBe('4.48');
			expect(formatRatio(21)).toBe('21.00');
		});
	});

	describe('isLargeText', () => {
		it('treats 24px+ regular text as large', () => {
			expect(isLargeText(24, false)).toBe(true);
			expect(isLargeText(23, false)).toBe(false);
		});

		it('treats 18.67px+ bold text as large', () => {
			expect(isLargeText(18.6667, true)).toBe(true);
			expect(isLargeText(18.5, true)).toBe(false);
			expect(isLargeText(19, false)).toBe(false);
		});
	});

	describe('isBoldWeight', () => {
		it('handles keyword and numeric weights', () => {
			expect(isBoldWeight('bold')).toBe(true);
			expect(isBoldWeight('normal')).toBe(false);
			expect(isBoldWeight(700)).toBe(true);
			expect(isBoldWeight('600')).toBe(false);
			expect(isBoldWeight(undefined)).toBe(false);
		});
	});

	describe('parseAxeFontSize', () => {
		it("parses axe's pt-with-px format", () => {
			expect(parseAxeFontSize('10.0pt (13.3333px)')).toBeCloseTo(13.3333, 3);
		});

		it('parses bare px and pt values', () => {
			expect(parseAxeFontSize('13.5px')).toBe(13.5);
			expect(parseAxeFontSize('12pt')).toBe(16);
		});

		it('passes through numbers and rejects garbage', () => {
			expect(parseAxeFontSize(16)).toBe(16);
			expect(parseAxeFontSize('garbage')).toBeNull();
			expect(parseAxeFontSize(null)).toBeNull();
		});
	});

	describe('requiredLevel / requiredRatio', () => {
		it('maps rules to conformance levels', () => {
			expect(requiredLevel('color-contrast')).toBe('AA');
			expect(requiredLevel('color-contrast-enhanced')).toBe('AAA');
		});

		it('returns the right thresholds', () => {
			expect(requiredRatio('color-contrast', false)).toBe(4.5);
			expect(requiredRatio('color-contrast', true)).toBe(3.0);
			expect(requiredRatio('color-contrast-enhanced', false)).toBe(7.0);
			expect(requiredRatio('color-contrast-enhanced', true)).toBe(4.5);
		});
	});

	describe('describeMessageKey', () => {
		it('explains known axe message keys', () => {
			expect(describeMessageKey('bgImage')).toContain('background image');
			expect(describeMessageKey('bgGradient')).toContain('gradient');
		});

		it('falls back for unknown keys', () => {
			expect(describeMessageKey('mystery')).toContain('could not automatically determine');
			expect(describeMessageKey(undefined)).toContain('could not automatically determine');
		});
	});
});
