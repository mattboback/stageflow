import type {
	ReportMonitorSnapshot,
	ScanJobEventsPort,
	ScanJobReportPort,
	ScanJobStatusPort,
	SchedulerPort,
	StatusMonitorSnapshot,
	StreamTransportEvent
} from '$lib/stores/scan-monitor';
import type { ScanResult } from '$lib/types/scan';
import type { UnifiedReport } from '$lib/types/unified-report';

import { createScanJobMonitor } from '$lib/stores/scan-monitor';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

function createScanResult(overrides: Partial<ScanResult> = {}): ScanResult {
	return {
		id: 'job-123',
		state: 'SCANNING',
		created_at: new Date().toISOString(),
		updated_at: new Date().toISOString(),
		...overrides
	};
}

function createReport(): UnifiedReport {
	return {
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
	};
}

function createScheduler(): SchedulerPort & {
	advance(ms: number): void;
	pendingIntervals(): number;
	pendingTimeouts(): number;
} {
	let now = 0;
	let nextId = 1;
	const intervals = new Map<number, { ms: number; fn: () => void; nextRun: number }>();
	const timeouts = new Map<number, { at: number; fn: () => void }>();

	const advance = (ms: number) => {
		now += ms;
		let ran = true;
		while (ran) {
			ran = false;
			for (const [id, timeout] of [...timeouts.entries()]) {
				if (timeout.at <= now) {
					timeouts.delete(id);
					timeout.fn();
					ran = true;
				}
			}
			for (const interval of intervals.values()) {
				while (interval.nextRun <= now) {
					interval.nextRun += interval.ms;
					interval.fn();
					ran = true;
				}
			}
		}
	};

	return {
		every(ms, fn) {
			const id = nextId++;
			intervals.set(id, { ms, fn, nextRun: now + ms });
			return () => {
				intervals.delete(id);
			};
		},
		after(ms, fn) {
			const id = nextId++;
			timeouts.set(id, { at: now + ms, fn });
			return () => {
				timeouts.delete(id);
			};
		},
		advance,
		pendingIntervals() {
			return intervals.size;
		},
		pendingTimeouts() {
			return timeouts.size;
		}
	};
}

