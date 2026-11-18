import { describe, expect, it } from 'vitest';

import { hostedEvidenceExpired, hostedExpiryAt } from './hosted-retention';

const DAY_MS = 24 * 60 * 60 * 1000;

describe('hosted retention', () => {
	it('adds one day to the completion timestamp', () => {
		const completed = '2026-08-13T12:00:00.000Z';
		const expiry = hostedExpiryAt(completed);
		expect(expiry?.toISOString()).toBe(new Date(Date.parse(completed) + DAY_MS).toISOString());
	});

	it('treats missing or invalid timestamps as not expired', () => {
		expect(hostedExpiryAt(undefined)).toBeNull();
		expect(hostedEvidenceExpired('not-a-date')).toBe(false);
	});

	it('reports expiry against a supplied now', () => {
		const completed = '2026-08-13T12:00:00.000Z';
		const completedMs = Date.parse(completed);
		expect(hostedEvidenceExpired(completed, completedMs)).toBe(false);
		expect(hostedEvidenceExpired(completed, completedMs + DAY_MS)).toBe(true);
	});
});
