import { createScanContext } from './scan-context';

import { join } from 'node:path';

import type {
	IssueSeverity,
	LighthouseCategorySummary,
	PageScanResult,
	Provenance,
	ScanContext,
	ScanEventPublisher,
	ScanResults,
	ScanSummary,
	ScannerConfig,
	ScannerLifecycleHooks,
	ScannerLogger,
	ScannerMetadata
} from './types';

import { createLogger } from '../utils/logger';
import { ensureDir, writeJson } from '../utils/fs';
import { resultsPath as artifactResultsPath } from './artifact-paths';
import { BrowserManager } from './browser-manager';
import { ScannerFiles } from './scanner-files';
import { enforceOutputBudget } from './output-budget';
import { createLiteralRedactor } from './page-iterator-redaction';
import { PageIterator } from './page-iterator';
import { ScanStageLogger } from './scan-stage-logger';
import { buildScanCompletedTiming, withPublishCompletedTiming } from './scan-lifecycle';
import { WebServerFormatter } from './web-server-formatter';

export type { ScannerMetadata } from './types';

export abstract class ScannerBase {
	protected config!: ScannerConfig;
	protected browserManager: BrowserManager | null = null;
	protected pageIterator!: PageIterator;
	protected eventPublisher!: ScanEventPublisher;
	protected logger: ScannerLogger;
	protected hooks: ScannerLifecycleHooks;
	protected scanStartedAt!: string;
	private runStartedAt!: number;
	private redactError: (value: string) => string = (value) => value;
	private providedLogger: boolean;
	protected scanStageLogger: ScanStageLogger | null = null;
	private scanStageLogPath = '';
	private scanRecipePath = '';

	abstract readonly metadata: ScannerMetadata;
	abstract scanPage(context: ScanContext): Promise<PageScanResult>;

	constructor(hooks?: ScannerLifecycleHooks, logger?: ScannerLogger) {
		this.hooks = hooks ?? {};
		this.providedLogger = Boolean(logger);
		this.logger = logger ?? createLogger('Scanner');
	}

	async run(config: ScannerConfig): Promise<ScanResults> {
		this.ensureLogger();
		this.config = config;
		this.eventPublisher = new ScannerFiles(config.resultsDir, this.logger);
		this.scanStartedAt = new Date().toISOString();
		this.runStartedAt = Date.now();

		this.logger.info(`Starting ${this.metadata.name} scanner`, {
			jobId: config.jobId,
			version: this.metadata.version
		});

		try {
			await this.initialize();
			await this.hooks.onScanStart?.(config);

			const provenance = await this.pageIterator.loadProvenance();
			this.redactError = createLiteralRedactor(provenance);
			this.validateProvenance(provenance);
			this.scanStageLogger?.recordEvent('provenance_loaded', {
				pages: provenance.pages.filter((p) => !p.skip).length
			});
			this.scanStageLogger?.setMetrics({
				pages_total: provenance.pages.filter((p) => !p.skip).length
			});

			const { pageResults, pageIterationMs } = await this.iteratePages(provenance);

			const results = this.buildResults(provenance, pageResults);
			this.scanStageLogger?.setMetrics({
				pages_scanned: results.pages.length,
				total_issues: results.summary.totalIssues
			});

			const { durationMs: writeResultsMs } = await this.runPhase('writeResults', () =>
				this.writeResults(provenance, results)
			);

			if (this.scanStageLogger) {
				const artifacts = {
					results_key: artifactResultsPath(this.config.jobId, this.metadata.name)
				};

				this.scanStageLogger.setArtifacts(artifacts);

				const { stageLogPath } = await this.scanStageLogger.finalizeSuccess();
				this.scanStageLogPath = stageLogPath;
			}

			await enforceOutputBudget(this.config.resultsDir);
			await this.publishScanCompleted(results, {
				pageIterationMs,
				writeResultsMs
			});

			await this.hooks.onScanEnd?.(results);

			this.logger.info(`${this.metadata.name} scanner completed`, {
				jobId: config.jobId,
				pagesScanned: results.pages.length,
				totalIssues: results.summary.totalIssues,
				durationMs: results.durationMs
			});

			return results;
		} catch (error) {
			await this.handleRunError(error);
			throw error;
		} finally {
			await this.cleanup();
		}
	}

	private ensureLogger(): void {
		if (!this.providedLogger) {
			this.logger = createLogger(this.metadata.name);
		}
	}

