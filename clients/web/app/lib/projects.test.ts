import fs from 'node:fs';
import path from 'node:path';
import { describe, expect, it } from 'vitest';

import type { UnifiedReport } from './types/unified-report';
import {
	buildArchivedLocalRun,
	computeProjectDiff,
	fingerprintProjectConfiguration,
	sanitizeProjectConfiguration
} from './projects';

const fixture = JSON.parse(
	fs.readFileSync(
		path.resolve('../../libs/contracts/report/fixtures/unified-report.v2.all-scans.json'),
		'utf8'
	)
) as UnifiedReport;

describe('local run report archival', () => {
	it('backfills a completed run without replacing its completion timestamp', () => {
		const report = { ...fixture, meta: { ...fixture.meta, jobId: 'job-archive' } };
		const archived = buildArchivedLocalRun(
			{
				jobId: 'job-archive',
				projectId: 'project-1',
				configFingerprint: 'fingerprint',
				status: 'complete',
				createdAt: '2026-01-01T00:00:00.000Z',
				completedAt: '2026-01-01T00:01:00.000Z'
			},
			report,
			'job-archive',
			'2026-08-13T00:00:00.000Z'
		);

		expect(archived?.report).toBe(report);
		expect(archived?.completedAt).toBe('2026-01-01T00:01:00.000Z');
		expect(archived?.status).toBe('complete');
	});

	it('does not rewrite a completed run that already has a report', () => {
		const report = { ...fixture, meta: { ...fixture.meta, jobId: 'job-archive' } };
		expect(
			buildArchivedLocalRun(
				{
					jobId: 'job-archive',
					projectId: 'project-1',
					configFingerprint: 'fingerprint',
					status: 'complete',
					createdAt: '2026-01-01T00:00:00.000Z',
					report
				},
				report,
				'job-archive'
			)
		).toBeNull();
	});
});

describe('local project configuration', () => {
	it('removes credentials and execution-only input values before persistence', () => {
		const safe = sanitizeProjectConfiguration({
			urls: ['https://b.example', 'https://a.example'],
			browser: 'chromium',
			highlightStyle: 'solid',
			scanners: [
				{
					id: 'example-scanner',
					enabled: true,
					config: {
						goal: { objective: 'Complete checkout', inputValues: { email: 'secret@example.com' } },
						password: 'do-not-store',
						vision: { model: 'example/model' }
					}
				}
			]
		});

		expect(JSON.stringify(safe)).not.toContain('secret@example.com');
		expect(JSON.stringify(safe)).not.toContain('do-not-store');
		expect(safe.scanners[0]?.config).toEqual({
			goal: { objective: 'Complete checkout' },
			vision: { model: 'example/model' }
		});
	});

	it('redacts execution-only values duplicated in otherwise retained fields', () => {
		const canary = 'duplicate-execution-secret-canary';
		const safe = sanitizeProjectConfiguration({
			urls: ['https://example.com'],
			browser: 'chromium',
			highlightStyle: 'solid',
			scanners: [
				{
					id: 'example-scanner',
					enabled: true,
					config: {
						goal: {
							objective: `Paste ${canary} into the form`,
							inputValues: { token: canary },
							successCriteria: [
								{ type: 'text-contains', value: canary },
								{ type: 'url-contains', value: `/complete?token=${canary}` }
							]
						}
					}
				}
			]
		});

		expect(JSON.stringify(safe)).not.toContain(canary);
		expect(safe.scanners[0]?.config).toEqual({
			goal: {
				objective: 'Paste [redacted] into the form',
				successCriteria: [
					{ type: 'text-contains', value: '[redacted]' },
					{ type: 'url-contains', value: '/complete?token=[redacted]' }
				]
			}
		});
	});

	it('fingerprints equivalent configurations deterministically', async () => {
		const left = {
			urls: ['https://b.example', 'https://a.example'],
			browser: 'chromium' as const,
			highlightStyle: 'solid' as const,
			scanners: [
				{ id: 'seo', enabled: true, config: { z: 2, a: 1 } },
				{ id: 'axe', enabled: true }
			]
		};
		const right = {
			...left,
			urls: [...left.urls].reverse(),
			scanners: [...left.scanners].reverse()
		};

		expect(await fingerprintProjectConfiguration(left)).toBe(
			await fingerprintProjectConfiguration(right)
		);
		expect(
			await fingerprintProjectConfiguration({
				...right,
				scanners: [...right.scanners, { id: 'new-catalog-scanner', enabled: false }]
			})
		).toBe(await fingerprintProjectConfiguration(left));
	});
});

