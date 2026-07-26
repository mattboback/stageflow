import { readFileSync } from 'node:fs';
import path from 'node:path';
import { describe, expect, it } from 'vitest';
import {
	contrastRatio,
	formatRatio,
	oklchToRgb,
	rgbToHex,
	WCAG_THRESHOLDS,
	type ContrastLevel
} from '../lib/utils/contrast';

/*
 * Grades the design system with the product's own WCAG engine.
 *
 * axe already runs over five rendered surfaces at two viewports, but it can
 * only judge colors that are actually painted at the moment it looks. A
 * severity wash that no fixture issue triggers, or a filter chip nobody
 * activated, is never evaluated — and axe reports color-contrast as
 * *incomplete* rather than failing for text over a backdrop-filter, which is
 * exactly what .site-header and .stickyrun are. This table has no such holes:
 * it checks every pair unconditionally, in both themes, straight from the
 * token source.
 */

// Vitest runs with cwd at clients/web, matching app/test/load-fixture.ts.
const INSTRUMENT_CSS = path.resolve(process.cwd(), 'app/styles/instrument.css');

type Mode = 'light' | 'dark';
const MODES: readonly Mode[] = ['light', 'dark'];

/**
 * Largest linear-sRGB excursion tolerated before a token counts as fictional.
 *
 * Saturated colors sit slightly outside sRGB in OKLCH as a matter of course —
 * the worst in the current palette is --sev-serious-wash at 0.028 — and the
 * browser clips them harmlessly. The ceiling exists to catch a *derived* value
 * whose written lightness and chroma are not what renders, because a palette
 * tuned against numbers the screen never shows is not tuned at all.
 */
const GAMUT_TOLERANCE = 0.05;

function splitTopLevel(value: string): string[] {
	const parts: string[] = [];
	let depth = 0;
	let current = '';
	for (const char of value) {
		if (char === '(') depth++;
		if (char === ')') depth--;
		if (char === ',' && depth === 0) {
			parts.push(current.trim());
			current = '';
			continue;
		}
		current += char;
	}
	parts.push(current.trim());
	return parts;
}

