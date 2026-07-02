import { describe, expect, it } from 'vitest';

import { scoreBandFor } from './score-band';

describe('scoreBandFor', () => {
	it('maps scores to bands at the documented boundaries', () => {
		expect(scoreBandFor(100)?.tone).toBe('strong');
		expect(scoreBandFor(90)?.tone).toBe('strong');
		expect(scoreBandFor(89)?.tone).toBe('watch');
		expect(scoreBandFor(80)?.tone).toBe('watch');
		expect(scoreBandFor(79)?.tone).toBe('needs-work');
		expect(scoreBandFor(70)?.tone).toBe('needs-work');
		expect(scoreBandFor(69)?.tone).toBe('high-risk');
		expect(scoreBandFor(60)?.tone).toBe('high-risk');
		expect(scoreBandFor(59)?.tone).toBe('failing');
		expect(scoreBandFor(0)?.tone).toBe('failing');
	});

	it('returns null for absent or non-numeric scores', () => {
		expect(scoreBandFor(null)).toBeNull();
		expect(scoreBandFor(undefined)).toBeNull();
		expect(scoreBandFor(Number.NaN)).toBeNull();
	});
});
