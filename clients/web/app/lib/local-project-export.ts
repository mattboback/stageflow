import type { LocalBaseline, LocalProject, LocalProjectConfiguration, LocalRun } from './projects';
import { isUnifiedReport, sanitizeProjectConfiguration } from './projects';

export const LOCAL_PROJECT_EXPORT_SCHEMA = 'stageflow/local-project@v1' as const;

const LOCAL_RUN_STATUSES = new Set(['submitted', 'complete', 'failed']);

export interface LocalProjectExport {
	schema: typeof LOCAL_PROJECT_EXPORT_SCHEMA;
	exportedAt: string;
	project: LocalProject;
	baseline: LocalBaseline | null;
	runs: LocalRun[];
}

export function buildLocalProjectExport(
	project: LocalProject,
	baseline: LocalBaseline | null,
	runs: LocalRun[]
): LocalProjectExport {
	return {
		schema: LOCAL_PROJECT_EXPORT_SCHEMA,
		exportedAt: new Date().toISOString(),
		project: {
			...project,
			configuration: sanitizeProjectConfiguration(project.configuration)
		},
		baseline,
		runs
	};
}

export function parseLocalProjectExport(value: unknown): LocalProjectExport | null {
	if (!value || typeof value !== 'object') return null;
	const candidate = value as Partial<LocalProjectExport>;
	if (candidate.schema !== LOCAL_PROJECT_EXPORT_SCHEMA) return null;
	if (!candidate.project || typeof candidate.project !== 'object') return null;
	if (typeof candidate.project.id !== 'string' || typeof candidate.project.name !== 'string') {
		return null;
	}
	if (!isProjectConfiguration(candidate.project.configuration)) return null;
	if (!Array.isArray(candidate.runs)) return null;

	const runs: LocalRun[] = [];
	for (const run of candidate.runs) {
		const parsedRun = parseLocalRun(run);
		if (!parsedRun) return null;
		runs.push(parsedRun);
	}

	const baseline = parseLocalBaseline(candidate.baseline);
	if (baseline === undefined) return null;

	return {
		schema: LOCAL_PROJECT_EXPORT_SCHEMA,
		exportedAt:
			typeof candidate.exportedAt === 'string' ? candidate.exportedAt : new Date().toISOString(),
		project: {
			...candidate.project,
			configuration: sanitizeProjectConfiguration(candidate.project.configuration)
		},
		baseline,
		runs
	};
}

function isProjectConfiguration(value: unknown): value is LocalProjectConfiguration {
	if (!value || typeof value !== 'object') return false;
	const candidate = value as Partial<LocalProjectConfiguration>;
	return Array.isArray(candidate.urls) && Array.isArray(candidate.scanners);
}

function parseLocalRun(value: unknown): LocalRun | null {
	if (!value || typeof value !== 'object') return null;
	const candidate = value as Partial<LocalRun>;
	if (typeof candidate.jobId !== 'string' || candidate.jobId.trim() === '') return null;
	if (typeof candidate.projectId !== 'string') return null;
	if (typeof candidate.configFingerprint !== 'string') return null;
	if (typeof candidate.status !== 'string' || !LOCAL_RUN_STATUSES.has(candidate.status)) {
		return null;
	}
	if (typeof candidate.createdAt !== 'string') return null;
	if (candidate.report !== undefined && !isUnifiedReport(candidate.report)) return null;
	return candidate as LocalRun;
}

function parseLocalBaseline(
	value: LocalProjectExport['baseline'] | undefined
): LocalBaseline | null | undefined {
	if (value == null) return null;
	if (typeof value !== 'object') return undefined;
	if (typeof value.projectId !== 'string' || typeof value.jobId !== 'string') return undefined;
	if (!isUnifiedReport(value.report)) return undefined;
	return value;
}
