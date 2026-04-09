import type { ScanResult, ScanStatus } from '$lib/types/scan';

import type { SSEUpdate } from './scan-status/types';

import { scanHistoryStore } from './scan-history.svelte';
import {
	addLifecycleLog,
	applyStatusData,
	applyStatusUpdate,
	createScanJobStream,
	fetchScanJobStatus
} from './scan-status/job-stream';
import { MAX_LOG_LINES } from './scan-status/constants';

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

	const controller = {
		getJob: () => result,
		setJob: (job: ScanResult) => {
			result = job;
		},
		setStatus: (nextStatus: ScanStatus) => {
			status = nextStatus;
		},
		addLog,
		fetchErrorPrefix: 'scan-status',
		fallbackFetchErrorMessage: 'Network error',
		initialFetchErrorStatus: 'error' as const,
		onStatusData: (_data: ScanResult, nextStatus: ScanStatus, normalizedState: string) => {
			addLifecycleLog(
				addLog,
				normalizedState,
				{
					EXTRACTING: 'Verifying archive integrity...',
					SCANNING: 'Starting scanner execution...',
					COMPLETING: 'Finalizing reports and uploading artifacts...'
				},
				normalizedState !== 'SCANNING' || result?.progress?.current_page === 0
			);

			if (['complete', 'failed'].includes(nextStatus) && !statusUpdated) {
				statusUpdated = true;
				scanHistoryStore.updateStatus(id, nextStatus === 'complete' ? 'complete' : 'failed');
				cleanup();
			}
		},
		onUpdate: (update: SSEUpdate) => {
			if (update.type === 'complete' || update.type === 'failed') {
				void fetchStatus();
			}
		}
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

	const fetchStatus = async () => {
		await fetchScanJobStatus(id, controller);
	};

	const startSSE = () => {
		sseStream = createScanJobStream(controller, {
			jobId: id,
			sourceName: 'scan-status',
			onDone: () => {
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
		});
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
