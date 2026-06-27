import { useCallback, useEffect, useRef, useState } from 'react';

import type { ScanResult, ScanStatus, ScreenshotArtifact } from '../types/scan';
import type { UnifiedReport } from '../types/unified-report';

import { createSSEStream } from '../api/sse';
import { buildApiUrl } from '../api/utils';
import { applySseUpdate } from '../stores/scan-status/event-update';
import {
	formatErrorDetails,
	getLogMessage,
	normalizeStatus
} from '../stores/scan-status/log-messages';
import {
	applyScannerCompletionUpdate,
	normalizeScannerProgress
} from '../stores/scan-status/scanner-progress';
import type { SSEUpdate } from '../stores/scan-status/types';

type TransportState = 'idle' | 'streaming' | 'polling';

interface StatusSnapshot {
	status: ScanStatus;
	job: ScanResult | null;
	logs: string[];
	elapsedSeconds: number;
	transport: TransportState;
	error: string | null;
}

interface ReportSnapshot {
	status: ScanStatus;
	job: ScanResult | null;
	report: UnifiedReport | null;
	screenshots: ScreenshotArtifact[];
	logs: string[];
	transport: TransportState;
	error: string | null;
}

const emptyStatusSnapshot: StatusSnapshot = {
	status: 'loading',
	job: null,
	logs: [],
	elapsedSeconds: 0,
	transport: 'idle',
	error: null
};

const emptyReportSnapshot: ReportSnapshot = {
	status: 'loading',
	job: null,
	report: null,
	screenshots: [],
	logs: [],
	transport: 'idle',
	error: null
};

const REPORT_RETRY_MS = 2000;
const REPORT_MAX_ATTEMPTS = 30;

function isTerminal(status: ScanStatus): boolean {
	return status === 'complete' || status === 'failed' || status === 'error';
}

function addLog<T extends { logs: string[] }>(snapshot: T, message: string | null): T {
	if (!message || snapshot.logs.includes(message)) {
		return snapshot;
	}

	return { ...snapshot, logs: [...snapshot.logs, message].slice(-50) };
}

async function fetchStatus(jobId: string): Promise<ScanResult> {
	const response = await fetch(buildApiUrl(`/api/v1/jobs/${jobId}`));
	if (!response.ok) {
		throw new Error(response.status === 404 ? 'Job not found' : 'Failed to fetch job status');
	}

	return normalizeScannerProgress((await response.json()) as ScanResult);
}

async function fetchReport(jobId: string): Promise<UnifiedReport | null> {
	const response = await fetch(buildApiUrl(`/api/v1/jobs/${jobId}/results`), {
		redirect: 'follow'
	});

	if ([400, 404, 409].includes(response.status)) {
		return null;
	}

	if (!response.ok) {
		throw new Error(`Report fetch failed: ${response.status}`);
	}

	const report = (await response.json()) as UnifiedReport;
	if (!report.meta.jobId || !report.version) {
		throw new Error('Report JSON did not match the expected aggregated schema.');
	}

	return report;
}

function failureMessage(data: { error?: string; error_details?: string }): string {
	const details = formatErrorDetails(data.error_details);
	const errorMessage = data.error ?? 'Unknown error';
	return `CRITICAL: Job failed - ${errorMessage}${details ? ` (${details})` : ''}`;
}

function statusLog(data: ScanResult): string | null {
	const status = normalizeStatus(data.state);
	if (status === 'failed') {
		return failureMessage(data);
	}

	return getLogMessage((data.state || '').toUpperCase(), data);
}

function updateLog(update: SSEUpdate): string | null {
	const status = normalizeStatus(update.state);
	if (status === 'failed') {
		return failureMessage(update);
	}

	return getLogMessage((update.state || '').toUpperCase(), update);
}

