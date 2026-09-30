import fs from 'node:fs';
import path from 'node:path';
import { describe, expect, it } from 'vitest';
import type { UnifiedReport } from './types/unified-report';
import { computeProjectDiff, fingerprintProjectConfiguration } from './projects';

const fixture = JSON.parse(
	fs.readFileSync(
		path.resolve('../../libs/contracts/report/fixtures/unified-report.v2.all-scans.json'),
		'utf8'
	)
) as UnifiedReport;
function coveredReport(): UnifiedReport {
	const report = structuredClone(fixture);
	report.meta.scanContext = {
		expectedPages: ['page-v2-a', 'page-v2-b'],
		identityScope: 'path',
		inputType: 'urls',
		purpose: 'full',
		authenticated: false,
		scannerConfigFingerprints: { axe: 'axe-settings' },
		pageConfigFingerprints: { 'page-v2-a': 'page-settings', 'page-v2-b': 'page-settings' }
	};
	report.pages = ['a', 'b'].map((id) => ({ ...fixture.pages[0]!, id: `page-v2-${id}` }));
	report.issues = ['a', 'b'].map((id) => ({
		...fixture.issues[0]!,
		id,
		scanner: 'axe',
		pageId: `page-v2-${id}`,
		occurrences: []
	}));
	report.scanners = [
		{
			...fixture.scanners[0]!,
			id: 'axe',
			status: 'success',
			toolVersion: '1',
			pages: ['a', 'b'].map((id) => ({
				pageId: `page-v2-${id}`,
				status: 'success',
				finalUrl: `https://production.example/${id}`
			}))
		}
	];
	report.summary.score = 70;
	return report;
}

describe('product comparison coverage', () => {
	it.each([
		['zip', 'path', 'http://127.0.0.1:30001/a', 'http://127.0.0.1:30099/a', 2],
		['urls', 'path', 'https://production.example/a', 'https://staging.example/a', 2],
		['urls', 'origin', 'https://production.example/a', 'https://staging.example/a', 1],
		['urls', 'path', 'https://production.example/a', 'https://production.example/other', 1],
		['urls', 'path', 'https://production.example/a?q=1', 'https://production.example/a?q=2', 1]
	] as const)(
		'matches %s %s final URLs %s → %s',
		(inputType, identityScope, left, right, fixed) => {
			const before = coveredReport(),
				after = coveredReport();
			before.meta.scanContext = { ...before.meta.scanContext!, inputType, identityScope };
			after.meta.scanContext = { ...after.meta.scanContext!, inputType, identityScope };
			before.scanners[0]!.pages![0]!.finalUrl = left;
			after.scanners[0]!.pages![0]!.finalUrl = right;
			after.issues = [];
			const result = computeProjectDiff('before', before, 'after', after);
			expect(result.delta.fixedIssues).toBe(fixed);
			if (fixed === 1) {
				expect(result.notComparedPages).toContain('page-v2-a');
				expect(result.delta.scoreDelta).toBeUndefined();
			}
		}
	);
	it('compares successful pages from a partial scanner without fixing failed pages', () => {
		const before = coveredReport(),
			after = coveredReport();
		after.issues = [];
		after.scanners[0]!.status = 'partial';
		after.scanners[0]!.pages![1]!.status = 'failed';
		const result = computeProjectDiff('before', before, 'after', after);
		expect(result.fixed.map((issue) => issue.id)).toEqual(['a']);
		expect(result.notComparedPages).toContain('page-v2-b');
		expect(result.delta.scoreDelta).toBeUndefined();
	});
	it('never fixes an expected page whose coverage was omitted', () => {
		const before = coveredReport(),
			after = coveredReport();
		after.issues = [];
		after.pages = after.pages.slice(0, 1);
		after.scanners[0]!.pages = after.scanners[0]!.pages!.slice(0, 1);
		const result = computeProjectDiff('before', before, 'after', after);
		expect(result.fixed.map((issue) => issue.id)).toEqual(['a']);
		expect(result.notComparedPages).toContain('page-v2-b');
		expect(result.delta.scoreDelta).toBeUndefined();
	});
	it('profile toggles preserve raw findings and suppress score comparison', () => {
		const before = coveredReport(),
			after = coveredReport();
		after.meta.profile = 'staging';
		const result = computeProjectDiff('before', before, 'after', after);
		expect(result.delta).toEqual({ newIssues: 0, fixedIssues: 0, unchangedIssues: 2 });
	});
	it('omitted production profile has the explicit production fingerprint', async () => {
		const config = {
			urls: ['https://example.com'],
			scanners: [{ id: 'axe', enabled: true }],
			browser: 'chromium' as const,
			highlightStyle: 'solid' as const
		};
		expect(await fingerprintProjectConfiguration(config)).toBe(
			await fingerprintProjectConfiguration({ ...config, profile: 'production' })
		);
		expect(await fingerprintProjectConfiguration({ ...config, profile: 'staging' })).not.toBe(
			await fingerprintProjectConfiguration(config)
		);
	});
});

it('does not count a failed rule evaluation as a fix on an otherwise successful page', () => {
	const before = coveredReport(),
		after = coveredReport();
	before.issues[0]!.ruleId = 'image-alt';
	after.issues = [];
	before.scanners[0]!.pages![0]!.rules = [{ ruleId: 'image-alt', outcome: 'fail' }];
	after.scanners[0]!.pages![0]!.rules = [{ ruleId: 'image-alt', outcome: 'error' }];
	const diff = computeProjectDiff('before', before, 'after', after);
	expect(diff.fixed.map((issue) => issue.id)).toEqual(['b']);
	expect(diff.notComparedPages).toContain('page-v2-a');
	expect(diff.delta.scoreDelta).toBeUndefined();
});
