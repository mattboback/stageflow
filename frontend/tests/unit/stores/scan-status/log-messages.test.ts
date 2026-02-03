/**
 * Log Messages Utility Tests
 */

import type { StatusLike } from '$lib/stores/scan-status/log-messages';

import {
	formatErrorDetails,
	getLogMessage,
	normalizeStatus
} from '$lib/stores/scan-status/log-messages';
import { describe, expect, it } from 'vitest';

describe('log-messages', () => {
	describe('formatErrorDetails', () => {
		it('returns null for undefined input', () => {
			expect(formatErrorDetails(undefined)).toBeNull();
		});

		it('returns null for empty string', () => {
			expect(formatErrorDetails('')).toBeNull();
		});

		it('returns null for whitespace-only string', () => {
			expect(formatErrorDetails('   \t\n  ')).toBeNull();
		});

		it('compacts whitespace in details', () => {
			expect(formatErrorDetails('Error   with\n\tmultiple   spaces')).toBe(
				'Error with multiple spaces'
			);
		});

		it('trims leading and trailing whitespace', () => {
			expect(formatErrorDetails('  Error message  ')).toBe('Error message');
		});

		it('truncates long messages', () => {
			const longMessage = 'A'.repeat(200);
			const result = formatErrorDetails(longMessage);
			// Truncates to 157 chars + '…' = 158 total (slice(0, 157) + ellipsis)
			expect(result).toHaveLength(158);
			expect(result?.endsWith('…')).toBe(true);
		});

		it('does not truncate messages at exactly 160 characters', () => {
			const exactMessage = 'A'.repeat(160);
			expect(formatErrorDetails(exactMessage)).toBe(exactMessage);
		});

		it('does not truncate short messages', () => {
			const shortMessage = 'Short error';
			expect(formatErrorDetails(shortMessage)).toBe(shortMessage);
		});
	});

	describe('normalizeStatus', () => {
		it('returns pending for undefined', () => {
			expect(normalizeStatus(undefined)).toBe('pending');
		});

		it('returns pending for empty string', () => {
			expect(normalizeStatus('')).toBe('pending');
		});

		it('normalizes done to complete', () => {
			expect(normalizeStatus('done')).toBe('complete');
			expect(normalizeStatus('DONE')).toBe('complete');
		});

		it('returns failed for failed status', () => {
			expect(normalizeStatus('failed')).toBe('failed');
			expect(normalizeStatus('FAILED')).toBe('failed');
		});

		it('returns pending for pending status', () => {
			expect(normalizeStatus('pending')).toBe('pending');
			expect(normalizeStatus('PENDING')).toBe('pending');
		});

		it('returns pending for ready_to_scan', () => {
			expect(normalizeStatus('ready_to_scan')).toBe('pending');
			expect(normalizeStatus('READY_TO_SCAN')).toBe('pending');
		});

		it('returns scanning for scanning status', () => {
			expect(normalizeStatus('scanning')).toBe('scanning');
		});

		it('returns extracting for extracting status', () => {
			expect(normalizeStatus('extracting')).toBe('extracting');
		});

		it('returns completing for completing status', () => {
			expect(normalizeStatus('completing')).toBe('completing');
		});

		it('returns processing for unknown statuses', () => {
			expect(normalizeStatus('unknown')).toBe('processing');
			expect(normalizeStatus('initializing')).toBe('processing');
			expect(normalizeStatus('custom-status')).toBe('processing');
		});

		it('is case insensitive', () => {
			expect(normalizeStatus('Scanning')).toBe('scanning');
			expect(normalizeStatus('EXTRACTING')).toBe('extracting');
			expect(normalizeStatus('Completing')).toBe('completing');
		});
	});

	describe('getLogMessage', () => {
		const emptyData: StatusLike = {};

		it('returns queue message for PENDING', () => {
			const message = getLogMessage('PENDING', emptyData);
			expect(message).toContain('queued');
		});

		it('returns extraction message for EXTRACTING', () => {
			const message = getLogMessage('EXTRACTING', emptyData);
			expect(message).toContain('Extracting');
		});

		it('returns structure analysis message for READY_TO_SCAN', () => {
			const message = getLogMessage('READY_TO_SCAN', emptyData);
			expect(message).toContain('structure');
		});

		it('returns progress message for SCANNING with progress', () => {
			const data: StatusLike = {
				progress: { current_page: 5, total_pages: 10 }
			};
			const message = getLogMessage('SCANNING', data);
			expect(message).toContain('5/10');
		});

		it('handles alternative progress field names', () => {
			const data: StatusLike = {
				progress: { currentPage: 3, totalPages: 8 }
			};
			const message = getLogMessage('SCANNING', data);
			expect(message).toContain('3/8');
		});

		it('returns null for SCANNING without progress', () => {
			expect(getLogMessage('SCANNING', emptyData)).toBeNull();
		});

		it('returns report generation message for COMPLETING', () => {
			const message = getLogMessage('COMPLETING', emptyData);
			expect(message).toContain('reports');
		});

		it('returns cleanup message for DONE', () => {
			const message = getLogMessage('DONE', emptyData);
			expect(message).toContain('complete');
		});

		it('returns error message for FAILED', () => {
			const data: StatusLike = {
				error: 'Timeout error'
			};
			const message = getLogMessage('FAILED', data);
			expect(message).toContain('CRITICAL');
			expect(message).toContain('Timeout error');
		});

		it('includes error details when available', () => {
			const data: StatusLike = {
				error: 'Main error',
				error_details: 'Additional context'
			};
			const message = getLogMessage('FAILED', data);
			expect(message).toContain('Main error');
			expect(message).toContain('Additional context');
		});

		it('uses Unknown error when error is empty', () => {
			const data: StatusLike = {};
			const message = getLogMessage('FAILED', data);
			expect(message).toContain('Unknown error');
		});

		it('returns null for unknown state', () => {
			expect(getLogMessage('UNKNOWN_STATE', emptyData)).toBeNull();
		});

		it('returns null for empty state', () => {
			expect(getLogMessage('', emptyData)).toBeNull();
		});
	});
});
