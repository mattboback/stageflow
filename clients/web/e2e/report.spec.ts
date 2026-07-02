import fs from 'node:fs';
import path from 'node:path';

import { expect, test } from '@playwright/test';

const FIXTURE_PATH = path.resolve(
	process.cwd(),
	'../../libs/contracts/report/fixtures/unified-report.v2.all-scans.json'
);

const report = JSON.parse(fs.readFileSync(FIXTURE_PATH, 'utf8'));

// Must match meta.jobId inside the fixture — the report hook rejects mismatches.
const JOB_ID = report.meta.jobId as string;

const jobStatus = {
	id: JOB_ID,
	state: 'done',
	violations: report.summary.totalIssues,
	artifacts: {},
	created_at: '2026-01-01T00:00:00Z',
	updated_at: '2026-01-01T00:00:12Z'
};

test('report page renders the fixture report end to end', async ({ page }) => {
	await page.route(`**/api/v1/jobs/${JOB_ID}`, (route) =>
		route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify(jobStatus)
		})
	);
	await page.route(`**/api/v1/jobs/${JOB_ID}/results**`, (route) =>
		route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify(report)
		})
	);
	await page.route(`**/api/v1/jobs/${JOB_ID}/stream**`, (route) =>
		route.fulfill({ status: 404, body: 'no stream' })
	);

	const pageErrors: string[] = [];
	page.on('pageerror', (err) => {
		// React #418 (hydration mismatch) fires when a non-prerendered /scan/*
		// route hydrates from the SPA shell; tracked separately from this smoke.
		if (String(err).includes('418')) return;
		pageErrors.push(String(err));
	});

	await page.goto(`/scan/${JOB_ID}/report?section=overview`);

	// Header: score gauge plus page/scanner counts from the fixture.
	await expect(page.getByRole('heading', { name: 'Scan report' })).toBeVisible();
	await expect(
		page.getByText(
			`${report.summary.pagesScanned} pages · ${report.scanners.length} scanners`
		)
	).toBeVisible();

	// Overview: the dashboard summarizes issue patterns and scanner status.
	await expect(page.getByRole('heading', { name: /issue patterns? found/ })).toBeVisible();
	await expect(page.getByRole('heading', { name: 'Scanner Status' })).toBeVisible();

	expect(pageErrors).toEqual([]);
});
