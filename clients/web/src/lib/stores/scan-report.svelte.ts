import type { ScanResult, ScanStatus, ScreenshotArtifact } from '$lib/types/scan';
import type { UnifiedReport } from '$lib/types/unified-report';

import { buildApiUrl } from '$lib/api/utils';
import { SvelteSet } from 'svelte/reactivity';

import type { SSEUpdate } from './scan-status/types';

import { MAX_LOG_LINES } from './scan-status/constants';
import { addLifecycleLog, createScanJobStream, fetchScanJobStatus } from './scan-status/job-stream';

export function createScanReportStore(id: string) {
	let status = $state<ScanStatus>('loading');
	let job = $state<ScanResult | null>(null);
	let report = $state<UnifiedReport | null>(null);
	let screenshots = $state<ScreenshotArtifact[]>([]);
	let logs = $state<string[]>([]);
	let error = $state<string | null>(null);

	let sseStream: { close: () => void } | null = null;
	const logSet = new SvelteSet<string>();
	let fetchReportInFlight = false;
	let started = false;
	let reportRetryTimeout: ReturnType<typeof setTimeout> | null = null;
	let reportRetryAttempts = 0;
	let reportRetryDelayMs = 800;
	let pollingInterval: ReturnType<typeof setInterval> | null = null;
	const POLL_INTERVAL_MS = 5000;
	const MAX_REPORT_RETRY_ATTEMPTS = 30;
	const MAX_REPORT_RETRY_DELAY_MS = 10_000;
	const isReportPendingStatus = (statusCode: number): boolean =>
		statusCode === 400 || statusCode === 404 || statusCode === 409;

	const addLog = (msg: string) => {
		if (logSet.has(msg)) return;
		logSet.add(msg);
		logs = [...logs, msg].slice(-MAX_LOG_LINES);
	};

	const clearReportRetry = () => {
		if (reportRetryTimeout) {
			clearTimeout(reportRetryTimeout);
			reportRetryTimeout = null;
		}
	};

	const scheduleReportRetry = () => {
		if (report) return;
		if (status !== 'complete') return;
		if (reportRetryTimeout) return;

		if (reportRetryAttempts >= MAX_REPORT_RETRY_ATTEMPTS) {
			const message = 'Aggregated report is taking longer than expected. Refresh to retry.';
			addLog(`WARN: ${message}`);
			error = message;
			return;
		}

		if (reportRetryAttempts === 0) {
			addLog('Scan complete. Generating aggregated report…');
		}

		const delayMs = reportRetryDelayMs;
		reportRetryAttempts += 1;
		reportRetryDelayMs = Math.min(MAX_REPORT_RETRY_DELAY_MS, Math.round(reportRetryDelayMs * 1.7));

		reportRetryTimeout = setTimeout(() => {
			reportRetryTimeout = null;
			void fetchReport();
		}, delayMs);
	};

	const stopPolling = () => {
		if (pollingInterval) {
			clearInterval(pollingInterval);
			pollingInterval = null;
		}
	};

	const closeSSEStream = () => {
		if (sseStream) {
			sseStream.close();
			sseStream = null;
		}
	};

	const controller = {
		getJob: () => job,
		setJob: (nextJob: ScanResult) => {
			job = nextJob;
		},
		setStatus: (nextStatus: ScanStatus) => {
			status = nextStatus;
		},
		addLog,
		fetchErrorPrefix: 'scan-report',
		fallbackFetchErrorMessage: 'Failed to fetch job status',
		initialFetchErrorStatus: 'error' as const,
		onStatusData: (data: ScanResult, nextStatus: ScanStatus, normalizedState: string) => {
			addLifecycleLog(
				addLog,
				normalizedState,
				{
					EXTRACTING: 'Verifying archive integrity...',
					SCANNING: '[axe-core] Injecting accessibility engine...',
					COMPLETING: 'Uploading artifacts to secure storage...'
				},
				normalizedState !== 'SCANNING' || Boolean(data.progress)
			);

			screenshots = data.artifacts?.screenshots ?? [];
			if (nextStatus === 'complete') {
				stopPolling();
				if (!report) {
					void fetchReport();
				}
			} else if (nextStatus === 'failed') {
				stopPolling();
			} else {
				clearReportRetry();
				if (!sseStream) {
					startPolling();
				}
			}
		},
		onStatusFetchSuccess: () => {
			error = null;
		},
		onStatusFetchError: (message: string) => {
			error = message;
		},
		onUpdate: (update: SSEUpdate) => {
			if (update.type === 'complete' || update.type === 'failed') {
				void fetchStatus();
			}
		}
	};

	const cleanup = () => {
		clearReportRetry();
		stopPolling();
		closeSSEStream();
	};

	const fetchStatus = async () => {
		await fetchScanJobStatus(id, controller);
		if (status === 'failed') {
			report = null;
		}
	};

	const startPolling = () => {
		if (pollingInterval) return;
		pollingInterval = setInterval(() => {
			if (status === 'complete' || status === 'failed') {
				stopPolling();
				return;
			}
			void fetchStatus();
		}, POLL_INTERVAL_MS);
	};

	const fetchReport = async () => {
		if (fetchReportInFlight) return;
		fetchReportInFlight = true;
		try {
			const res = await fetch(buildApiUrl(`/api/v1/jobs/${id}/results`), {
				redirect: 'follow'
			});
			if (!res.ok) {
				if (isReportPendingStatus(res.status)) {
					scheduleReportRetry();
					return;
				}
				throw new Error(`Report fetch failed: ${res.status}`);
			}
			const data = (await res.json()) as UnifiedReport;
			if (!data.meta.jobId || !data.version) {
				throw new Error('Report JSON did not match the expected aggregated schema.');
			}
			report = data;
			error = null;
			clearReportRetry();

			// Completion artifacts can lag slightly behind the aggregated report redirect.
			// Refresh the final job payload once the report is ready so page screenshots
			// and overlay metadata render without a manual refresh.
			if (status === 'complete' && screenshots.length === 0) {
				await fetchStatus();
			}
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Failed to load report';
			console.error('[scan-report] Failed to fetch aggregated report:', {
				jobId: id,
				error: err
			});
			error = message;
		} finally {
			fetchReportInFlight = false;
		}
	};

	const startSSE = () => {
		sseStream = createScanJobStream(controller, {
			jobId: id,
			sourceName: 'scan-report',
			onDone: () => {
				closeSSEStream();
			},
			onError: (err) => {
				if (err.parseError) {
					status = 'error';
					cleanup();
					startPolling();
					void fetchStatus();
				} else if (err.terminal) {
					cleanup();
					startPolling();
					void fetchStatus();
				} else {
					closeSSEStream();
					startPolling();
					void fetchStatus();
				}
			}
		});
	};

	const refreshArtifacts = async () => {
		await fetchStatus();
	};

	const start = () => {
		if (started) return;
		started = true;
		status = 'loading';
		report = null;
		logs = [];
		logSet.clear();
		error = null;
		clearReportRetry();
		reportRetryAttempts = 0;
		reportRetryDelayMs = 800;
		void fetchStatus();
		if (typeof EventSource !== 'undefined') {
			startSSE();
		} else {
			startPolling();
		}
	};

	return {
		get status() {
			return status;
		},
		get job() {
			return job;
		},
		get report() {
			return report;
		},
		get screenshots() {
			return screenshots;
		},
		get logs() {
			return logs;
		},
		get error() {
			return error;
		},
		start,
		cleanup,
		refreshArtifacts
	};
}