function loadGoldenReport(name: string): UnifiedReport {
	const fixture = JSON.parse(
		fs.readFileSync(
			path.resolve(process.cwd(), `../../qa/fixtures/scan-diff-golden/${name}`),
			'utf8'
		)
	) as { report: UnifiedReport };
	return fixture.report;
}

describe('computeProjectDiff', () => {
	it('matches the committed Go golden regression semantics', () => {
		const baseline = loadGoldenReport('golden-baseline-report.json');
		const current = loadGoldenReport('golden-regression-report.json');
		const expected = JSON.parse(
			fs.readFileSync(
				path.resolve(
					process.cwd(),
					'../../qa/fixtures/scan-diff-golden/golden-regression-diff.json'
				),
				'utf8'
			)
		) as {
			delta: { newIssues: number; fixedIssues: number; unchangedIssues: number };
			new: { id: string }[];
			fixed: { id: string }[];
		};

		const result = computeProjectDiff('baseline-job', baseline, 'current-job', current);

		expect(result.schema).toBe('stageflow/diff@v1');
		expect(result.delta.newIssues).toBe(expected.delta.newIssues);
		expect(result.delta.fixedIssues).toBe(expected.delta.fixedIssues);
		expect(result.delta.unchangedIssues).toBe(expected.delta.unchangedIssues);
		expect(result.new.map((issue) => issue.id)).toEqual(expected.new.map((issue) => issue.id));
		expect(result.fixed.map((issue) => issue.id)).toEqual(expected.fixed.map((issue) => issue.id));
	});

	it('sorts issue IDs and computes score changes like the Go implementation', () => {
		const base = loadGoldenReport('golden-regression-report.json');
		const first = base.issues[0];
		if (!first) {
			throw new Error('golden-regression-report.json must contain at least one issue');
		}
		const baseline = {
			...base,
			summary: { ...base.summary, score: 92, totalIssues: 2 },
			issues: [
				{
					...first,
					id: 'z-fixed',
					pageId: 'page-v2-test',
					occurrences: [{ elementId: 'occ-v2-z-fixed', selector: '#z-fixed' }]
				},
				{
					...first,
					id: 'shared',
					pageId: 'page-v2-test',
					occurrences: [{ elementId: 'occ-v2-shared', selector: '#shared' }]
				}
			]
		};
		const current = {
			...base,
			summary: { ...base.summary, score: 88, totalIssues: 3 },
			issues: [
				{
					...first,
					id: 'shared',
					pageId: 'page-v2-test',
					occurrences: [{ elementId: 'occ-v2-shared', selector: '#shared' }]
				},
				{
					...first,
					id: 'z-new',
					pageId: 'page-v2-test',
					occurrences: [{ elementId: 'occ-v2-z-new', selector: '#z-new' }]
				},
				{
					...first,
					id: 'a-new',
					pageId: 'page-v2-test',
					occurrences: [{ elementId: 'occ-v2-a-new', selector: '#a-new' }]
				}
			]
		};

		const result = computeProjectDiff('base', baseline, 'current', current);
		expect(result.delta).toEqual({
			scoreDelta: -4,
			newIssues: 2,
			fixedIssues: 1,
			unchangedIssues: 1
		});
		expect(result.new.map((issue) => issue.id)).toEqual(['a-new', 'z-new']);
		expect(result.fixed.map((issue) => issue.id)).toEqual(['z-fixed']);
	});

	// Mirrors TestComputeDiff_SkipsScannersThatDidNotFinish in libs/go/diff.
	it('skips scanners that did not finish in either report, like the Go implementation', () => {
		const base = loadGoldenReport('golden-regression-report.json');
		const first = base.issues[0];
		if (!first) {
			throw new Error('golden-regression-report.json must contain at least one issue');
		}
		const issue = (id: string, scanner: string) => ({
			...first,
			id,
			scanner,
			pageId: 'page-v2-test',
			occurrences: [{ elementId: `occ-v2-${id}`, selector: `#${id}` }]
		});
		const baseline = {
			...base,
			summary: { ...base.summary, score: 80, totalIssues: 3 },
			scanners: [
				{ id: 'axe', status: 'success' as const },
				{ id: 'seo', status: 'success' as const },
				{ id: 'lighthouse', status: 'skipped' as const }
			],
			issues: [issue('axe-1', 'axe'), issue('axe-2', 'axe'), issue('seo-1', 'seo')]
		};
		const current = {
			...base,
			summary: { ...base.summary, score: 95, totalIssues: 2 },
			scanners: [
				{ id: 'axe', status: 'failed' as const },
				{ id: 'seo', status: 'success' as const },
				{ id: 'lighthouse', status: 'success' as const }
			],
			issues: [issue('seo-2', 'seo'), issue('lh-1', 'lighthouse')]
		};

		const result = computeProjectDiff('baseline-job', baseline, 'current-job', current);

		expect(result.skippedScanners).toEqual(['axe', 'lighthouse']);
		expect(result.fixed.map((entry) => entry.id)).toEqual(['seo-1']);
		expect(result.new.map((entry) => entry.id)).toEqual(['seo-2']);
		expect(result.delta.scoreDelta).toBeUndefined();
		expect(result.baseline.score).toBeUndefined();
		expect(result.current.score).toBeUndefined();
	});

	it('compares nothing from the golden pair once axe failed in the current report', () => {
		const baseline = loadGoldenReport('golden-baseline-report.json');
		const golden = loadGoldenReport('golden-regression-report.json');
		const current = {
			...golden,
			scanners: golden.scanners.map((scanner) =>
				scanner.id === 'axe' ? { ...scanner, status: 'failed' as const } : scanner
			)
		};

		const result = computeProjectDiff('baseline-job', baseline, 'current-job', current);

		expect(result.skippedScanners).toEqual(['axe']);
		expect(result.delta).toEqual({ newIssues: 0, fixedIssues: 0, unchangedIssues: 0 });
	});

	it('leaves skippedScanners out when every scanner finished', () => {
		const baseline = loadGoldenReport('golden-baseline-report.json');
		const current = loadGoldenReport('golden-regression-report.json');

		expect(computeProjectDiff('b', baseline, 'c', current)).not.toHaveProperty('skippedScanners');
	});
});

