import type { Page } from '@playwright/test';
import type { UnifiedReport } from '../app/lib/types/unified-report';
import { expect, loadReportFixture, test } from './fixtures';

const SOURCE_JOB = 'e2e-verification-source';
const TARGET_JOB = 'e2e-verification-target';
const PAGE_ID = 'page-v2-verification-main';
const SUBJECT_ID = 'occ-v2-verification-button';
const ISSUE_ID = 'e2e-verification-contrast';
const PAGE_URL = 'https://staging.example.test/pricing';
const SCANNER_OPTIONS = { runOnly: { type: 'tag', values: ['wcag2aa'] } };

function modernReport(inputType: 'urls' | 'zip' = 'urls', authenticated = false): UnifiedReport {
	const report = loadReportFixture();
	const templatePage = report.pages[0];
	const templateIssue = report.issues[0];
	if (!templatePage || !templateIssue) throw new Error('Expected report fixture');
	report.meta.jobId = SOURCE_JOB;
	report.meta.baseUrl = 'https://staging.example.test';
	report.meta.profile = 'staging';
	report.meta.scanContext = {
		expectedPages: [PAGE_ID, 'page-v2-other-origin'],
		identityScope: 'origin',
		scannerConfigFingerprints: { axe: 'original-scanner-settings', seo: 'other-scanner' },
		scannerConfigurations: { axe: SCANNER_OPTIONS, seo: {} },
		pageConfigFingerprints: { [PAGE_ID]: 'original-page-settings' },
		verificationEligiblePages: { [PAGE_ID]: true },
		viewport: { width: 1280, height: 720 },
		purpose: 'full',
		authenticated,
		inputType
	};
	report.pages = [
		{ ...templatePage, id: PAGE_ID, url: PAGE_URL, path: '/pricing', issueCount: 1 },
		{
			...templatePage,
			id: 'page-v2-other-origin',
			url: 'https://other.example.test/',
			issueCount: 0
		}
	];
	report.issues = [
		{
			...templateIssue,
			id: ISSUE_ID,
			pageId: PAGE_ID,
			pageUrl: PAGE_URL,
			occurrences: [{ ...templateIssue.occurrences?.[0], elementId: SUBJECT_ID }]
		}
	];
	report.summary.totalIssues = 1;
	report.scanners = report.scanners.filter(
		(scanner) => scanner.id === 'axe' || scanner.id === 'seo'
	);
	for (const scanner of report.scanners) {
		scanner.status = 'success';
		scanner.toolVersion = 'e2e-scanner-1.0';
		scanner.pages = [
			{
				pageId: PAGE_ID,
				status: 'success',
				finalUrl: PAGE_URL,
				rules: [{ ruleId: templateIssue.ruleId, outcome: 'fail', subjectIds: [SUBJECT_ID] }]
			}
		];
	}
	return report;
}

function targetedReport(source: UnifiedReport): UnifiedReport {
	const result = structuredClone(source);
	const context = result.meta.scanContext;
	const issue = source.issues[0];
	if (!context || !issue) throw new Error('Expected modern execution context');
	result.meta.jobId = TARGET_JOB;
	context.purpose = 'verification';
	context.expectedPages = [PAGE_ID];
	result.pages = result.pages.filter((page) => page.id === PAGE_ID);
	result.issues = [];
	result.summary.totalIssues = 0;
	result.summary.score = 99;
	result.scanners = result.scanners.filter((scanner) => scanner.id === 'axe');
	for (const scanner of result.scanners) {
		scanner.issueCount = 0;
		scanner.pages = [
			{
				pageId: PAGE_ID,
				status: 'success',
				finalUrl: PAGE_URL,
				rules: [{ ruleId: issue.ruleId, outcome: 'pass', subjectIds: [SUBJECT_ID] }]
			}
		];
	}
	return result;
}

async function mockReports(page: Page, source: UnifiedReport) {
	for (const report of [source, targetedReport(source)]) {
		const id = report.meta.jobId;
		await page.route(`**/api/v1/jobs/${id}`, (route) =>
			route.fulfill({
				json: {
					id,
					state: 'done',
					violations: report.summary.totalIssues,
					artifacts: {},
					created_at: '2026-09-30T00:00:00Z',
					updated_at: '2026-09-30T00:00:12Z'
				}
			})
		);
		await page.route(`**/api/v1/jobs/${id}/results**`, (route) => route.fulfill({ json: report }));
		await page.route(`**/api/v1/jobs/${id}/stream**`, (route) =>
			route.fulfill({
				contentType: 'text/event-stream',
				body: 'event: done\ndata: {}\n\n'
			})
		);
	}
}

async function openVerification(page: Page) {
	await page.goto(`/scan/${SOURCE_JOB}/report?section=issues`);
	await page.locator('button.irow').first().click();
	await expect(page.getByRole('dialog')).toBeVisible();
	await page.getByRole('button', { name: 'Verify this fix', exact: true }).click();
	const dialog = page.getByRole('dialog', { name: 'Verify this fix', exact: true });
	await expect(dialog).toBeVisible();
	return dialog;
}

