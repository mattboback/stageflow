import type { Page } from '@playwright/test';
import { expect, loadReportFixture, test } from './fixtures';

const report = loadReportFixture();

const firstPage = report.pages[0];
if (!firstPage) {
	throw new Error('unified-report.v2.all-scans.json must contain at least one page');
}

// The fixture ships no manual-review findings; add one so the review-next
// CTA (a queue-driven surface) is exercised. Lighthouse info + no occurrences
// is the shape getIssueKind treats as manual review.
report.issues.push({
	id: 'e2e-manual-review-1',
	scanner: 'lighthouse',
	ruleId: 'custom-controls-labels',
	title: 'Custom controls have associated labels',
	description: 'Verify that custom interactive controls expose accessible labels.',
	severity: 'info',
	pageId: firstPage.id,
	pageUrl: firstPage.url,
	elementCount: 0,
	occurrences: []
});

// Axe incomplete contrast findings have a concrete element but still require
// a human decision, so they belong in the same review queue.
report.issues.push({
	id: 'e2e-incomplete-contrast-1',
	scanner: 'axe',
	ruleId: 'color-contrast',
	title: 'Elements must have sufficient color contrast',
	description: 'Verify the rendered text and background colors.',
	severity: 'serious',
	pageId: firstPage.id,
	pageUrl: firstPage.url,
	elementCount: 1,
	occurrences: [{ selector: '.hero-copy', html: '<p class="hero-copy">Welcome</p>' }],
	scannerData: {
		axeIncomplete: true,
		contrastData: {
			fgColor: '#000000',
			bgColor: '#ffffff',
			contrastRatio: 21,
			fontSize: '16px',
			fontWeight: '400',
			messageKey: 'bgImage'
		}
	}
});

// Must match meta.jobId inside the fixture — the report hook rejects mismatches.
const JOB_ID = report.meta.jobId;

const jobStatus = {
	id: JOB_ID,
	state: 'done',
	violations: report.summary.totalIssues,
	artifacts: {},
	created_at: '2026-01-01T00:00:00Z',
	updated_at: '2026-01-01T00:00:12Z'
};

async function mockReportRoutes(page: Page) {
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
		route.fulfill({
			status: 200,
			contentType: 'text/event-stream',
			body: 'event: done\ndata: {}\n\n'
		})
	);
}

async function mockScannerCatalog(page: Page) {
	await page.route('**/api/v1/scanners', (route) =>
		route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify({
				scanners: [
					{
						id: 'axe',
						name: 'Axe Accessibility',
						version: '1.0.0',
						description: 'WCAG 2.1 rules, powered by axe-core',
						categories: ['accessibility'],
						aliases: [],
						enabled: true,
						builtIn: true,
						capabilities: {}
					}
				],
				categories: ['accessibility']
			})
		})
	);
}

function collectPageErrors(page: Page): string[] {
	const pageErrors: string[] = [];
	page.on('pageerror', (err) => {
		pageErrors.push(String(err));
	});
	return pageErrors;
}

test.describe('employer-facing product claims', () => {
	test('homepage describes the supported runtime and owned report formats', async ({ page }) => {
		await page.goto('/');

		const selfHost = page.locator('.selfhost');
		await expect(selfHost.getByText('Rootless Podman isolation', { exact: true })).toBeVisible();
		await expect(
			selfHost.getByText('Self-host on supported Linux infrastructure', { exact: true })
		).toBeVisible();
		await expect(
			selfHost.getByText('Reports you own: HTML and JSON', { exact: true })
		).toBeVisible();
		await expect(selfHost.getByText('Docker-based', { exact: false })).toHaveCount(0);
		await expect(selfHost.getByText('SARIF', { exact: false })).toHaveCount(0);
	});

	test('404 metadata uses page terminology', async ({ page }) => {
		await page.goto('/route-that-does-not-exist');

		await expect(page).toHaveTitle('404 · Page not found — StageFlow');
		await expect(page.getByText('Channel not found', { exact: false })).toHaveCount(0);
	});
});

