import { act, renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { useScanReport, useScanStatus } from './useScanMonitor';

type Listener = (event: { data: string }) => void;

/* The two hooks only use the named-event API of EventSource, so a minimal fake
   lets a test push status and update frames the way the API streams them. */
class FakeEventSource {
	static CLOSED = 2;
	static instances: FakeEventSource[] = [];
	readyState = 1;
	onopen: (() => void) | null = null;
	onerror: ((event: unknown) => void) | null = null;
	private listeners = new Map<string, Listener[]>();

	url: string;

	constructor(url: string) {
		this.url = url;
		FakeEventSource.instances.push(this);
	}

	addEventListener(type: string, listener: Listener) {
		this.listeners.set(type, [...(this.listeners.get(type) ?? []), listener]);
	}

	close() {
		this.readyState = FakeEventSource.CLOSED;
	}

	emit(type: string, data: unknown) {
		for (const listener of this.listeners.get(type) ?? []) listener({ data: JSON.stringify(data) });
	}
}

const JOB = 'job-1';
const SHOT = { page_id: 'p1', scanner_id: 'axe', url: '/shot.webp' };
const REPORT = { version: '2.2', meta: { jobId: JOB }, issues: [], pages: [] };

function job(state: string, extra: Record<string, unknown> = {}) {
	return { id: JOB, state, scanners: {}, ...extra };
}

function json(body: unknown, status = 200) {
	return Promise.resolve(new Response(JSON.stringify(body), { status }));
}

/* Routes /jobs/{id} and /jobs/{id}/results to queued responses; the last queued
   response repeats. */
function mockApi(statuses: unknown[], results: { body: unknown; status: number }[] = []) {
	const calls: string[] = [];
	vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
		const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url;
		calls.push(url);
		if (url.endsWith(`/api/v1/jobs/${JOB}/results`)) {
			const next = results.length > 1 ? results.shift() : results[0];
			return next ? json(next.body, next.status) : json({}, 404);
		}
		if (url.endsWith(`/api/v1/jobs/${JOB}`)) {
			const next = statuses.length > 1 ? statuses.shift() : statuses[0];
			return json(next);
		}
		throw new Error(`Unexpected request ${url}`);
	});
	return calls;
}

beforeEach(() => {
	FakeEventSource.instances = [];
	vi.stubGlobal('EventSource', FakeEventSource);
});

afterEach(() => {
	vi.useRealTimers();
	vi.unstubAllGlobals();
	vi.restoreAllMocks();
});

describe('useScanReport', () => {
	it('loads the report and screenshots for a completed job', async () => {
		mockApi([job('DONE', { artifacts: { screenshots: [SHOT] } })], [{ body: REPORT, status: 200 }]);
		const { result } = renderHook(() => useScanReport(JOB));

		await waitFor(() => expect(result.current.report).not.toBeNull());
		expect(result.current.status).toBe('complete');
		expect(result.current.screenshots).toEqual([SHOT]);
		expect(result.current.error).toBeNull();
	});

	it('retries until the aggregated report exists', async () => {
		vi.useFakeTimers({ shouldAdvanceTime: true });
		const calls = mockApi(
			[job('DONE')],
			[
				{ body: {}, status: 404 },
				{ body: REPORT, status: 200 }
			]
		);
		const { result } = renderHook(() => useScanReport(JOB));

		await waitFor(() => expect(calls.filter((url) => url.endsWith('/results'))).toHaveLength(1));
		expect(result.current.report).toBeNull();
		await act(async () => {
			await vi.advanceTimersByTimeAsync(2000);
		});
		await waitFor(() => expect(result.current.report).not.toBeNull());
	});

	it('never requests a report for a failed job', async () => {
		const calls = mockApi([job('FAILED', { error: 'boom' })]);
		const { result } = renderHook(() => useScanReport(JOB));

		await waitFor(() => expect(result.current.status).toBe('failed'));
		expect(result.current.report).toBeNull();
		expect(calls.some((url) => url.endsWith('/results'))).toBe(false);
	});

	it('loads the report once the stream reports completion', async () => {
		mockApi([job('SCANNING'), job('DONE')], [{ body: REPORT, status: 200 }]);
		const { result } = renderHook(() => useScanReport(JOB));

		await waitFor(() => expect(result.current.status).toBe('scanning'));
		expect(result.current.report).toBeNull();
		act(() => FakeEventSource.instances[0]?.emit('update', { state: 'DONE' }));
		await waitFor(() => expect(result.current.report).not.toBeNull());
		expect(result.current.status).toBe('complete');
	});

	it('keeps one screenshots identity across renders when there are none', async () => {
		mockApi([job('DONE')], [{ body: REPORT, status: 200 }]);
		const { result, rerender } = renderHook(() => useScanReport(JOB));

		await waitFor(() => expect(result.current.report).not.toBeNull());
		const first = result.current.screenshots;
		rerender();
		// Callers use screenshots as an effect dependency; a fresh [] each render loops.
		expect(result.current.screenshots).toBe(first);
	});

	it('falls back to polling without EventSource', async () => {
		vi.stubGlobal('EventSource', undefined);
		mockApi([job('SCANNING')]);
		const { result } = renderHook(() => useScanReport(JOB));

		await waitFor(() => expect(result.current.transport).toBe('polling'));
	});
});

describe('useScanStatus', () => {
	it('follows the stream to completion', async () => {
		mockApi([job('SCANNING'), job('DONE')]);
		const { result } = renderHook(() => useScanStatus(JOB));

		await waitFor(() => expect(result.current.status).toBe('scanning'));
		act(() => FakeEventSource.instances[0]?.emit('update', { state: 'DONE' }));
		await waitFor(() => expect(result.current.status).toBe('complete'));
		expect(result.current.result?.id).toBe(JOB);
	});
});
