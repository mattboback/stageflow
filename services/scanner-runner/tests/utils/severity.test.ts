import { describe, expect, it } from 'vitest';

import { SEVERITY_LEVELS, normalizeSeverity } from '../../src/utils/severity';

describe('SEVERITY_LEVELS', () => {
	it('contains all expected severity levels', () => {
		expect(SEVERITY_LEVELS).toContain('critical');
		expect(SEVERITY_LEVELS).toContain('serious');
		expect(SEVERITY_LEVELS).toContain('moderate');
		expect(SEVERITY_LEVELS).toContain('minor');
		expect(SEVERITY_LEVELS).toContain('info');
	});

	it('has levels in order from most to least severe', () => {
		expect(SEVERITY_LEVELS[0]).toBe('critical');
		expect(SEVERITY_LEVELS[4]).toBe('info');
	});
});

describe('normalizeSeverity', () => {
	it('normalizes uppercase to lowercase', () => {
		expect(normalizeSeverity('CRITICAL')).toBe('critical');
		expect(normalizeSeverity('SERIOUS')).toBe('serious');
		expect(normalizeSeverity('MODERATE')).toBe('moderate');
		expect(normalizeSeverity('MINOR')).toBe('minor');
		expect(normalizeSeverity('INFO')).toBe('info');
	});

	it('trims whitespace', () => {
		expect(normalizeSeverity('  moderate  ')).toBe('moderate');
		expect(normalizeSeverity('\tcritical\n')).toBe('critical');
	});

	it('handles mixed case', () => {
		expect(normalizeSeverity('CrItIcAl')).toBe('critical');
		expect(normalizeSeverity('Serious')).toBe('serious');
	});

	it('returns fallback for invalid values', () => {
		expect(normalizeSeverity('unknown', 'minor')).toBe('minor');
		expect(normalizeSeverity('invalid')).toBe('info'); // default fallback
		expect(normalizeSeverity('warning', 'serious')).toBe('serious');
	});

	it('returns fallback for null/undefined', () => {
		expect(normalizeSeverity(null)).toBe('info');
		expect(normalizeSeverity(undefined)).toBe('info');
		expect(normalizeSeverity(null, 'critical')).toBe('critical');
	});

	it('returns fallback for empty string', () => {
		expect(normalizeSeverity('')).toBe('info');
		expect(normalizeSeverity('   ')).toBe('info');
	});
});
