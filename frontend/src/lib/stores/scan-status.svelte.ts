import type { ScanResult, ScanStatus } from '$lib/types/scan';

import { buildApiUrl } from '$lib/api/utils';
import { SvelteSet } from 'svelte/reactivity';

import type { SSEUpdate } from './scan-status/types';

import { scanHistoryStore } from './scan-history.svelte';
import {
	MAX_LOG_LINES,
	MAX_SSE_PARSE_ERRORS,
	MAX_SSE_RECONNECT_ATTEMPTS
} from './scan-status/constants';
import { getLogMessage, normalizeStatus } from './scan-status/log-messages';

export function createScanStatusStore(id: string) {
	let status = $state<ScanStatus>('loading');
	let result = $state<ScanResult | null>(null);
	let elapsed = $state(0);
	let logs = $state<string[]>([]);

	let elapsedInterval: ReturnType<typeof setInterval> | null = null;
	let eventSource: EventSource | null = null;
	const logSet = new SvelteSet<string>();
	let statusUpdated = false;
	let reportedSSEError = false;

	const addLog = (msg: string) => {
		if (logSet.has(msg)) {
			return;
		}
		logSet.add(msg);
		logs = [...logs, msg].slice(-MAX_LOG_LINES);
	};

	const cleanup = () => {
		if (elapsedInterval) {
			clearInterval(elapsedInterval);
			elapsedInterval = null;
		}
		if (eventSource) {
			eventSource.close();
			eventSource = null;
		}
	};

	const handleStatusData = (data: ScanResult): boolean => {
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

		result = data;
		const newStatus = normalizeStatus(data.state);
		status = newStatus;

		// Update history if done or failed (only once)
		if (['complete', 'failed'].includes(newStatus) && !statusUpdated) {
			statusUpdated = true;
			scanHistoryStore.updateStatus(id, newStatus === 'complete' ? 'complete' : 'failed');
			cleanup();
			return true;
		}
		return false;
	};

	const handleSSEUpdate = (update: SSEUpdate) => {
		reportedSSEError = false;
		const normalizedState = (update.state || '').toUpperCase();
		const logMsg = getLogMessage(normalizedState, update);
		if (logMsg) {
			addLog(logMsg);
		}

		// Update result with progress info
		if (result) {
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
				: result.progress;
			result = {
				...result,
				state: update.state,
				progress: newProgress,
				error: update.error ?? result.error,
				error_details: update.error_details ?? result.error_details,
				last_stage: update.stage ?? result.last_stage
			};
		}

		const newStatus = normalizeStatus(update.state);
		status = newStatus;

		// If complete or failed, fetch full status for artifacts
		if (update.type === 'complete' || update.type === 'failed') {
			void fetchStatus();
		}
	};

	const fetchStatus = async () => {
		try {
			const res = await fetch(buildApiUrl(`/api/v1/jobs/${id}`));
			if (!res.ok) {
				throw new Error(res.status === 404 ? 'Job not found' : 'Network error');
			}
			const data = (await res.json()) as ScanResult;
			handleStatusData(data);
		} catch (err) {
			const errorMessage = err instanceof Error ? err.message : String(err);
			console.error('[scan-status] Failed to fetch job status:', { jobId: id, error: err });
			addLog(`ERROR: ${errorMessage}. Refresh to retry.`);
			if (status === 'loading') {
				status = 'error';
			}
		}
	};

	const startSSE = () => {
		const url = buildApiUrl(`/api/v1/jobs/${id}/stream`);
		eventSource = new EventSource(url);
		addLog('Connecting to live status stream...');
		let sseParseErrors = 0;
		let sseReconnectAttempts = 0;

		eventSource.onopen = () => {
			reportedSSEError = false;
			addLog('Live status stream connected.');
		};

		const handleSSEParseError = (eventType: string, err: unknown, rawData: string) => {
			sseParseErrors++;
			console.error(`[scan-status] Failed to parse SSE ${eventType} event:`, {
				jobId: id,
				parseErrors: sseParseErrors,
				error: err,
				rawData: rawData.slice(0, 200) // Truncate for logging
			});

			// If too many parse errors, stop: without valid SSE data we can't provide live updates.
			if (sseParseErrors >= MAX_SSE_PARSE_ERRORS) {
				console.warn('[scan-status] Too many SSE parse errors, stopping stream');
				status = 'error';
				addLog('ERROR: Live updates failed (invalid stream data). Refresh to retry.');
				cleanup();
			}
		};

		eventSource.addEventListener('status', (event) => {
			try {
				const data = JSON.parse(String(event.data)) as ScanResult;
				sseParseErrors = 0; // Reset on successful parse
				reportedSSEError = false;
				handleStatusData(data);
			} catch (err) {
				handleSSEParseError('status', err, String(event.data));
			}
		});

		eventSource.addEventListener('update', (event) => {
			try {
				const data = JSON.parse(String(event.data)) as SSEUpdate;
				sseParseErrors = 0; // Reset on successful parse
				handleSSEUpdate(data);
			} catch (err) {
				handleSSEParseError('update', err, String(event.data));
			}
		});

		// Handle "done" event - server sends this before closing for completed jobs
		eventSource.addEventListener('done', () => {
			eventSource?.close();
			eventSource = null;
			// Don't reconnect - this is a terminal stream closure.
		});

		eventSource.onerror = (event) => {
			// EventSource reconnects automatically on transient errors.
			// But if the connection is fully closed (server gone, job done, etc.)
			// we need to stop retrying and fall back to a one-time fetch.
			if (eventSource?.readyState === EventSource.CLOSED) {
				sseReconnectAttempts++;
				if (sseReconnectAttempts >= MAX_SSE_RECONNECT_ATTEMPTS) {
					console.warn('[scan-status] SSE connection closed after max reconnect attempts', {
						jobId: id,
						attempts: sseReconnectAttempts
					});
					addLog('WARN: Connection lost. Fetching latest status...');
					cleanup();
					void fetchStatus();
					return;
				}
			}

			// Log once per disconnect so we don't spam.
			if (!statusUpdated && !reportedSSEError) {
				reportedSSEError = true;
				console.warn('[scan-status] SSE connection error, waiting for reconnect', {
					jobId: id,
					readyState: eventSource?.readyState,
					reconnectAttempts: sseReconnectAttempts,
					event
				});
				addLog('WARN: Lost connection to live updates. Reconnecting...');
			}
		};
	};

	const start = () => {
		statusUpdated = false;
		reportedSSEError = false;

		elapsedInterval = setInterval(() => {
			elapsed = elapsed + 1;
		}, 1000);

		if (typeof EventSource !== 'undefined') {
			startSSE();
		} else {
			addLog('ERROR: Your browser does not support live updates (SSE).');
			void fetchStatus();
		}
	};

	return {
		get status() {
			return status;
		},
		get result() {
			return result;
		},
		get elapsed() {
			return elapsed;
		},
		get logs() {
			return logs;
		},
		start,
		cleanup
	};
}
