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
