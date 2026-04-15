import type { ReportMonitorSnapshot } from '$lib/stores/scan-monitor';

import { createScanReportStore } from '$lib/stores/scan-report.svelte';
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

function createSnapshot(overrides: Partial<ReportMonitorSnapshot> = {}): ReportMonitorSnapshot {
	return {
		kind: 'report',
		jobId: 'job-123',
		status: 'loading',
		job: null,
		report: null,
		screenshots: [],
		logs: [],
		transport: 'idle',
		error: null,
		...overrides
	};
}

describe('Scan Report Store', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		monitorMocks.subscribe.mockImplementation(
			(listener: (snapshot: ReportMonitorSnapshot) => void) => {
				listener(createSnapshot());
				return vi.fn();
			}
		);
	});

	afterEach(() => {
		vi.restoreAllMocks();
	});

	it('initializes from the monitor snapshot', () => {
		const store = createScanReportStore('job-123');

		expect(store.status).toBe('loading');
		expect(store.job).toBeNull();
		expect(store.report).toBeNull();
		expect(store.screenshots).toEqual([]);
		expect(store.logs).toEqual([]);
		expect(store.error).toBeNull();
	});

	it('mirrors report monitor updates', () => {
		monitorMocks.subscribe.mockImplementation(
			(listener: (snapshot: ReportMonitorSnapshot) => void) => {
				listener(
					createSnapshot({
						status: 'complete',
						logs: ['Scan complete. Generating aggregated report...'],
						report: {
							version: '2.0.0',
							meta: { jobId: 'job-123' },
							issues: [],
							pages: [],
							scanners: [],
							summary: {
								totalIssues: 0,
								bySeverity: { critical: 0, serious: 0, moderate: 0, minor: 0, info: 0 },
								pagesScanned: 0,
								pagesWithIssues: 0
							}
						},
						screenshots: [
							{
								kind: 'page_overview',
								artifact_id: 'page-overview:axe:url-1',
								scanner_id: 'axe',
								page_id: 'url-1',
								url: '/scanner-artifacts/job-123/axe/url-1/screenshots/page-overview.webp'
							}
						]
					})
				);
				return vi.fn();
			}
		);

		const store = createScanReportStore('job-123');

		expect(store.status).toBe('complete');
		expect(store.report?.meta.jobId).toBe('job-123');
		expect(store.screenshots).toHaveLength(1);
		expect(store.logs).toEqual(['Scan complete. Generating aggregated report...']);
	});

	it('delegates lifecycle methods to the monitor', async () => {
		const unsubscribe = vi.fn();
		monitorMocks.subscribe.mockImplementation(
			(listener: (snapshot: ReportMonitorSnapshot) => void) => {
				listener(createSnapshot());
				return unsubscribe;
			}
		);
		monitorMocks.refreshArtifacts.mockResolvedValue(undefined);

		const store = createScanReportStore('job-123');
		store.start();
		await store.refreshArtifacts();
		store.cleanup();

		expect(monitorMocks.start).toHaveBeenCalledTimes(1);
		expect(monitorMocks.refreshArtifacts).toHaveBeenCalledTimes(1);
		expect(monitorMocks.stop).toHaveBeenCalledTimes(1);
		expect(unsubscribe).toHaveBeenCalledTimes(1);
	});
});
