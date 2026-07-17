import fs from 'node:fs';
import path from 'node:path';
import { describe, expect, it } from 'vitest';

import type { UnifiedReport } from './types/unified-report';
import {
	computeProjectDiff,
	fingerprintProjectConfiguration,
	sanitizeProjectConfiguration
} from './projects';

describe('local project configuration', () => {
	it('removes credentials and AI input values before persistence', () => {
		const safe = sanitizeProjectConfiguration({
			urls: ['https://b.example', 'https://a.example'],
			browser: 'chromium',
			highlightStyle: 'solid',
			scanners: [
				{
					id: 'ai-navigator',
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
		expect(safe.scanners[0].config).toEqual({
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
					id: 'ai-navigator',
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
		expect(safe.scanners[0].config).toEqual({
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
		fs.readFileSync(path.resolve(process.cwd(), `../../qa/fixtures/project-golden/${name}`), 'utf8')
	) as { report: UnifiedReport };
	return fixture.report;
}

describe('computeProjectDiff', () => {
	it('matches the committed Go golden regression semantics', () => {
		const baseline = loadGoldenReport('golden-baseline-report.json');
		const current = loadGoldenReport('golden-regression-report.json');
		const expected = JSON.parse(
			fs.readFileSync(
				path.resolve(process.cwd(), '../../qa/fixtures/project-golden/golden-regression-diff.json'),
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
		const baseline = {
			...base,
			summary: { ...base.summary, score: 92, totalIssues: 2 },
			issues: [{ ...first, id: 'z-fixed' }, { ...first, id: 'shared' }]
		};
		const current = {
			...base,
			summary: { ...base.summary, score: 88, totalIssues: 3 },
			issues: [
				{ ...first, id: 'shared' },
				{ ...first, id: 'z-new' },
				{ ...first, id: 'a-new' }
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
});
