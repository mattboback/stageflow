import { existsSync, readFileSync } from 'node:fs';
import { execFileSync } from 'node:child_process';
import path from 'node:path';
import { describe, expect, it } from 'vitest';

import { buildOccurrenceModeReport, splitManualChecks } from '../report';
import { computeProjectDiff, hasStableReportIdentity, isUnifiedReport } from '../projects';
import { getPageOverviewUrl } from '../report/screenshots';
import { DEMO_SUMMARY } from './demo-summary';
import { DEMO_CROP } from './demo-crop';
import DEMO_TERMINAL from './demo-terminal.json';
import type { ScreenshotArtifact } from '../types/scan';
import type { UnifiedReport } from '../types/unified-report';

/*
 * Holds `just demo-fixture` to the shape /demo needs.
 *
 * The failure this exists to prevent is silent. VisualReviewPanel filters out
 * any overlay element whose issueId does not resolve to a real issue -- no
 * error, no warning -- so a fixture that lost its element↔issue mapping renders
 * as a screenshot with no markers, which looks like a design choice rather than
 * a bug. The committed contract fixture has no pageOverview at all, which is
 * exactly that empty state; if this ever regenerates into the same condition,
 * the most substantial feature in the product would demo as a blank picture.
 *
 * Runs at the end of build-demo-fixture.sh, so a bad regeneration is caught
 * before it is committed rather than after it is deployed.
 */

// Vitest runs with cwd at clients/web, matching app/test/load-fixture.ts.
const DEMO_DIR = path.resolve(process.cwd(), 'public/demo');

function readJson<T>(name: string): T {
	const file = path.join(DEMO_DIR, name);
	if (!existsSync(file)) {
		throw new Error(`${name} is missing — run 'just demo-fixture'`);
	}
	return JSON.parse(readFileSync(file, 'utf8')) as T;
}

const report = readJson<UnifiedReport>('report.json');
const baseline = readJson<UnifiedReport>('baseline.json');
const comparison = computeProjectDiff('demo-baseline', baseline, 'demo', report);
const screenshots = readJson<ScreenshotArtifact[]>('screenshots.json');

