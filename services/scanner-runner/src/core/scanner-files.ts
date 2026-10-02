import { appendFile, mkdir, open, rename, rm, stat } from 'node:fs/promises';
import { join } from 'node:path';
import type {
	PageScanResult,
	ScanEventPublisher,
	ScanResults,
	ScanTiming,
	ScannerLogger
} from './types';
import { createLogger } from '../utils/logger';

export const MAX_PROGRESS_BYTES = 8 * 1024 * 1024;
export const MAX_OUTCOME_BYTES = 1024 * 1024;
const MAX_ERROR_BYTES = 4096;
export interface ScannerOutcome {
	status: 'success' | 'partial' | 'failed';
	error: string;
	pages_scanned: number;
	summary: {
		total_violations: number;
		by_severity: Record<string, number>;
		pages_scanned: number;
		critical_issues: number;
		serious_issues: number;
		moderate_issues: number;
		minor_issues: number;
	};
	timing: {
		total_ms: number;
		page_iteration_ms: number;
		write_results_ms: number;
		publish_completed_ms: number;
		finalization_ms: number;
	};
}
/** Progress is disposable; a terminal outcome must reach disk before exit. */
export class ScannerFiles implements ScanEventPublisher {
	private progressBytes = 0;
	private progressInitialized = false;
	private pagesScanned = 0;
	private readonly startedAt = Date.now();
	private progressQueue: Promise<void> = Promise.resolve();
	private readonly logger: ScannerLogger;
	constructor(
		private readonly outputDir: string,
		logger?: ScannerLogger
	) {
		this.logger = logger ?? createLogger('ScannerFiles');
	}
	publishPageCompleted(result: PageScanResult, index: number, total: number): Promise<void> {
		this.pagesScanned += 1;
		const line =
			JSON.stringify({
				timestamp: new Date().toISOString(),
				page_id: result.pageId,
				pages_scanned: this.pagesScanned,
				total_pages: total,
				page_index: index
			}) + '\n';
		this.progressQueue = this.progressQueue.then(async () => {
			if (
				Buffer.byteLength(line) > 16 * 1024 ||
				this.progressBytes + Buffer.byteLength(line) > MAX_PROGRESS_BYTES
			) {
				return;
			}
			try {
				const progressPath = join(this.outputDir, 'progress.jsonl');
				if (!this.progressInitialized) {
					try {
						this.progressBytes = (await stat(progressPath)).size;
					} catch (error) {
						if ((error as NodeJS.ErrnoException).code !== 'ENOENT') {
							throw error;
						}
					}
					this.progressInitialized = true;
				}
				if (this.progressBytes + Buffer.byteLength(line) > MAX_PROGRESS_BYTES) {
					return;
				}
				await appendFile(progressPath, line, { mode: 0o600 });
				this.progressBytes += Buffer.byteLength(line);
			} catch (error) {
				this.logger.warn('Progress could not be recorded', {
					error: error instanceof Error ? error.message : String(error)
				});
			}
		});
		return Promise.resolve();
	}
	async publishScanCompleted(results: ScanResults, timing?: ScanTiming): Promise<void> {
		const severity = results.summary.bySeverity;
		await this.writeOutcome({
			status: results.summary.pagesFailed > 0 ? 'partial' : 'success',
			error:
				results.summary.pagesFailed > 0 ? `${results.summary.pagesFailed} page audits failed` : '',
			pages_scanned: results.pages.length,
			summary: {
				total_violations: results.summary.totalIssues,
				by_severity: severity,
				pages_scanned: results.pages.length,
				critical_issues: severity.critical,
				serious_issues: severity.serious,
				moderate_issues: severity.moderate,
				minor_issues: severity.minor
			},
			timing: this.timing(timing)
		});
	}
	async publishScanFailed(error: string): Promise<void> {
		await this.writeOutcome({
			status: 'failed',
			error: Buffer.from(error).subarray(0, MAX_ERROR_BYTES).toString('utf8'),
			pages_scanned: this.pagesScanned,
			summary: {
				total_violations: 0,
				by_severity: { critical: 0, serious: 0, moderate: 0, minor: 0, info: 0 },
				pages_scanned: this.pagesScanned,
				critical_issues: 0,
				serious_issues: 0,
				moderate_issues: 0,
				minor_issues: 0
			},
			timing: this.timing()
		});
	}
	private timing(timing?: ScanTiming): ScannerOutcome['timing'] {
		return {
			total_ms: timing?.totalMs ?? Date.now() - this.startedAt,
			page_iteration_ms: timing?.pageIterationMs ?? 0,
			write_results_ms: timing?.writeResultsMs ?? 0,
			publish_completed_ms: 0,
			finalization_ms: timing?.finalizationMs ?? 0
		};
	}
	private async writeOutcome(outcome: ScannerOutcome): Promise<void> {
		const bytes = Buffer.from(JSON.stringify(outcome));
		if (bytes.byteLength > MAX_OUTCOME_BYTES) {
			throw new Error('Scanner outcome exceeds the artifact limit');
		}
		await mkdir(this.outputDir, { recursive: true });
		const temp = join(this.outputDir, `.outcome-${process.pid}.tmp`);
		try {
			const file = await open(temp, 'wx', 0o600);
			try {
				await file.writeFile(bytes);
				await file.sync();
			} finally {
				await file.close();
			}
			await rename(temp, join(this.outputDir, 'outcome.json'));
			const directory = await open(this.outputDir, 'r');
			try {
				await directory.sync();
			} finally {
				await directory.close();
			}
		} finally {
			await rm(temp, { force: true });
		}
	}
	async close(): Promise<void> {
		await this.progressQueue;
	}
}