function readTokens(mode: Mode): Map<string, string> {
	const css = readFileSync(INSTRUMENT_CSS, 'utf8');
	/*
	 * Anchored to the rule at the start of a line, not to the first occurrence
	 * of the string. The file's own header comment mentions :root[data-theme],
	 * and matching that instead silently reads the wrong block — which is
	 * exactly what happened when @font-face landed between the comment and the
	 * real rule, and every obligation started failing at once.
	 */
	const opening = /^:root\s*\{/m.exec(css);
	if (!opening) throw new Error('instrument.css has no :root rule');
	const start = opening.index;
	const rootBlock = css.slice(start, css.indexOf('\n}', start));

	const tokens = new Map<string, string>();
	for (const match of rootBlock.matchAll(/(--[\w-]+):\s*([^;]+);/g)) {
		const [, name, rawValue] = match;
		if (name === undefined || rawValue === undefined) continue;

		const value = rawValue.trim();
		const lightDark = /^light-dark\(([\s\S]+)\)$/.exec(value);
		if (lightDark?.[1]) {
			const sides = splitTopLevel(lightDark[1]);
			const picked = mode === 'light' ? sides[0] : sides[1];
			if (picked !== undefined) tokens.set(name, picked);
			continue;
		}
		tokens.set(name, value);
	}
	return tokens;
}

function resolve(tokens: Map<string, string>, name: string): string {
	const raw = tokens.get(name);
	if (raw === undefined) {
		throw new Error(`${name} is not declared in instrument.css :root`);
	}
	// Tokens may alias other tokens; follow the chain rather than failing.
	const alias = /^var\((--[\w-]+)\)$/.exec(raw);
	return alias?.[1] ? resolve(tokens, alias[1]) : raw;
}

function colorOf(tokens: Map<string, string>, name: string) {
	const color = oklchToRgb(resolve(tokens, name));
	if (!color) {
		throw new Error(`${name} is not an oklch() color: ${resolve(tokens, name)}`);
	}
	return color;
}

/** One contrast obligation the palette owes, in both themes. */
interface Obligation {
	fg: string;
	bg: string;
	level: ContrastLevel;
	/** 'large' relaxes AA to 3:1 — for display type and non-text marks. */
	size: 'normal' | 'large';
	why: string;
}

/*
 * Severity is four roles, not one color, and each role has a different debt.
 * The base hue is a *mark* (dot, border, overlay) and owes 3:1 as non-text;
 * -ink is the text variant and owes the full 4.5:1 on both the wash it sits in
 * and the plain surface. That split is why amber and green have a separate
 * -ink at all: their base is too light to be read as words.
 */
const SEVERITIES = [
	{ name: 'critical', wash: '--sev-critical-wash', fill: '--sev-critical' },
	{ name: 'serious', wash: '--sev-serious-wash', fill: '--sev-serious' },
	{ name: 'moderate', wash: '--sev-moderate-wash', fill: '--sev-moderate-ink' },
	{ name: 'minor', wash: '--sev-minor-wash', fill: '--sev-minor-ink' },
	{ name: 'info', wash: '--sev-info-wash', fill: '--sev-info' },
	{ name: 'pass', wash: '--sev-minor-wash', fill: '--sev-pass-ink' }
] as const;

const OBLIGATIONS: Obligation[] = [
	...SEVERITIES.flatMap((sev): Obligation[] => [
		{
			fg: `--sev-${sev.name}-ink`,
			bg: sev.wash,
			level: 'AA',
			size: 'normal',
			why: `${sev.name} badge text on its tinted pill`
		},
		{
			fg: `--sev-${sev.name}-ink`,
			bg: '--surface',
			level: 'AA',
			size: 'normal',
			why: `${sev.name} text on a plain card`
		},
		{
			fg: '--on-severity',
			bg: sev.fill,
			level: 'AA',
			size: 'normal',
			why: `label on the filled ${sev.name} chip`
		},
		{
			fg: `--sev-${sev.name}`,
			bg: '--surface',
			level: 'AA',
			size: 'large',
			why: `${sev.name} dot and border, SC 1.4.11 non-text`
		}
	]),

	// Body and interface text.
	{ fg: '--ink', bg: '--ground', level: 'AA', size: 'normal', why: 'body on the page' },
	{ fg: '--ink', bg: '--surface', level: 'AA', size: 'normal', why: 'body on a card' },
	{ fg: '--ink-strong', bg: '--ground', level: 'AA', size: 'normal', why: 'headings' },
	{ fg: '--ink-strong', bg: '--surface', level: 'AA', size: 'normal', why: 'headings on a card' },
	{ fg: '--ink-muted', bg: '--ground', level: 'AA', size: 'normal', why: 'secondary text' },
	{ fg: '--ink-muted', bg: '--surface', level: 'AA', size: 'normal', why: 'secondary on a card' },
	{
		fg: '--ink-faint',
		bg: '--surface',
		level: 'AA',
		size: 'large',
		// The token's own comment reserves it for large/decorative use, so it is
		// held to the large-text floor and must never carry prose.
		why: 'placeholder and decorative text, large only'
	},

	// Signal teal.
	{ fg: '--on-signal', bg: '--signal-strong', level: 'AA', size: 'normal', why: 'primary button' },
	{ fg: '--signal-ink', bg: '--signal-wash', level: 'AA', size: 'normal', why: 'active nav link' },
	{ fg: '--signal-ink', bg: '--surface', level: 'AA', size: 'normal', why: 'link on a card' },
	{ fg: '--signal-ink', bg: '--ground', level: 'AA', size: 'normal', why: 'link on the page' },
	{ fg: '--signal', bg: '--surface', level: 'AA', size: 'large', why: 'focus ring, SC 1.4.11' },

	/*
	 * Input and ghost-button borders. 'large' is this table's 3:1 floor, which
	 * is the non-text threshold SC 1.4.11 asks for — these are not text, they
	 * are the only thing identifying where a control begins. All three
	 * backgrounds are listed because an input appears on the page, on a card,
	 * and inside a sunk panel, and a border tuned against the lightest of
	 * those disappears on the other two.
	 */
	{
		fg: '--line-strong',
		bg: '--surface',
		level: 'AA',
		size: 'large',
		why: 'input border on a card, SC 1.4.11'
	},
	{
		fg: '--line-strong',
		bg: '--ground',
		level: 'AA',
		size: 'large',
		why: 'input border on the page, SC 1.4.11'
	},
	{
		fg: '--line-strong',
		bg: '--surface-sunk',
		level: 'AA',
		size: 'large',
		why: 'input border in a sunk panel, SC 1.4.11'
	},

	// Emphatic variants used by verdict controls.
	{
		fg: '--sev-critical-ink-strong',
		bg: '--sev-critical-wash',
		level: 'AA',
		size: 'normal',
		why: 'fail verdict button'
	},
	{
		fg: '--sev-pass-ink-strong',
		bg: '--sev-minor-wash',
		level: 'AA',
		size: 'normal',
		why: 'pass verdict button'
	},
	{
		fg: '--on-severity',
		bg: '--sev-critical-press',
		level: 'AA',
		size: 'normal',
		why: 'danger button hover'
	},

	// Advisory amber.
	{
		fg: '--warn-rule',
		bg: '--warn-wash',
		level: 'AA',
		size: 'normal',
		why: 'sensitive-data notice'
	},

	/*
	 * The terminal island is the only surface whose foreground/background pair
	 * appears nowhere else, because it deliberately stays dark in both themes.
	 * Nothing else in the table would catch a regression here.
	 */
	{ fg: '--terminal-ink', bg: '--terminal-bg', level: 'AA', size: 'normal', why: 'CLI output' },
	{
		fg: '--terminal-ink-faint',
		bg: '--terminal-bg',
		level: 'AA',
		size: 'normal',
		why: 'CLI empty state'
	},
	{ fg: '--terminal-prompt', bg: '--terminal-bg', level: 'AA', size: 'normal', why: 'CLI prompt' },
	{
		fg: '--terminal-ok',
		bg: '--terminal-bg',
		level: 'AA',
		size: 'normal',
		why: 'CLI success mark'
	},
	{ fg: '--terminal-warn', bg: '--terminal-bg', level: 'AA', size: 'normal', why: 'CLI warning' }
];

/*
 * Without this, a light-dark() parser that quietly stopped splitting would
 * make the dark suite re-check the light palette and still report 92 passes.
 * Every token below is one whose whole purpose is to invert.
 */
describe('theme resolution', () => {
	const light = readTokens('light');
	const dark = readTokens('dark');

	it.each(['--ground', '--surface', '--ink', '--ink-strong', '--line', '--on-severity'])(
		'%s resolves to a different value per theme',
		(token) => {
			expect(resolve(dark, token)).not.toEqual(resolve(light, token));
		}
	);

	it('only uses light-dark() with colors on both sides', () => {
		/*
		 * light-dark() is defined over <color> only. A numeric one still parses
		 * as a custom property, so nothing complains — it fails later, at the
		 * point of use, and the declaration is simply dropped. That is a silent
		 * dead hover or a missing radius, found by eye or not at all.
		 */
		const css = readFileSync(INSTRUMENT_CSS, 'utf8');
		const rootBlock = css.slice(css.indexOf(':root'), css.indexOf('\n}', css.indexOf(':root')));
		const offenders: string[] = [];

		for (const match of rootBlock.matchAll(/(--[\w-]+):\s*light-dark\(([\s\S]+?)\);/g)) {
			const [, name, inner] = match;
			if (name === undefined || inner === undefined) continue;
			for (const side of splitTopLevel(inner)) {
				// Shadows are a color inside a length list, so check the color part.
				const color = /(oklch\([^)]*\)|transparent)/.exec(side)?.[1];
				if (!color || (color !== 'transparent' && !oklchToRgb(color))) {
					offenders.push(`${name}: ${side}`);
				}
			}
		}
		expect(offenders, `not a color:\n${offenders.join('\n')}`).toEqual([]);
	});

	it('keeps the terminal island dark in both themes', () => {
		// It is literal stdout. If it ever inverts, home renders a near-white
		// block of near-white text.
		const lightBg = colorOf(light, '--terminal-bg');
		const darkBg = colorOf(dark, '--terminal-bg');
		const lum = (c: { r: number; g: number; b: number }) =>
			0.2126 * c.r + 0.7152 * c.g + 0.0722 * c.b;
		expect(lum(lightBg)).toBeLessThan(128);
		expect(lum(darkBg)).toBeLessThan(128);
	});
});

