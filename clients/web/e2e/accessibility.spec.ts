import AxeBuilder from '@axe-core/playwright';
import type { Page } from '@playwright/test';

import { expect, loadReportFixture, test } from './fixtures';

/*
 * Pin the emulated OS preference.
 *
 * :root declares `color-scheme: light dark`, so an untouched document follows
 * whatever prefers-color-scheme the browser reports. A CI image that reports
 * dark would make the "light" half of this matrix silently run dark twice and
 * ship an entirely unaudited light theme. This line is the difference between
 * a real matrix and a fake one.
 */
test.use({ colorScheme: 'light' });

const fixture = loadReportFixture();
const jobId = fixture.meta.jobId;

const firstIssue = fixture.issues[0];
if (!firstIssue) {
	throw new Error('unified-report.v2.all-scans.json must contain at least one issue');
}
const issueId = firstIssue.id;

interface Surface {
	name: string;
	path: string;
	/** Waits for the surface to actually be on screen before axe looks at it. */
	ready?: (page: Page) => Promise<void>;
}

const SURFACES: Surface[] = [
	{ name: 'home', path: '/' },
	{ name: 'projects', path: '/projects' },
	{
		name: 'playground',
		path: '/playground',
		ready: async (page) => {
			await expect(page.getByRole('heading', { name: 'Configure a scan' })).toBeVisible();
		}
	},
	{
		name: 'report',
		path: `/scan/${jobId}/report?section=issues`,
		ready: async (page) => {
			await expect(page.getByRole('tab', { name: /Findings/ })).toBeVisible();
		}
	},
	{
		/*
		 * The demo renders the full report viewer off committed static assets,
		 * so it is the one surface here that exercises the real thing with no
		 * mocked API in front of it.
		 */
		name: 'demo report',
		path: '/demo',
		ready: async (page) => {
			await expect(page.getByRole('tab', { name: /Findings/ })).toBeVisible();
		}
	},
	{
		name: 'issue modal',
		path: `/scan/${jobId}/report?section=issues&issue=${encodeURIComponent(issueId)}`,
		ready: async (page) => {
			await expect(page.getByRole('dialog', { name: 'Issue details' })).toBeVisible();
		}
	}
];

async function mockCatalog(page: Page) {
	await page.route('**/api/v1/scanners', (route) =>
		route.fulfill({
			contentType: 'application/json',
			body: JSON.stringify({
				scanners: [
					{
						id: 'axe',
						name: 'Axe Accessibility',
						version: '1.0.0',
						description: 'WCAG accessibility checks',
						categories: ['accessibility'],
						aliases: [],
						enabled: true,
						builtIn: true,
						capabilities: {
							supportsConcurrency: true,
							estimatedTimePerPage: 5000
						}
					}
				],
				categories: ['accessibility']
			})
		})
	);
}

async function mockReport(page: Page) {
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
		route.fulfill({ contentType: 'application/json', body: JSON.stringify(fixture) })
	);
	await page.route(`**/api/v1/jobs/${jobId}/stream**`, (route) =>
		route.fulfill({
			status: 200,
			contentType: 'text/event-stream',
			body: 'event: done\ndata: {}\n\n'
		})
	);
}

async function expectNoSeriousViolations(page: Page, surface: string) {
	const result = await new AxeBuilder({ page })
		.withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
		.analyze();
	const violations = result.violations.filter(
		(violation) => violation.impact === 'serious' || violation.impact === 'critical'
	);
	expect(
		violations.map(({ id, impact, nodes }) => ({
			id,
			impact,
			targets: nodes.map((node) => node.target)
		})),
		`${surface} has serious or critical accessibility violations`
	).toEqual([]);
}

/*
 * Both themes off one page load.
 *
 * axe only grades the colors actually painted at the moment it runs, so a
 * theme it never sees is a theme nobody checks. Flipping the attribute in
 * place rather than reloading keeps navigations at 10 while doubling the
 * audits -- navigation dominates the wall clock here, not analysis.
 *
 * The attribute is set directly instead of through the toggle: this asserts
 * the palette, and routing it through React would make a rendering bug in the
 * control look like a contrast failure on every surface.
 */
async function auditBothThemes(page: Page, surface: string) {
	await expectNoSeriousViolations(page, `${surface} (light)`);

	await page.evaluate(() => {
		document.documentElement.dataset.theme = 'dark';
	});
	await page.waitForFunction(
		() => getComputedStyle(document.documentElement).colorScheme === 'dark'
	);
	await expectNoSeriousViolations(page, `${surface} (dark)`);
}

for (const viewport of [
	{ name: 'desktop', width: 1440, height: 1000 },
	{ name: 'mobile', width: 390, height: 844 }
] as const) {
	test(`${viewport.name} product surfaces pass serious accessibility checks in both themes`, async ({
		page
	}) => {
		/* Six surfaces × two themes = twelve axe runs in one body; under parallel
		   workers that legitimately brushes the default 30s and flakes. */
		test.slow();
		await page.setViewportSize(viewport);
		await mockCatalog(page);
		await mockReport(page);

		for (const surface of SURFACES) {
			await page.goto(surface.path);
			await surface.ready?.(page);
			await auditBothThemes(page, `${viewport.name} ${surface.name}`);
		}
	});
}

/*
 * The only end-to-end exercise of choice -> storage -> init script -> paint.
 * The unit tests cover the store and the matrix above covers the palette;
 * neither one proves a returning visitor gets what they picked.
 */
