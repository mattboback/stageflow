import type { IssueSeverity } from '../core/types';

export const SEVERITY_LEVELS = ['critical', 'serious', 'moderate', 'minor', 'info'] as const;

export function normalizeSeverity(
	value: string | undefined | null,
	fallback: IssueSeverity = 'info'
): IssueSeverity {
	if (!value) {
		return fallback;
	}

	const normalized = value.trim().toLowerCase();
	if ((SEVERITY_LEVELS as readonly string[]).includes(normalized)) {
		return normalized as IssueSeverity;
	}

	return fallback;
}

export function incrementSeverityCount(
	counts: Record<Exclude<IssueSeverity, 'info'>, number>,
	severity: IssueSeverity
): void {
	if (severity === 'info') {
		return;
	}
	counts[severity] += 1;
}
