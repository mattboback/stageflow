import type { ScanResult, ScanStatus, ScreenshotArtifact } from '$lib/types/scan';
import type { UnifiedReport } from '$lib/types/unified-report';

import { createScanJobMonitor } from './scan-monitor';

export function createScanReportStore(id: string) {
	let status = $state<ScanStatus>('loading');
	let job = $state<ScanResult | null>(null);
	let report = $state<UnifiedReport | null>(null);
	let screenshots = $state<ScreenshotArtifact[]>([]);
	let logs = $state<string[]>([]);
	let error = $state<string | null>(null);

	const monitor = createScanJobMonitor({
		kind: 'report',
		jobId: id
	});

	const unsubscribe = monitor.subscribe((snapshot) => {
		status = snapshot.status;
		job = snapshot.job;
		report = snapshot.report;
		screenshots = snapshot.screenshots;
		logs = snapshot.logs;
		error = snapshot.error;
	});

	const cleanup = () => {
		monitor.stop();
		unsubscribe();
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
		start: () => {
			monitor.start();
		},
		cleanup,
		refreshArtifacts: () => monitor.refreshArtifacts()
	};
}