test('an explicit theme choice survives a reload without a flash of the wrong one', async ({
	page
}) => {
	await page.goto('/');

	await page.getByRole('radio', { name: 'Dark' }).check();

	await expect.poll(() => page.evaluate(() => document.documentElement.dataset.theme)).toBe('dark');
	await expect
		.poll(() => page.evaluate(() => localStorage.getItem('stageflow-theme')))
		.toBe('dark');

	await page.reload();

	await expect(page.getByRole('radio', { name: 'Dark' })).toBeChecked();
	expect(await page.evaluate(() => document.documentElement.dataset.theme)).toBe('dark');

	/*
	 * No flash, structurally rather than by timing: the served document carries
	 * the init script inside <head>, so the attribute is on <html> before the
	 * parser reaches <body> and long before the deferred React bundle runs.
	 * Nothing can paint light first.
	 */
	const document_ = await (await page.request.get('/')).text();
	const scriptAt = document_.indexOf('stageflow-theme');
	const bodyAt = document_.indexOf('<body');
	expect(scriptAt, 'the theme init script is missing from the served document').toBeGreaterThan(-1);
	expect(scriptAt, 'the theme init script must run before <body> is parsed').toBeLessThan(bodyAt);
});

/*
 * The narrow-viewport nav, which the surface matrix cannot reach: axe only
 * grades what is painted, and the panel does not exist until it is opened.
 * Below 960px this IS the primary navigation -- the links are display:none --
 * so a regression here removes every route off the home page on a phone.
 */
test.describe('the narrow-screen menu', () => {
	test.use({ viewport: { width: 390, height: 844 } });

	test('is the primary navigation, traps focus, and closes on Escape', async ({ page }) => {
		await page.goto('/');

		// Scoped to the header: the footer sitemap carries the same link names and
		// is visible at every width.
		const header = page.locator('.site-header');
		const trigger = page.getByRole('button', { name: 'Menu' });
		await expect(trigger).toHaveAttribute('aria-expanded', 'false');
		// The inline row is hidden at this width, so the menu is the only way through.
		await expect(header.getByRole('link', { name: 'Configure a scan' })).toBeHidden();

		await trigger.click();
		await expect(trigger).toHaveAttribute('aria-expanded', 'true');
		await expect(header.getByRole('link', { name: 'Configure a scan' })).toBeVisible();
		await expect(header.getByRole('link', { name: 'Demo report' })).toBeVisible();

		await expectNoSeriousViolations(page, 'mobile menu (light)');
		await page.evaluate(() => {
			document.documentElement.dataset.theme = 'dark';
		});
		await page.waitForFunction(
			() => getComputedStyle(document.documentElement).colorScheme === 'dark'
		);
		await expectNoSeriousViolations(page, 'mobile menu (dark)');

		// Focus starts inside the panel and Tab cycles rather than escaping to the
		// page behind it.
		await expect(header.getByRole('link', { name: 'Projects' })).toBeFocused();
		for (let i = 0; i < 8; i += 1) await page.keyboard.press('Tab');
		await expect(page.locator('.navmenu__panel :focus')).toHaveCount(1);

		await page.keyboard.press('Escape');
		await expect(trigger).toHaveAttribute('aria-expanded', 'false');
		// Focus comes home, or the next Tab restarts from the top of the document.
		await expect(trigger).toBeFocused();
	});

	test('a link navigates and leaves the menu closed behind it', async ({ page }) => {
		await page.goto('/');
		const header = page.locator('.site-header');

		await page.getByRole('button', { name: 'Menu' }).click();
		await header.getByRole('link', { name: 'Projects' }).click();
		await expect(page).toHaveURL(/\/projects$/);

		// /projects keeps the marketing header, so the trigger is still there and
		// the panel must not have survived the navigation.
		await expect(page.getByRole('button', { name: 'Menu' })).toHaveAttribute(
			'aria-expanded',
			'false'
		);
		await expect(page.locator('.navmenu__panel')).toHaveCount(0);
	});
});

/*
 * The band between the phone tests above and the 1440px surface matrix, which
 * neither one covered. The inline row needs ~950px to lay out and nothing in it
 * shrinks or wraps, so while the swap sat at 640px every width from 641 to ~950
 * -- iPad portrait included -- pushed the GitHub button off-screen and scrolled
 * the whole page sideways. Both ends of the swap are pinned here because moving
 * the breakpoint in either direction reintroduces one side of the bug.
 */
test.describe('the nav swap band', () => {
	const horizontalOverflow = (page: Page) =>
		page.evaluate(
			() => document.documentElement.scrollWidth - document.documentElement.clientWidth
		);

	for (const width of [700, 768, 820, 900] as const) {
		test(`at ${width}px the menu replaces the inline row and nothing overflows`, async ({
			page
		}) => {
			await page.setViewportSize({ width, height: 900 });
			await page.goto('/');

			const header = page.locator('.site-header');
			await expect(page.getByRole('button', { name: 'Menu' })).toBeVisible();
			await expect(header.getByRole('link', { name: 'Configure a scan' })).toBeHidden();
			expect(await horizontalOverflow(page)).toBe(0);
		});
	}

	test('at 1024px the inline row is back and the trigger is gone', async ({ page }) => {
		await page.setViewportSize({ width: 1024, height: 900 });
		await page.goto('/');

		const header = page.locator('.site-header');
		await expect(header.getByRole('link', { name: 'Configure a scan' })).toBeVisible();
		await expect(page.getByRole('button', { name: 'Menu' })).toBeHidden();
		expect(await horizontalOverflow(page)).toBe(0);
	});
});
