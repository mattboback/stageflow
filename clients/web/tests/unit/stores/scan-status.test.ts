import { createSSEStream } from '$lib/api/sse';
import { scanHistoryStore } from '$lib/stores/scan-history.svelte';
import { createScanStatusStore } from '$lib/stores/scan-status.svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('$lib/stores/scan-history.svelte', () => ({
	scanHistoryStore: {
		updateStatus: vi.fn()
	}
}));

vi.mock('$lib/api/sse', () => ({
	createSSEStream: vi.fn(() => ({ close: vi.fn() }))
}));

vi.mock('$lib/api/utils', () => ({
	buildApiUrl: (path: string) => path
}));

type FetchFn = (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>;

const jsonResponse = (payload: unknown, status = 200): Response =>
	new Response(JSON.stringify(payload), {
		status,
		headers: { 'content-type': 'application/json' }
	});

async function flushPromises(times = 4): Promise<void> {
	for (let i = 0; i < times; i += 1) {
		await Promise.resolve();
	}
}

describe('Scan Status Store', () => {
	let fetchMock: ReturnType<typeof vi.fn<FetchFn>>;

	beforeEach(() => {
		vi.useFakeTimers();
		vi.clearAllMocks();
		fetchMock = vi.fn<FetchFn>();
		vi.stubGlobal('fetch', fetchMock);
	});

	afterEach(() => {
		vi.useRealTimers();
		vi.unstubAllGlobals();
		vi.restoreAllMocks();
	});

	it('initializes with default values', () => {
		const store = createScanStatusStore('job-123');
		expect(store.status).toBe('loading');
		expect(store.result).toBeNull();
		expect(store.elapsed).toBe(0);
		expect(store.logs).toEqual([]);
	});

	it('starts elapsed timer on start when live updates are supported', () => {
		class AvailableEventSource {}
		vi.stubGlobal('EventSource', AvailableEventSource);

		const createSSEStreamMock = vi.mocked(createSSEStream);
		const close = vi.fn();
		createSSEStreamMock.mockReturnValue({ close });

		const store = createScanStatusStore('job-123');
		store.start();

		expect(store.elapsed).toBe(0);
		vi.advanceTimersByTime(2500);
		expect(store.elapsed).toBe(2);
		expect(createSSEStreamMock).toHaveBeenCalledTimes(1);

		store.cleanup();
		expect(close).toHaveBeenCalledTimes(1);
	});

	it('fetches initial status when EventSource is undefined', async () => {
		vi.stubGlobal('EventSource', undefined);

		const store = createScanStatusStore('job-123');
		const statusResponse = {
			state: 'scanning',
			progress: { current_page: 0, total_pages: 10 }
		};
		fetchMock.mockResolvedValueOnce(jsonResponse(statusResponse));

		store.start();
		await vi.advanceTimersByTimeAsync(1);
		await flushPromises();

		expect(fetchMock).toHaveBeenCalledWith('/api/v1/jobs/job-123');
		expect(store.status).toBe('scanning');
		expect(store.result).toEqual(statusResponse);

		store.cleanup();
	});

	it('handles fetch errors during fallback', async () => {
		vi.stubGlobal('EventSource', undefined);

		const store = createScanStatusStore('job-123');
		fetchMock.mockRejectedValueOnce(new Error('Network failure'));

		store.start();
		await flushPromises();

		expect(store.status).toBe('error');
		expect(store.logs.some((line) => line.includes('Network failure'))).toBe(true);

		store.cleanup();
	});

	it('updates history store when a job completes', async () => {
		vi.stubGlobal('EventSource', undefined);

		const store = createScanStatusStore('job-123');
		fetchMock.mockResolvedValueOnce(jsonResponse({ state: 'done' }));

		store.start();
		await vi.advanceTimersByTimeAsync(1);
		await flushPromises();

		expect(store.status).toBe('complete');
		expect(vi.mocked(scanHistoryStore.updateStatus)).toHaveBeenCalledWith('job-123', 'complete');

		store.cleanup();
	});

	it('tracks completed and remaining scanners from SSE status and updates', async () => {
		class AvailableEventSource {}
		vi.stubGlobal('EventSource', AvailableEventSource);

		const createSSEStreamMock = vi.mocked(createSSEStream);
		const close = vi.fn();
		createSSEStreamMock.mockReturnValue({ close });

		const store = createScanStatusStore('job-123');
		store.start();

		const handlers = createSSEStreamMock.mock.calls[0]?.[1];
		if (!handlers) {
			throw new Error('expected createSSEStream handlers');
		}

		handlers.onStatus?.({
			id: 'job-123',
			state: 'SCANNING',
			progress: { current_page: 0, total_pages: 2, percentage: 0 },
			expected_scanners: ['axe', 'lighthouse'],
			completed_scanners: [],
			remaining_scanners: ['axe', 'lighthouse'],
			created_at: new Date().toISOString(),
			updated_at: new Date().toISOString()
		});

		handlers.onUpdate?.({
			type: 'scanner_complete',
			state: 'SCANNING',
			scanner_type: 'axe',
			pages_scanned: 1,
			violations: 3,
			timing: {
				total_ms: 2100,
				page_iteration_ms: 1800,
				write_results_ms: 100,
				upload_artifacts_ms: 100,
				publish_completed_ms: 100,
				finalization_ms: 200
			}
		});

		await flushPromises();

		expect(store.result?.completed_scanners).toEqual(['axe']);
		expect(store.result?.remaining_scanners).toEqual(['lighthouse']);
		expect(store.logs.some((line) => line.includes('[axe] Complete in 2.1s'))).toBe(true);

		store.cleanup();
	});
});
