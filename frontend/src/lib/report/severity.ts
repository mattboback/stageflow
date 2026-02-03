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
			return 'bg-slate-400 text-white';
	}
}

/**
 * Returns Tailwind classes for a severity-highlighted container (light background + border).
 * Dark mode compatible.
 */
export function getSeverityBorderClass(severity?: string | null): string {
	switch (severity) {
		case 'critical':
			return 'border-red-200 bg-red-50 dark:border-red-800 dark:bg-red-950';
		case 'serious':
			return 'border-orange-200 bg-orange-50 dark:border-orange-800 dark:bg-orange-950';
		case 'moderate':
			return 'border-amber-200 bg-amber-50 dark:border-amber-800 dark:bg-amber-950';
		case 'minor':
			return 'border-blue-200 bg-blue-50 dark:border-blue-800 dark:bg-blue-950';
		case 'info':
			return 'border-purple-200 bg-purple-50 dark:border-purple-800 dark:bg-purple-950';
		default:
			return 'border-slate-200 bg-slate-50 dark:border-slate-700 dark:bg-slate-900';
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
			return 'border-slate-400 bg-slate-200/30';
	}
}

/**
 * Returns a CSS color value for SVG stroke based on severity.
 * Used for SVG-based overlays where Tailwind classes don't apply.
 */
export function getSeverityStrokeColor(severity?: string | null): string {
	switch (severity) {
		case 'critical':
			return 'rgba(239, 68, 68, 0.95)'; // red-500
		case 'serious':
			return 'rgba(249, 115, 22, 0.95)'; // orange-500
		case 'moderate':
			return 'rgba(245, 158, 11, 0.95)'; // amber-500
		case 'minor':
			return 'rgba(59, 130, 246, 0.95)'; // blue-500
		case 'info':
			return 'rgba(168, 85, 247, 0.95)'; // purple-500
		default:
			return 'rgba(148, 163, 184, 0.95)'; // slate-400
	}
}

/**
 * Returns a CSS color value for SVG fill based on severity (for focus/hover states).
 * Used for SVG-based overlays where Tailwind classes don't apply.
 */
export function getSeverityFillColor(severity?: string | null): string {
	switch (severity) {
		case 'critical':
			return 'rgba(239, 68, 68, 0.15)'; // red-500 @ 15%
		case 'serious':
			return 'rgba(249, 115, 22, 0.15)'; // orange-500 @ 15%
		case 'moderate':
			return 'rgba(245, 158, 11, 0.15)'; // amber-500 @ 15%
		case 'minor':
			return 'rgba(59, 130, 246, 0.15)'; // blue-500 @ 15%
		case 'info':
			return 'rgba(168, 85, 247, 0.15)'; // purple-500 @ 15%
		default:
			return 'rgba(148, 163, 184, 0.15)'; // slate-400 @ 15%
	}
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
					: 'border-red-200 text-red-600 hover:bg-red-50 dark:border-red-800 dark:text-red-400 dark:hover:bg-red-950'
			);
		case 'serious':
			return cn(
				base,
				isActive
					? 'border-orange-500 bg-orange-500 text-white'
					: 'border-orange-200 text-orange-600 hover:bg-orange-50 dark:border-orange-800 dark:text-orange-400 dark:hover:bg-orange-950'
			);
		case 'moderate':
			return cn(
				base,
				isActive
					? 'border-amber-500 bg-amber-500 text-white'
					: 'border-amber-200 text-amber-600 hover:bg-amber-50 dark:border-amber-800 dark:text-amber-400 dark:hover:bg-amber-950'
			);
		case 'minor':
			return cn(
				base,
				isActive
					? 'border-blue-500 bg-blue-500 text-white'
					: 'border-blue-200 text-blue-600 hover:bg-blue-50 dark:border-blue-800 dark:text-blue-400 dark:hover:bg-blue-950'
			);
		case 'info':
			return cn(
				base,
				isActive
					? 'border-purple-500 bg-purple-500 text-white'
					: 'border-purple-200 text-purple-600 hover:bg-purple-50 dark:border-purple-800 dark:text-purple-400 dark:hover:bg-purple-950'
			);
		default:
			return cn(base, 'border-line text-ink-muted hover:bg-surface-muted');
	}
}
