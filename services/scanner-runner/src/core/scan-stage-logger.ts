import { createHash } from 'node:crypto';
import { writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';

import type { ScannerConfig } from './types';

import { ensureDir } from '../utils/fs';

const scanStageSchema = 'stageflow.stages.scan.v1';
const scanRecipeSchema = 'stageflow.recipes.scan.v1';

type ScanStageStatus = 'succeeded' | 'failed';

interface ScanStageEvent {
	ts: string;
	type: string;
	details?: Record<string, unknown>;
}

interface ScanStageMetrics {
	pages_total?: number;
	pages_scanned?: number;
	total_issues?: number;
}

interface ScanStageArtifacts {
	provenance_key?: string;
	results_key?: string;
}

interface ScanRecipe {
	schema: string;
	job_id: string;
	stage: 'scan';
	scanner_type: string;
	generated_at: string;
	input: {
		provenance_path: string;
		scan_urls?: boolean;
	};
	results_dir: string;
	environment: Record<string, unknown>;
}

interface ScanStageRef {
	bucket: string;
	path: string;
	sha256?: string;
}

interface ScanStageFailure {
	stage: string;
	message: string;
	details?: string;
}

interface ScanStageLog {
	schema: string;
	job_id: string;
	stage: 'scan';
	scanner_type: string;
	attempt: number;
	status: ScanStageStatus;
	started_at: string;
	completed_at: string;
	duration_ms: number;
	recipe_ref: ScanStageRef;
	metrics: ScanStageMetrics;
	events: ScanStageEvent[];
	artifacts: ScanStageArtifacts;
	failure?: ScanStageFailure;
}

export class ScanStageLogger {
	private readonly config: ScannerConfig;

	private readonly startedAt: Date;
	private readonly recipeObjectPath: string;
	private readonly stageLogObjectPath: string;
	private readonly recipeLocalPath: string;
	private readonly stageLogLocalPath: string;

	private recipeHash = '';
	private events: ScanStageEvent[] = [];
	private metrics: ScanStageMetrics = {};
	private artifacts: ScanStageArtifacts = {};

	constructor(config: ScannerConfig) {
		this.config = config;

		this.startedAt = new Date();

		this.recipeObjectPath = `${config.jobId}/recipes/scan.${config.scannerName}.json`;
		this.stageLogObjectPath = `${config.jobId}/stages/scan.${config.scannerName}.log.json`;

		this.recipeLocalPath = join(
			tmpdir(),
			'stageflow',
			config.jobId,
			config.scannerName,
			'recipes',
			'scan.json'
		);
		this.stageLogLocalPath = join(
			tmpdir(),
			'stageflow',
			config.jobId,
			config.scannerName,
			'stages',
			'scan.log.json'
		);
	}

	async start(): Promise<{ recipePath: string }> {
		await ensureDir(dirname(this.recipeLocalPath));
		await ensureDir(dirname(this.stageLogLocalPath));

		const recipe: ScanRecipe = {
			schema: scanRecipeSchema,
			job_id: this.config.jobId,
			stage: 'scan',
			scanner_type: this.config.scannerName,
			generated_at: new Date().toISOString(),
			input: {
				provenance_path: this.config.provenancePath,
				scan_urls: Boolean(process.env.SCAN_URLS)
			},
			results_dir: this.config.resultsDir,
			environment: {
				nats_url: this.config.messaging.url
			}
		};

		const json = JSON.stringify(recipe, null, 2);
		this.recipeHash = sha256Hex(json);
		await writeFile(this.recipeLocalPath, json, {
			encoding: 'utf8',
			mode: 0o600
		});

		this.events.push({
			ts: this.startedAt.toISOString(),
			type: 'stage_started'
		});

		return { recipePath: '' };
	}

	recordEvent(type: string, details?: Record<string, unknown>): void {
		this.events.push({
			ts: new Date().toISOString(),
			type,
			...(details !== undefined ? { details } : {})
		});
	}

	setMetrics(metrics: Partial<ScanStageMetrics>): void {
		this.metrics = { ...this.metrics, ...metrics };
	}

	setArtifacts(artifacts: Partial<ScanStageArtifacts>): void {
		this.artifacts = { ...this.artifacts, ...artifacts };
	}

	async finalizeSuccess(): Promise<{ stageLogPath: string }> {
		return this.finalize('succeeded');
	}

	async finalizeFailure(failure: ScanStageFailure): Promise<{ stageLogPath: string }> {
		return this.finalize('failed', failure);
	}

	private async finalize(
		status: ScanStageStatus,
		failure?: ScanStageFailure
	): Promise<{ stageLogPath: string }> {
		const completedAt = new Date();

		this.events = this.events.filter(
			(e) => e.type !== 'stage_completed' && e.type !== 'stage_failed'
		);

		const completionDetails = failure?.message ? { message: failure.message } : undefined;
		this.events.push({
			ts: completedAt.toISOString(),
			type: status === 'failed' ? 'stage_failed' : 'stage_completed',
			...(completionDetails !== undefined ? { details: completionDetails } : {})
		});

		const log: ScanStageLog = {
			schema: scanStageSchema,
			job_id: this.config.jobId,
			stage: 'scan',
			scanner_type: this.config.scannerName,
			attempt: 1,
			status,
			started_at: this.startedAt.toISOString(),
			completed_at: completedAt.toISOString(),
			duration_ms: completedAt.getTime() - this.startedAt.getTime(),
			recipe_ref: {
				bucket: '',
				path: this.recipeObjectPath,
				sha256: this.recipeHash
			},
			metrics: this.metrics,
			events: this.events,
			artifacts: this.artifacts,
			...(failure && status === 'failed' ? { failure } : {})
		};

		const json = JSON.stringify(log, null, 2);
		await writeFile(this.stageLogLocalPath, json, {
			encoding: 'utf8',
			mode: 0o600
		});

		return { stageLogPath: '' };
	}
}

function sha256Hex(text: string): string {
	return createHash('sha256').update(text).digest('hex');
}
