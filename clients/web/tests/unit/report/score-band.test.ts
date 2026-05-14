import { scoreBandFor } from '$lib/report/score-band';
import { describe, expect, it } from 'vitest';

describe('scoreBandFor', () => {
	it('returns null for null/undefined/NaN', () => {
		expect(scoreBandFor(null)).toBeNull();
		expect(scoreBandFor(undefined)).toBeNull();
		expect(scoreBandFor(Number.NaN)).toBeNull();
	});

	it('returns Strong for 90+', () => {
		expect(scoreBandFor(94)).toEqual({ tone: 'strong', label: 'Strong' });
		expect(scoreBandFor(100)).toEqual({ tone: 'strong', label: 'Strong' });
		expect(scoreBandFor(90)).toEqual({ tone: 'strong', label: 'Strong' });
	});

	it('returns Watch for 80-89', () => {
		expect(scoreBandFor(89)).toEqual({ tone: 'watch', label: 'Watch' });
		expect(scoreBandFor(80)).toEqual({ tone: 'watch', label: 'Watch' });
	});

	it('returns Needs work for 70-79', () => {
		expect(scoreBandFor(75)).toEqual({ tone: 'needs-work', label: 'Needs work' });
	});

	it('returns High risk for 60-69', () => {
		expect(scoreBandFor(65)).toEqual({ tone: 'high-risk', label: 'High risk' });
	});

	it('returns Failing for <60', () => {
		expect(scoreBandFor(42)).toEqual({ tone: 'failing', label: 'Failing' });
		expect(scoreBandFor(0)).toEqual({ tone: 'failing', label: 'Failing' });
	});
});