	private async iteratePages(
		provenance: Provenance
	): Promise<{ pageResults: PageScanResult[]; pageIterationMs: number }> {
		const hooks = this.hooks;
		const stageLogger = this.scanStageLogger;
		const pageIteratorCallbacks = {
			onPageComplete: async (result: PageScanResult, index: number, total: number) => {
				await enforceOutputBudget(this.config.resultsDir);
				await this.eventPublisher.publishPageCompleted(result, index + 1, total);
				await hooks.onPageEnd?.(result);
			},
			onAuditEvent: (event: { type: string; details?: Record<string, unknown> }) => {
				stageLogger?.recordEvent(event.type, event.details);
			},
			...(hooks.onPageStart
				? {
						onPageStart: async (pageEntry: Provenance['pages'][number]) => {
							await hooks.onPageStart?.(pageEntry);
						}
					}
				: {}),
			...(hooks.onError
				? {
						onPageError: async (error: Error, pageEntry: Provenance['pages'][number]) => {
							await hooks.onError?.(error, { pageEntry });
						}
					}
				: {})
		};
		const { value: pageResults, durationMs: pageIterationMs } = await this.runPhase(
			'pageIteration',
			() =>
				this.pageIterator.iteratePages(
					provenance,
					(context) => this.scanPage(context),
					pageIteratorCallbacks
				)
		);

		return { pageResults, pageIterationMs };
	}

	private async publishScanCompleted(
		results: ScanResults,
		timing: {
			pageIterationMs: number;
			writeResultsMs: number;
		}
	): Promise<{ publishCompletedMs: number }> {
		// The five timing components are disjoint phases that must sum to <= totalMs
		// (enforced by the scan.completed contract validator). publishCompletedMs is 0
		// here because we are still inside the publish phase, and finalizationMs is the
		// unaccounted remainder (provenance load/upload, stage-log finalize, teardown)
		// rather than a roll-up of the write/upload phases — double-counting those made
		// the component sum exceed totalMs and the orchestrator rejected the event.
		const timingForEvent = buildScanCompletedTiming({
			totalMs: Date.now() - this.runStartedAt,
			pageIterationMs: timing.pageIterationMs,
			writeResultsMs: timing.writeResultsMs
		});

		const { durationMs: publishCompletedMs } = await this.runPhase('publishScanCompleted', () =>
			this.eventPublisher.publishScanCompleted(results, timingForEvent, {
				...(this.scanStageLogPath ? { stageLogPath: this.scanStageLogPath } : {}),
				...(this.scanRecipePath ? { recipePath: this.scanRecipePath } : {})
			})
		);

		this.logPipelineTiming(withPublishCompletedTiming(timingForEvent, publishCompletedMs));

		return { publishCompletedMs };
	}

	private async handleRunError(error: unknown): Promise<void> {
		const errorMessage = this.redactError(error instanceof Error ? error.message : String(error));
		this.logger.error('Scanner failed', { error: errorMessage });

		if (this.scanStageLogger) {
			try {
				this.scanStageLogger.recordEvent('scan_failed', {
					error: errorMessage
				});
				const failure = {
					stage: 'scan',
					message: errorMessage,
					...(error instanceof Error && error.stack !== undefined
						? { details: this.redactError(error.stack) }
						: {})
				};
				const { stageLogPath } = await this.scanStageLogger.finalizeFailure(failure);
				this.scanStageLogPath = stageLogPath;
			} catch (stageErr) {
				this.logger.warn('Failed to finalize scan stage log', {
					error: stageErr instanceof Error ? stageErr.message : String(stageErr)
				});
			}
		}

		await this.eventPublisher.publishScanFailed(
			errorMessage,
			error instanceof Error && error.stack ? this.redactError(error.stack) : undefined,
			{
				...(this.scanStageLogPath ? { stageLogPath: this.scanStageLogPath } : {}),
				...(this.scanRecipePath ? { recipePath: this.scanRecipePath } : {})
			}
		);

		if (this.hooks.onError) {
			await this.hooks.onError(error instanceof Error ? error : new Error(errorMessage));
		}
	}

	protected async initialize(): Promise<void> {
		await ensureDir(this.config.resultsDir);

		const browserManager = new BrowserManager(this.config.browser, this.logger);
		this.browserManager = browserManager;

		this.pageIterator = new PageIterator(browserManager, this.config, this.logger);

		this.scanStageLogger = new ScanStageLogger(this.config);
		const { recipePath } = await this.scanStageLogger.start();
		this.scanRecipePath = recipePath;
	}

	protected validateProvenance(provenance: Provenance): void {
		if (!provenance.version) {
			throw new Error('Provenance version is required');
		}
		if (!provenance.job_id) {
			throw new Error('Provenance job_id is required');
		}
		if (!Array.isArray(provenance.pages) || provenance.pages.length === 0) {
			throw new Error('Provenance does not contain any pages to scan');
		}
		// base_url is only required if pages don't have full URLs
		const allPagesHaveUrls = provenance.pages.every(
			(page) => page.url && /^https?:\/\//i.test(page.url)
		);
		if (!provenance.base_url && !allPagesHaveUrls) {
			throw new Error("Provenance base_url is required when pages don't have full URLs");
		}
	}

