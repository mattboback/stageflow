import type { ScanResult } from '$lib/types/scan';

import type { SSEUpdate } from '../scan-status/types';

import { applySseUpdate } from '../scan-status/event-update';
import { formatErrorDetails, getLogMessage, normalizeStatus } from '../scan-status/log-messages';
import {
	applyScannerCompletionUpdate,
	normalizeScannerProgress
} from '../scan-status/scanner-progress';

import {
	createReportSnapshot,
	createStatusSnapshot,
	DEFAULT_POLL_INTERVAL_MS,
	DEFAULT_REPORT_RETRY,
	REPORT_LIFECYCLE_MESSAGES,
	STATUS_LIFECYCLE_MESSAGES
} from './defaults';
import {
	addLifecycleLog,
	hasHistoryPort,
	hasReportPort,
	isReportSnapshot,
	isTerminalStatus,
	readFailureMessage,
	readFetchStatusError,
	shouldAddLifecycleLog
} from './helpers';
import { browserScheduler, createEventsPort, createReportPort, createStatusPort } from './ports';
import type {
	CreateMonitorOptions,
	MonitorSubscription,
	ReportMonitorDependencyOverrides,
	ReportMonitorDependencies,
	ReportMonitorSnapshot,
	ScanJobMonitor,
	StatusMonitorDependencies,
	StatusMonitorDependencyOverrides,
	StatusMonitorOptions,
	StatusMonitorSnapshot,
	TerminalStatus,
	TransportState,
	StreamTransportEvent
} from './types';

function resolveStatusDependencies(
	deps: StatusMonitorDependencyOverrides = {}
): StatusMonitorDependencies {
	return {
		statusPort: deps.statusPort ?? createStatusPort(),
		eventsPort: deps.eventsPort ?? createEventsPort(),
		scheduler: deps.scheduler ?? browserScheduler(),
		...(deps.historyPort ? { historyPort: deps.historyPort } : {})
	};
}

function resolveReportDependencies(
	deps: ReportMonitorDependencyOverrides = {}
): ReportMonitorDependencies {
	return {
		statusPort: deps.statusPort ?? createStatusPort(),
		eventsPort: deps.eventsPort ?? createEventsPort(),
		scheduler: deps.scheduler ?? browserScheduler(),
		reportPort: deps.reportPort ?? createReportPort()
	};
}

