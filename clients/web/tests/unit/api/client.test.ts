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

	it('submits ZIP scans with scanner config and screenshot flag in form data', async () => {
		const fetchMock = vi
			.spyOn(globalThis, 'fetch')
			.mockResolvedValue(mockJsonResponse(201, { job_id: 'job-zip' }));
		const file = new File(['zip-bytes'], 'site.zip', { type: 'application/zip' });

		await expect(
			submitScanJob({
				mode: 'zip',
				file,
				urls: [],
				scanners: [{ id: 'axe', enabled: true, config: { tags: ['wcag2a'] } }],
				highlightStyle: 'solid',
				screenshot: false
			})
		).resolves.toEqual({ job_id: 'job-zip' });

		const [, init] = fetchMock.mock.calls[0] ?? [];
		const formData = init?.body as FormData;
		expect(init?.method).toBe('POST');
		expect(formData.get('file')).toBe(file);
		expect(formData.get('modules')).toBe('axe');
		expect(formData.get('screenshot')).toBe('false');
		expect(formData.get('scanner_configs')).toBe(JSON.stringify({ axe: { tags: ['wcag2a'] } }));
	});

	it('rejects ZIP submission without a selected file', async () => {
		await expect(
			submitScanJob({
				mode: 'zip',
				file: null,
				urls: [],
				scanners: SCANNERS,
				highlightStyle: 'dashed'
			})
		).rejects.toThrow('Select a file');
	});

	it('rejects URL submission without URLs', async () => {
		await expect(
			submitScanJob({
				mode: 'url',
				file: null,
				urls: [],
				scanners: SCANNERS,
				highlightStyle: 'dashed'
			})
		).rejects.toThrow('Enter a URL');
	});

	it('uses top-level API error messages when present', async () => {
		vi.spyOn(globalThis, 'fetch').mockResolvedValue(
			mockJsonResponse(400, {
				message: 'Project limit reached'
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
		).rejects.toThrow('Project limit reached');
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

	it('uses the dedicated file size message for 413 responses', async () => {
		vi.spyOn(globalThis, 'fetch').mockResolvedValue(mockJsonResponse(413, {}));

		await expect(
			submitScanJob({
				mode: 'url',
				file: null,
				urls: ['https://example.com'],
				scanners: SCANNERS,
				highlightStyle: 'dashed'
			})
		).rejects.toThrow('File too large. Maximum size is 100MB.');
	});

	it('falls back for unrecognized non-success statuses', async () => {
		vi.spyOn(globalThis, 'fetch').mockResolvedValue(mockJsonResponse(409, {}));

		await expect(
			submitScanJob({
				mode: 'url',
				file: null,
				urls: ['https://example.com'],
				scanners: SCANNERS,
				highlightStyle: 'dashed'
			})
		).rejects.toThrow('Scan failed. Please try again.');
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

	it('rejects successful responses that do not include a job id', async () => {
		vi.spyOn(globalThis, 'fetch').mockResolvedValue(mockJsonResponse(200, { message: 'created' }));

		await expect(
			submitScanJob({
				mode: 'url',
				file: null,
				urls: ['https://example.com'],
				scanners: SCANNERS,
				highlightStyle: 'dashed'
			})
		).rejects.toThrow('No job ID returned. Please try again.');
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

	it('rejects when the scanner catalog request fails', async () => {
		vi.spyOn(globalThis, 'fetch').mockRejectedValue(new TypeError('Failed to fetch'));

		await expect(fetchScanners()).rejects.toThrow(
			'Scanner catalog failed to load. Failed to fetch. Refresh to retry.'
		);
	});

	it('rethrows caller aborts instead of wrapping them as catalog failures', async () => {
		const controller = new AbortController();
		controller.abort();
		vi.spyOn(globalThis, 'fetch').mockRejectedValue(new DOMException('Aborted', 'AbortError'));

		await expect(fetchScanners(controller.signal)).rejects.toMatchObject({ name: 'AbortError' });
	});

	it('uses a network error fallback for non-Error scanner catalog failures', async () => {
		vi.spyOn(globalThis, 'fetch').mockRejectedValue('connection reset');

		await expect(fetchScanners()).rejects.toThrow(
			'Scanner catalog failed to load. Network error. Refresh to retry.'
		);
	});

	it('rejects when the scanner catalog returns an error', async () => {
		vi.spyOn(globalThis, 'fetch').mockResolvedValue(mockJsonResponse(500, {}));

		await expect(fetchScanners()).rejects.toThrow(
			'Scanner catalog failed to load (500). Refresh to retry.'
		);
	});

	it('returns only enabled scanners from the scanner catalog', async () => {
		vi.spyOn(globalThis, 'fetch').mockResolvedValue(
			mockJsonResponse(200, {
				categories: ['accessibility'],
				scanners: [
					{ id: 'axe', enabled: true },
					{ id: 'stale-disabled', enabled: false }
				]
			})
		);

		const result = await fetchScanners();

		expect(result.categories).toEqual(['accessibility']);
		expect(result.scanners.map((scanner) => scanner.id)).toEqual(['axe']);
	});
});
