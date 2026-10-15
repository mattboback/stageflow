import type { Locator, Page } from '@playwright/test';
import { DEMO_CROP } from '../app/lib/demo/demo-crop';
import { expect, loadReportFixture, test } from './fixtures';

if (!DEMO_CROP) throw new Error('The mobile layout suite requires the committed demo finding.');
const demoIssueUrl = `/demo?section=issues&issue=${DEMO_CROP.issueId}`;

async function expectViewportFits(page: Page, width: number) {
	await expect(page.locator('meta[name="viewport"]')).toHaveCount(1);
	await expect(page.locator('meta[name="viewport"]')).toHaveAttribute(
		'content',
		'width=device-width, initial-scale=1'
	);
	const sizes = await page.evaluate(() => ({
		viewport: window.innerWidth,
		client: document.documentElement.clientWidth,
		scroll: document.documentElement.scrollWidth
	}));
	expect(sizes.viewport, 'the browser must use the phone viewport, not a scaled desktop').toBe(
		width
	);
	expect(sizes.scroll, 'the document must not overflow horizontally').toBeLessThanOrEqual(
		sizes.client + 1
	);
}

async function expectControlsFit(container: Locator, controls: Locator) {
	const box = await container.boundingBox();
	expect(box).not.toBeNull();
	if (!box) throw new Error('The control container is not rendered.');
	for (const control of await controls.all()) {
		const rect = await control.boundingBox();
		expect(rect).not.toBeNull();
		if (!rect) throw new Error('A required control is not rendered.');
		expect(rect.x).toBeGreaterThanOrEqual(box.x - 1);
		expect(rect.x + rect.width).toBeLessThanOrEqual(box.x + box.width + 1);
		expect(rect.height, 'phone controls keep a 44px touch target').toBeGreaterThanOrEqual(44);
	}
}

