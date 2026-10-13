import { isUnifiedReport } from './report-validation';
import { fingerprintProjectConfiguration, type LocalProject, type LocalRun } from './projects';
import type { UnifiedReport } from './types/unified-report';

const DEFAULT_SCANNERS = ['axe', 'lighthouse', 'link-checker', 'security-headers', 'seo'];

function record(value: unknown): Record<string, unknown> | null {
	return value !== null && typeof value === 'object' && !Array.isArray(value)
		? (value as Record<string, unknown>)
		: null;
}

function stable(value: unknown): unknown {
	if (Array.isArray(value)) return value.map(stable);
	const object = record(value);
	return object
		? Object.fromEntries(
				Object.keys(object)
					.sort()
					.map((key) => [key, stable(object[key])])
			)
		: value;
}

function same(left: unknown, right: unknown): boolean {
	return JSON.stringify(stable(left)) === JSON.stringify(stable(right));
}

function urls(values: string[]): string[] {
	return [
		...new Set(
			values.map((value) => {
				const url = new URL(value);
				if (!['http:', 'https:'].includes(url.protocol) || url.username || url.password)
					throw new Error('Project reports must use HTTP URLs without credentials.');
				url.hash = '';
				return url.href;
			})
		)
	].sort();
}

/** Import only complete, unfiltered envelopes; never follow their remote links. */
export function parseCliReport(value: unknown): UnifiedReport {
	const envelope = record(value);
	const filters = record(envelope?.filters);
	const job = record(envelope?.job);
	if (envelope?.schema !== 'stageflow-cli/report@v1' || !isUnifiedReport(envelope.report))
		throw new Error('Choose a StageFlow CLI JSON report (stageflow-cli/report@v1).');
	const report = envelope.report;
	if (!job || job.id !== report.meta.jobId || !report.meta.jobId.trim())
		throw new Error('The report job identity is missing or inconsistent.');
	if (
		!filters ||
		filters.truncated !== false ||
		filters.issues_returned !== report.issues.length ||
		filters.issues_total !== report.issues.length ||
		['severities', 'categories'].some(
			(key) =>
				filters[key] !== undefined && (!Array.isArray(filters[key]) || filters[key].length > 0)
		)
	)
		throw new Error(
			'Import a complete, unfiltered report. Use check or scan --format json --max-issues 0.'
		);
	return report;
}

export async function buildImportedCliRun(
	project: LocalProject,
	report: UnifiedReport
): Promise<LocalRun> {
	const config = project.configuration;
	const context = report.meta.scanContext;
	if (
		!context ||
		context.purpose !== 'full' ||
		context.authenticated ||
		context.inputType !== 'urls'
	)
		throw new Error('Import a complete public-page scan with recorded scan settings.');
	if (
		context.viewport?.width !== 1280 ||
		context.viewport.height !== 720 ||
		report.pages.some((page) => context.verificationEligiblePages?.[page.id] !== true)
	)
		throw new Error('This report uses page settings that the saved project cannot replay.');
	if (
		(report.meta.profile ?? 'production') !== (config.profile ?? 'production') ||
		!same(urls(config.urls), urls(report.pages.map((page) => page.url))) ||
		!same([...context.expectedPages].sort(), report.pages.map((page) => page.id).sort())
	)
		throw new Error('The report pages or assessment profile do not match this project.');
	const enabled = config.scanners.filter((scanner) => scanner.enabled);
	const expected = enabled.length ? enabled.map((scanner) => scanner.id).sort() : DEFAULT_SCANNERS;
	if (
		!same(expected, report.scanners.map((scanner) => scanner.id).sort()) ||
		expected.some(
			(id) =>
				!context.scannerConfigFingerprints[id] ||
				!same(
					enabled.find((scanner) => scanner.id === id)?.config ?? {},
					context.scannerConfigurations?.[id]
				)
		)
	)
		throw new Error('The report scanners or scanner settings do not match this project.');
	const scannedAt = report.meta.scannedAt;
	const createdAt =
		scannedAt && Number.isFinite(Date.parse(scannedAt)) ? scannedAt : new Date().toISOString();
	return {
		projectId: project.id,
		jobId: report.meta.jobId,
		configFingerprint: await fingerprintProjectConfiguration(config),
		status: 'complete',
		createdAt,
		completedAt: report.meta.completedAt ?? createdAt,
		totalIssues: report.summary.totalIssues,
		...(report.summary.score == null ? {} : { score: report.summary.score }),
		report,
		archived: true,
		screenshots: []
	};
}
