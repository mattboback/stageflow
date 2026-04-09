import type { ScanResult, ScanStatus } from '$lib/types/scan';

import { createSSEStream } from '$lib/api/sse';
import { buildApiUrl } from '$lib/api/utils';

import type { SSEUpdate } from './types';

import { applySseUpdate } from './event-update';
import { getLogMessage, normalizeStatus } from './log-messages';
import { applyScannerCompletionUpdate, normalizeScannerProgress } from './scanner-progress';

export interface ScanJobStateController<TJob extends ScanResult | null> {
	getJob: () => TJob;
	setJob: (job: ScanResult) => void;
	setStatus: (status: ScanStatus) => void;
	addLog: (message: string) => void;
	fetchErrorPrefix: string;
	fallbackFetchErrorMessage: string;
	initialFetchErrorStatus?: ScanStatus;
	onStatusData?: (data: ScanResult, nextStatus: ScanStatus, normalizedState: string) => void;
	onStatusFetchSuccess?: (data: ScanResult) => void;
	onStatusFetchError?: (message: string, error: unknown) => void;
	onUpdate?: (update: SSEUpdate, nextStatus: ScanStatus, normalizedState: string) => void;
}

export interface ScanJobStreamOptions {
	jobId: string;
	sourceName: string;
	onDone?: () => void;
	onError: (err: { message: string; parseError?: boolean; terminal?: boolean }) => void;
}

export function addLifecycleLog(
	addLog: (message: string) => void,
	normalizedState: string,
	messageMap: Partial<Record<string, string>>,
	condition?: boolean
) {
	if (condition === false) {
		return;
	}

	const message = messageMap[normalizedState];
	if (message) {
		addLog(message);
	}
}

export function applyStatusData<TJob extends ScanResult | null>(
	controller: ScanJobStateController<TJob>,
	data: ScanResult
): { status: ScanStatus; normalizedState: string } {
	const normalizedState = (data.state || '').toUpperCase();
	const logMessage = getLogMessage(normalizedState, data);
	if (logMessage) {
		controller.addLog(logMessage);
	}

	const job = normalizeScannerProgress(data);
	const status = normalizeStatus(data.state);
	controller.setJob(job);
	controller.setStatus(status);
	controller.onStatusData?.(data, status, normalizedState);

	return { status, normalizedState };
}

export function applyStatusUpdate<TJob extends ScanResult | null>(
	controller: ScanJobStateController<TJob>,
	update: SSEUpdate
): { status: ScanStatus; normalizedState: string } {
	const normalizedState = (update.state || '').toUpperCase();
	const logMessage = getLogMessage(normalizedState, update);
	if (logMessage) {
		controller.addLog(logMessage);
	}

	const currentJob = controller.getJob();
	if (currentJob) {
		controller.setJob(applyScannerCompletionUpdate(applySseUpdate(currentJob, update), update));
	}

	const status = normalizeStatus(update.state);
	controller.setStatus(status);
	controller.onUpdate?.(update, status, normalizedState);

	return { status, normalizedState };
}

export async function fetchScanJobStatus<TJob extends ScanResult | null>(
	jobId: string,
	controller: ScanJobStateController<TJob>
) {
	try {
		const res = await fetch(buildApiUrl(`/api/v1/jobs/${jobId}`));
		if (!res.ok) {
			throw new Error(res.status === 404 ? 'Job not found' : controller.fallbackFetchErrorMessage);
		}

		const data = (await res.json()) as ScanResult;
		controller.onStatusFetchSuccess?.(data);
		applyStatusData(controller, data);
	} catch (err) {
		const message = err instanceof Error ? err.message : controller.fallbackFetchErrorMessage;
		console.error(`[${controller.fetchErrorPrefix}] Failed to fetch job status:`, {
			jobId,
			error: err
		});
		controller.addLog(`ERROR: ${message}. Refresh to retry.`);
		controller.onStatusFetchError?.(message, err);
		if (controller.initialFetchErrorStatus) {
			controller.setStatus(controller.initialFetchErrorStatus);
		}
	}
}

export function createScanJobStream<TJob extends ScanResult | null>(
	controller: ScanJobStateController<TJob>,
	options: ScanJobStreamOptions
) {
	return createSSEStream<ScanResult, SSEUpdate>(
		options.jobId,
		{
			onStatus: (data) => {
				applyStatusData(controller, data);
			},
			onUpdate: (update) => {
				applyStatusUpdate(controller, update);
			},
			onDone: () => {
				options.onDone?.();
			},
			onError: options.onError
		},
		{
			sourceName: options.sourceName,
			onLog: controller.addLog
		}
	);
}
