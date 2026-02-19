import type { ScanResult, ScanStatus } from '$lib/types/scan';

import { createSSEStream } from '$lib/api/sse';
import { buildApiUrl } from '$lib/api/utils';

import type { SSEUpdate } from './scan-status/types';

import { scanHistoryStore } from './scan-history.svelte';
import {
	MAX_LOG_LINES
} from './scan-status/constants';
import { getLogMessage, normalizeStatus } from './scan-status/log-messages';

export function createScanStatusStore(id: string) {
	let status = $state<ScanStatus>('loading');
	let result = $state<ScanResult | null>(null);
	let elapsed = $state(0);
	let logs = $state<string[]>([]);

	let elapsedInterval: ReturnType<typeof setInterval> | null = null;
	let sseStream: { close: () => void } | null = null;
	let statusUpdated = false;
	let started = false;

	const addLog = (msg: string) => {
		logs = [...logs, msg].slice(-MAX_LOG_LINES);
	};

	const cleanup = () => {
		if (elapsedInterval) {
			clearInterval(elapsedInterval);
			elapsedInterval = null;
		}
		if (sseStream) {
			sseStream.close();
			sseStream = null;
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
		} else if (normalizedState === 'SCANNING' && data.progress?.current_page === 0) {
			addLog('Starting scanner execution...');
		} else if (normalizedState === 'COMPLETING') {
			addLog('Finalizing reports and uploading artifacts...');
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
		sseStream = createSSEStream<ScanResult, SSEUpdate>(
			id,
			{
				onStatus: handleStatusData,
				onUpdate: handleSSEUpdate,
				onDone: () => {
					// Don't reconnect - this is a terminal stream closure.
					sseStream = null;
				},
				onError: (err) => {
					if (err.parseError) {
						status = 'error';
						cleanup();
					} else if (err.terminal) {
						cleanup();
						void fetchStatus();
					}
				}
			},
			{
				sourceName: 'scan-status',
				onLog: addLog
			}
		);
	};

	const start = () => {
		if (started) return;
		started = true;
		statusUpdated = false;

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
