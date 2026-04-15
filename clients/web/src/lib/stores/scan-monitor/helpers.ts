import type { ScanResult, ScanStatus } from '$lib/types/scan';

import { formatErrorDetails } from '../scan-status/log-messages';

import type {
	CreateMonitorOptions,
	ReportMonitorDependencies,
	ReportMonitorSnapshot,
	StatusMonitorDependencies,
	StatusMonitorSnapshot,
	TerminalStatus
} from './types';

export function isTerminalStatus(status: ScanStatus): status is TerminalStatus {
	return status === 'complete' || status === 'failed';
}

export function isReportSnapshot(
	snapshot: StatusMonitorSnapshot | ReportMonitorSnapshot
): snapshot is ReportMonitorSnapshot {
	return snapshot.kind === 'report';
}

export function hasHistoryPort(
	deps: StatusMonitorDependencies | ReportMonitorDependencies
): deps is StatusMonitorDependencies {
	return 'historyPort' in deps;
}

export function hasReportPort(
	deps: StatusMonitorDependencies | ReportMonitorDependencies
): deps is ReportMonitorDependencies {
	return 'reportPort' in deps;
}

export function addLifecycleLog(
	addLog: (message: string) => void,
	normalizedState: string,
	messageMap: Partial<Record<string, string>>,
	condition = true
) {
	if (!condition) {
		return;
	}

	const message = messageMap[normalizedState];
	if (message) {
		addLog(message);
	}
}

export function shouldAddLifecycleLog(
	kind: CreateMonitorOptions['kind'],
	normalizedState: string,
	data: ScanResult
) {
	if (normalizedState !== 'SCANNING') {
		return true;
	}

	if (kind === 'status') {
		return (data.progress?.current_page ?? 1) === 0;
	}

	return Boolean(data.progress);
}

export function readFetchStatusError(kind: CreateMonitorOptions['kind'], error: unknown) {
	if (error instanceof Error) {
		return error.message;
	}

	return kind === 'status' ? 'Network error' : 'Failed to fetch job status';
}

export function readFailureMessage(result: ScanResult) {
	const details = formatErrorDetails(result.error_details);
	const errorMessage = result.error ?? 'Unknown error';
	return `CRITICAL: Job failed - ${errorMessage}${details ? ` (${details})` : ''}`;
}