for (const width of [360, 390, 412]) {
	test.describe(`${width}px phone layout`, () => {
		test.use({ viewport: { width, height: 844 }, isMobile: true, hasTouch: true });

		test('homepage reflows and navigation remains reachable', async ({ page }, testInfo) => {
			await page.goto('/');
			await expect(page.getByRole('heading', { name: /Find what to fix/ })).toBeVisible();
			await expectViewportFits(page, width);
			const hero = page.locator('.hero__grid');
			expect(
				await hero.evaluate(
					(element) => getComputedStyle(element).gridTemplateColumns.split(' ').length
				)
			).toBe(1);
			expect(
				await page
					.locator('.basetrack')
					.evaluate((element) => getComputedStyle(element).gridTemplateColumns.split(' ').length)
			).toBe(1);
			await expect(page.getByRole('button', { name: 'Scan', exact: true })).toBeInViewport();
			await page.getByRole('button', { name: 'Menu', exact: true }).click();
			await expect(page.locator('.navmenu__panel')).toBeVisible();
			await page.keyboard.press('Escape');
			await expect(page.getByRole('button', { name: 'Menu', exact: true })).toBeFocused();
			await page.getByRole('button', { name: 'Menu', exact: true }).click();
			await page.locator('.navmenu__panel').getByRole('link', { name: 'Demo report' }).click();
			await expect(page).toHaveURL(/\/demo$/);
			await expect(page.locator('.navmenu__panel')).toHaveCount(0);
			await expectViewportFits(page, width);
			await testInfo.attach('phone-report', {
				body: await page.screenshot(),
				contentType: 'image/png'
			});
		});

		test('screenshot controls stay inside the panel at every zoom', async ({ page }, testInfo) => {
			await page.goto('/demo');
			const stage = page.locator('.vrev__stage');
			const controls = page.locator('.vrev__stage-bar button, .vrev__stage-open');
			await expect(page.getByRole('button', { name: 'Zoom in', exact: true })).toBeVisible();
			await expectControlsFit(stage, controls);
			await page.getByRole('button', { name: /Next finding on this page/ }).click();
			await page.getByRole('button', { name: 'Zoom to finding', exact: true }).click();
			await page.getByRole('button', { name: 'Zoom in', exact: true }).click();
			await expectControlsFit(stage, controls);
			await page.getByRole('button', { name: 'Zoom out', exact: true }).click();
			await page.getByRole('button', { name: 'Fit width', exact: true }).click();
			await expect(page.getByRole('button', { name: 'Fit width', exact: true })).toHaveAttribute(
				'aria-pressed',
				'true'
			);
			const lastPage = page.locator('.vrev__page-btn').last();
			await lastPage.click();
			await expect(lastPage).toHaveAttribute('aria-current', 'true');
			await expectControlsFit(stage, controls);
			await expectViewportFits(page, width);
			await testInfo.attach('phone-screenshot-controls', {
				body: await stage.screenshot(),
				contentType: 'image/png'
			});
		});

		test('finding evidence and notes share a scroller while close and tabs remain visible', async ({
			page
		}, testInfo) => {
			await page.goto(demoIssueUrl);
			const dialog = page.getByRole('dialog', { name: 'Issue details' });
			const content = dialog.locator('.imodal__content');
			const close = dialog.getByRole('button', { name: 'Close modal' });
			await expect(dialog).toBeVisible();
			await expect(close).toBeFocused();
			await expect(content).toHaveCSS('overflow-y', 'auto');
			expect(await content.evaluate((element) => element.clientHeight)).toBeGreaterThan(400);
			expect(
				await dialog
					.locator('.imodal__meta-bar')
					.evaluate((element) => getComputedStyle(element).gridTemplateColumns.split(' ').length)
			).toBe(1);
			await dialog.getByRole('tab', { name: 'Evidence', exact: true }).click();
			await dialog.getByLabel('Review note (optional)').fill('Checked on a phone.');
			await dialog.getByRole('button', { name: 'Save note', exact: true }).click();
			await expect(dialog.getByText('Note saved.')).toBeVisible();
			expect(await content.evaluate((element) => element.scrollTop)).toBeGreaterThan(0);
			await expect(close).toBeInViewport();
			await expect(dialog.getByRole('tab', { name: 'Fix', exact: true })).toBeInViewport();
			await dialog.getByRole('tab', { name: 'Fix', exact: true }).click();
			await expect.poll(() => content.evaluate((element) => element.scrollTop)).toBe(0);
			await expectViewportFits(page, width);
			await testInfo.attach('phone-finding', {
				body: await page.screenshot(),
				contentType: 'image/png'
			});
			await close.click();
			await expect(dialog).toHaveCount(0);
			await expect(page).not.toHaveURL(/[?&]issue=/);
			expect(await page.evaluate(() => document.body.style.overflow)).not.toBe('hidden');
			await page.goto(demoIssueUrl);
			await expect(dialog.getByLabel('Review note (optional)')).toHaveValue('Checked on a phone.');
			await page.keyboard.press('Escape');
			await expect(dialog).toHaveCount(0);
		});

		test('scan/report app header fits with its additional back button', async ({ page }) => {
			const report = loadReportFixture();
			const jobId = report.meta.jobId;
			await page.route(`**/api/v1/jobs/${jobId}`, (route) =>
				route.fulfill({
					contentType: 'application/json',
					body: JSON.stringify({
						id: jobId,
						state: 'done',
						artifacts: {},
						created_at: '2026-01-01T00:00:00Z',
						updated_at: '2026-01-01T00:00:12Z'
					})
				})
			);
			await page.route(`**/api/v1/jobs/${jobId}/results**`, (route) =>
				route.fulfill({ contentType: 'application/json', body: JSON.stringify(report) })
			);
			await page.route(`**/api/v1/jobs/${jobId}/stream**`, (route) =>
				route.fulfill({ contentType: 'text/event-stream', body: 'event: done\ndata: {}\n\n' })
			);
			await page.goto(`/scan/${jobId}/report`);
			await expect(page.getByRole('tab', { name: /Findings/ })).toBeVisible();
			const header = page.locator('.site-header');
			await expectControlsFit(
				header,
				header.locator('.navlink--back, .navmenu__trigger, .themepick__opt')
			);
			await expectViewportFits(page, width);
		});
	});
}

test('desktop keeps the two-column homepage and pinned report notes', async ({ page }) => {
	await page.setViewportSize({ width: 1440, height: 1000 });
	await page.goto('/');
	await expectViewportFits(page, 1440);
	await expect(page.getByRole('button', { name: 'Menu', exact: true })).toBeHidden();
	expect(
		await page
			.locator('.hero__grid')
			.evaluate((element) => getComputedStyle(element).gridTemplateColumns.split(' ').length)
	).toBe(2);
	await page.goto(demoIssueUrl);
	const dialog = page.getByRole('dialog', { name: 'Issue details' });
	await expect(dialog).toBeVisible();
	await expect(dialog.locator('.imodal__content')).toHaveCSS('display', 'contents');
	await expect(dialog.getByRole('button', { name: 'Save note', exact: true })).toBeInViewport();
	await expect(dialog.getByRole('button', { name: 'Close modal' })).toBeInViewport();
});