	protected buildResults(provenance: Provenance, pageResults: PageScanResult[]): ScanResults {
		const completedAt = new Date().toISOString();
		const startTime = new Date(this.scanStartedAt).getTime();
		const endTime = new Date(completedAt).getTime();

		const summary = this.buildSummary(pageResults);

		return {
			jobId: this.config.jobId,
			scanner: this.metadata.name,
			version: this.metadata.version,
			totalPages: provenance.pages.length,
			context: createScanContext(this.config, provenance),
			pages: pageResults,
			summary,
			startedAt: this.scanStartedAt,
			completedAt,
			durationMs: endTime - startTime
		};
	}

	protected buildSummary(pageResults: PageScanResult[]): ScanSummary {
		const allIssues = pageResults.flatMap((p) => p.issues);
		const successfulPages = pageResults.filter((p) => p.success);

		const bySeverity: Record<IssueSeverity, number> = {
			critical: 0,
			serious: 0,
			moderate: 0,
			minor: 0,
			info: 0
		};

		const byCategory: Record<string, number> = {};

		for (const issue of allIssues) {
			bySeverity[issue.severity] += 1;
			byCategory[issue.category] = (byCategory[issue.category] ?? 0) + 1;
		}

		const durations = successfulPages.map((p) => p.durationMs);
		const avgDurationMs =
			durations.length > 0 ? durations.reduce((a, b) => a + b, 0) / durations.length : 0;

		const lighthouseCategories = this.extractLighthouseCategories(pageResults);

		return {
			totalIssues: allIssues.length,
			bySeverity,
			byCategory,
			pagesScanned: pageResults.length,
			pagesFailed: pageResults.filter((p) => !p.success).length,
			pagesWithIssues: pageResults.filter((p) => p.issues.length > 0).length,
			avgDurationMs,
			...(lighthouseCategories !== undefined ? { lighthouseCategories } : {})
		};
	}

	protected extractLighthouseCategories(
		pageResults: PageScanResult[]
	): LighthouseCategorySummary[] | undefined {
		interface LHCategory {
			id: string;
			title: string;
			score: number | null;
		}

		interface CategoryAggregate {
			title: string;
			sum: number;
			count: number;
		}

		const lhCategories: Record<string, CategoryAggregate | undefined> = {};

		for (const page of pageResults) {
			const lhr = page.rawResults as {
				categories?: Record<string, LHCategory>;
			} | null;

			if (!lhr?.categories) {
				continue;
			}

			for (const cat of Object.values(lhr.categories)) {
				if (cat.score == null) {
					continue;
				}

				const existing = lhCategories[cat.id];
				if (existing) {
					existing.sum += cat.score;
					existing.count += 1;
				} else {
					lhCategories[cat.id] = { title: cat.title, sum: cat.score, count: 1 };
				}
			}
		}

		const entries = Object.entries(lhCategories).filter(
			(entry): entry is [string, CategoryAggregate] => Boolean(entry[1])
		);
		if (entries.length === 0) {
			return undefined;
		}

		return entries.map(([id, { title, sum, count }]) => ({
			id: id as LighthouseCategorySummary['id'],
			title,
			avgScore: Math.round((sum / count) * 100) / 100
		}));
	}

	protected async writeResults(provenance: Provenance, results: ScanResults): Promise<void> {
		const webServerFormat = new WebServerFormatter().format(provenance, results, this.metadata);

		const resultsPath = join(this.config.resultsDir, 'results.json');
		await writeJson(resultsPath, webServerFormat, { spaces: 2 });
		this.logger.info('Wrote results file', { path: resultsPath });
	}

	protected async cleanup(): Promise<void> {
		if (this.browserManager) {
			await this.browserManager.close();
			this.browserManager = null;
		}
		await this.eventPublisher.close();
	}

	private logPipelineTiming(phases: {
		totalMs: number;
		pageIterationMs: number;
		writeResultsMs: number;
		publishCompletedMs: number;
		finalizationMs: number;
	}): void {
		this.logger.info('Scan timing summary', {
			jobId: this.config.jobId,
			totalMs: phases.totalMs,
			pageIterationMs: phases.pageIterationMs,
			finalizationMs: phases.finalizationMs,
			writeResultsMs: phases.writeResultsMs,
			publishCompletedMs: phases.publishCompletedMs
		});
	}

	private async runPhase<T>(
		phase: string,
		fn: () => Promise<T>
	): Promise<{ value: T; durationMs: number }> {
		const startedAt = Date.now();
		const value = await fn();
		const durationMs = Date.now() - startedAt;

		this.logger.info('Phase complete', {
			phase,
			durationMs,
			jobId: this.config.jobId
		});

		return { value, durationMs };
	}
}