export function createScanJobMonitor(
	options: StatusMonitorOptions,
	deps?: StatusMonitorDependencyOverrides
): ScanJobMonitor<StatusMonitorSnapshot>;
export function createScanJobMonitor(
	options: import('./types').ReportMonitorOptions,
	deps?: ReportMonitorDependencyOverrides
): ScanJobMonitor<ReportMonitorSnapshot>;
export function createScanJobMonitor(
	options: CreateMonitorOptions,
	deps: StatusMonitorDependencyOverrides | ReportMonitorDependencyOverrides = {}
): ScanJobMonitor<StatusMonitorSnapshot | ReportMonitorSnapshot> {
	const resolvedDeps =
		options.kind === 'status'
			? resolveStatusDependencies(deps as StatusMonitorDependencyOverrides)
			: resolveReportDependencies(deps as ReportMonitorDependencyOverrides);
	const listeners = new Set<MonitorSubscription<StatusMonitorSnapshot | ReportMonitorSnapshot>>();
	const lifecycleMessages =
		options.kind === 'status'
			? { ...STATUS_LIFECYCLE_MESSAGES, ...options.lifecycleMessages }
			: { ...REPORT_LIFECYCLE_MESSAGES, ...options.lifecycleMessages };
	const logLimit = options.logLimit ?? 50;
	const reportRetry = {
		...DEFAULT_REPORT_RETRY,
		...(options.kind === 'report' ? options.reportRetry : {})
	};
	const pollIntervalMs =
		options.kind === 'report' ? (options.pollIntervalMs ?? DEFAULT_POLL_INTERVAL_MS) : 0;

	let snapshot: StatusMonitorSnapshot | ReportMonitorSnapshot =
		options.kind === 'status'
			? createStatusSnapshot(options.jobId)
			: createReportSnapshot(options.jobId);
	let active = false;
	let generation = 0;
	let streamHandle: { close(): void } | null = null;
	let cancelElapsedTimer: (() => void) | null = null;
	let cancelPollTimer: (() => void) | null = null;
	let cancelReportRetry: (() => void) | null = null;
	let reportRetryAttempts = 0;
	let reportRetryDelayMs = reportRetry.initialDelayMs;
	let reportFetchInFlight = false;
	let terminalHandled = false;
	const seenLogs = new Set<string>();

	const emit = () => {
		for (const listener of listeners) {
			listener(snapshot);
		}
	};

	const updateSnapshot = (
		updater: (
			current: StatusMonitorSnapshot | ReportMonitorSnapshot
		) => StatusMonitorSnapshot | ReportMonitorSnapshot
	) => {
		snapshot = updater(snapshot);
		emit();
	};

	const addLog = (message: string) => {
		if (!message || seenLogs.has(message)) {
			return;
		}

		seenLogs.add(message);
		updateSnapshot((current) => ({
			...current,
			logs: [...current.logs, message].slice(-logLimit)
		}));
	};

	const setError = (error: string | null) => {
		updateSnapshot((current) => ({
			...current,
			error
		}));
	};

	const setTransport = (transport: TransportState) => {
		if (snapshot.transport === transport) {
			return;
		}

		updateSnapshot((current) => ({
			...current,
			transport
		}));
	};

	const clearReportRetry = () => {
		if (cancelReportRetry) {
			cancelReportRetry();
			cancelReportRetry = null;
		}
	};

	const closeStream = () => {
		if (!streamHandle) {
			return;
		}

		streamHandle.close();
		streamHandle = null;
	};

	const stopPolling = () => {
		if (!cancelPollTimer) {
			return;
		}

		cancelPollTimer();
		cancelPollTimer = null;
	};

	const stopElapsedTimer = () => {
		if (!cancelElapsedTimer) {
			return;
		}

		cancelElapsedTimer();
		cancelElapsedTimer = null;
	};

	const reset = () => {
		seenLogs.clear();
		terminalHandled = false;
		reportFetchInFlight = false;
		reportRetryAttempts = 0;
		reportRetryDelayMs = reportRetry.initialDelayMs;
		clearReportRetry();
		stopPolling();
		stopElapsedTimer();
		closeStream();
		snapshot =
			options.kind === 'status'
				? createStatusSnapshot(options.jobId)
				: createReportSnapshot(options.jobId);
		emit();
	};

	const applyTerminalSideEffects = (status: TerminalStatus) => {
		if (terminalHandled) {
			return;
		}

		terminalHandled = true;
		stopPolling();
		stopElapsedTimer();
		closeStream();
		clearReportRetry();
		setTransport('idle');

		if (options.kind === 'status') {
			void (hasHistoryPort(resolvedDeps)
				? resolvedDeps.historyPort?.markTerminal(options.jobId, status)
				: undefined);
		}
	};

	const scheduleReportRetry = (token: number) => {
		if (!active || options.kind !== 'report') {
			return;
		}

		if (
			(isReportSnapshot(snapshot) && snapshot.report) ||
			snapshot.status !== 'complete' ||
			cancelReportRetry
		) {
			return;
		}

		if (reportRetryAttempts >= reportRetry.maxAttempts) {
			const message = 'Aggregated report is taking longer than expected. Refresh to retry.';
			addLog(`WARN: ${message}`);
			setError(message);
			return;
		}

		if (reportRetryAttempts === 0) {
			addLog('Scan complete. Generating aggregated report...');
		}

		const delayMs = reportRetryDelayMs;
		reportRetryAttempts += 1;
		reportRetryDelayMs = Math.min(reportRetry.maxDelayMs, Math.round(reportRetryDelayMs * 1.7));

		cancelReportRetry = resolvedDeps.scheduler.after(delayMs, () => {
			cancelReportRetry = null;
			void fetchReport(token);
		});
	};

	const setReportFetchError = (message: string) => {
		const retryableMessage = `${message}. Refresh to retry.`;
		addLog(`ERROR: ${retryableMessage}`);
		setError(retryableMessage);
	};

	const fetchReport = async (token = generation) => {
		if (
			options.kind !== 'report' ||
			reportFetchInFlight ||
			!active ||
			!hasReportPort(resolvedDeps)
		) {
			return;
		}

		reportFetchInFlight = true;
		let shouldRefreshArtifacts = false;

		try {
			const response = await resolvedDeps.reportPort.fetch(options.jobId);
			if (!active || generation !== token) {
				return;
			}

			if (response.state === 'pending') {
				scheduleReportRetry(token);
				return;
			}

			if (response.state === 'failed') {
				setReportFetchError(response.message);
				return;
			}

			clearReportRetry();
			updateSnapshot((current) => {
				if (!isReportSnapshot(current)) {
					return current;
				}

				return {
					...current,
					report: response.report,
					error: null
				};
			});

			shouldRefreshArtifacts =
				isReportSnapshot(snapshot) &&
				snapshot.status === 'complete' &&
				snapshot.screenshots.length === 0;
		} catch (error) {
			if (!active || generation !== token) {
				return;
			}

			const message = error instanceof Error ? error.message : 'Failed to load report';
			setReportFetchError(`Report fetch failed: ${message}`);
		} finally {
			if (generation === token) {
				reportFetchInFlight = false;
			}
		}

		if (shouldRefreshArtifacts && active && generation === token) {
			await refreshStatus(token);
		}
	};

	const processStatusSnapshot = (data: ScanResult, token: number) => {
		const normalizedState = (data.state || '').toUpperCase();
		const nextStatus = normalizeStatus(data.state);
		const nextJob = normalizeScannerProgress(data);
		const logMessage =
			nextStatus === 'failed' ? readFailureMessage(data) : getLogMessage(normalizedState, data);

		if (logMessage) {
			addLog(logMessage);
		}

		updateSnapshot((current) => {
			const base = {
				...current,
				status: nextStatus,
				job: nextJob,
				error: null
			};

			if (!isReportSnapshot(current)) {
				return base;
			}

			return {
				...base,
				screenshots: data.artifacts?.screenshots ?? [],
				report: nextStatus === 'failed' ? null : current.report
			};
		});

		addLifecycleLog(
			addLog,
			normalizedState,
			lifecycleMessages,
			shouldAddLifecycleLog(options.kind, normalizedState, data)
		);

		if (isTerminalStatus(nextStatus)) {
			applyTerminalSideEffects(nextStatus);
		}

		if (
			options.kind === 'report' &&
			nextStatus === 'complete' &&
			(!isReportSnapshot(snapshot) || !snapshot.report)
		) {
			void fetchReport(token);
		}
	};

	const processUpdate = (update: SSEUpdate, token: number) => {
		const normalizedState = (update.state || '').toUpperCase();
		const nextStatus = normalizeStatus(update.state);
		const errorDetails = update.error_details ? formatErrorDetails(update.error_details) : null;
		const logMessage =
			nextStatus === 'failed'
				? `CRITICAL: Job failed - ${update.error ?? 'Unknown error'}${errorDetails ? ` (${errorDetails})` : ''}`
				: getLogMessage(normalizedState, update);

		if (logMessage) {
			addLog(logMessage);
		}

		updateSnapshot((current) => {
			const nextJob = current.job
				? applyScannerCompletionUpdate(applySseUpdate(current.job, update), update)
				: current.job;
			const base = {
				...current,
				status: nextStatus,
				job: nextJob
			};

			if (!isReportSnapshot(current)) {
				return base;
			}

			return {
				...base,
				screenshots: nextJob?.artifacts?.screenshots ?? current.screenshots,
				report: nextStatus === 'failed' ? null : current.report
			};
		});

		if (isTerminalStatus(nextStatus)) {
			applyTerminalSideEffects(nextStatus);
			void refreshStatus(token);
		}
	};

	const handleStreamTransport = (event: StreamTransportEvent, token: number) => {
		if (!active || generation !== token) {
			return;
		}

		if (event.type === 'connected') {
			setTransport('streaming');
			if (options.kind === 'status') {
				addLog('Live status stream connected.');
			}
			return;
		}

		if (event.type === 'retrying') {
			addLog('WARN: Lost connection to live updates. Reconnecting...');
			return;
		}

		closeStream();

		if (event.reason === 'invalid-message') {
			addLog('ERROR: Live updates failed (invalid stream data). Refresh to retry.');
			if (options.kind === 'status') {
				updateSnapshot((current) => ({
					...current,
					status: 'error',
					error: 'Live updates failed (invalid stream data).'
				}));
				stopElapsedTimer();
				setTransport('idle');
				return;
			}
		}

		if (options.kind === 'status') {
			addLog('WARN: Connection lost. Fetching latest status...');
			setTransport('idle');
			void refreshStatus(token);
			return;
		}

		addLog(
			event.reason === 'closed'
				? 'WARN: Connection closed. Switching to polling...'
				: 'WARN: Connection lost. Switching to polling...'
		);
		setTransport('polling');
		startPolling(token);
		void refreshStatus(token);
	};

	const startElapsedTimer = () => {
		if (options.kind !== 'status' || cancelElapsedTimer) {
			return;
		}

		cancelElapsedTimer = resolvedDeps.scheduler.every(1000, () => {
			if (!active || isTerminalStatus(snapshot.status) || snapshot.status === 'error') {
				stopElapsedTimer();
				return;
			}

			updateSnapshot((current) => {
				if (current.kind !== 'status') {
					return current;
				}

				return {
					...current,
					elapsedSeconds: current.elapsedSeconds + 1
				};
			});
		});
	};

	const startPolling = (token: number) => {
		if (options.kind !== 'report' || cancelPollTimer || !active) {
			return;
		}

		setTransport('polling');
		cancelPollTimer = resolvedDeps.scheduler.every(pollIntervalMs, () => {
			if (!active || generation !== token) {
				stopPolling();
				return;
			}

			if (isTerminalStatus(snapshot.status)) {
				stopPolling();
				return;
			}

			void refreshStatus(token);
		});
	};

	const startStream = (token: number) => {
		if (!active || streamHandle || !resolvedDeps.eventsPort.supportsStreaming()) {
			return;
		}

		streamHandle = resolvedDeps.eventsPort.open(options.jobId, {
			onStatus: (data) => {
				if (!active || generation !== token) {
					return;
				}

				processStatusSnapshot(data, token);
			},
			onUpdate: (update) => {
				if (!active || generation !== token) {
					return;
				}

				processUpdate(update, token);
			},
			onTransport: (event) => {
				handleStreamTransport(event, token);
			}
		});
	};

	async function refreshStatus(token = generation) {
		try {
			const result = await resolvedDeps.statusPort.fetch(options.jobId);
			if (!active || generation !== token) {
				return;
			}

			processStatusSnapshot(result, token);
		} catch (error) {
			if (!active || generation !== token) {
				return;
			}

			const message = readFetchStatusError(options.kind, error);
			addLog(`ERROR: ${message}. Refresh to retry.`);
			updateSnapshot((current) => ({
				...current,
				status: 'error',
				error: message
			}));
		}
	}

	const refreshArtifacts = async () => {
		await refreshStatus(generation);
	};

	const start = () => {
		if (active) {
			return;
		}

		active = true;
		generation += 1;
		const token = generation;
		reset();

		if (options.kind === 'status') {
			startElapsedTimer();
			if (resolvedDeps.eventsPort.supportsStreaming()) {
				startStream(token);
			} else {
				addLog('ERROR: Your browser does not support live updates (SSE).');
				void refreshStatus(token);
			}
			return;
		}

		if (resolvedDeps.eventsPort.supportsStreaming()) {
			startStream(token);
		} else {
			startPolling(token);
		}

		void refreshStatus(token);
	};

	const stop = () => {
		if (!active) {
			return;
		}

		active = false;
		generation += 1;
		clearReportRetry();
		stopPolling();
		stopElapsedTimer();
		closeStream();
		setTransport('idle');
	};

	return {
		start,
		stop,
		refreshStatus,
		getSnapshot() {
			return snapshot;
		},
		subscribe(listener) {
			listeners.add(listener);
			listener(snapshot);
			return () => {
				listeners.delete(listener);
			};
		},
		refreshArtifacts
	};
}
