import type { ScannerSelection } from '$lib/types/scan';

import { submitScanJob } from '$lib/api/client';
import { afterEach, describe, expect, it, vi } from 'vitest';

const SCANNERS: ScannerSelection[] = [{ id: 'axe', enabled: true }];

function mockJsonResponse(status: number, body: unknown): Response {
	return {
		ok: status >= 200 && status < 300,
		status,
		json: vi.fn().mockResolvedValue(body)
	} as unknown as Response;
}

describe('api/client submitScanJob', () => {
	afterEach(() => {
		vi.restoreAllMocks();
	});

	it('returns job id for successful URL submit', async () => {
		const fetchMock = vi
			.spyOn(globalThis, 'fetch')
			.mockResolvedValue(mockJsonResponse(201, { job_id: 'job-123' }));

		await expect(
			submitScanJob({
				mode: 'url',
				file: null,
				urls: ['https://example.com'],
				scanners: SCANNERS,
				highlightStyle: 'dashed'
			})
		).resolves.toEqual({ job_id: 'job-123' });

		expect(fetchMock).toHaveBeenCalledOnce();
	});

	it('surfaces structured validation message with suggestion/details', async () => {
		vi.spyOn(globalThis, 'fetch').mockResolvedValue(
			mockJsonResponse(400, {
				error: {
					code: 'VALIDATION_FAILED',
					message: 'URL exceeds maximum length',
					suggestion: 'Split this scan into smaller batches.',
					details: 'Maximum length: 2048 characters.'
				}
			})
		);

		await expect(
			submitScanJob({
				mode: 'url',
				file: null,
				urls: ['https://example.com'],
				scanners: SCANNERS,
				highlightStyle: 'solid'
			})
		).rejects.toThrow(
			'URL exceeds maximum length Split this scan into smaller batches. Maximum length: 2048 characters.'
		);
	});

	it('uses structured message for 422 errors', async () => {
		vi.spyOn(globalThis, 'fetch').mockResolvedValue(
			mockJsonResponse(422, {
				error: {
					code: 'UNSUPPORTED_MODULE',
					message: 'Unsupported scanner module: unknown-scanner'
				}
			})
		);

		await expect(
			submitScanJob({
				mode: 'url',
				file: null,
				urls: ['https://example.com'],
				scanners: SCANNERS,
				highlightStyle: 'dashed'
			})
		).rejects.toThrow('Unsupported scanner module: unknown-scanner');
	});

	it('falls back to generic server error when payload is not JSON', async () => {
		vi.spyOn(globalThis, 'fetch').mockResolvedValue({
			ok: false,
			status: 500,
			json: vi.fn().mockRejectedValue(new Error('invalid json'))
		} as unknown as Response);

		await expect(
			submitScanJob({
				mode: 'url',
				file: null,
				urls: ['https://example.com'],
				scanners: SCANNERS,
				highlightStyle: 'dashed'
			})
		).rejects.toThrow('Server error. Please try again in a moment.');
	});
});