test('report page renders the fixture report end to end', async ({ page }) => {
	await mockReportRoutes(page);
	const pageErrors = collectPageErrors(page);

	await page.goto(`/scan/${JOB_ID}/report?section=overview`);

	// Header: host from baseUrl (Daylight Gauge redesign) plus compact stats.
	const host = new URL(report.meta.baseUrl as string).host;
	const totals = page.getByLabel('Scan totals');
	await expect(page.getByRole('heading', { name: host })).toBeVisible();
	await expect(totals.getByText('Pages scanned', { exact: true })).toBeVisible();
	await expect(
		totals.getByText(`${report.summary.pagesWithIssues.toLocaleString()} with findings`)
	).toBeVisible();
	await expect(totals.getByText('Human review')).toBeVisible();

	// Review landing: review-next CTA, filter toolbar, three-panel visual review.
	await expect(page.getByText(/findings need review|findings reviewed/)).toBeVisible();
	await expect(page.getByLabel('Scan summary').getByText('Severity')).toBeVisible();
	await expect(page.getByRole('heading', { name: 'Pages', exact: true })).toBeVisible();
	await expect(page.getByRole('heading', { name: 'Findings on this page' })).toBeVisible();

	expect(pageErrors).toEqual([]);
});

test('review queue records contrast and generic human decisions', async ({ page }) => {
	await mockReportRoutes(page);
	await page.goto(`/scan/${JOB_ID}/report`);

	await page.getByRole('button', { name: /Start review/ }).click();
	await expect(page.getByRole('tab', { name: 'Verify' })).toHaveAttribute('aria-selected', 'true');
	await page.getByRole('button', { name: 'Mark pass' }).click();

	await page.getByRole('button', { name: 'Next issue (j)' }).click();
	await expect(page.getByRole('tab', { name: 'Review', exact: true })).toHaveAttribute(
		'aria-selected',
		'true'
	);
	await page.getByRole('button', { name: 'Mark pass' }).click();
	await page.getByRole('button', { name: 'Close modal' }).click();

	await expect(page.locator('.rnext__count').getByText('All 2 findings reviewed')).toBeVisible();
	const totals = page.getByLabel('Scan totals');
	await expect(totals.getByText('all 2 findings reviewed')).toBeVisible();
	await expect(totals.getByText(/need a human check/)).toHaveCount(0);
	await expect(page.getByText('needs review', { exact: true })).toHaveCount(0);
	// Both findings were marked pass above, and both are attached to the same page,
	// so the visual review list shows a badge for each.
	await expect(page.getByText('reviewed · pass', { exact: true })).toHaveCount(2);
});

test('review decisions synchronize without loss across browser tabs', async ({ context, page }) => {
	const otherPage = await context.newPage();
	await mockReportRoutes(page);
	await mockReportRoutes(otherPage);
	await Promise.all([
		page.goto(`/scan/${JOB_ID}/report`),
		otherPage.goto(`/scan/${JOB_ID}/report`)
	]);

	await page.getByRole('button', { name: /Start review/ }).click();
	await page.getByRole('button', { name: 'Mark pass' }).click();
	await page.getByRole('button', { name: 'Close modal' }).click();

	await expect(otherPage.getByText('1 of 2 findings still need review')).toBeVisible();
	await otherPage.getByRole('button', { name: /Continue review/ }).click();
	await expect(otherPage.getByRole('tab', { name: 'Review', exact: true })).toHaveAttribute(
		'aria-selected',
		'true'
	);
	await otherPage.getByRole('button', { name: 'Mark fail' }).click();
	await otherPage.getByRole('button', { name: 'Close modal' }).click();

	await expect(page.locator('.rnext__count').getByText('All 2 findings reviewed')).toBeVisible();
	await page.reload();
	await expect(page.locator('.rnext__count').getByText('All 2 findings reviewed')).toBeVisible();
});

