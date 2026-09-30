import { parsePortableAudit, PORTABLE_AUDIT_SCHEMA } from './portable-audit';
import { parseVerificationRequest } from './verification';
import { normalizeReviewVerdicts, type ReviewVerdictsByJob } from './report/review-verdict';
import type { LocalBaseline, LocalProject, LocalProjectConfiguration, LocalRun } from './projects';
import { isUnifiedReport, sanitizeProjectConfiguration } from './projects';

export const LOCAL_PROJECT_EXPORT_SCHEMA = 'stageflow/local-project@v2' as const;

const LOCAL_RUN_STATUSES = new Set(['submitted', 'complete', 'failed']);

export interface LocalProjectExport {
	schema: typeof LOCAL_PROJECT_EXPORT_SCHEMA;
	exportedAt: string;
	project: LocalProject;
	baseline: LocalBaseline | null;
	runs: LocalRun[];
	reviews: ReviewVerdictsByJob;
}

export function buildLocalProjectExport(
	project: LocalProject,
	baseline: LocalBaseline | null,
	runs: LocalRun[],
	reviews: ReviewVerdictsByJob = {}
): LocalProjectExport {
	return {
		schema: LOCAL_PROJECT_EXPORT_SCHEMA,
		exportedAt: new Date().toISOString(),
		project: {
			...project,
			configuration: sanitizeProjectConfiguration(project.configuration)
		},
		baseline,
		runs: runs.map((run) => {
			const { verification, ...rest } = run;
			const safe = parseVerificationRequest(verification);
			return { ...rest, ...(safe?.jobId === run.jobId ? { verification: safe } : {}) };
		}),
		reviews: normalizeReviewVerdicts(reviews)
	};
}

export function parseLocalProjectExport(value: unknown): LocalProjectExport | null {
	if (!value || typeof value !== 'object') return null;
	const candidate = value as Omit<Partial<LocalProjectExport>, 'schema'> & { schema?: string };
	if (
		candidate.schema !== LOCAL_PROJECT_EXPORT_SCHEMA &&
		candidate.schema !== 'stageflow/local-project@v1'
	)
		return null;
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
		runs: runs.map((run) => {
			const { verification, ...rest } = run;
			const safe = parseVerificationRequest(verification);
			return { ...rest, ...(safe?.jobId === run.jobId ? { verification: safe } : {}) };
		}),
		reviews: normalizeReviewVerdicts(candidate.reviews)
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
	if (
		candidate.report !== undefined &&
		(!isUnifiedReport(candidate.report) || candidate.report.meta?.jobId !== candidate.jobId)
	)
		return null;
	if (candidate.screenshots !== undefined) {
		if (
			!candidate.report ||
			!parsePortableAudit({
				schema: PORTABLE_AUDIT_SCHEMA,
				jobId: candidate.jobId,
				createdAt: candidate.createdAt,
				report: candidate.report,
				screenshots: candidate.screenshots,
				reviews: {},
				missingArtifacts: []
			})
		)
			return null;
	}
	const { verification, ...rest } = candidate;
	const safe = parseVerificationRequest(verification);
	if (verification !== undefined && (!safe || safe.jobId !== candidate.jobId)) return null;
	return { ...rest, ...(safe ? { verification: safe } : {}) } as LocalRun;
}

function parseLocalBaseline(
	value: LocalProjectExport['baseline'] | undefined
): LocalBaseline | null | undefined {
	if (value == null) return null;
	if (typeof value !== 'object') return undefined;
	if (typeof value.projectId !== 'string' || typeof value.jobId !== 'string') return undefined;
	if (
		!isUnifiedReport(value.report) ||
		value.report.meta?.jobId !== value.jobId ||
		value.report.meta.scanContext?.purpose === 'verification'
	)
		return undefined;
	if (typeof value.configFingerprint !== 'string' || typeof value.createdAt !== 'string')
		return undefined;
	return value;
}
