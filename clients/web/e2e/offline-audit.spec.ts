import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { pathToFileURL } from 'node:url';
import { renderPortableAuditHTML } from '../app/lib/portable-audit';
import { expect, loadReportFixture, test } from './fixtures';

test('hosted export embeds the bundled viewer without requesting obsolete asset URLs', async ({
	page
}) => {
	const report = loadReportFixture();
	const jobId = report.meta.jobId;
	const obsoleteRequests: string[] = [];
	page.on('request', (request) => {
		if (/\/assets\/offline-viewer\.(js|css)$/.test(request.url()))
			obsoleteRequests.push(request.url());
	});
	await page.route(`**/api/v1/jobs/${jobId}`, (route) =>
		route.fulfill({
			json: {
				id: jobId,
				state: 'done',
				artifacts: {},
				created_at: '2026-10-05T00:00:00Z',
				updated_at: '2026-10-05T00:00:12Z'
			}
		})
	);
	await page.route(`**/api/v1/jobs/${jobId}/results**`, (route) => route.fulfill({ json: report }));
	await page.route(`**/api/v1/jobs/${jobId}/stream**`, (route) =>
		route.fulfill({ contentType: 'text/event-stream', body: 'event: done\ndata: {}\n\n' })
	);
	await page.goto(`/scan/${jobId}/report`);
	const downloaded = page.waitForEvent('download');
	await page.getByRole('button', { name: 'Download interactive audit' }).click();
	const filename = await (await downloaded).path();
	expect(filename).toBeTruthy();
	const [html, javascript, css] = await Promise.all([
		readFile(filename, 'utf8'),
		readFile('dist/offline-viewer/offline-viewer.js', 'utf8'),
		readFile('dist/offline-viewer/offline-viewer.css', 'utf8')
	]);
	expect(html).toContain(javascript.replace(/<\/script/gi, '<\\/script'));
	expect(html).toContain(css.replace(/<\/style/gi, '<\\/style'));
	expect(obsoleteRequests).toEqual([]);
});

test('interactive HTML opens without a network and preserves edits after download and reopen', async ({
	page,
	context
}) => {
	// This full round trip includes editing, downloading, and booting a second offline viewer.
	test.slow();
	const directory = await mkdtemp(join(tmpdir(), 'stageflow-offline-'));
	try {
		const report = loadReportFixture();
		const pageInfo = report.pages[0]!;
		report.issues = [
			{
				id: 'offline-contrast',
				scanner: 'axe',
				ruleId: 'color-contrast',
				severity: 'moderate',
				title: 'Check text contrast offline',
				description: 'Review captured text and background colors.',
				pageId: pageInfo.id,
				pageUrl: pageInfo.url,
				elementCount: 1,
				occurrences: [
					{
						selector: 'p',
						html: '<p>Example</p>',
						boundingBox: { x: 0, y: 0, width: 1, height: 1 }
					}
				],
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
			}
		];
		report.summary.totalIssues = 1;
		const [javascript, css] = await Promise.all([
			readFile('dist/offline-viewer/offline-viewer.js', 'utf8'),
			readFile('dist/offline-viewer/offline-viewer.css', 'utf8')
		]);
		const filename = join(directory, 'audit.html');
		await writeFile(
			filename,
			renderPortableAuditHTML(
				{
					schema: 'stageflow/portable-audit@v1',
					jobId: report.meta.jobId,
					createdAt: '2026-09-30T12:00:00Z',
					report,
					reviews: {},
					missingArtifacts: [],
					screenshots: [
						{
							kind: 'page_overview',
							artifact_id: 'screenshot',
							scanner_id: 'axe',
							page_id: pageInfo.id,
							url: 'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jRZkAAAAASUVORK5CYII='
						}
					]
				},
				{ javascript, css }
			)
		);
		const networkRequests: string[] = [];
		page.on('request', (request) => {
			if (/^https?:/.test(request.url())) networkRequests.push(request.url());
		});
		await context.setOffline(true);
		await page.goto(pathToFileURL(filename).href);
		await expect(page.getByText('Interactive offline audit', { exact: true })).toBeVisible();
		// The viewer bundle has to carry the report stylesheets, not just the tokens:
		// without them the review workspace and the modal render as plain flow text.
		await expect(page.locator('.vrev')).toHaveCSS('display', 'grid');
		await page.getByRole('button', { name: /Review human checks/ }).click();
		await expect(page.getByRole('dialog')).toBeVisible();
		await expect(page.locator('.imodal__backdrop')).toHaveCSS('position', 'fixed');
		await expect(page.getByRole('tab', { name: 'Verify' })).toHaveAttribute(
			'aria-selected',
			'true'
		);
		await page.getByRole('button', { name: 'Mark pass' }).click();
		await page.getByLabel('Review note (optional)').fill('Confirmed offline </script> safely.');
		await page.getByRole('button', { name: 'Save note' }).click();
		await page.getByRole('button', { name: 'Close modal' }).click();
		await expect(page.getByText(/Unsaved review changes/)).toBeVisible();
		const downloaded = page.waitForEvent('download');
		await page.getByRole('button', { name: 'Download updated report' }).click();
		const updated = join(directory, 'updated.html');
		await (await downloaded).saveAs(updated);
		await page.goto(pathToFileURL(updated).href);
		await page.getByRole('tab', { name: /Findings/ }).click();
		await page.locator('button.irow').first().click();
		await expect(page.getByLabel('Review note (optional)')).toHaveValue(
			'Confirmed offline </script> safely.'
		);
		await expect(page.getByText('Verified · pass', { exact: true })).toBeVisible();
		expect(networkRequests).toEqual([]);
	} finally {
		await rm(directory, { recursive: true, force: true });
	}
});
