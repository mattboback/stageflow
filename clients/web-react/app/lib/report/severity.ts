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

export function getSeverityContainerClass(severity?: string | null): string {
	return `sev-container${severityToken(severity)}`;
}

export function getSeverityDotClass(severity?: string | null): string {
	return `sev-dot${severityToken(severity)}`;
}

export function getSeverityBadgeClass(severity?: string | null): string {
	return `sev-badge${severityToken(severity)}`;
}

export function getSeverityBorderClass(severity?: string | null): string {
	return `sev-border${severityToken(severity)}`;
}

export function getSeverityOverlayClass(severity?: string | null): string {
	return `sev-overlay${severityToken(severity)}`;
}

export function getSeverityStrokeColor(severity?: string | null): string {
	switch (severity) {
		case 'critical':
			return 'rgba(239, 68, 68, 0.95)'; // red
		case 'serious':
			return 'rgba(249, 115, 22, 0.95)'; // orange
		case 'moderate':
			return 'rgba(245, 158, 11, 0.95)'; // amber
		case 'minor':
			return 'rgba(59, 130, 246, 0.95)'; // blue
		case 'info':
			return 'rgba(168, 85, 247, 0.95)'; // purple
		default:
			return 'rgba(148, 163, 184, 0.95)'; // slate
	}
}

export function getSeverityFillColor(severity?: string | null): string {
	switch (severity) {
		case 'critical':
			return 'rgba(239, 68, 68, 0.15)';
		case 'serious':
			return 'rgba(249, 115, 22, 0.15)';
		case 'moderate':
			return 'rgba(245, 158, 11, 0.15)';
		case 'minor':
			return 'rgba(59, 130, 246, 0.15)';
		case 'info':
			return 'rgba(168, 85, 247, 0.15)';
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
