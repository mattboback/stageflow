import type { ScanResult, ScanStatus, ScreenshotArtifact } from '../../types/scan';
import type { UnifiedReport } from '../../types/unified-report';

import type { SSEUpdate } from '../scan-status/types';

export type TerminalStatus = Extract<ScanStatus, 'complete' | 'failed'>;
export type TransportState = 'idle' | 'streaming' | 'polling';
export type StreamTransportReason = 'closed' | 'retries-exceeded' | 'invalid-message';

export type StreamTransportEvent =
	| { type: 'connected' }
	| { type: 'retrying' }
	| { type: 'exhausted'; reason: StreamTransportReason };

export interface ScanJobStatusPort {
	fetch(jobId: string): Promise<ScanResult>;
}

export interface ScanJobEventsPort {
	supportsStreaming(): boolean;
	open(
		jobId: string,
		sink: {
			onStatus(snapshot: ScanResult): void;
			onUpdate(event: SSEUpdate): void;
			onTransport(event: StreamTransportEvent): void;
		}
	): { close(): void };
}

export interface ScanJobReportPort {
	fetch(
		jobId: string
	): Promise<
		| { state: 'ready'; report: UnifiedReport }
		| { state: 'pending' }
		| { state: 'failed'; message: string }
	>;
}

export interface ScanHistoryPort {
	markTerminal(jobId: string, status: TerminalStatus): void | Promise<void>;
}

export interface SchedulerPort {
	every(ms: number, fn: () => void): () => void;
	after(ms: number, fn: () => void): () => void;
}

export interface StatusMonitorSnapshot {
	kind: 'status';
	jobId: string;
	status: ScanStatus;
	job: ScanResult | null;
	logs: string[];
	elapsedSeconds: number;
	transport: TransportState;
	error: string | null;
}

export interface ReportMonitorSnapshot {
	kind: 'report';
	jobId: string;
	status: ScanStatus;
	job: ScanResult | null;
	report: UnifiedReport | null;
	screenshots: ScreenshotArtifact[];
	logs: string[];
	transport: TransportState;
	error: string | null;
}

export interface StatusMonitorOptions {
	kind: 'status';
	jobId: string;
	logLimit?: number;
	lifecycleMessages?: Partial<Record<'EXTRACTING' | 'SCANNING' | 'COMPLETING', string>>;
}

export interface ReportRetryOptions {
	initialDelayMs: number;
	maxDelayMs: number;
	maxAttempts: number;
}

export interface ReportMonitorOptions {
	kind: 'report';
	jobId: string;
	logLimit?: number;
	pollIntervalMs?: number;
	reportRetry?: Partial<ReportRetryOptions>;
	lifecycleMessages?: Partial<Record<'EXTRACTING' | 'SCANNING' | 'COMPLETING', string>>;
}

export type CreateMonitorOptions = StatusMonitorOptions | ReportMonitorOptions;

export interface SharedDependencies {
	statusPort: ScanJobStatusPort;
	eventsPort: ScanJobEventsPort;
	scheduler: SchedulerPort;
}

export interface StatusMonitorDependencies extends SharedDependencies {
	historyPort?: ScanHistoryPort;
}

export interface ReportMonitorDependencies extends SharedDependencies {
	reportPort: ScanJobReportPort;
}

export type StatusMonitorDependencyOverrides = Partial<StatusMonitorDependencies>;
export type ReportMonitorDependencyOverrides = Partial<ReportMonitorDependencies>;

export type MonitorSubscription<TSnapshot> = (snapshot: TSnapshot) => void;

export interface ScanJobMonitor<TSnapshot> {
	start(): void;
	stop(): void;
	refreshStatus(): Promise<void>;
	getSnapshot(): TSnapshot;
	subscribe(listener: MonitorSubscription<TSnapshot>): () => void;
	refreshArtifacts(): Promise<void>;
}