test('verify fix sends only the responsible page and scanner and renders a targeted result', async ({
	page
}) => {
	const source = modernReport();
	await mockReports(page, source);
	const submissions: unknown[] = [];
	await page.route('**/api/v1/jobs/urls', (route) => {
		submissions.push(route.request().postDataJSON());
		return route.fulfill({ json: { job_id: TARGET_JOB } });
	});
	const dialog = await openVerification(page);
	await dialog.getByRole('button', { name: 'Run verification', exact: true }).click();
	await expect(page).toHaveURL(`/scan/${TARGET_JOB}/report`);
	await expect(page.getByRole('region', { name: 'Targeted verification result' })).toContainText(
		'Targeted verification: Fixed'
	);
	expect(submissions).toEqual([
		{
			urls: [PAGE_URL],
			modules: ['axe'],
			scanner_configs: { axe: SCANNER_OPTIONS },
			identity_scope: 'origin',
			profile: 'staging',
			screenshot: true,
			verification: {
				source_job_id: SOURCE_JOB,
				page_id: PAGE_ID,
				rule_id: source.issues[0]?.ruleId,
				scanner: 'axe',
				subject_id: SUBJECT_ID
			}
		}
	]);
	await expect(page.getByText('Staging profile', { exact: true })).toBeVisible();
	await expect(page.locator('.rhead .gauge')).toHaveCount(0);
	await expect(page.getByRole('button', { name: 'Save in this browser' })).toHaveCount(0);
	await expect(page.getByRole('button', { name: 'Promote as baseline' })).toHaveCount(0);
	await expect(page.getByRole('link', { name: 'Return to the original report' })).toHaveAttribute(
		'href',
		`/scan/${SOURCE_JOB}/report`
	);
	// The comparison request must still work after a normal browser reload.
	await page.reload();
	await expect(page.getByRole('region', { name: 'Targeted verification result' })).toContainText(
		'Targeted verification: Fixed'
	);
});

test('ZIP verification blocks submission until a fresh build is selected', async ({ page }) => {
	await mockReports(page, modernReport('zip'));
	const submissions: string[] = [];
	await page.route('**/api/v1/jobs/zip', (route) => {
		submissions.push(route.request().url());
		return route.fulfill({ json: { job_id: TARGET_JOB } });
	});
	const dialog = await openVerification(page);
	await expect(dialog.getByText('Upload a fresh ZIP build to verify this change.')).toBeVisible();
	await expect(dialog.getByRole('button', { name: 'Run verification' })).toBeDisabled();
	await dialog.getByLabel('Fresh ZIP build').setInputFiles({
		name: 'fresh-build.zip',
		mimeType: 'application/zip',
		buffer: Buffer.from('PK\u0003\u0004')
	});
	await expect(dialog.getByRole('button', { name: 'Run verification' })).toBeEnabled();
	await dialog.getByLabel('Fresh ZIP build').setInputFiles([]);
	await expect(dialog.getByRole('button', { name: 'Run verification' })).toBeDisabled();
	expect(submissions).toEqual([]);
});

test('protected-page verification requires fresh complete authentication and cannot bypass it', async ({
	page
}) => {
	await mockReports(page, modernReport('urls', true));
	const submissions: unknown[] = [];
	await page.route('**/api/v1/jobs/urls', (route) => {
		submissions.push(route.request().postDataJSON());
		return route.fulfill({ json: { job_id: TARGET_JOB } });
	});
	const dialog = await openVerification(page);
	await expect(dialog.getByText(/Enter fresh credentials for this verification/)).toBeVisible();
	await expect(dialog.getByRole('button', { name: 'Run verification' })).toBeDisabled();
	const acknowledgment = dialog.getByRole('checkbox', { name: /throwaway test account/ });
	if (await acknowledgment.count()) await acknowledgment.check();
	await dialog.getByLabel('Login URL').fill('https://staging.example.test/login');
	await dialog.getByLabel('Username / email').fill('temporary-test-user');
	await expect(dialog.getByRole('button', { name: 'Run verification' })).toBeDisabled();
	await dialog.getByLabel('Password', { exact: false }).first().fill('fresh-test-only-password');
	await expect(dialog.getByRole('button', { name: 'Run verification' })).toBeEnabled();
	await dialog.getByRole('button', { name: 'Remove', exact: true }).click();
	await expect(dialog.getByRole('button', { name: 'Run verification' })).toBeDisabled();
	expect(submissions).toEqual([]);
	const stored = await page.evaluate(() => JSON.stringify({ ...window.localStorage }));
	expect(stored).not.toContain('fresh-test-only-password');
	expect(stored).not.toContain('temporary-test-user');
});
