import type { ScanResult } from '$lib/types/scan';
import type { UnifiedReport } from '$lib/types/unified-report';

import { createSSEStream } from '$lib/api/sse';
import { buildApiUrl } from '$lib/api/utils';

import type { SSEUpdate } from '../scan-status/types';

import type {
	ScanJobEventsPort,
	ScanJobReportPort,
	ScanJobStatusPort,
	SchedulerPort
} from './types';

export function browserScheduler(): SchedulerPort {
	return {
		every(ms, fn) {
			const handle = setInterval(fn, ms);
			return () => clearInterval(handle);
		},
		after(ms, fn) {
			const handle = setTimeout(fn, ms);
			return () => clearTimeout(handle);
		}
	};
}

export function createStatusPort(): ScanJobStatusPort {
	return {
		async fetch(jobId) {
			const response = await fetch(buildApiUrl(`/api/v1/jobs/${jobId}`));
			if (!response.ok) {
				throw new Error(response.status === 404 ? 'Job not found' : 'Failed to fetch job status');
			}

			return (await response.json()) as ScanResult;
		}
	};
}

export function createReportPort(): ScanJobReportPort {
	return {
		async fetch(jobId) {
			try {
				const response = await fetch(buildApiUrl(`/api/v1/jobs/${jobId}/results`), {
					redirect: 'follow'
				});
				if (!response.ok) {
					if ([400, 404, 409].includes(response.status)) {
						return { state: 'pending' as const };
					}

					return {
						state: 'failed' as const,
						message: `Report fetch failed: ${response.status}`
					};
				}

				const report = (await response.json()) as UnifiedReport;
				if (!report.meta.jobId || !report.version) {
					return {
						state: 'failed' as const,
						message: 'Report JSON did not match the expected aggregated schema.'
					};
				}

				return { state: 'ready' as const, report };
			} catch (error) {
				const message = error instanceof Error ? error.message : 'Failed to load report';
				return { state: 'failed' as const, message };
			}
		}
	};
}

export function createEventsPort(): ScanJobEventsPort {
	return {
		supportsStreaming() {
			return typeof EventSource !== 'undefined';
		},
		open(jobId, sink) {
			return createSSEStream<ScanResult, SSEUpdate>(
				jobId,
				{
					onStatus: sink.onStatus,
					onUpdate: sink.onUpdate,
					onDone: () => undefined,
					onError: (error) => {
						if (error.kind === 'transient') {
							sink.onTransport({ type: 'retrying' });
							return;
						}

						if (error.kind === 'parse') {
							sink.onTransport({ type: 'exhausted', reason: 'invalid-message' });
							return;
						}

						sink.onTransport({
							type: 'exhausted',
							reason: error.kind === 'closed' ? 'closed' : 'retries-exceeded'
						});
					}
				},
				{
					sourceName: 'scan-job-monitor',
					onOpen: () => {
						sink.onTransport({ type: 'connected' });
					}
				}
			);
		}
	};
}
