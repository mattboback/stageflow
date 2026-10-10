import { useCallback, useEffect, useRef, useState } from 'react';

import type { ScanResult, ScanStatus, ScreenshotArtifact } from '../types/scan';
import type { UnifiedReport } from '../types/unified-report';

import { createSSEStream } from '../api/sse';
import { isUnifiedReport } from '../report-validation';
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

type TransportState = 'idle' | 'streaming' | 'reconnecting' | 'polling';

interface StatusSnapshot {
	status: ScanStatus;
	job: ScanResult | null;
	logs: string[];
	elapsedSeconds: number;
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

/* Callers pass screenshots into effect dependencies, so "none" must keep one
   identity across renders. */
const NO_SCREENSHOTS: ScreenshotArtifact[] = [];

const REPORT_RETRY_MS = 2000;
const REPORT_MAX_ATTEMPTS = 30;

function isTerminal(status: ScanStatus): boolean {
	return (
		status === 'complete' || status === 'failed' || status === 'error' || status === 'cancelled'
	);
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

	const report: unknown = await response.json();
	if (!isUnifiedReport(report)) {
		throw new Error('Report JSON did not match the expected aggregated schema.');
	}

	if (report.meta.jobId !== jobId) {
		throw new Error('Report job ID did not match the requested scan.');
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

	if (status === 'cancelled') {
		return 'Scan cancelled. Scanner pods are being torn down.';
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

export function useScanStatus(jobId: string, options: { trackElapsed?: boolean } = {}) {
	const trackElapsed = options.trackElapsed ?? true;
	const [snapshot, setSnapshot] = useState<StatusSnapshot>(emptyStatusSnapshot);
	const [attempt, setAttempt] = useState(0);
	const pollRef = useRef<number | null>(null);

	/* Tears down and re-subscribes the whole monitor (stream, polling, timer)
	   without a document reload. */
	const retry = useCallback(() => setAttempt((current) => current + 1), []);

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

		let closed = false;
		let stream: { close(): void } | null = null;
		const isActive = () => !closed;
		const isCurrentResult = (result: ScanResult) => isActive() && result.id === jobId;
		queueMicrotask(() => {
			if (isActive()) {
				setSnapshot(
					attempt === 0
						? emptyStatusSnapshot
						: addLog({ ...emptyStatusSnapshot }, 'Retrying connection...')
				);
			}
		});
		/* Time since the job started, not since this page opened: the page says
		   it is safe to leave, so coming back must not restart the clock. */
		const timer = trackElapsed
			? window.setInterval(() => {
					const now = Date.now();
					setSnapshot((current) => {
						if (isTerminal(current.status)) return current;
						const startedAt = current.job ? Date.parse(current.job.created_at) : Number.NaN;
						return {
							...current,
							elapsedSeconds: Number.isFinite(startedAt)
								? Math.max(0, Math.floor((now - startedAt) / 1000))
								: current.elapsedSeconds + 1
						};
					});
				}, 1000)
			: null;

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
						`ERROR: ${message}. Retry the connection to try again.`
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
			pollRef.current = window.setInterval(() => {
				void refresh();
			}, 5000);
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
								addLog(
									{ ...current, transport: 'reconnecting' },
									'WARN: Lost connection to live updates. Reconnecting...'
								)
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
							pollRef.current = window.setInterval(() => {
								void refresh();
							}, 5000);
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
			if (timer !== null) window.clearInterval(timer);
			stream?.close();
			clearPoll();
		};
	}, [attempt, clearPoll, jobId, trackElapsed]);

	return {
		status: snapshot.status,
		result: snapshot.job,
		elapsed: snapshot.elapsedSeconds,
		logs: snapshot.logs,
		transport: snapshot.transport,
		error: snapshot.error,
		retry
	};
}

/*
 * The status monitor plus the aggregated report, which exists only once the job
 * is complete and may lag the DONE event by a few seconds.
 */
export function useScanReport(jobId: string) {
	const {
		status,
		result,
		logs,
		transport,
		error: statusError
	} = useScanStatus(jobId, { trackElapsed: false });
	const [loaded, setLoaded] = useState<{
		jobId: string;
		report: UnifiedReport | null;
		error: string | null;
	}>({ jobId, report: null, error: null });
	const [reportAttempt, setReportAttempt] = useState(0);
	const retryReport = useCallback(() => {
		setLoaded({ jobId, report: null, error: null });
		setReportAttempt((current) => current + 1);
	}, [jobId]);

	useEffect(() => {
		if (status !== 'complete') {
			return;
		}

		let cancelled = false;
		let attempts = 0;
		let retryTimer: number | null = null;

		const load = async () => {
			try {
				const report = await fetchReport(jobId);
				if (cancelled) {
					return;
				}
				if (report) {
					setLoaded({ jobId, report, error: null });
					return;
				}
				if (attempts >= REPORT_MAX_ATTEMPTS) {
					setLoaded({
						jobId,
						report: null,
						error: 'Aggregated report is taking longer than expected. Retry loading the report.'
					});
					return;
				}
				attempts += 1;
				retryTimer = window.setTimeout(() => void load(), REPORT_RETRY_MS);
			} catch (error) {
				if (cancelled) {
					return;
				}
				const message = error instanceof Error ? error.message : 'Failed to load report';
				setLoaded({ jobId, report: null, error: `${message}. Retry loading the report.` });
			}
		};

		void load();

		return () => {
			cancelled = true;
			if (retryTimer !== null) window.clearTimeout(retryTimer);
		};
	}, [jobId, status, reportAttempt]);

	const job = result?.id === jobId ? result : null;
	const isCurrentJob = !result || job !== null;
	const current = loaded.jobId === jobId ? loaded : null;
	const report = status === 'complete' ? (current?.report ?? null) : null;
	const error = current?.error ?? statusError;

	return {
		status: isCurrentJob ? status : 'loading',
		job,
		report,
		screenshots: job?.artifacts?.screenshots ?? NO_SCREENSHOTS,
		logs: isCurrentJob ? logs : [],
		transport: isCurrentJob ? transport : 'idle',
		error: isCurrentJob ? error : null,
		reportError: isCurrentJob && status === 'complete' ? (current?.error ?? null) : null,
		retryReport
	};
}
