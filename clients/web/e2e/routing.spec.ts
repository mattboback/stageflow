import { DEMO_CROP } from '../app/lib/demo/demo-crop';
import { expect, test } from './fixtures';

/*
 * qa/serve-build.mjs mirrors nginx.conf, so these status codes are the ones
 * stageflow.org returns: prerendered pages without a trailing-slash redirect,
 * the SPA shell for scan routes, and a real 404 for anything else.
 */
test.describe('routing', () => {
	for (const path of ['/', '/projects', '/playground', '/privacy', '/demo']) {
		test(`${path} is a 200 with no redirect`, async ({ page }) => {
			const response = await page.request.get(path, { maxRedirects: 0 });
			expect(response.status()).toBe(200);
		});
	}

	test('scan routes serve the client shell', async ({ page }) => {
		for (const path of ['/scan/abc123', '/scan/abc123/report']) {
			const response = await page.request.get(path, { maxRedirects: 0 });
			expect(response.status(), path).toBe(200);
		}
	});

	/*
	 * The prerendered HTML has no query string. A first render that showed
	 * something different because of one would not match it, and React would
	 * discard the page with a hydration error, which the browserErrors fixture
	 * fails on.
	 */
	test.describe('a query string on a prerendered page', () => {
		test('opens the report section it names', async ({ page }) => {
			await page.goto('/demo?section=pages');
			await expect(page.getByRole('tab', { name: /Pages/ })).toHaveAttribute(
				'aria-selected',
				'true'
			);
		});

		test('opens the issue it names', async ({ page }) => {
			const issueId = DEMO_CROP?.issueId;
			expect(issueId, 'the demo fixture has a located finding').toBeDefined();
			await page.goto(`/demo?section=issues&issue=${issueId ?? ''}`);
			await expect(page.getByRole('dialog', { name: 'Issue details' })).toBeVisible();
		});

		test('seeds the scan form with the URL it carries', async ({ page }) => {
			await page.route('**/api/v1/scanners', (route) =>
				route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: JSON.stringify({ scanners: [], categories: [] })
				})
			);
			await page.goto(`/playground?url=${encodeURIComponent('https://sample.stageflow.org/')}`);
			await expect(page.getByRole('textbox', { name: 'URL 1', exact: true })).toHaveValue(
				'https://sample.stageflow.org/'
			);
		});
	});

	test.describe('an unknown path', () => {
		// Chromium logs the document's own 404 status as a console error.
		test.use({ allowedBrowserErrors: [/status of 404/] });

		test('is a 404 that renders the not-found page', async ({ page }) => {
			const response = await page.goto('/no-such-page');
			expect(response?.status()).toBe(404);
			await expect(page.getByRole('heading', { name: 'Page not found.' })).toBeVisible();
		});
	});
});