export function useScanStatus(jobId: string) {
	const [snapshot, setSnapshot] = useState<StatusSnapshot>(emptyStatusSnapshot);
	const pollRef = useRef<number | null>(null);

	const clearPoll = useCallback(() => {
		if (pollRef.current !== null) {
			window.clearInterval(pollRef.current);
			pollRef.current = null;
		}
	}, []);

	useEffect(() => {
		if (!jobId) {
			return;
		}

		setSnapshot(emptyStatusSnapshot);

		let closed = false;
		let stream: { close(): void } | null = null;
		const isActive = () => !closed;
		const isCurrentResult = (result: ScanResult) => isActive() && result.id === jobId;
		const timer = window.setInterval(() => {
			setSnapshot((current) =>
				isTerminal(current.status)
					? current
					: { ...current, elapsedSeconds: current.elapsedSeconds + 1 }
			);
		}, 1000);

		const applyStatus = (result: ScanResult) => {
			if (!isCurrentResult(result)) {
				return;
			}

			const nextStatus = normalizeStatus(result.state);
			setSnapshot((current) =>
				addLog(
					{
						...current,
						status: nextStatus,
						job: result,
						error: null
					},
					statusLog(result)
				)
			);

			if (isTerminal(nextStatus)) {
				clearPoll();
				stream?.close();
			}
		};

		const refresh = async () => {
			try {
				applyStatus(await fetchStatus(jobId));
			} catch (error) {
				if (!isActive()) {
					return;
				}
				const message = error instanceof Error ? error.message : 'Network error';
				setSnapshot((current) =>
					addLog(
						{ ...current, status: 'error', error: message },
						`ERROR: ${message}. Refresh to retry.`
					)
				);
			}
		};

		void refresh();

		if (typeof EventSource === 'undefined') {
			queueMicrotask(() => {
				if (!isActive()) {
					return;
				}

				setSnapshot((current) =>
					addLog(
						{ ...current, transport: 'polling' },
						'ERROR: Your browser does not support live updates (SSE). Switching to polling...'
					)
				);
			});
			pollRef.current = window.setInterval(refresh, 5000);
		} else {
			stream = createSSEStream<ScanResult, SSEUpdate>(
				jobId,
				{
					onStatus: applyStatus,
					onUpdate(update) {
						if (!isActive()) {
							return;
						}

						setSnapshot((current) => {
							const job = current.job
								? applyScannerCompletionUpdate(applySseUpdate(current.job, update), update)
								: current.job;
							return addLog(
								{
									...current,
									status: normalizeStatus(update.state),
									job
								},
								updateLog(update)
							);
						});

						if (isTerminal(normalizeStatus(update.state))) {
							clearPoll();
							stream?.close();
							void refresh();
						}
					},
					onDone: () => undefined,
					onError(error) {
						if (!isActive()) {
							return;
						}

						if (error.kind === 'transient') {
							setSnapshot((current) =>
								addLog(current, 'WARN: Lost connection to live updates. Reconnecting...')
							);
							void refresh();
							return;
						}

						stream?.close();
						setSnapshot((current) =>
							addLog(
								{ ...current, transport: 'polling' },
								'WARN: Connection lost. Switching to polling...'
							)
						);
						if (pollRef.current === null) {
							pollRef.current = window.setInterval(refresh, 5000);
						}
						void refresh();
					}
				},
				{
					sourceName: 'scan-status',
					onOpen: () => {
						if (!isActive()) {
							return;
						}

						clearPoll();
						setSnapshot((current) =>
							addLog({ ...current, transport: 'streaming' }, 'Live status stream connected.')
						);
					}
				}
			);
		}

		return () => {
			closed = true;
			window.clearInterval(timer);
			stream?.close();
			clearPoll();
		};
	}, [clearPoll, jobId]);

	return {
		status: snapshot.status,
		result: snapshot.job,
		elapsed: snapshot.elapsedSeconds,
		logs: snapshot.logs,
		transport: snapshot.transport,
		error: snapshot.error
	};
}

