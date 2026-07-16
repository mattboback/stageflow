import fs from 'node:fs';
import path from 'node:path';

import AxeBuilder from '@axe-core/playwright';
import type { Page } from '@playwright/test';

import { expect, test } from './fixtures';

const fixture = JSON.parse(
	fs.readFileSync(
		path.resolve(process.cwd(), '../../libs/contracts/report/fixtures/unified-report.v2.all-scans.json'),
		'utf8'
	)
);
const jobId = fixture.meta.jobId as string;
const issueId = fixture.issues[0]?.id as string;

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
		violations.map(({ id, impact, nodes }) => ({ id, impact, targets: nodes.map((node) => node.target) })),
		`${surface} has serious or critical accessibility violations`
	).toEqual([]);
}

for (const viewport of [
	{ name: 'desktop', width: 1440, height: 1000 },
	{ name: 'mobile', width: 390, height: 844 }
] as const) {
	test(`${viewport.name} product surfaces pass serious accessibility checks`, async ({ page }) => {
		await page.setViewportSize(viewport);
		await mockCatalog(page);
		await mockReport(page);

		await page.goto('/');
		await expectNoSeriousViolations(page, `${viewport.name} home`);

		await page.goto('/projects');
		await expectNoSeriousViolations(page, `${viewport.name} projects`);

		await page.goto('/playground');
		await expect(page.getByRole('heading', { name: 'Configure a scan' })).toBeVisible();
		await expectNoSeriousViolations(page, `${viewport.name} playground`);

		await page.goto(`/scan/${jobId}/report?section=issues`);
		await expect(page.getByRole('tab', { name: /Findings/ })).toBeVisible();
		await expectNoSeriousViolations(page, `${viewport.name} report`);

		await page.goto(`/scan/${jobId}/report?section=issues&issue=${encodeURIComponent(issueId)}`);
		await expect(page.getByRole('dialog', { name: 'Issue details' })).toBeVisible();
		await expectNoSeriousViolations(page, `${viewport.name} issue modal`);
	});
}
