import { describe, expect, it } from 'vitest';

import {
	SEVERITY_LEVELS,
	compareSeverity,
	getWorstSeverity,
	normalizeSeverity,
	severitySortIndex
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

describe('severitySortIndex', () => {
	it('places critical first and info last in display order', () => {
		expect(severitySortIndex('critical')).toBe(0);
		expect(severitySortIndex('critical')).toBeLessThan(severitySortIndex('serious'));
		expect(severitySortIndex('serious')).toBeLessThan(severitySortIndex('moderate'));
		expect(severitySortIndex('moderate')).toBeLessThan(severitySortIndex('minor'));
		expect(severitySortIndex('minor')).toBeLessThan(severitySortIndex('info'));
	});

	it('places unknown severities after every real level', () => {
		expect(severitySortIndex('bogus')).toBeGreaterThan(severitySortIndex('info'));
		expect(severitySortIndex(null)).toBeGreaterThan(severitySortIndex('info'));
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
