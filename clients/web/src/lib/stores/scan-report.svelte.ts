import type { ScanResult, ScanStatus, ScreenshotArtifact } from '$lib/types/scan';
import type { UnifiedReport } from '$lib/types/unified-report';

import { createSSEStream } from '$lib/api/sse';
import { buildApiUrl } from '$lib/api/utils';
import { SvelteSet } from 'svelte/reactivity';

import type { SSEUpdate } from './scan-status/types';

import { MAX_LOG_LINES } from './scan-status/constants';
import { getLogMessage, normalizeStatus } from './scan-status/log-messages';
import {
	applyScannerCompletionUpdate,
	normalizeScannerProgress
} from './scan-status/scanner-progress';

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

	const cleanup = () => {
		clearReportRetry();
		stopPolling();
		closeSSEStream();
	};

	const handleStatusData = (data: ScanResult) => {
		const normalizedState = (data.state || '').toUpperCase();
		const logMsg = getLogMessage(normalizedState, data);
		if (logMsg) {
			addLog(logMsg);
		}

		if (normalizedState === 'EXTRACTING') {
			addLog('Verifying archive integrity...');
		} else if (normalizedState === 'SCANNING' && data.progress) {
			addLog('[axe-core] Injecting accessibility engine...');
		} else if (normalizedState === 'COMPLETING') {
			addLog('Uploading artifacts to secure storage...');
		}

		job = normalizeScannerProgress(data);
		status = normalizeStatus(data.state);
		screenshots = data.artifacts?.screenshots ?? [];
		if (status === 'complete') {
			stopPolling();
			void fetchReport();
		} else if (status === 'failed') {
			stopPolling();
		} else {
			clearReportRetry();
			if (!sseStream) {
				startPolling();
			}
		}
	};

	const fetchStatus = async () => {
		try {
			const res = await fetch(buildApiUrl(`/api/v1/jobs/${id}`));
			if (!res.ok) {
				throw new Error(res.status === 404 ? 'Job not found' : 'Failed to fetch job status');
			}
			const data = (await res.json()) as ScanResult;
			error = null;
			handleStatusData(data);
			if (status === 'failed') {
				report = null;
			}
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Failed to fetch job status';
			console.error('[scan-report] Failed to fetch job status:', { jobId: id, error: err });
			addLog(`ERROR: ${message}. Refresh to retry.`);
			error = message;
			if (status === 'loading') {
				status = 'error';
			}
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

	const handleSSEUpdate = (update: SSEUpdate) => {
		const normalizedState = (update.state || '').toUpperCase();
		const logMsg = getLogMessage(normalizedState, update);
		if (logMsg) {
			addLog(logMsg);
		}

		if (job) {
			const newProgress = update.progress
				? (() => {
						const currentPage = update.progress.currentPage;
						const totalPages = update.progress.totalPages;
						const rawPercentage = totalPages > 0 ? (currentPage / totalPages) * 100 : 0;
						return {
							current_page: currentPage,
							total_pages: totalPages,
							percentage: Math.max(0, Math.min(100, Math.round(rawPercentage)))
						};
					})()
				: job.progress;
			const nextJob = {
				...job,
				state: update.state,
				progress: newProgress,
				error: update.error ?? job.error,
				error_details: update.error_details ?? job.error_details,
				last_stage: update.stage ?? job.last_stage
			};
			job = applyScannerCompletionUpdate(nextJob, update);
		}

		status = normalizeStatus(update.state);

		if (update.type === 'complete' || update.type === 'failed') {
			void fetchStatus();
		}
	};

	const startSSE = () => {
		sseStream = createSSEStream<ScanResult, SSEUpdate>(
			id,
			{
				onStatus: (data) => {
					stopPolling();
					handleStatusData(data);
				},
				onUpdate: (data) => {
					stopPolling();
					handleSSEUpdate(data);
				},
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
						// Do not rely on EventSource reconnect internals; switch to polling immediately.
						closeSSEStream();
						startPolling();
						void fetchStatus();
					}
				}
			},
			{
				sourceName: 'scan-report',
				onLog: addLog
			}
		);
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