export function useScanReport(jobId: string) {
	const [snapshot, setSnapshot] = useState<ReportSnapshot>(emptyReportSnapshot);
	const pollRef = useRef<number | null>(null);
	const reportRetryRef = useRef<number | null>(null);

	const clearPoll = useCallback(() => {
		if (pollRef.current !== null) {
			window.clearInterval(pollRef.current);
			pollRef.current = null;
		}
	}, []);

	const clearReportRetry = useCallback(() => {
		if (reportRetryRef.current !== null) {
			window.clearTimeout(reportRetryRef.current);
			reportRetryRef.current = null;
		}
	}, []);

	const refreshArtifacts = useCallback(async () => {
		if (!jobId) {
			return;
		}

		try {
			const result = await fetchStatus(jobId);
			setSnapshot((current) => ({
				...current,
				status: normalizeStatus(result.state),
				job: result,
				screenshots:
					result.artifacts?.screenshots ?? (current.job?.id === jobId ? current.screenshots : []),
				error: null
			}));
		} catch (error) {
			const message = error instanceof Error ? error.message : 'Failed to fetch job status';
			setSnapshot((current) =>
				addLog({ ...current, error: message }, `ERROR: ${message}. Refresh to retry.`)
			);
		}
	}, [jobId]);

	useEffect(() => {
		if (!jobId) {
			return;
		}

		setSnapshot(emptyReportSnapshot);

		let closed = false;
		let stream: { close(): void } | null = null;
		let reportRetryAttempts = 0;
		const isActive = () => !closed;
		const isCurrentResult = (result: ScanResult) => isActive() && result.id === jobId;

		const scheduleReportRetry = () => {
			if (!isActive() || reportRetryRef.current !== null) {
				return;
			}

			if (reportRetryAttempts >= REPORT_MAX_ATTEMPTS) {
				const message = 'Aggregated report is taking longer than expected. Refresh to retry.';
				setSnapshot((current) => addLog({ ...current, error: message }, `WARN: ${message}`));
				return;
			}

			reportRetryAttempts += 1;
			reportRetryRef.current = window.setTimeout(() => {
				reportRetryRef.current = null;
				void loadReport();
			}, REPORT_RETRY_MS);
		};

		const loadReport = async () => {
			if (!isActive()) {
				return;
			}

			try {
				const report = await fetchReport(jobId);
				if (!isActive() || (report && report.meta.jobId !== jobId)) {
					return;
				}
				if (!report) {
					scheduleReportRetry();
					return;
				}
				clearPoll();
				clearReportRetry();
				setSnapshot((current) => ({ ...current, report, error: null }));
			} catch (error) {
				if (!isActive()) {
					return;
				}
				const message = error instanceof Error ? error.message : 'Failed to load report';
				setSnapshot((current) =>
					addLog(
						{ ...current, error: `${message}. Refresh to retry.` },
						`ERROR: ${message}. Refresh to retry.`
					)
				);
			}
		};

		const applyStatus = (result: ScanResult) => {
			if (!isCurrentResult(result)) {
				return;
			}

			const nextStatus = normalizeStatus(result.state);
			setSnapshot((current) =>
				addLog(
					{
						...current,
						status: nextStatus,
						job: result,
						screenshots:
							result.artifacts?.screenshots ??
							(current.job?.id === jobId ? current.screenshots : []),
						report:
							nextStatus === 'failed' || nextStatus === 'error'
								? null
								: current.report?.meta.jobId === jobId
									? current.report
									: null,
						error: null
					},
					statusLog(result)
				)
			);

			if (nextStatus === 'complete') {
				clearPoll();
				void loadReport();
			} else if (isTerminal(nextStatus)) {
				clearReportRetry();
				clearPoll();
			}

			if (isTerminal(nextStatus)) {
				stream?.close();
			}
		};

		const refresh = async () => {
			try {
				const result = await fetchStatus(jobId);
				applyStatus(result);
			} catch (error) {
				if (!isActive()) {
					return;
				}
				const message = error instanceof Error ? error.message : 'Failed to fetch job status';
				setSnapshot((current) =>
					addLog(
						{ ...current, status: 'error', error: message },
						`ERROR: ${message}. Refresh to retry.`
					)
				);
			}
		};

		void refresh();

		if (typeof EventSource !== 'undefined') {
			stream = createSSEStream<ScanResult, SSEUpdate>(
				jobId,
				{
					onStatus: applyStatus,
					onUpdate(update) {
						if (!isActive()) {
							return;
						}

						setSnapshot((current) => {
							const job = current.job?.id === jobId
								? applyScannerCompletionUpdate(applySseUpdate(current.job, update), update)
								: null;
							return addLog(
								{
									...current,
									status: normalizeStatus(update.state),
									job,
									screenshots:
										job?.artifacts?.screenshots ??
										(current.job?.id === jobId ? current.screenshots : []),
									report:
										current.report?.meta.jobId === jobId &&
										normalizeStatus(update.state) !== 'failed'
											? current.report
											: null
								},
								updateLog(update)
							);
						});
						if (isTerminal(normalizeStatus(update.state))) {
							void refresh();
						}
					},
					onDone: () => undefined,
					onError(error) {
						if (!isActive()) {
							return;
						}

						if (error.kind === 'transient') {
							setSnapshot((current) =>
								addLog(current, 'WARN: Lost connection to live updates. Reconnecting...')
							);
							void refresh();
							return;
						}

						stream?.close();
						setSnapshot((current) =>
							addLog(
								{ ...current, transport: 'polling' },
								'WARN: Connection lost. Switching to polling...'
							)
						);
						if (pollRef.current === null) {
							pollRef.current = window.setInterval(refresh, 5000);
						}
						void refresh();
					}
				},
				{
					sourceName: 'scan-report',
					onOpen: () => {
						if (!isActive()) {
							return;
						}

						clearPoll();
						setSnapshot((current) => ({ ...current, transport: 'streaming' }));
					}
				}
			);
		} else {
			queueMicrotask(() => {
				if (!isActive()) {
					return;
				}

				setSnapshot((current) => ({ ...current, transport: 'polling' }));
			});
			pollRef.current = window.setInterval(refresh, 5000);
		}

		return () => {
			closed = true;
			stream?.close();
			clearPoll();
			clearReportRetry();
		};
	}, [clearPoll, clearReportRetry, jobId]);

	const isCurrentJob = !snapshot.job || snapshot.job.id === jobId;
	const job = isCurrentJob ? snapshot.job : null;
	const report = snapshot.report?.meta.jobId === jobId ? snapshot.report : null;

	return {
		status: isCurrentJob ? snapshot.status : 'loading',
		job,
		report,
		screenshots: job ? snapshot.screenshots : [],
		logs: isCurrentJob ? snapshot.logs : [],
		transport: isCurrentJob ? snapshot.transport : 'idle',
		error: isCurrentJob ? snapshot.error : null,
		refreshArtifacts
	};
}
