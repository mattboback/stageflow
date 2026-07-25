export interface Rgba {
	r: number;
	g: number;
	b: number;
	a: number;
}

export type ContrastLevel = 'AA' | 'AAA';

export const WCAG_THRESHOLDS: Record<ContrastLevel, { normal: number; large: number }> = {
	AA: { normal: 4.5, large: 3.0 },
	AAA: { normal: 7.0, large: 4.5 }
};

const HEX_PATTERN = /^#?([0-9a-f]{3,4}|[0-9a-f]{6}|[0-9a-f]{8})$/i;
const RGB_PATTERN =
	/^rgba?\(\s*([\d.]+)\s*[, ]\s*([\d.]+)\s*[, ]\s*([\d.]+)\s*(?:[,/]\s*([\d.]+%?)\s*)?\)$/i;

function clampChannel(value: number): number {
	return Math.min(255, Math.max(0, Math.round(value)));
}

export function parseColor(input: string | null | undefined): Rgba | null {
	if (!input) {
		return null;
	}
	const value = input.trim();

	const hexMatch = HEX_PATTERN.exec(value);
	if (hexMatch) {
		let hex = hexMatch[1];
		if (hex === undefined) {
			return null;
		}
		if (hex.length === 3 || hex.length === 4) {
			hex = hex
				.split('')
				.map((c) => c + c)
				.join('');
		}
		const r = parseInt(hex.slice(0, 2), 16);
		const g = parseInt(hex.slice(2, 4), 16);
		const b = parseInt(hex.slice(4, 6), 16);
		const a = hex.length === 8 ? parseInt(hex.slice(6, 8), 16) / 255 : 1;
		return { r, g, b, a };
	}

	const rgbMatch = RGB_PATTERN.exec(value);
	if (rgbMatch) {
		const r = clampChannel(Number(rgbMatch[1]));
		const g = clampChannel(Number(rgbMatch[2]));
		const b = clampChannel(Number(rgbMatch[3]));
		const alpha: string | undefined = rgbMatch[4];
		let a = 1;
		if (alpha !== undefined) {
			a = alpha.endsWith('%') ? Number(alpha.slice(0, -1)) / 100 : Number(alpha);
		}
		if ([r, g, b, a].some((n) => Number.isNaN(n))) {
			return null;
		}
		return { r, g, b, a: Math.min(1, Math.max(0, a)) };
	}

	return null;
}

export function rgbToHex({ r, g, b }: Pick<Rgba, 'r' | 'g' | 'b'>): string {
	const toHex = (n: number) => clampChannel(n).toString(16).padStart(2, '0');
	return `#${toHex(r)}${toHex(g)}${toHex(b)}`;
}

export function compositeOver(fg: Rgba, bg: Rgba): Rgba {
	if (fg.a >= 1) {
		return fg;
	}
	const blend = (f: number, b: number) => f * fg.a + b * (1 - fg.a);
	return { r: blend(fg.r, bg.r), g: blend(fg.g, bg.g), b: blend(fg.b, bg.b), a: 1 };
}

export function relativeLuminance({ r, g, b }: Pick<Rgba, 'r' | 'g' | 'b'>): number {
	const linear = (channel: number) => {
		const c = channel / 255;
		return c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4);
	};
	return 0.2126 * linear(r) + 0.7152 * linear(g) + 0.0722 * linear(b);
}

export function contrastRatio(c1: Rgba, c2: Rgba): number {
	const composited = c1.a < 1 ? compositeOver(c1, { ...c2, a: 1 }) : c1;
	const l1 = relativeLuminance(composited);
	const l2 = relativeLuminance(c2);
	const lighter = Math.max(l1, l2);
	const darker = Math.min(l1, l2);
	return (lighter + 0.05) / (darker + 0.05);
}

export function contrastRatioFromStrings(
	fg: string | null | undefined,
	bg: string | null | undefined
): number | null {
	const fgColor = parseColor(fg);
	const bgColor = parseColor(bg);
	if (!fgColor || !bgColor) {
		return null;
	}
	return contrastRatio(fgColor, bgColor);
}

export function formatRatio(ratio: number): string {
	return (Math.round(ratio * 100) / 100).toFixed(2);
}

export function isLargeText(fontSizePx: number, bold: boolean): boolean {
	const pt = fontSizePx * 0.75;
	const epsilon = 0.01;
	return pt >= 18 - epsilon || (bold && pt >= 14 - epsilon);
}

export function isBoldWeight(fontWeight: string | number | null | undefined): boolean {
	if (fontWeight === null || fontWeight === undefined) {
		return false;
	}
	if (typeof fontWeight === 'number') {
		return fontWeight >= 700;
	}
	const normalized = fontWeight.trim().toLowerCase();
	if (normalized === 'bold' || normalized === 'bolder') {
		return true;
	}
	const numeric = Number(normalized);
	return !Number.isNaN(numeric) && numeric >= 700;
}

export function parseAxeFontSize(fontSize: string | number | null | undefined): number | null {
	if (typeof fontSize === 'number') {
		return Number.isFinite(fontSize) ? fontSize : null;
	}
	if (!fontSize) {
		return null;
	}

	const pxMatch = /([\d.]+)\s*px/i.exec(fontSize);
	if (pxMatch) {
		const px = Number(pxMatch[1]);
		return Number.isNaN(px) ? null : px;
	}
	const ptMatch = /([\d.]+)\s*pt/i.exec(fontSize);
	if (ptMatch) {
		const pt = Number(ptMatch[1]);
		return Number.isNaN(pt) ? null : pt / 0.75;
	}
	const bare = Number(fontSize);
	return Number.isNaN(bare) ? null : bare;
}

export function requiredLevel(ruleId: string | null | undefined): ContrastLevel {
	return ruleId?.includes('color-contrast-enhanced') ? 'AAA' : 'AA';
}

const MESSAGE_KEY_DESCRIPTIONS: Record<string, string> = {
	bgImage:
		'The text sits on a background image, so the effective background color could not be determined automatically.',
	bgGradient:
		'The text sits on a gradient background, so a single background color could not be determined automatically.',
	imgNode:
		'The element contains an image, so the background behind the text could not be determined automatically.',
	bgOverlap:
		'Another element overlaps this text, so the background color could not be determined automatically.',
	fgAlpha:
		'The text color is semi-transparent, so the effective contrast could not be determined automatically.',
	elmPartiallyObscured:
		'The element is partially obscured by other content, so contrast could not be determined automatically.',
	elmPartiallyObscuring:
		'The element partially overlaps other content, so contrast could not be determined automatically.',
	pseudoContent:
		'A pseudo-element may be affecting the background, so contrast could not be determined automatically.',
	complexTextShadows:
		'Text shadows affect the rendered contrast, so it could not be determined automatically.',
	outsideViewport:
		'The element was outside the viewport, so its rendered colors could not be determined automatically.',
	equalRatio:
		'The text and background colors are identical, which usually means the real background comes from elsewhere.',
	shortTextContent:
		'The element contains only punctuation or very short text, which may not be meaningful content.'
};

export function describeMessageKey(messageKey: string | null | undefined): string {
	if (messageKey && MESSAGE_KEY_DESCRIPTIONS[messageKey]) {
		return MESSAGE_KEY_DESCRIPTIONS[messageKey];
	}
	return 'axe-core could not automatically determine the contrast for this element.';
}
