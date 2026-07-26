import type { IssueSeverity } from '../types/unified-report';
import { cn } from '../utils';

export const SEVERITY_LEVELS = [
	'critical',
	'serious',
	'moderate',
	'minor',
	'info'
] as const satisfies IssueSeverity[];

export type SeverityLevel = (typeof SEVERITY_LEVELS)[number];

/**
 * Display order, NOT severity magnitude: critical is 0 because it sorts first.
 *
 * The Go CLI's severityRank in clients/cli/internal/render/filter.go deliberately
 * runs the other way (critical is highest) because it gates on `>= threshold`.
 * Two same-named functions with opposite orderings is a trap, so this one is named
 * for what it produces — a sort index — and callers should prefer compareSeverity.
 */
const SEVERITY_SORT_INDEX: Record<SeverityLevel, number> = {
	critical: 0,
	serious: 1,
	moderate: 2,
	minor: 3,
	info: 4
};

/** Unrecognized severities sort after every known level. */
const UNKNOWN_SEVERITY_SORT_INDEX = SEVERITY_LEVELS.length;

/**
 * Validates an already-normalized severity from the report contract.
 *
 * Deliberately stricter than the scanner-runner's normalizeSeverity, which trims,
 * lowercases, and falls back to a default because it handles raw scanner output.
 * By the time a report reaches the browser the values are schema-validated, so an
 * unexpected one is a signal worth surfacing as null rather than silently coercing.
 */
export function normalizeSeverity(value?: string | null): SeverityLevel | null {
	if (!value) return null;
	return (SEVERITY_LEVELS as readonly string[]).includes(value) ? (value as SeverityLevel) : null;
}

/** Position of a severity in display order; lower sorts first. */
export function severitySortIndex(value?: string | null): number {
	const normalized = normalizeSeverity(value);
	if (!normalized) return UNKNOWN_SEVERITY_SORT_INDEX;
	return SEVERITY_SORT_INDEX[normalized];
}

/** The most severe of `values`, or null if none are recognized. */
export function getWorstSeverity(values: (string | null | undefined)[]): SeverityLevel | null {
	let worst: SeverityLevel | null = null;
	for (const value of values) {
		const normalized = normalizeSeverity(value);
		if (!normalized) continue;
		if (!worst || severitySortIndex(normalized) < severitySortIndex(worst)) {
			worst = normalized;
		}
	}
	return worst;
}

/** Comparator placing more severe issues first. */
export function compareSeverity(a?: string | null, b?: string | null): number {
	return severitySortIndex(a) - severitySortIndex(b);
}

function severityToken(severity?: string | null): string {
	const normalized = normalizeSeverity(severity);
	return normalized ? ` sev-${normalized}` : '';
}

export function getSeverityDotClass(severity?: string | null): string {
	return `sev-dot${severityToken(severity)}`;
}

export function getSeverityBadgeClass(severity?: string | null): string {
	return `sev-badge${severityToken(severity)}`;
}

/*
 * SVG presentation attributes can't resolve CSS custom properties, so these
 * mirror the --sev-* tokens in instrument.css (sRGB approximations).
 *
 * These deliberately do NOT follow the theme. They are painted on top of a
 * screenshot of someone else's page, which does not change when our chrome
 * does — a marker tuned for our dark surface would be wrong over a light
 * screenshot. Leave them fixed.
 */
export function getSeverityStrokeColor(severity?: string | null): string {
	switch (severity) {
		case 'critical':
			return 'rgba(198, 44, 41, 0.95)'; // red — --sev-critical
		case 'serious':
			return 'rgba(199, 93, 22, 0.95)'; // orange — --sev-serious
		case 'moderate':
			return 'rgba(175, 123, 15, 0.95)'; // amber — --sev-moderate
		case 'minor':
			return 'rgba(64, 133, 60, 0.95)'; // green — --sev-minor
		case 'info':
			return 'rgba(63, 113, 191, 0.95)'; // blue — --sev-info
		default:
			return 'rgba(148, 163, 184, 0.95)'; // slate
	}
}

export function getSeverityFillColor(severity?: string | null): string {
	switch (severity) {
		case 'critical':
			return 'rgba(198, 44, 41, 0.15)';
		case 'serious':
			return 'rgba(199, 93, 22, 0.15)';
		case 'moderate':
			return 'rgba(175, 123, 15, 0.15)';
		case 'minor':
			return 'rgba(64, 133, 60, 0.15)';
		case 'info':
			return 'rgba(63, 113, 191, 0.15)';
		default:
			return 'rgba(148, 163, 184, 0.15)';
	}
}

export function getSeverityChipClass(
	severity: string,
	isActive: boolean,
	size: 'sm' | 'md' = 'sm'
): string {
	const normalized = normalizeSeverity(severity);
	return cn(
		'sev-chip',
		size === 'md' && 'size-md',
		normalized && `sev-${normalized}`,
		isActive && 'active'
	);
}
