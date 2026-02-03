import type { ScanResult, ScanStatus, ScreenshotArtifact } from '$lib/types/scan';
import type { UnifiedReport } from '$lib/types/unified-report';

import { buildApiUrl } from '$lib/api/utils';
import { SvelteSet } from 'svelte/reactivity';

import type { SSEUpdate } from './scan-status/types';

import {
	MAX_LOG_LINES,
	MAX_SSE_PARSE_ERRORS,
	MAX_SSE_RECONNECT_ATTEMPTS
} from './scan-status/constants';
import { getLogMessage, normalizeStatus } from './scan-status/log-messages';

export function createScanReportStore(id: string) {
	let status = $state<ScanStatus>('loading');
	let job = $state<ScanResult | null>(null);
	let report = $state<UnifiedReport | null>(null);
	let screenshots = $state<ScreenshotArtifact[]>([]);
	let logs = $state<string[]>([]);
	let error = $state<string | null>(null);

	let eventSource: EventSource | null = null;
	const logSet = new SvelteSet<string>();
	let reportedSSEError = false;
	let fetchReportInFlight = false;
	let reportRetryTimeout: ReturnType<typeof setTimeout> | null = null;
	let reportRetryAttempts = 0;
	let reportRetryDelayMs = 800;
	let pollingInterval: ReturnType<typeof setInterval> | null = null;
	const POLL_INTERVAL_MS = 5000;
	const MAX_REPORT_RETRY_ATTEMPTS = 30;
	const MAX_REPORT_RETRY_DELAY_MS = 10_000;

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

	const cleanup = () => {
		clearReportRetry();
		stopPolling();
		if (eventSource) {
			eventSource.close();
			eventSource = null;
		}
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

		job = data;
		status = normalizeStatus(data.state);
		screenshots = data.artifacts?.screenshots ?? [];
		if (status === 'complete') {
			stopPolling();
			void fetchReport();
		} else if (status === 'failed') {
			stopPolling();
		} else {
			clearReportRetry();
			if (!eventSource) {
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
				if (res.status === 400 || res.status === 404) {
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
		reportedSSEError = false;
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
			job = {
				...job,
				state: update.state,
				progress: newProgress,
				error: update.error ?? job.error,
				error_details: update.error_details ?? job.error_details,
				last_stage: update.stage ?? job.last_stage
			};
		}

		status = normalizeStatus(update.state);

		if (update.type === 'complete' || update.type === 'failed') {
			void fetchStatus();
		}
	};

	const startSSE = () => {
		const url = buildApiUrl(`/api/v1/jobs/${id}/stream`);
		eventSource = new EventSource(url);
		let sseParseErrors = 0;
		let sseReconnectAttempts = 0;

		const handleSSEParseError = (eventType: string, err: unknown, rawData: string) => {
			sseParseErrors++;
			console.error(`[scan-report] Failed to parse SSE ${eventType} event:`, {
				jobId: id,
				parseErrors: sseParseErrors,
				error: err,
				rawData: rawData.slice(0, 200)
			});

			if (sseParseErrors >= MAX_SSE_PARSE_ERRORS) {
				console.warn('[scan-report] Too many SSE parse errors, stopping stream');
				status = 'error';
				addLog('ERROR: Live updates failed (invalid stream data). Refresh to retry.');
				cleanup();
				startPolling();
				void fetchStatus();
			}
		};

		eventSource.addEventListener('status', (event) => {
			try {
				const data = JSON.parse(String(event.data)) as ScanResult;
				sseParseErrors = 0;
				reportedSSEError = false;
				handleStatusData(data);
			} catch (err) {
				handleSSEParseError('status', err, String(event.data));
			}
		});

		eventSource.addEventListener('update', (event) => {
			try {
				const data = JSON.parse(String(event.data)) as SSEUpdate;
				sseParseErrors = 0;
				handleSSEUpdate(data);
			} catch (err) {
				handleSSEParseError('update', err, String(event.data));
			}
		});

		eventSource.addEventListener('done', () => {
			eventSource?.close();
			eventSource = null;
		});

		eventSource.onerror = (event) => {
			// EventSource reconnects automatically on transient errors.
			// But if the connection is fully closed (server gone, job done, etc.)
			// we need to stop retrying and fall back to polling.
			if (eventSource?.readyState === EventSource.CLOSED) {
				sseReconnectAttempts++;
				if (sseReconnectAttempts >= MAX_SSE_RECONNECT_ATTEMPTS) {
					console.warn('[scan-report] SSE connection closed after max reconnect attempts', {
						jobId: id,
						attempts: sseReconnectAttempts
					});
					addLog('WARN: Connection lost. Switching to polling...');
					cleanup();
					startPolling();
					void fetchStatus();
					return;
				}
			}

			// Log once per disconnect so we don't spam.
			if (!reportedSSEError) {
				reportedSSEError = true;
				console.warn('[scan-report] SSE connection error, waiting for reconnect', {
					jobId: id,
					readyState: eventSource?.readyState,
					reconnectAttempts: sseReconnectAttempts,
					event
				});
				addLog('WARN: Lost connection to live updates. Reconnecting...');
				if (eventSource?.readyState === EventSource.CLOSED) {
					startPolling();
				}
			}
		};
	};

	const refreshArtifacts = async () => {
		await fetchStatus();
	};

	const start = () => {
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
