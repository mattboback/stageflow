import type { ScannerSelection } from '$lib/types/scan';

import { fetchScanners, fetchWithTimeout, submitScanJob } from '$lib/api/client';
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

	it('preserves caller abort when AbortSignal.any is unavailable', async () => {
		const originalAny = Reflect.get(AbortSignal, 'any');
		Object.defineProperty(AbortSignal, 'any', {
			value: undefined,
			configurable: true,
			writable: true
		});

		try {
			vi.spyOn(globalThis, 'fetch').mockImplementation((_input, init) => {
				return new Promise((_resolve, reject) => {
					const signal = init?.signal;
					if (!signal) {
						reject(new Error('Missing request signal'));
						return;
					}
					if (signal.aborted) {
						reject(new DOMException('Aborted', 'AbortError'));
						return;
					}
					signal.addEventListener(
						'abort',
						() => {
							reject(new DOMException('Aborted', 'AbortError'));
						},
						{ once: true }
					);
				});
			});

			const callerController = new AbortController();
			const request = fetchWithTimeout('/api/v1/jobs', { signal: callerController.signal }, 60_000);
			callerController.abort();

			await expect(request).rejects.toMatchObject({ name: 'AbortError' });
		} finally {
			Object.defineProperty(AbortSignal, 'any', {
				value: originalAny,
				configurable: true,
				writable: true
			});
		}
	});
});

describe('api/client fetchScanners', () => {
	afterEach(() => {
		vi.restoreAllMocks();
	});

	it('falls back to built-in scanners when the scanner catalog request fails', async () => {
		vi.spyOn(globalThis, 'fetch').mockRejectedValue(new TypeError('Failed to fetch'));

		const result = await fetchScanners();

		expect(result.categories).toEqual([]);
		expect(result.scanners.slice(0, 3).map((scanner) => scanner.id)).toEqual([
			'axe',
			'lighthouse',
			'link-checker'
		]);
		expect(result.scanners.every((scanner) => scanner.enabled)).toBe(true);
	});

	it('falls back to built-in scanners when the scanner catalog returns an error', async () => {
		vi.spyOn(globalThis, 'fetch').mockResolvedValue(mockJsonResponse(500, {}));

		const result = await fetchScanners();

		expect(result.scanners.map((scanner) => scanner.id)).toContain('security-headers');
		expect(result.scanners.map((scanner) => scanner.id)).toContain('spelling-grammar');
	});
});
