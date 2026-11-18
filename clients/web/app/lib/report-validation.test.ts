import { describe, expect, it } from 'vitest';
import { loadAllScansFixture } from '../test/load-fixture';
import { isProjectConfiguration, isUnifiedReport } from './report-validation';

describe('report structural validation', () => {
	it('accepts a legacy 2.x report without recent optional metadata and allows extra fields', () => {
		const value = loadAllScansFixture();
		delete value.meta.scanContext;
		delete value.assessment;
		delete value.manualChecks;
		value.version = '2.0.0';
		expect(isUnifiedReport({ ...value, futureField: {} })).toBe(true);
	});

	it.each([
		['meta', null],
		['pages', [null]],
		['scanners', [{ id: 'axe', status: 'unknown' }]],
		['issues', [{ id: 'issue' }]],
		['summary', { totalIssues: 1 }],
		['errors', [{ scope: 'global', code: 'error', message: 'bad', retryable: 'yes' }]],
		['artifacts', [{ id: 'artifact', type: 'image', path: 12 }]]
	])('rejects malformed %s', (field, value) => {
		expect(isUnifiedReport({ ...loadAllScansFixture(), [field]: value })).toBe(false);
	});

	it('rejects malformed nested optional issue, page, and coverage data', () => {
		const report = loadAllScansFixture();
		const issue = report.issues[0]!;
		const page = report.pages[0]!;
		const scanner = report.scanners[0]!;
		expect(
			isUnifiedReport({
				...report,
				issues: [{ ...issue, occurrences: [{ boundingBox: { x: 1 } }] }]
			})
		).toBe(false);
		expect(
			isUnifiedReport({
				...report,
				pages: [
					{
						...page,
						pageOverview: {
							screenshotFilename: 'page.png',
							pageWidth: 100,
							pageHeight: 100,
							elements: [null]
						}
					}
				]
			})
		).toBe(false);
		expect(
			isUnifiedReport({
				...report,
				scanners: [
					{
						...scanner,
						pages: [
							{
								pageId: 'page',
								status: 'success',
								rules: [{ ruleId: 'rule', outcome: 'pass', subjectIds: [null] }]
							}
						]
					}
				]
			})
		).toBe(false);
		expect(
			isUnifiedReport({ ...report, meta: { ...report.meta, scanContext: { expectedPages: [] } } })
		).toBe(false);
	});
});

describe('configuration structural validation', () => {
	it('accepts legacy engine/highlight settings for downstream normalization', () => {
		expect(
			isProjectConfiguration({
				urls: ['https://example.com'],
				scanners: [{ id: 'axe', enabled: true, config: { password: 'secret' } }],
				browser: 'firefox',
				highlightStyle: 'outline'
			})
		).toBe(true);
	});
	it.each([
		{ urls: [null], scanners: [] },
		{ urls: [], scanners: [null] },
		{ urls: [], scanners: [{ id: 1, enabled: true }] },
		{ urls: [], scanners: [{ id: 'axe', enabled: 'yes' }] },
		{ urls: [], scanners: [{ id: 'axe', enabled: true, config: [] }] },
		{ urls: [], scanners: [{ id: 'axe', enabled: true, config: null }] }
	])('rejects malformed configuration %#', (value) => {
		expect(isProjectConfiguration(value)).toBe(false);
	});
});
