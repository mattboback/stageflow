import type { StatusMonitorSnapshot } from '$lib/stores/scan-monitor';

import { createScanStatusStore } from '$lib/stores/scan-status.svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const { monitorMocks } = vi.hoisted(() => ({
	monitorMocks: {
		start: vi.fn(),
		stop: vi.fn(),
		refreshStatus: vi.fn(),
		refreshArtifacts: vi.fn(),
		getSnapshot: vi.fn(),
		subscribe: vi.fn()
	}
}));

vi.mock('$lib/stores/scan-monitor', () => ({
	createScanJobMonitor: vi.fn(() => ({
		start: monitorMocks.start,
		stop: monitorMocks.stop,
		refreshStatus: monitorMocks.refreshStatus,
		refreshArtifacts: monitorMocks.refreshArtifacts,
		getSnapshot: monitorMocks.getSnapshot,
		subscribe: monitorMocks.subscribe
	}))
}));

function createSnapshot(overrides: Partial<StatusMonitorSnapshot> = {}): StatusMonitorSnapshot {
	return {
		kind: 'status',
		jobId: 'job-123',
		status: 'loading',
		job: null,
		logs: [],
		elapsedSeconds: 0,
		transport: 'idle',
		error: null,
		...overrides
	};
}

describe('Scan Status Store', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		monitorMocks.subscribe.mockImplementation(
			(listener: (snapshot: StatusMonitorSnapshot) => void) => {
				listener(createSnapshot());
				return vi.fn();
			}
		);
	});

	afterEach(() => {
		vi.restoreAllMocks();
	});

	it('initializes from the monitor snapshot', () => {
		const store = createScanStatusStore('job-123');

		expect(store.status).toBe('loading');
		expect(store.result).toBeNull();
		expect(store.elapsed).toBe(0);
		expect(store.logs).toEqual([]);
	});

	it('mirrors monitor updates', () => {
		monitorMocks.subscribe.mockImplementation(
			(listener: (snapshot: StatusMonitorSnapshot) => void) => {
				listener(
					createSnapshot({
						status: 'scanning',
						elapsedSeconds: 12,
						logs: ['Live status stream connected.'],
						job: {
							id: 'job-123',
							state: 'SCANNING',
							created_at: new Date().toISOString(),
							updated_at: new Date().toISOString()
						}
					})
				);
				return vi.fn();
			}
		);

		const store = createScanStatusStore('job-123');

		expect(store.status).toBe('scanning');
		expect(store.elapsed).toBe(12);
		expect(store.logs).toEqual(['Live status stream connected.']);
		expect(store.result?.state).toBe('SCANNING');
	});

	it('delegates start and cleanup to the monitor', () => {
		const unsubscribe = vi.fn();
		monitorMocks.subscribe.mockImplementation(
			(listener: (snapshot: StatusMonitorSnapshot) => void) => {
				listener(createSnapshot());
				return unsubscribe;
			}
		);

		const store = createScanStatusStore('job-123');
		store.start();
		store.cleanup();

		expect(monitorMocks.start).toHaveBeenCalledTimes(1);
		expect(monitorMocks.stop).toHaveBeenCalledTimes(1);
		expect(unsubscribe).toHaveBeenCalledTimes(1);
	});
});
