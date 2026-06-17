import type { IssueSeverity } from '$lib/types/unified-report';

import { chipVariants } from '$lib/components/ui';
import { cn } from '$lib/utils';

export const SEVERITY_LEVELS = [
	'critical',
	'serious',
	'moderate',
	'minor',
	'info'
] as const satisfies IssueSeverity[];

export type SeverityLevel = (typeof SEVERITY_LEVELS)[number];

const SEVERITY_RANK: Record<SeverityLevel, number> = {
	critical: 0,
	serious: 1,
	moderate: 2,
	minor: 3,
	info: 4
};

/**
 * Canonical severity primaries as `r, g, b` triples — these MIRROR the
 * `--color-severity-*` tokens in app.css. They live here (rather than as
 * `var(--color-severity-*)`) because they feed SVG `stroke`/`fill` presentation
 * attributes, which do not resolve CSS custom properties. Keep in sync with
 * app.css; CSS-context severity colors (bars, dots, tracks) read the tokens directly.
 */
const SEVERITY_PRIMARY_RGB: Record<SeverityLevel, string> = {
	critical: '239, 68, 68', // red-500   #ef4444
	serious: '249, 115, 22', // orange-500 #f97316
	moderate: '245, 158, 11', // amber-500 #f59e0b
	minor: '59, 130, 246', // blue-500  #3b82f6
	info: '168, 85, 247' // purple-500 #a855f7
};
/** Neutral fallback — mirrors --color-ink-faint (#6f6961). */
const SEVERITY_NEUTRAL_RGB = '111, 105, 97';

function severityRgb(severity?: string | null): string {
	const level = normalizeSeverity(severity);
	return level ? SEVERITY_PRIMARY_RGB[level] : SEVERITY_NEUTRAL_RGB;
}

export function normalizeSeverity(value?: string | null): SeverityLevel | null {
	if (!value) return null;
	return (SEVERITY_LEVELS as readonly string[]).includes(value) ? (value as SeverityLevel) : null;
}

export function severityRank(value?: string | null): number {
	const normalized = normalizeSeverity(value);
	if (!normalized) return 99;
	return SEVERITY_RANK[normalized];
}

export function getWorstSeverity(values: (string | null | undefined)[]): SeverityLevel | null {
	let worst: SeverityLevel | null = null;
	for (const value of values) {
		const normalized = normalizeSeverity(value);
		if (!normalized) continue;
		if (!worst || severityRank(normalized) < severityRank(worst)) {
			worst = normalized;
		}
	}
	return worst;
}

export function compareSeverity(a?: string | null, b?: string | null): number {
	return severityRank(a) - severityRank(b);
}

/**
 * Returns Tailwind classes for a softened severity container (Nexus variant).
 * Uses pastel bg + subtle border — saturation reserved for dots/icons.
 */
export function getSeverityContainerClass(severity?: string | null): string {
	switch (severity) {
		case 'critical':
			return 'bg-red-50 border-red-100';
		case 'serious':
			return 'bg-orange-50 border-orange-100';
		case 'moderate':
			return 'bg-amber-50 border-amber-100';
		case 'minor':
			return 'bg-blue-50 border-blue-100';
		case 'info':
			return 'bg-purple-50 border-purple-100';
		default:
			return 'bg-surface-muted border-line';
	}
}

/**
 * Returns Tailwind class for a small saturated severity dot/badge.
 */
export function getSeverityDotClass(severity?: string | null): string {
	switch (severity) {
		case 'critical':
			return 'bg-red-500';
		case 'serious':
			return 'bg-orange-500';
		case 'moderate':
			return 'bg-amber-500';
		case 'minor':
			return 'bg-blue-500';
		case 'info':
			return 'bg-purple-500';
		default:
			return 'bg-ink-faint';
	}
}

/**
 * Returns Tailwind classes for a severity badge (solid background).
 */
