import { expect, test } from './fixtures';

/*
 * The only spec that runs against real data with nothing mocked.
 *
 * Every other spec routes the API and feeds it the contract fixture, which
 * proves the components work on a shape. This proves the shape the generator
 * actually produces still drives them -- and it is the route an employer
 * lands on, so a regression here is the one that matters most.
 */
test.describe('the demo report', () => {
	test('opens from the home page and renders a real scan', async ({ page }) => {
		await page.goto('/');
		await page.getByRole('link', { name: /Open the report/ }).click();

		await expect(page).toHaveURL(/\/demo$/);
		await expect(page.getByRole('heading', { level: 1, name: 'stageflow.org' })).toBeVisible();
		// Findings, not "no findings" -- the empty branch is the failure mode.
		await expect(page.getByRole('tab', { name: /Findings/ })).toBeVisible();
		await expect(page.getByRole('tab', { name: /Artifacts/ })).toBeVisible();
	});

	test('the page index leads into visual review for that page', async ({ page }) => {
		await page.goto('/demo');

		await page.getByRole('tab', { name: /Pages/ }).click();
		const cards = page.getByRole('button', { name: /findings$/ });
		await expect(cards.first()).toBeVisible();
		// The tab only exists above one page; a single-page report resolves it
		// back to review, so this also pins the fixture at more than one page.
		expect(await cards.count()).toBeGreaterThan(1);

		await cards.first().click();
		await expect(page).toHaveURL(/[?&]page=/);
		await expect(page.getByRole('heading', { name: 'Findings on this page' })).toBeVisible();
	});

	test('the next-finding control walks the markers on the screenshot', async ({ page }) => {
		await page.goto('/demo');

		const next = page.getByRole('button', { name: /Next finding on this page/ });
		await expect(next).toBeVisible();

		// The active marker is the one drawn with the heavier stroke, which is the
		// only externally visible signal that the walk moved. Asserting zero first
		// is what makes the second assertion mean anything.
		const active = page.locator('.vrev__marker[stroke-width="6"]');
		await expect(active).toHaveCount(0);
		await next.click();
		await expect(active).toHaveCount(1);
	});

	test('serves the fixture it claims to render', async ({ page }) => {
		const response = await page.request.get('/demo/report.json');
		expect(response.ok()).toBe(true);

		const report = (await response.json()) as { meta: { jobId: string }; issues: unknown[] };
		expect(report.meta.jobId).toBe('demo');
		expect(report.issues.length).toBeGreaterThan(0);
	});
});
