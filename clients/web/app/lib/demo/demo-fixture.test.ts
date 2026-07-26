import { existsSync, readFileSync } from 'node:fs';
import path from 'node:path';
import { describe, expect, it } from 'vitest';

import { buildOccurrenceModeReport } from '../report';
import { isUnifiedReport } from '../projects';
import { getPageOverviewUrl } from '../report/screenshots';
import { DEMO_SUMMARY } from './demo-summary';
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
const screenshots = readJson<ScreenshotArtifact[]>('screenshots.json');

describe('the committed /demo fixture', () => {
	it('is a report the app would accept from the API', () => {
		expect(isUnifiedReport(report)).toBe(true);
	});

	it('is not addressed as a live job', () => {
		// A real job id here would invite someone to paste it into
		// /scan/<id>/report and get a 404, and would key review verdicts on a
		// job that no longer exists.
		expect(report.meta.jobId).toBe('demo');
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
	 * The trap the plan for this fixture called out by name.
	 *
	 * buildOccurrenceModeReport rewrites element.issueId, falling back to a
	 * `ruleId|selector` index when the direct mapping misses. A generator that
	 * blanks `selector` (qa/record-report-gif.mjs does exactly that) kills the
	 * fallback, and any element whose nodeIndex exceeds the occurrence count
	 * keeps a stale id. So the assertion has to run POST-remap: checking the raw
	 * fixture would pass while the rendered page shows nothing.
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
		const raw = JSON.stringify(report);
		expect(raw).not.toContain('resultsPath');
		expect(raw).not.toContain('reportPath');
	});

	it('matches the figures the home page prints', () => {
		// DEMO_SUMMARY is duplicated on purpose: the home page will not pull 90 KB
		// of JSON in to print four numbers. This is what stops the duplicate from
		// silently describing an older scan.
		const displayed = buildOccurrenceModeReport(report);
		expect(DEMO_SUMMARY.pages).toBe(report.pages.length);
		expect(DEMO_SUMMARY.scannersRun).toBe(report.scanners.length);
		expect(DEMO_SUMMARY.totalIssues).toBe(displayed.issues.length);
		expect(DEMO_SUMMARY.serious).toBe(
			displayed.issues.filter((issue) => issue.severity === 'serious').length
		);
		expect(DEMO_SUMMARY.moderate).toBe(
			displayed.issues.filter((issue) => issue.severity === 'moderate').length
		);
	});

	it('leaks nothing from the machine that generated it', () => {
		const raw = `${JSON.stringify(report)}${JSON.stringify(screenshots)}`;
		for (const forbidden of ['localhost', '127.0.0.1', 'X-Amz-', 'minio']) {
			expect(raw, `${forbidden} survived into the committed fixture`).not.toContain(forbidden);
		}
	});
});
