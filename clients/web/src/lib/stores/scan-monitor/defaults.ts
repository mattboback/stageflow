import type { ReportMonitorSnapshot, ReportRetryOptions, StatusMonitorSnapshot } from './types';

export const STATUS_LIFECYCLE_MESSAGES: Record<'EXTRACTING' | 'SCANNING' | 'COMPLETING', string> = {
	EXTRACTING: 'Verifying archive integrity...',
	SCANNING: 'Starting scanner execution...',
	COMPLETING: 'Finalizing reports and uploading artifacts...'
};

export const REPORT_LIFECYCLE_MESSAGES: Record<'EXTRACTING' | 'SCANNING' | 'COMPLETING', string> = {
	EXTRACTING: 'Verifying archive integrity...',
	SCANNING: '[axe-core] Injecting accessibility engine...',
	COMPLETING: 'Uploading artifacts to secure storage...'
};

export const DEFAULT_POLL_INTERVAL_MS = 5_000;

export const DEFAULT_REPORT_RETRY: ReportRetryOptions = {
	initialDelayMs: 800,
	maxDelayMs: 10_000,
	maxAttempts: 30
};

export function createStatusSnapshot(jobId: string): StatusMonitorSnapshot {
	return {
		kind: 'status',
		jobId,
		status: 'loading',
		job: null,
		logs: [],
		elapsedSeconds: 0,
		transport: 'idle',
		error: null
	};
}

export function createReportSnapshot(jobId: string): ReportMonitorSnapshot {
	return {
		kind: 'report',
		jobId,
		status: 'loading',
		job: null,
		report: null,
		screenshots: [],
		logs: [],
		transport: 'idle',
		error: null
	};
}
