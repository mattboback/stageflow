import { formatDuration, formatTimestamp } from '$lib/utils/date';
import { describe, expect, it } from 'vitest';

describe('formatTimestamp', () => {
	it('returns null for null input', () => {
		expect(formatTimestamp(null)).toBeNull();
	});

	it('returns null for undefined input', () => {
		expect(formatTimestamp(undefined)).toBeNull();
	});

	it('returns null for empty string', () => {
		expect(formatTimestamp('')).toBeNull();
	});

	it('returns original value for unparseable string', () => {
		expect(formatTimestamp('not-a-date')).toBe('not-a-date');
	});

	it('formats a valid ISO timestamp', () => {
		const result = formatTimestamp('2025-01-15T10:30:00Z');
		expect(result).toBeTruthy();
		expect(result).not.toBe('2025-01-15T10:30:00Z');
	});
});

describe('formatDuration', () => {
	it('returns null for null input', () => {
		expect(formatDuration(null)).toBeNull();
	});

	it('returns null for undefined input', () => {
		expect(formatDuration(undefined)).toBeNull();
	});

	it('returns null for zero', () => {
		expect(formatDuration(0)).toBeNull();
	});

	it('formats milliseconds to seconds', () => {
		expect(formatDuration(2500)).toBe('2.5s');
	});

	it('formats sub-second durations', () => {
		expect(formatDuration(100)).toBe('0.1s');
	});

	it('formats whole second durations', () => {
		expect(formatDuration(3000)).toBe('3.0s');
	});
});
