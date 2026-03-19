/**
 * Event Publisher
 *
 * NATS-based event publishing for scanner progress and completion events.
 */

import {
	type JetStreamClient,
	type NatsConnection,
	StringCodec,
	connect as connectNats,
} from "nats";

import type {
	MessagingConfig,
	PageScanResult,
	ScanEventPublisher,
	ScanResults,
	ScanTiming,
	ScannerLogger,
} from "./types";

import { createLogger } from "../utils/logger";
import { reportPath, resultsPath } from "./artifact-paths";

const DEFAULT_SUBJECTS = {
	pageCompleted: "scan.events.page.completed",
	scanCompleted: "scan.events.completed",
	scanFailed: "scan.events.failed",
};

export interface EventEnvelope<T = unknown> {
	event: string;
	job_id: string;
	request_id?: string;
	run_id?: string;
	timestamp: string;
	producer: string;
	payload: T;
}

export class NatsEventPublisher implements ScanEventPublisher {
	private connection: NatsConnection | null = null;
	private jetstream: JetStreamClient | null = null;
	private readonly codec = StringCodec();
	private readonly jobId: string;
	private readonly requestId?: string;
	private readonly runId?: string;
	private readonly scannerName: string;
	private readonly subjects: MessagingConfig["subjects"];
	private readonly logger: ScannerLogger;

	constructor(
		jobId: string,
		scannerName: string,
		subjects?: Partial<MessagingConfig["subjects"]>,
		logger?: ScannerLogger,
		correlation?: { requestId?: string; runId?: string },
	) {
		this.jobId = jobId;
		this.scannerName = scannerName;
		this.subjects = { ...DEFAULT_SUBJECTS, ...subjects };
		this.logger = logger ?? createLogger("EventPublisher");
		this.requestId = correlation?.requestId;
		this.runId = correlation?.runId;
	}

	async connect(url: string): Promise<void> {
		this.logger.info("Connecting to NATS", { url });
		this.connection = await connectNats({
			servers: url,
			name: `scanner-${this.scannerName}-${this.jobId}`,
			reconnect: true,
			maxReconnectAttempts: -1,
			reconnectTimeWait: 1000,
			pingInterval: 20_000,
			maxPingOut: 5,
		});
		this.jetstream = this.connection.jetstream();
		this.logger.info("Connected to NATS");
	}

	async publishPageCompleted(
		result: PageScanResult,
		index: number,
		total: number,
	): Promise<void> {
		await this.publish(this.subjects.pageCompleted, "scan.page.completed", {
			job_id: this.jobId,
			scanner_type: this.scannerName,
			page_id: result.pageId,
			page_index: index,
			total_pages: total,
		});
	}

	async publishScanCompleted(
		results: ScanResults,
		timing?: ScanTiming,
		artifacts?: {
			stageLogPath?: string;
			recipePath?: string;
			reportPath?: string;
		},
	): Promise<void> {
		await this.publish(this.subjects.scanCompleted, "scan.completed", {
			job_id: this.jobId,
			scanner_type: this.scannerName,
			results_path: resultsPath(this.jobId, this.scannerName),
			report_path:
				artifacts?.reportPath ?? reportPath(this.jobId, this.scannerName),
			stage_log_path: artifacts?.stageLogPath,
			recipe_path: artifacts?.recipePath,
			total_pages_scanned: results.pages.length,
			summary: {
				total_violations: results.summary.totalIssues,
				by_severity: results.summary.bySeverity,
				pages_scanned: results.summary.pagesScanned,
				critical_issues: results.summary.bySeverity.critical,
				serious_issues: results.summary.bySeverity.serious,
				moderate_issues: results.summary.bySeverity.moderate,
				minor_issues: results.summary.bySeverity.minor,
			},
			timing: timing
				? {
						total_ms: timing.totalMs,
						page_iteration_ms: timing.pageIterationMs,
						write_results_ms: timing.writeResultsMs,
						upload_artifacts_ms: timing.uploadArtifactsMs,
						publish_completed_ms: timing.publishCompletedMs,
						finalization_ms: timing.finalizationMs,
					}
				: undefined,
		});
	}

	async publishScanFailed(
		error: string,
		details?: string,
		artifacts?: { stageLogPath?: string; recipePath?: string },
	): Promise<void> {
		await this.publish(
			this.subjects.scanFailed,
			"scan.failed",
			{
				job_id: this.jobId,
				scanner_type: this.scannerName,
				error,
				error_details: details,
				stage_log_path: artifacts?.stageLogPath,
				recipe_path: artifacts?.recipePath,
			},
			true,
		);
	}

	async close(): Promise<void> {
		if (!this.connection) {
			return;
		}

		try {
			await this.connection.drain();
			this.logger.info("NATS connection drained");
		} catch (err) {
			this.logger.warn("Failed to drain NATS connection", {
				error: err instanceof Error ? err.message : String(err),
			});
			try {
				await this.connection.close();
			} catch {
				// Ignore close errors
			}
		} finally {
			this.connection = null;
			this.jetstream = null;
		}
	}

	private async publish(
		subject: string,
		event: string,
		payload: unknown,
		suppressErrors = false,
	): Promise<void> {
		if (!this.jetstream) {
			if (suppressErrors) {
				this.logger.warn("Cannot publish - not connected to NATS", { event });
				return;
			}
			throw new Error("Not connected to NATS");
		}

		const envelope: EventEnvelope = {
			event,
			job_id: this.jobId,
			request_id: this.requestId,
			run_id: this.runId,
			timestamp: new Date().toISOString(),
			producer: this.scannerName,
			payload,
		};

		try {
			await this.jetstream.publish(
				subject,
				this.codec.encode(JSON.stringify(envelope)),
			);
			this.logger.debug("Published event", { subject, event });
		} catch (err) {
			this.logger.error("Failed to publish event", {
				subject,
				event,
				error: err instanceof Error ? err.message : String(err),
			});
			if (!suppressErrors) {
				throw err;
			}
		}
	}
}

export class NoOpEventPublisher implements ScanEventPublisher {
	async publishPageCompleted(): Promise<void> {
		return Promise.resolve();
	}

	async publishScanCompleted(
		_results: ScanResults,
		_timing?: ScanTiming,
		_artifacts?: {
			stageLogPath?: string;
			recipePath?: string;
			reportPath?: string;
		},
	): Promise<void> {
		return Promise.resolve();
	}

	async publishScanFailed(
		_error?: string,
		_details?: string,
		_artifacts?: { stageLogPath?: string; recipePath?: string },
	): Promise<void> {
		return Promise.resolve();
	}

	async close(): Promise<void> {
		return Promise.resolve();
	}
}
