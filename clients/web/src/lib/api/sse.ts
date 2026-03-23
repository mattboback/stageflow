import { buildApiUrl } from '$lib/api/utils';
import {
	MAX_SSE_PARSE_ERRORS,
	MAX_SSE_RECONNECT_ATTEMPTS
} from '$lib/stores/scan-status/constants';

export interface SSECallbacks<TStatus, TUpdate> {
	onStatus: (data: TStatus) => void;
	onUpdate: (data: TUpdate) => void;
	onDone: () => void;
	onError: (err: { message: string; parseError?: boolean; terminal?: boolean }) => void;
}

export interface SSEOptions {
	sourceName?: string;
	onLog?: (msg: string) => void;
}

export function createSSEStream<TStatus, TUpdate>(
	jobId: string,
	callbacks: SSECallbacks<TStatus, TUpdate>,
	options?: SSEOptions
) {
	const sourceName = options?.sourceName ?? 'sse-stream';
	const url = buildApiUrl(`/api/v1/jobs/${jobId}/stream`);
	let eventSource: EventSource | null = new EventSource(url);
	let sseParseErrors = 0;
	let sseReconnectAttempts = 0;
	let reportedSSEError = false;

	const handleParseError = (eventType: string, err: unknown, rawData: string) => {
		sseParseErrors++;
		console.error(`[${sourceName}] Failed to parse SSE ${eventType} event:`, {
			jobId,
			parseErrors: sseParseErrors,
			error: err,
			rawData: rawData.slice(0, 200)
		});

		if (sseParseErrors >= MAX_SSE_PARSE_ERRORS) {
			console.warn(`[${sourceName}] Too many SSE parse errors, stopping stream`);
			options?.onLog?.('ERROR: Live updates failed (invalid stream data). Refresh to retry.');
			callbacks.onError({
				message: 'Too many parse errors',
				parseError: true,
				terminal: true
			});
			close();
		}
	};

	eventSource.onopen = () => {
		reportedSSEError = false;
		sseReconnectAttempts = 0;
		if (sourceName === 'scan-status') {
			options?.onLog?.('Live status stream connected.');
		}
	};

	eventSource.addEventListener('status', (event) => {
		try {
			const data = JSON.parse(String(event.data)) as TStatus;
			sseParseErrors = 0;
			reportedSSEError = false;
			callbacks.onStatus(data);
		} catch (err) {
			handleParseError('status', err, String(event.data));
		}
	});

	eventSource.addEventListener('update', (event) => {
		try {
			const data = JSON.parse(String(event.data)) as TUpdate;
			sseParseErrors = 0;
			reportedSSEError = false;
			callbacks.onUpdate(data);
		} catch (err) {
			handleParseError('update', err, String(event.data));
		}
	});

	eventSource.addEventListener('done', () => {
		callbacks.onDone();
		close();
	});

	eventSource.onerror = (event) => {
		if (eventSource?.readyState === EventSource.CLOSED) {
			console.warn(`[${sourceName}] SSE connection closed permanently`, {
				jobId
			});
			options?.onLog?.(
				sourceName === 'scan-status'
					? 'WARN: Connection closed. Fetching latest status...'
					: 'WARN: Connection closed. Switching to polling...'
			);
			callbacks.onError({
				message: 'Connection closed permanently',
				terminal: true
			});
			close();
			return;
		}

		sseReconnectAttempts++;
		if (sseReconnectAttempts >= MAX_SSE_RECONNECT_ATTEMPTS) {
			console.warn(`[${sourceName}] SSE connection failed after max reconnect attempts`, {
				jobId,
				attempts: sseReconnectAttempts
			});
			options?.onLog?.(
				sourceName === 'scan-status'
					? 'WARN: Connection lost. Fetching latest status...'
					: 'WARN: Connection lost. Switching to polling...'
			);
			callbacks.onError({
				message: 'Max reconnect attempts reached',
				terminal: true
			});
			close();
			return;
		}

		if (!reportedSSEError) {
			reportedSSEError = true;
			console.warn(`[${sourceName}] SSE connection error, waiting for reconnect`, {
				jobId,
				readyState: eventSource?.readyState,
				reconnectAttempts: sseReconnectAttempts,
				event
			});
			options?.onLog?.('WARN: Lost connection to live updates. Reconnecting...');
			callbacks.onError({ message: 'Connection error', terminal: false });
		}
	};

	const close = () => {
		if (eventSource) {
			eventSource.close();
			eventSource = null;
		}
	};

	return { close };
}
