import { formatDuration, formatTimestamp } from '$lib/utils/date';
import { afterEach, describe, expect, it, vi } from 'vitest';

afterEach(() => {
	vi.restoreAllMocks();
});

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

	it('falls back to ISO string when locale formatting throws', () => {
		vi.spyOn(Date.prototype, 'toLocaleString').mockImplementation(() => {
			throw new Error('format failed');
		});
		expect(formatTimestamp('2025-01-15T10:30:00Z')).toBe('2025-01-15T10:30:00.000Z');
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