describe('complete occurrence comparisons', () => {
	it('counts one new occurrence when a sixth failing image is added', () => {
		const base = loadGoldenReport('golden-regression-report.json');
		const first = base.issues[0]!;
		const occurrences = Array.from({ length: 5 }, (_, index) => ({
			elementId: `occ-v2-${index}`,
			selector: `#image-${index}`
		}));
		const baseline = {
			...base,
			issues: [{ ...first, id: 'image-alt', pageId: 'page-v2-root', occurrences }]
		};
		const current = {
			...base,
			issues: [
				{
					...first,
					id: 'image-alt',
					pageId: 'page-v2-root',
					occurrences: [...occurrences, { elementId: 'occ-v2-5', selector: '#image-5' }]
				}
			]
		};
		const result = computeProjectDiff('b', baseline, 'c', current);
		expect(result.delta.newIssues).toBe(1);
		expect(result.delta.fixedIssues).toBe(0);
		expect(result.delta.unchangedIssues).toBe(5);
		expect(result.new[0]?.occurrences?.[0]?.elementId).toBe('occ-v2-5');
	});
	it('excludes partial page coverage on both sides without apparent fixes', () => {
		const base = loadGoldenReport('golden-regression-report.json');
		const scanner = base.issues[0]!.scanner;
		const current = {
			...base,
			scanners: base.scanners.map((value) =>
				value.id === scanner ? { ...value, status: 'partial' as const } : value
			),
			issues: base.issues.filter((issue) => issue.scanner !== scanner)
		};
		const result = computeProjectDiff('b', base, 'c', current);
		expect(result.skippedScanners).toContain(scanner);
		expect(result.fixed).toHaveLength(0);
		expect(result.delta.scoreDelta).toBeUndefined();
	});
	it('requires a fresh baseline when stable identity replaces positional IDs', () => {
		const base = loadGoldenReport('golden-regression-report.json');
		const legacy = {
			...base,
			pages: base.pages.map((page, index) => ({ ...page, id: `url-${index + 1}` })),
			issues: base.issues.map((issue, index) => ({
				...issue,
				pageId: 'url-1',
				occurrences: (issue.occurrences ?? []).map((occurrence) => ({
					...occurrence,
					elementId: `legacy-${index}`
				}))
			}))
		};
		const result = computeProjectDiff('b', legacy, 'c', base);
		expect(result.incompatibleIdentity).toBe(true);
		expect(result.new).toHaveLength(0);
		expect(result.fixed).toHaveLength(0);
	});
});