describe.each(MODES)('%s theme', (mode) => {
	const tokens = readTokens(mode);

	it.each(OBLIGATIONS)('$fg on $bg meets $level ($why)', ({ fg, bg, level, size }) => {
		const foreground = colorOf(tokens, fg);
		const background = colorOf(tokens, bg);
		const ratio = contrastRatio(foreground, background);
		const required = WCAG_THRESHOLDS[level][size];

		expect(
			ratio,
			`${fg} ${rgbToHex(foreground)} on ${bg} ${rgbToHex(background)} ` +
				`is ${formatRatio(ratio)}:1, needs ${required.toFixed(1)}:1 (${level} ${size})`
		).toBeGreaterThanOrEqual(required);
	});

	it('keeps every color token inside sRGB', () => {
		const escapees: string[] = [];
		for (const [name] of tokens) {
			const color = oklchToRgb(resolve(tokens, name));
			if (!color) continue;
			const { r, g, b } = color.linear;
			const excursion = Math.max(0, -r, -g, -b, r - 1, g - 1, b - 1);
			if (excursion > GAMUT_TOLERANCE) {
				escapees.push(`${name} (${excursion.toFixed(4)} outside)`);
			}
		}
		expect(
			escapees,
			`these clip to a color that is not what was written:\n${escapees.join('\n')}`
		).toEqual([]);
	});
});