describe('the committed /demo fixture', () => {
	it('is a report the app would accept from the API', () => {
		expect(isUnifiedReport(report)).toBe(true);
		expect(isUnifiedReport(baseline)).toBe(true);
	});

	it('validates both real reports against the shared schema and integrity checks', () => {
		const validator = path.resolve(process.cwd(), '../../libs/contracts/report/schema/validate.js');
		for (const name of ['baseline.json', 'report.json']) {
			expect(() =>
				execFileSync('node', [validator, path.join(DEMO_DIR, name)], { stdio: 'pipe' })
			).not.toThrow();
		}
	});

	it('runs all five supported scanners to completion in both versions', () => {
		for (const fixture of [baseline, report]) {
			expect(fixture.scanners.map((scanner) => scanner.id).sort()).toEqual([
				'axe',
				'lighthouse',
				'link-checker',
				'security-headers',
				'seo'
			]);
			expect(fixture.scanners.every((scanner) => scanner.status === 'success')).toBe(true);
			expect(hasStableReportIdentity(fixture)).toBe(true);
		}
	});

	it('shows real regressions and fixes without excluded scanners', () => {
		expect(comparison.delta.newIssues).toBeGreaterThan(0);
		expect(comparison.delta.fixedIssues).toBeGreaterThan(0);
		expect(comparison.skippedScanners ?? []).toEqual([]);
		expect(comparison.incompatibleIdentity).toBeUndefined();
	});

	it('lets successful axe scans own overlapping accessibility rules in both real reports', () => {
		const duplicateRules = new Set([
			'seo-images-missing-alt',
			'seo-missing-title',
			'seo-missing-language',
			'seo-heading-hierarchy',
			'link-checker-no-text-links'
		]);
		for (const fixture of [baseline, report]) {
			if (
				!fixture.scanners.some((scanner) => scanner.id === 'axe' && scanner.status === 'success')
			) {
				continue;
			}
			expect(
				fixture.issues
					.filter((issue) => duplicateRules.has(issue.ruleId))
					.map((issue) => issue.ruleId),
				`${fixture.meta.jobId}: overlapping scanner rules survived a successful axe scan`
			).toEqual([]);
		}
	});

	it('is not addressed as a live job', () => {
		// A real job id here would invite someone to paste it into
		// /scan/<id>/report and get a 404, and would key review verdicts on a
		// job that no longer exists.
		expect(report.meta.jobId).toBe('demo');
		expect(baseline.meta.jobId).toBe('demo-baseline');
	});

	it('has findings to show', () => {
		expect(report.issues.length).toBeGreaterThan(0);
		expect(report.pages.length).toBeGreaterThan(1);
	});

	it('exercises more than one scanner', () => {
		const scanners = new Set(report.issues.map((issue) => issue.scanner));
		expect(scanners.size).toBeGreaterThan(1);
	});

	it('gives every page a screenshot that exists on disk', () => {
		for (const page of report.pages) {
			const url = getPageOverviewUrl(screenshots, page.id);
			expect(url, `page ${page.id} has no page_overview screenshot`).toBeTruthy();
			expect(url).toMatch(/^\/demo\/.+\.webp$/);

			const file = path.join(DEMO_DIR, path.basename(url ?? ''));
			expect(existsSync(file), `${url} is referenced but not committed`).toBe(true);
		}
	});

	it('keeps the overlay geometry it needs to draw markers', () => {
		for (const page of report.pages) {
			const overview = page.pageOverview;
			expect(overview, `page ${page.id} has no pageOverview`).toBeTruthy();
			// VisualReviewPanel builds viewBox="0 0 pageWidth pageHeight". Zero or
			// missing dimensions collapse the SVG and nothing renders.
			expect(overview?.pageWidth).toBeGreaterThan(0);
			expect(overview?.pageHeight).toBeGreaterThan(0);
		}
	});

	/*
	 * Overlay IDs must remain valid after occurrence expansion.
	 *
	 * buildOccurrenceModeReport rewrites element.issueId, falling back to a
	 * `ruleId|selector` index when the direct mapping misses. A generator that
	 * blanks `selector` kills the fallback, and any element whose nodeIndex
	 * exceeds the occurrence count keeps a stale id. So the assertion has to
	 * run POST-remap: checking the raw fixture would pass while the rendered
	 * page shows nothing.
	 */
	it('resolves every overlay marker to a real issue after occurrence expansion', () => {
		const displayed = buildOccurrenceModeReport(report);
		const issueIds = new Set(displayed.issues.map((issue) => issue.id));

		let markers = 0;
		for (const page of displayed.pages) {
			for (const element of page.pageOverview?.elements ?? []) {
				markers += 1;
				expect(
					issueIds.has(element.issueId),
					`page ${page.id} marker points at ${element.issueId}, which no issue has`
				).toBe(true);
			}
		}

		// At least one marker somewhere, or the visual review panel is a
		// screenshot viewer. Not "every page has one": a genuinely clean page is
		// a legitimate outcome of scanning a site that has been fixed, and this
		// fixture is a real scan, not a staged one.
		expect(markers, 'no page has a single overlay marker').toBeGreaterThan(0);
	});

	it('carries no dead object-store references', () => {
		// These are keys into a bucket, so they are useless statically -- and they
		// embed the real job id that meta.jobId was rewritten to hide.
		const raw = JSON.stringify([baseline, report]);
		expect(raw).not.toContain('resultsPath');
		expect(raw).not.toContain('reportPath');
	});

	it('matches the figures the home page prints', () => {
		// DEMO_SUMMARY is duplicated on purpose: the home page will not pull 90 KB
		// of JSON in to print four numbers. This is what stops the duplicate from
		// silently describing an older scan.
		// Through the same transforms ReportView applies, so the home page and
		// /demo agree on what counts as a finding.
		const displayed = buildOccurrenceModeReport(splitManualChecks(report).report);
		expect(DEMO_SUMMARY.pages).toBe(report.pages.length);
		expect(DEMO_SUMMARY.scannersRun).toBe(report.scanners.length);
		expect(DEMO_SUMMARY.totalIssues).toBe(displayed.issues.length);
		expect(DEMO_SUMMARY.serious).toBe(
			displayed.issues.filter((issue) => issue.severity === 'serious').length
		);
		expect(DEMO_SUMMARY.newIssues).toBe(comparison.delta.newIssues);
		expect(DEMO_SUMMARY.fixedIssues).toBe(comparison.delta.fixedIssues);
		expect(DEMO_SUMMARY.unchangedIssues).toBe(comparison.delta.unchangedIssues);
		expect(DEMO_SUMMARY.baselineScore).toBe(baseline.summary.score ?? null);
		expect(DEMO_SUMMARY.currentScore).toBe(report.summary.score ?? null);
		expect(DEMO_SUMMARY.scoreDelta).toBe(comparison.delta.scoreDelta ?? null);
		// demo_terminal_test.go checks the transcript against the CLI itself; this
		// checks it describes the same comparison as /demo.
		expect(DEMO_TERMINAL.command).toBe(
			'stageflow diff baseline.json current.json --fail-on-new=any'
		);
		expect(DEMO_TERMINAL.exitCode).toBe(comparison.delta.newIssues > 0 ? 1 : 0);
		expect(DEMO_TERMINAL.output).toContain(`New issues: ${comparison.delta.newIssues}\n`);
		expect(DEMO_TERMINAL.output).toContain(`Fixed issues: ${comparison.delta.fixedIssues}\n`);
		expect(DEMO_TERMINAL.output).toContain(`Unchanged: ${comparison.delta.unchangedIssues}\n`);
		for (const issue of comparison.new) expect(DEMO_TERMINAL.output).toContain(issue.title);
		expect(DEMO_SUMMARY.moderate).toBe(
			displayed.issues.filter((issue) => issue.severity === 'moderate').length
		);
	});

	it('shows a screenshot crop for a real new occurrence on the home page', () => {
		const crop = DEMO_CROP;
		expect(crop).not.toBeNull();
		if (!crop) throw new Error('The sample deployment needs a captured new finding');
		// The featured example must stay a recognizable control when regenerated.
		// The old first-element selection brought the tiny careers arrow back.
		expect(crop.element.ruleId).toBe('select-name');
		expect(crop.element.selector).toBe('#billing-cycle');
		expect(crop.title).toBe('Select element must have an accessible name');
		const displayed = buildOccurrenceModeReport(splitManualChecks(report).report);
		const issue = displayed.issues.find((value) => value.id === crop.issueId);
		expect(issue).toBeTruthy();
		const newIds = new Set(
			comparison.new.flatMap(
				(value) => value.occurrences?.map((occurrence) => occurrence.elementId) ?? []
			)
		);
		expect(issue?.occurrences?.some((occurrence) => newIds.has(occurrence.elementId))).toBe(true);
		const page = displayed.pages.find((value) => value.id === issue?.pageId);
		expect(page?.pageOverview?.pageWidth).toBe(crop.pageWidth);
		expect(page?.pageOverview?.pageHeight).toBe(crop.pageHeight);
		expect(page?.pageOverview?.elements).toContainEqual(crop.element);
		expect(
			screenshots.some((shot) => shot.url === crop.screenshot && shot.page_id === issue?.pageId)
		).toBe(true);
		expect(existsSync(path.join(DEMO_DIR, path.basename(crop.screenshot)))).toBe(true);
	});

	it('leaks nothing from the machine that generated it', () => {
		const raw = `${JSON.stringify(baseline)}${JSON.stringify(report)}${JSON.stringify(screenshots)}`;
		for (const forbidden of [
			'localhost',
			'127.0.0.1',
			'X-Amz-',
			'minio',
			'/workspace/',
			'"storage_state":',
			'"auth":',
			'"password":'
		]) {
			expect(raw, `${forbidden} survived into the committed fixture`).not.toContain(forbidden);
		}
	});
});