export function getSeverityBadgeClass(severity?: string | null): string {
	switch (severity) {
		case 'critical':
			return 'bg-red-600 text-white';
		case 'serious':
			return 'bg-orange-500 text-white';
		case 'moderate':
			return 'bg-amber-500 text-white';
		case 'minor':
			return 'bg-blue-500 text-white';
		case 'info':
			return 'bg-purple-500 text-white';
		default:
			return 'bg-ink-faint text-white';
	}
}

/**
 * Returns Tailwind classes for a severity-highlighted container (light background + border).
 */
export function getSeverityBorderClass(severity?: string | null): string {
	switch (severity) {
		case 'critical':
			return 'border-red-200 bg-red-50';
		case 'serious':
			return 'border-orange-200 bg-orange-50';
		case 'moderate':
			return 'border-amber-200 bg-amber-50';
		case 'minor':
			return 'border-blue-200 bg-blue-50';
		case 'info':
			return 'border-purple-200 bg-purple-50';
		default:
			return 'border-line bg-surface-muted';
	}
}

/**
 * Returns Tailwind classes for overlay markers on page screenshots.
 */
export function getSeverityOverlayClass(severity?: string | null): string {
	switch (severity) {
		case 'critical':
			return 'border-red-500 bg-red-500/10';
		case 'serious':
			return 'border-orange-500 bg-orange-500/10';
		case 'moderate':
			return 'border-amber-500 bg-amber-500/10';
		case 'minor':
			return 'border-blue-500 bg-blue-500/10';
		case 'info':
			return 'border-purple-500 bg-purple-500/10';
		default:
			return 'border-ink-faint bg-ink-faint/15';
	}
}

/**
 * Returns a CSS color value for SVG stroke based on severity.
 * Used for SVG-based overlays where Tailwind classes don't apply.
 */
export function getSeverityStrokeColor(severity?: string | null): string {
	return `rgba(${severityRgb(severity)}, 0.95)`;
}

/**
 * Returns a CSS color value for SVG fill based on severity (for focus/hover states).
 * Used for SVG-based overlays where Tailwind classes don't apply.
 */
export function getSeverityFillColor(severity?: string | null): string {
	return `rgba(${severityRgb(severity)}, 0.15)`;
}

/**
 * Returns Tailwind classes for interactive severity filter chips.
 * Supports active/inactive states and multiple sizes.
 */
export function getSeverityChipClass(
	severity: string,
	isActive: boolean,
	size: 'sm' | 'md' = 'sm'
): string {
	const base = chipVariants({ caps: true, interactive: true, size });

	if (severity === 'all') {
		return cn(
			base,
			isActive
				? 'border-ink bg-ink text-surface'
				: 'border-line text-ink-muted hover:bg-surface-muted'
		);
	}

	switch (severity) {
		case 'critical':
			return cn(
				base,
				isActive
					? 'border-red-600 bg-red-600 text-white'
					: 'border-red-200 text-red-600 hover:bg-red-50'
			);
		case 'serious':
			return cn(
				base,
				isActive
					? 'border-orange-500 bg-orange-500 text-white'
					: 'border-orange-200 text-orange-600 hover:bg-orange-50'
			);
		case 'moderate':
			return cn(
				base,
				isActive
					? 'border-amber-500 bg-amber-500 text-white'
					: 'border-amber-200 text-amber-600 hover:bg-amber-50'
			);
		case 'minor':
			return cn(
				base,
				isActive
					? 'border-blue-500 bg-blue-500 text-white'
					: 'border-blue-200 text-blue-600 hover:bg-blue-50'
			);
		case 'info':
			return cn(
				base,
				isActive
					? 'border-purple-500 bg-purple-500 text-white'
					: 'border-purple-200 text-purple-600 hover:bg-purple-50'
			);
		default:
			return cn(base, 'border-line text-ink-muted hover:bg-surface-muted');
	}
}
