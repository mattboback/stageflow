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
