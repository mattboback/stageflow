import { describe, expect, it } from 'vitest';

import { loadAllScansFixture } from '../test/load-fixture';
import {
	LOCAL_PROJECT_EXPORT_SCHEMA,
	buildLocalProjectExport,
	parseLocalProjectExport
} from './local-project-export';
import { LOCAL_PROJECT_SCHEMA_VERSION, type LocalProject, type LocalRun } from './projects';

const project: LocalProject = {
	id: 'project-1',
	schemaVersion: LOCAL_PROJECT_SCHEMA_VERSION,
	name: 'Example',
	configuration: {
		urls: ['https://example.com'],
		scanners: [{ id: 'axe', enabled: true }],
		browser: 'chromium',
		highlightStyle: 'solid'
	},
	createdAt: '2026-08-13T00:00:00.000Z',
	updatedAt: '2026-08-13T00:00:00.000Z'
};

describe('local project export', () => {
	it('round-trips a project with a stored report', () => {
		const report = loadAllScansFixture();
		const run: LocalRun = {
			jobId: report.meta.jobId,
			projectId: project.id,
			configFingerprint: 'abc',
			status: 'complete',
			createdAt: '2026-08-13T00:00:00.000Z',
			report
		};
		const exported = buildLocalProjectExport(project, null, [run]);
		expect(exported.schema).toBe(LOCAL_PROJECT_EXPORT_SCHEMA);
		expect(parseLocalProjectExport(exported)?.runs[0]?.report?.meta.jobId).toBe(report.meta.jobId);
	});

	it('rejects a foreign payload', () => {
		expect(parseLocalProjectExport({ schema: 'nope' })).toBeNull();
	});

	it('rejects a schema-matching payload with no configuration', () => {
		expect(
			parseLocalProjectExport({
				schema: LOCAL_PROJECT_EXPORT_SCHEMA,
				project: { id: project.id, name: project.name },
				runs: []
			})
		).toBeNull();
	});

	it('rejects a run that is missing a jobId', () => {
		expect(
			parseLocalProjectExport({
				schema: LOCAL_PROJECT_EXPORT_SCHEMA,
				project,
				runs: [
					{
						projectId: project.id,
						configFingerprint: 'abc',
						status: 'complete',
						createdAt: '2026-08-13T00:00:00.000Z'
					}
				]
			})
		).toBeNull();
	});
});

describe('project reviews compatibility', () => {
	it('exports review notes independently of the raw report', () => {
		const reviews = { job: { issue: { note: 'Check keyboard navigation', at: '2026-09-30' } } };
		const exported = buildLocalProjectExport(project, null, [], reviews);
		expect(exported.schema).toBe('stageflow/local-project@v2');
		expect(parseLocalProjectExport(exported)?.reviews).toEqual(reviews);
	});
	it('accepts v1 project exports without reviews', () => {
		const exported = {
			...buildLocalProjectExport(project, null, []),
			schema: 'stageflow/local-project@v1',
			reviews: undefined
		};
		expect(parseLocalProjectExport(exported)?.reviews).toEqual({});
	});
});

it('rejects targeted reports promoted through an imported baseline', () => {
	const report = loadAllScansFixture();
	report.meta.scanContext = {
		expectedPages: [],
		identityScope: 'path',
		scannerConfigFingerprints: {},
		purpose: 'verification',
		authenticated: false,
		inputType: 'urls'
	};
	const baseline = {
		projectId: project.id,
		jobId: report.meta.jobId,
		configFingerprint: 'fingerprint',
		createdAt: 'today',
		report
	};
	expect(parseLocalProjectExport(buildLocalProjectExport(project, baseline, []))).toBeNull();
});

it('rejects imported run evidence that can request remote or active content', () => {
	const report = loadAllScansFixture();
	const run = {
		projectId: project.id,
		jobId: report.meta.jobId,
		configFingerprint: 'fingerprint',
		status: 'complete' as const,
		createdAt: 'today',
		report,
		screenshots: [
			{
				artifact_id: 'overview',
				kind: 'page_overview' as const,
				scanner_id: 'axe',
				page_id: report.pages[0]?.id ?? 'page',
				url: 'https://attacker.invalid/tracking'
			}
		]
	};
	expect(parseLocalProjectExport(buildLocalProjectExport(project, null, [run]))).toBeNull();
	const screenshot = run.screenshots[0];
	if (!screenshot) throw new Error('Expected screenshot fixture');
	screenshot.url = 'data:image/svg+xml;base64,PHN2Zz4=';
	expect(parseLocalProjectExport(buildLocalProjectExport(project, null, [run]))).toBeNull();
});

it('preserves a targeted request while stripping credential-shaped extra fields', () => {
	const report = loadAllScansFixture();
	const issue = report.issues[0];
	if (!issue) throw new Error('Expected a finding fixture');
	const verification = {
		jobId: report.meta.jobId,
		sourceJobId: 'original-run',
		createdAt: 'today',
		issue,
		sourceScannerVersion: '1.0',
		expectedFinalUrl: issue.pageUrl,
		sourceContext: {
			expectedPages: [issue.pageId],
			identityScope: 'path' as const,
			scannerConfigFingerprints: { [issue.scanner]: 'scanner-fingerprint' },
			pageConfigFingerprints: { [issue.pageId]: 'page-fingerprint' },
			purpose: 'full' as const,
			authenticated: true,
			inputType: 'urls' as const,
			scannerConfigurations: { [issue.scanner]: { token: 'do-not-export' } }
		}
	};
	const run: LocalRun = {
		jobId: report.meta.jobId,
		projectId: project.id,
		configFingerprint: 'config',
		status: 'complete',
		createdAt: 'today',
		report,
		verification
	};
	const exported = buildLocalProjectExport(project, null, [run]);
	expect(JSON.stringify(exported)).not.toContain('do-not-export');
	const parsed = parseLocalProjectExport(exported);
	expect(parsed?.runs[0]?.verification?.sourceJobId).toBe('original-run');
	expect(parsed?.runs[0]?.verification?.sourceContext.authenticated).toBe(true);
	if (!exported.runs[0]?.verification) throw new Error('Expected serialized request');
	exported.runs[0].verification.jobId = 'different-job';
	expect(parseLocalProjectExport(exported)).toBeNull();
});

it('rejects invalid nested archived run and baseline reports in both export versions', () => {
	const report = loadAllScansFixture();
	const run: LocalRun = {
		jobId: report.meta.jobId,
		projectId: project.id,
		configFingerprint: 'abc',
		status: 'complete',
		createdAt: 'today',
		report
	};
	for (const schema of ['stageflow/local-project@v1', LOCAL_PROJECT_EXPORT_SCHEMA]) {
		const exported = { ...buildLocalProjectExport(project, null, [run]), schema };
		expect(parseLocalProjectExport(exported)?.runs[0]?.report).toEqual(report);
		expect(
			parseLocalProjectExport({
				...exported,
				runs: [{ ...run, report: { ...report, issues: [null] } }]
			})
		).toBeNull();
		expect(
			parseLocalProjectExport({
				...exported,
				baseline: {
					projectId: project.id,
					jobId: report.meta.jobId,
					configFingerprint: 'abc',
					createdAt: 'today',
					report: { ...report, pages: [null] }
				}
			})
		).toBeNull();
		expect(
			parseLocalProjectExport({
				...exported,
				project: { ...project, configuration: { urls: [123], scanners: [null] } }
			})
		).toBeNull();
	}
});
