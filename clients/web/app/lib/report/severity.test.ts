import { describe, expect, it } from 'vitest';

import {
	SEVERITY_LEVELS,
	compareSeverity,
	getWorstSeverity,
	normalizeSeverity,
	severityRank
} from './severity';

describe('normalizeSeverity', () => {
	it('accepts every canonical severity level', () => {
		for (const level of SEVERITY_LEVELS) {
			expect(normalizeSeverity(level)).toBe(level);
		}
	});

	it('rejects unknown, empty, and missing values', () => {
		expect(normalizeSeverity('catastrophic')).toBeNull();
		expect(normalizeSeverity('')).toBeNull();
		expect(normalizeSeverity(null)).toBeNull();
		expect(normalizeSeverity(undefined)).toBeNull();
	});
});

describe('severityRank', () => {
	it('ranks critical as most severe and info as least', () => {
		expect(severityRank('critical')).toBe(0);
		expect(severityRank('critical')).toBeLessThan(severityRank('serious'));
		expect(severityRank('serious')).toBeLessThan(severityRank('moderate'));
		expect(severityRank('moderate')).toBeLessThan(severityRank('minor'));
		expect(severityRank('minor')).toBeLessThan(severityRank('info'));
	});

	it('ranks unknown severities after every real level', () => {
		expect(severityRank('bogus')).toBeGreaterThan(severityRank('info'));
		expect(severityRank(null)).toBeGreaterThan(severityRank('info'));
	});
});

describe('getWorstSeverity', () => {
	it('picks the most severe value and ignores unknowns', () => {
		expect(getWorstSeverity(['minor', 'serious', 'bogus', null])).toBe('serious');
		expect(getWorstSeverity(['info', 'critical', 'moderate'])).toBe('critical');
	});

	it('returns null when nothing normalizes', () => {
		expect(getWorstSeverity([])).toBeNull();
		expect(getWorstSeverity(['bogus', null, undefined])).toBeNull();
	});
});

describe('compareSeverity', () => {
	it('sorts a mixed list from most to least severe', () => {
		const sorted = ['info', 'critical', 'minor', 'serious'].sort(compareSeverity);
		expect(sorted).toEqual(['critical', 'serious', 'minor', 'info']);
	});
});