describe('scan-monitor', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	afterEach(() => {
		vi.restoreAllMocks();
	});

	it('tracks status snapshots and scanner completion updates at the boundary', async () => {
		const scheduler = createScheduler();
		const fetchStatus = vi.fn<ScanJobStatusPort['fetch']>().mockResolvedValue(createScanResult());
		let streamSink: Parameters<ScanJobEventsPort['open']>[1] | undefined;
		const eventsPort: ScanJobEventsPort = {
			supportsStreaming: () => true,
			open: (_jobId, sink) => {
				streamSink = sink;
				return { close: vi.fn() };
			}
		};

		const monitor = createScanJobMonitor(
			{ kind: 'status', jobId: 'job-123' },
			{ statusPort: { fetch: fetchStatus }, eventsPort, scheduler }
		);

		monitor.start();
		expect(streamSink).toBeDefined();

		streamSink?.onStatus(
			createScanResult({
				state: 'SCANNING',
				progress: { current_page: 0, total_pages: 2, percentage: 0 },
				expected_scanners: ['axe', 'lighthouse'],
				completed_scanners: []
			})
		);
		streamSink?.onUpdate({
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

		const snapshot = monitor.getSnapshot() as StatusMonitorSnapshot;
		expect(snapshot.status).toBe('scanning');
		expect(snapshot.job?.completed_scanners).toEqual(['axe']);
		expect(snapshot.job?.remaining_scanners).toEqual(['lighthouse']);
		expect(snapshot.logs.some((line) => line.includes('[axe] Complete in 2.1s'))).toBe(true);
	});

	it('falls back to a status refresh when status streaming is exhausted', async () => {
		const scheduler = createScheduler();
		const fetchStatus = vi
			.fn<ScanJobStatusPort['fetch']>()
			.mockResolvedValue(createScanResult({ state: 'DONE' }));
		let streamSink: Parameters<ScanJobEventsPort['open']>[1] | undefined;
		const close = vi.fn();
		const eventsPort: ScanJobEventsPort = {
			supportsStreaming: () => true,
			open: (_jobId, sink) => {
				streamSink = sink;
				return { close };
			}
		};
		const historyPort = { markTerminal: vi.fn() };

		const monitor = createScanJobMonitor(
			{ kind: 'status', jobId: 'job-123' },
			{ statusPort: { fetch: fetchStatus }, eventsPort, scheduler, historyPort }
		);

		monitor.start();
		streamSink?.onTransport({ type: 'exhausted', reason: 'closed' });
		await Promise.resolve();

		expect(close).toHaveBeenCalledTimes(1);
		expect(fetchStatus).toHaveBeenCalledTimes(1);
		expect(historyPort.markTerminal).toHaveBeenCalledWith('job-123', 'complete');
		expect((monitor.getSnapshot() as StatusMonitorSnapshot).status).toBe('complete');
	});

	it('polls report status when streaming is unavailable and stops on cleanup', async () => {
		const scheduler = createScheduler();
		const fetchStatus = vi
			.fn<ScanJobStatusPort['fetch']>()
			.mockResolvedValue(createScanResult({ state: 'RUNNING' }));
		const eventsPort: ScanJobEventsPort = {
			supportsStreaming: () => false,
			open: () => {
				throw new Error('should not open stream');
			}
		};
		const reportPort: ScanJobReportPort = {
			fetch: vi.fn().mockResolvedValue({ state: 'pending' })
		};

		const monitor = createScanJobMonitor(
			{ kind: 'report', jobId: 'job-123' },
			{ statusPort: { fetch: fetchStatus }, eventsPort, reportPort, scheduler }
		);

		monitor.start();
		await Promise.resolve();
		expect(fetchStatus).toHaveBeenCalledTimes(1);

		scheduler.advance(5000);
		await Promise.resolve();
		expect(fetchStatus).toHaveBeenCalledTimes(2);

		monitor.stop();
		scheduler.advance(5000);
		await Promise.resolve();
		expect(fetchStatus).toHaveBeenCalledTimes(2);
	});

	it('retries report loading and refreshes artifacts when the report becomes ready', async () => {
		const scheduler = createScheduler();
		const fetchStatus = vi
			.fn<ScanJobStatusPort['fetch']>()
			.mockResolvedValueOnce(
				createScanResult({
					state: 'DONE',
					artifacts: {
						report_json: '/scanner-artifacts/job-123/report.json',
						report_html: '/scanner-artifacts/job-123/report.html'
					}
				})
			)
			.mockResolvedValueOnce(
				createScanResult({
					state: 'DONE',
					artifacts: {
						report_json: '/scanner-artifacts/job-123/report.json',
						report_html: '/scanner-artifacts/job-123/report.html',
						screenshots: [
							{
								kind: 'page_overview',
								artifact_id: 'page-overview:axe:url-1',
								scanner_id: 'axe',
								page_id: 'url-1',
								url: '/scanner-artifacts/job-123/axe/url-1/screenshots/page-overview.webp'
							}
						]
					}
				})
			);
		const reportPort: ScanJobReportPort = {
			fetch: vi
				.fn<ScanJobReportPort['fetch']>()
				.mockResolvedValueOnce({ state: 'pending' })
				.mockResolvedValueOnce({ state: 'ready', report: createReport() })
		};
		let streamSink: Parameters<ScanJobEventsPort['open']>[1] | undefined;
		const eventsPort: ScanJobEventsPort = {
			supportsStreaming: () => true,
			open: (_jobId, sink) => {
				streamSink = sink;
				return { close: vi.fn() };
			}
		};

		const monitor = createScanJobMonitor(
			{ kind: 'report', jobId: 'job-123' },
			{ statusPort: { fetch: fetchStatus }, eventsPort, reportPort, scheduler }
		);

		monitor.start();
		await Promise.resolve();
		await Promise.resolve();
		expect(fetchStatus).toHaveBeenCalledTimes(1);
		expect(reportPort.fetch).toHaveBeenCalledTimes(1);

		scheduler.advance(800);
		await Promise.resolve();
		await Promise.resolve();

		expect(reportPort.fetch).toHaveBeenCalledTimes(2);
		expect(fetchStatus).toHaveBeenCalledTimes(2);

		const snapshot = monitor.getSnapshot() as ReportMonitorSnapshot;
		expect(snapshot.status).toBe('complete');
		expect(snapshot.report?.meta.jobId).toBe('job-123');
		expect(snapshot.screenshots).toHaveLength(1);
		expect(snapshot.logs).toContain('Scan complete. Generating aggregated report...');
		expect(streamSink).toBeDefined();
	});

	it('stops timers and closes streams on stop', async () => {
		const scheduler = createScheduler();
		const fetchStatus = vi.fn<ScanJobStatusPort['fetch']>().mockResolvedValue(createScanResult());
		const close = vi.fn();
		const eventsPort: ScanJobEventsPort = {
			supportsStreaming: () => true,
			open: () => ({ close })
		};

		const monitor = createScanJobMonitor(
			{ kind: 'report', jobId: 'job-123' },
			{
				statusPort: { fetch: fetchStatus },
				eventsPort,
				reportPort: { fetch: vi.fn().mockResolvedValue({ state: 'pending' }) },
				scheduler
			}
		);

		monitor.start();
		monitor.stop();

		expect(close).toHaveBeenCalledTimes(1);
		expect(scheduler.pendingIntervals()).toBe(0);
		expect(scheduler.pendingTimeouts()).toBe(0);
	});
});
