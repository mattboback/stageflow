import { act, renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { loadAllScansFixture } from '../../test/load-fixture';

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
const fixture = loadAllScansFixture();
const REPORT = { ...fixture, meta: { ...fixture.meta, jobId: JOB } };

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

	it.each([
		['HTTP failure', {}, 500, 'Report fetch failed: 500'],
		['invalid schema', { version: '2.2', meta: { jobId: JOB } }, 200, 'expected aggregated schema'],
		['another job', { ...REPORT, meta: { ...REPORT.meta, jobId: 'other-job' } }, 200, 'job ID']
	])('exposes report acquisition errors for %s', async (_label, body, status, message) => {
		mockApi([job('DONE')], [{ body, status }]);
		const { result } = renderHook(() => useScanReport(JOB));
		await waitFor(() => expect(result.current.reportError).toContain(message));
		expect(result.current.status).toBe('complete');
		expect(result.current.report).toBeNull();
		expect(result.current.error).toBe(result.current.reportError);
	});

	it('exposes malformed JSON as a report error', async () => {
		vi.spyOn(globalThis, 'fetch').mockImplementation((input) =>
			(typeof input === 'string' ? input : input instanceof URL ? input.href : input.url).endsWith(
				'/results'
			)
				? Promise.resolve(new Response('{broken-json'))
				: json(job('DONE'))
		);
		const { result } = renderHook(() => useScanReport(JOB));
		await waitFor(() => expect(result.current.reportError).not.toBeNull());
		expect(result.current.status).toBe('complete');
		expect(result.current.report).toBeNull();
	});

	it('cancels pending report retries on unmount', async () => {
		vi.useFakeTimers({ shouldAdvanceTime: true });
		const calls = mockApi([job('DONE')], [{ body: {}, status: 404 }]);
		const { result, unmount } = renderHook(() => useScanReport(JOB));
		await waitFor(() => expect(result.current.status).toBe('complete'));
		const count = calls.length;
		unmount();
		await act(async () => {
			await vi.advanceTimersByTimeAsync(6000);
		});
		expect(calls).toHaveLength(count);
	});

	it('retries report acquisition without resubscribing or submitting another job', async () => {
		const calls = mockApi(
			[job('DONE')],
			[
				{ body: {}, status: 500 },
				{ body: REPORT, status: 200 }
			]
		);
		const { result } = renderHook(() => useScanReport(JOB));
		await waitFor(() => expect(result.current.reportError).not.toBeNull());
		act(() => result.current.retryReport());
		expect(result.current.reportError).toBeNull();
		await waitFor(() => expect(result.current.report).toEqual(REPORT));
		expect(calls.filter((url) => url.endsWith(`/jobs/${JOB}`))).toHaveLength(1);
		expect(FakeEventSource.instances).toHaveLength(1);
	});

	it('exposes exhaustion and restarts the not-ready retry window', async () => {
		vi.useFakeTimers({ shouldAdvanceTime: true });
		const calls = mockApi([job('DONE')], [{ body: {}, status: 404 }]);
		const { result } = renderHook(() => useScanReport(JOB));
		await waitFor(() => expect(result.current.status).toBe('complete'));
		await act(async () => {
			await vi.advanceTimersByTimeAsync(60000);
		});
		expect(result.current.reportError).toContain('taking longer than expected');
		const count = calls.length;
		act(() => result.current.retryReport());
		await act(async () => {
			await vi.advanceTimersByTimeAsync(2000);
		});
		expect(result.current.reportError).toBeNull();
		expect(calls.length).toBeGreaterThan(count);
	});

	it('ignores a late report after navigation to another job', async () => {
		let resolveReport!: (response: Response) => void;
		vi.spyOn(globalThis, 'fetch').mockImplementation((input) => {
			const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url;
			if (url.endsWith('/results'))
				return new Promise((resolve) => {
					resolveReport = resolve;
				});
			return json({ id: url.endsWith('/job-2') ? 'job-2' : JOB, state: 'DONE', scanners: {} });
		});
		const { result, rerender } = renderHook(({ id }) => useScanReport(id), {
			initialProps: { id: JOB }
		});
		await waitFor(() => expect(result.current.status).toBe('complete'));
		const firstResolve = resolveReport;
		rerender({ id: 'job-2' });
		await act(async () => {
			firstResolve(new Response(JSON.stringify(REPORT)));
			await Promise.resolve();
		});
		expect(result.current.report).toBeNull();
		expect(result.current.reportError).toBeNull();
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

	it('says it is reconnecting while the stream is down, then live again', async () => {
		mockApi([job('SCANNING')]);
		const { result } = renderHook(() => useScanStatus(JOB));

		await waitFor(() => expect(result.current.status).toBe('scanning'));
		const stream = FakeEventSource.instances[0];
		act(() => stream?.onopen?.());
		await waitFor(() => expect(result.current.transport).toBe('streaming'));
		act(() => stream?.onerror?.({}));
		await waitFor(() => expect(result.current.transport).toBe('reconnecting'));
		act(() => stream?.onopen?.());
		await waitFor(() => expect(result.current.transport).toBe('streaming'));
	});

	it('counts elapsed time from when the job started, not from page load', async () => {
		vi.useFakeTimers({ shouldAdvanceTime: true });
		vi.setSystemTime(new Date('2026-10-02T00:02:00Z'));
		mockApi([job('SCANNING', { created_at: '2026-10-02T00:00:00Z' })]);
		const { result } = renderHook(() => useScanStatus(JOB));

		await waitFor(() => expect(result.current.status).toBe('scanning'));
		await act(async () => {
			await vi.advanceTimersByTimeAsync(1000);
		});
		expect(result.current.elapsed).toBeGreaterThanOrEqual(120);
		expect(result.current.elapsed).toBeLessThan(130);
	});
});
