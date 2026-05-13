import {
	SEVERITY_LEVELS,
	compareSeverity,
	getSeverityBadgeClass,
	getSeverityBorderClass,
	getSeverityChipClass,
	getSeverityContainerClass,
	getSeverityDotClass,
	getSeverityOverlayClass,
	getSeverityStrokeColor,
	getWorstSeverity,
	normalizeSeverity,
	severityRank
} from '$lib/report/severity';
import { describe, expect, it } from 'vitest';

describe('severity utilities', () => {
	describe('SEVERITY_LEVELS', () => {
		it('contains all expected severity levels in order', () => {
			expect(SEVERITY_LEVELS).toEqual(['critical', 'serious', 'moderate', 'minor', 'info']);
		});
	});

	describe('normalizeSeverity', () => {
		it('returns null for empty or null input', () => {
			expect(normalizeSeverity(null)).toBeNull();
			expect(normalizeSeverity(undefined)).toBeNull();
			expect(normalizeSeverity('')).toBeNull();
		});

		it('returns the severity level for valid inputs', () => {
			expect(normalizeSeverity('critical')).toBe('critical');
			expect(normalizeSeverity('serious')).toBe('serious');
			expect(normalizeSeverity('moderate')).toBe('moderate');
			expect(normalizeSeverity('minor')).toBe('minor');
			expect(normalizeSeverity('info')).toBe('info');
		});

		it('returns null for invalid severity values', () => {
			expect(normalizeSeverity('invalid')).toBeNull();
			expect(normalizeSeverity('CRITICAL')).toBeNull(); // case sensitive
		});
	});

	describe('severityRank', () => {
		it('returns correct rank for each severity level', () => {
			expect(severityRank('critical')).toBe(0);
			expect(severityRank('serious')).toBe(1);
			expect(severityRank('moderate')).toBe(2);
			expect(severityRank('minor')).toBe(3);
			expect(severityRank('info')).toBe(4);
		});

		it('returns 99 for invalid or null values', () => {
			expect(severityRank(null)).toBe(99);
			expect(severityRank('unknown')).toBe(99);
		});
	});

	describe('getWorstSeverity', () => {
		it('returns null for empty array', () => {
			expect(getWorstSeverity([])).toBeNull();
		});

		it('returns the worst (lowest rank) severity', () => {
			expect(getWorstSeverity(['moderate', 'critical', 'minor'])).toBe('critical');
			expect(getWorstSeverity(['info', 'moderate'])).toBe('moderate');
			expect(getWorstSeverity(['info'])).toBe('info');
		});

		it('handles null values in the array', () => {
			expect(getWorstSeverity([null, 'moderate', undefined])).toBe('moderate');
		});
	});

	describe('compareSeverity', () => {
		it('returns negative when first is more severe', () => {
			expect(compareSeverity('critical', 'serious')).toBeLessThan(0);
		});

		it('returns positive when first is less severe', () => {
			expect(compareSeverity('info', 'critical')).toBeGreaterThan(0);
		});

		it('returns 0 when severities are equal', () => {
			expect(compareSeverity('moderate', 'moderate')).toBe(0);
		});
	});

	describe('getSeverityBadgeClass', () => {
		it('returns correct classes for each severity', () => {
			expect(getSeverityBadgeClass('critical')).toContain('bg-red');
			expect(getSeverityBadgeClass('serious')).toContain('bg-orange');
			expect(getSeverityBadgeClass('moderate')).toContain('bg-amber');
			expect(getSeverityBadgeClass('minor')).toContain('bg-blue');
			expect(getSeverityBadgeClass('info')).toContain('bg-purple');
		});

		it('returns slate for unknown severity', () => {
			expect(getSeverityBadgeClass('unknown')).toContain('bg-slate');
			expect(getSeverityBadgeClass(null)).toContain('bg-slate');
		});
	});

	describe('getSeverityBorderClass', () => {
		it('returns correct classes for each severity', () => {
			expect(getSeverityBorderClass('critical')).toContain('border-red');
			expect(getSeverityBorderClass('serious')).toContain('border-orange');
			expect(getSeverityBorderClass('moderate')).toContain('border-amber');
			expect(getSeverityBorderClass('minor')).toContain('border-blue');
			expect(getSeverityBorderClass('info')).toContain('border-purple');
		});

		it('returns slate for unknown severity', () => {
			expect(getSeverityBorderClass('unknown')).toContain('border-slate');
		});
	});

	describe('getSeverityOverlayClass', () => {
		it('returns correct classes for each severity', () => {
			expect(getSeverityOverlayClass('critical')).toContain('border-red');
			expect(getSeverityOverlayClass('critical')).toContain('bg-red');
			expect(getSeverityOverlayClass('serious')).toContain('border-orange');
			expect(getSeverityOverlayClass('moderate')).toContain('border-amber');
			expect(getSeverityOverlayClass('minor')).toContain('border-blue');
			expect(getSeverityOverlayClass('info')).toContain('border-purple');
		});
	});

	describe('getSeverityStrokeColor', () => {
		it('returns correct RGBA color for critical severity', () => {
			const color = getSeverityStrokeColor('critical');
			expect(color).toBe('rgba(239, 68, 68, 0.95)');
		});

		it('returns correct RGBA color for serious severity', () => {
			const color = getSeverityStrokeColor('serious');
			expect(color).toBe('rgba(249, 115, 22, 0.95)');
		});

		it('returns correct RGBA color for moderate severity', () => {
			const color = getSeverityStrokeColor('moderate');
			expect(color).toBe('rgba(245, 158, 11, 0.95)');
		});

		it('returns correct RGBA color for minor severity', () => {
			const color = getSeverityStrokeColor('minor');
			expect(color).toBe('rgba(59, 130, 246, 0.95)');
		});

		it('returns correct RGBA color for info severity', () => {
			const color = getSeverityStrokeColor('info');
			expect(color).toBe('rgba(168, 85, 247, 0.95)');
		});

		it('returns slate color for unknown severity', () => {
			const color = getSeverityStrokeColor('unknown');
			expect(color).toBe('rgba(148, 163, 184, 0.95)');
		});

		it('returns slate color for null severity', () => {
			const color = getSeverityStrokeColor(null);
			expect(color).toBe('rgba(148, 163, 184, 0.95)');
		});

		it('returns slate color for undefined severity', () => {
			const color = getSeverityStrokeColor(undefined);
			expect(color).toBe('rgba(148, 163, 184, 0.95)');
		});

		it('returns valid CSS rgba values', () => {
			for (const severity of SEVERITY_LEVELS) {
				const color = getSeverityStrokeColor(severity);
				expect(color).toMatch(/^rgba\(\d+,\s*\d+,\s*\d+,\s*[\d.]+\)$/);
			}
		});
	});

	describe('getSeverityChipClass', () => {
		it('returns different classes for active vs inactive state', () => {
			const activeClass = getSeverityChipClass('critical', true);
			const inactiveClass = getSeverityChipClass('critical', false);
			expect(activeClass).not.toBe(inactiveClass);
		});

		it('returns special classes for "all" filter', () => {
			const allActive = getSeverityChipClass('all', true);
			const allInactive = getSeverityChipClass('all', false);
			expect(allActive).toContain('bg-ink');
			expect(allInactive).toContain('border-line');
		});

		it('returns correct classes for each severity when active', () => {
			expect(getSeverityChipClass('critical', true)).toContain('bg-red');
			expect(getSeverityChipClass('serious', true)).toContain('bg-orange');
			expect(getSeverityChipClass('moderate', true)).toContain('bg-amber');
			expect(getSeverityChipClass('minor', true)).toContain('bg-blue');
			expect(getSeverityChipClass('info', true)).toContain('bg-purple');
		});
	});

	describe('getSeverityContainerClass', () => {
		it('returns soft tint bg + subtle border for each severity', () => {
			expect(getSeverityContainerClass('critical')).toBe('bg-red-50 border-red-100');
			expect(getSeverityContainerClass('serious')).toBe('bg-orange-50 border-orange-100');
			expect(getSeverityContainerClass('moderate')).toBe('bg-amber-50 border-amber-100');
			expect(getSeverityContainerClass('minor')).toBe('bg-blue-50 border-blue-100');
			expect(getSeverityContainerClass('info')).toBe('bg-purple-50 border-purple-100');
		});

		it('returns slate for unknown severity', () => {
			expect(getSeverityContainerClass('unknown')).toBe('bg-slate-50 border-slate-100');
			expect(getSeverityContainerClass(null)).toBe('bg-slate-50 border-slate-100');
			expect(getSeverityContainerClass(undefined)).toBe('bg-slate-50 border-slate-100');
		});
	});

	describe('getSeverityDotClass', () => {
		it('returns saturated dot color for each severity', () => {
			expect(getSeverityDotClass('critical')).toBe('bg-red-500');
			expect(getSeverityDotClass('serious')).toBe('bg-orange-500');
			expect(getSeverityDotClass('moderate')).toBe('bg-amber-500');
			expect(getSeverityDotClass('minor')).toBe('bg-blue-500');
			expect(getSeverityDotClass('info')).toBe('bg-purple-500');
		});

		it('returns slate for unknown severity', () => {
			expect(getSeverityDotClass('unknown')).toBe('bg-slate-400');
			expect(getSeverityDotClass(null)).toBe('bg-slate-400');
			expect(getSeverityDotClass(undefined)).toBe('bg-slate-400');
		});
	});
});