test.describe('mobile report (390×844)', () => {
	test.use({ viewport: { width: 390, height: 844 } });

	test('review tab leads with the review-next CTA', async ({ page }) => {
		await mockReportRoutes(page);
		const pageErrors = collectPageErrors(page);

		await page.goto(`/scan/${JOB_ID}/report`);

		await expect(page.getByText(/findings need review/)).toBeVisible();
		await expect(page.getByRole('button', { name: /Start review/ })).toBeVisible();

		expect(pageErrors).toEqual([]);
	});

	test('findings tab uses a filter sheet instead of a sidebar', async ({ page }) => {
		await mockReportRoutes(page);

		await page.goto(`/scan/${JOB_ID}/report?section=issues`);

		// The sidebar is hidden behind the Filters button on small screens.
		const filtersBtn = page.getByRole('button', { name: /^Filters/ });
		await expect(filtersBtn).toBeVisible();
		await expect(page.getByLabel('Finding filters')).toBeHidden();

		await filtersBtn.click();
		await expect(page.getByLabel('Finding filters')).toBeVisible();
		await page.getByRole('button', { name: /Show \d+ finding/ }).click();
		await expect(page.getByLabel('Finding filters')).toBeHidden();
	});
});

test.describe('mobile configure page (390×844)', () => {
	test.use({ viewport: { width: 390, height: 844 } });

	test('exactly one Run button, in the sticky action bar', async ({ page }) => {
		await mockScannerCatalog(page);

		await page.goto('/playground');

		// Hidden buttons drop out of the accessibility tree, so exactly one
		// Run button being resolvable means exactly one is offered.
		const runButtons = page.getByRole('button', { name: /Run scan/ });
		await expect(runButtons).toHaveCount(1);
		await expect(
			page.locator('.stickyrun').getByRole('button', { name: /Run scan/ })
		).toBeVisible();
	});

	test('hidden incomplete URL auth does not block a ZIP scan', async ({ page }) => {
		await mockScannerCatalog(page);
		await page.route('**/api/v1/jobs/zip', (route) =>
			route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({ job_id: 'zip-job-1' })
			})
		);

		await page.goto('/playground');
		await page.getByRole('button', { name: 'Set up' }).click();
		await page.getByRole('button', { name: 'ZIP upload' }).click();
		await page.locator('input[type="file"]').setInputFiles({
			name: 'site.zip',
			mimeType: 'application/zip',
			buffer: Buffer.from('zip fixture')
		});

		const submitted = page.waitForRequest(
			(request) => request.method() === 'POST' && request.url().endsWith('/api/v1/jobs/zip')
		);
		await page
			.locator('.stickyrun')
			.getByRole('button', { name: /Run scan/ })
			.click();
		await submitted;
	});
});

test('a zero-finding report celebrates the all-clear instead of an empty queue', async ({
	page
}) => {
	const cleanReport = loadReportFixture();
	const CLEAN_JOB_ID = 'e2e-clean-run';
	cleanReport.meta.jobId = CLEAN_JOB_ID;
	cleanReport.issues = [];
	cleanReport.errors = [];
	cleanReport.summary.totalIssues = 0;
	cleanReport.summary.pagesWithIssues = 0;
	cleanReport.summary.score = 100;
	cleanReport.summary.bySeverity = { critical: 0, serious: 0, moderate: 0, minor: 0 };

	await page.route(`**/api/v1/jobs/${CLEAN_JOB_ID}`, (route) =>
		route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify({ ...jobStatus, id: CLEAN_JOB_ID, violations: 0 })
		})
	);
	await page.route(`**/api/v1/jobs/${CLEAN_JOB_ID}/results**`, (route) =>
		route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify(cleanReport)
		})
	);
	await page.route(`**/api/v1/jobs/${CLEAN_JOB_ID}/stream**`, (route) =>
		route.fulfill({
			status: 200,
			contentType: 'text/event-stream',
			body: 'event: done\ndata: {}\n\n'
		})
	);

	await page.goto(`/scan/${CLEAN_JOB_ID}/report`);

	await expect(page.getByRole('heading', { name: 'All clear.' })).toBeVisible();
	await expect(page.getByText('0 findings')).toBeVisible();
	await expect(page.getByText('nothing to fix')).toBeVisible();
	await expect(page.getByRole('link', { name: /Run another scan/ })).toBeVisible();
	await expect(page.getByText(/findings need review/)).toHaveCount(0);
});
