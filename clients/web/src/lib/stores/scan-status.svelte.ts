import type { ScanResult, ScanStatus } from '$lib/types/scan';

import { scanHistoryStore } from './scan-history.svelte';
import { createScanJobMonitor } from './scan-monitor';

export function createScanStatusStore(id: string) {
	let status = $state<ScanStatus>('loading');
	let result = $state<ScanResult | null>(null);
	let elapsed = $state(0);
	let logs = $state<string[]>([]);

	const monitor = createScanJobMonitor(
		{
			kind: 'status',
			jobId: id
		},
		{
			historyPort: {
				markTerminal(jobId, nextStatus) {
					scanHistoryStore.updateStatus(jobId, nextStatus);
				}
			}
		}
	);

	const unsubscribe = monitor.subscribe((snapshot) => {
		status = snapshot.status;
		result = snapshot.job;
		elapsed = snapshot.elapsedSeconds;
		logs = snapshot.logs;
	});

	const cleanup = () => {
		monitor.stop();
		unsubscribe();
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
		start: () => {
			monitor.start();
		},
		cleanup
	};
}
