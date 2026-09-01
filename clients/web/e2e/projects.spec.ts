import type { Page } from '@playwright/test';
import { expect, loadReportFixture, test } from './fixtures';

const report = loadReportFixture();
const JOB_ID = report.meta.jobId;
const SECOND_JOB_ID = 'e2e-local-project-regression';
const secondReport = structuredClone(report);
secondReport.meta.jobId = SECOND_JOB_ID;

const templateIssue = report.issues[0];
if (!templateIssue) {
	throw new Error('unified-report.v2.all-scans.json must contain at least one issue');
}
secondReport.issues.push({
	...structuredClone(templateIssue),
	id: 'e2e-new-project-issue',
	title: 'New regression from the second project run'
});
secondReport.summary.totalIssues += 1;

interface MockProjectApiOptions {
	/* A second scanner the catalog can gain and lose mid-test, standing in for
	   any scanner a deployment enables or disables between runs. */
	includeOptionalScanner?: boolean;
}

async function mockProjectApi(
	page: Page,
	{ includeOptionalScanner = false }: MockProjectApiOptions = {}
) {
	let submissionCount = 0;
	let optionalScannerAvailable = includeOptionalScanner;
	let catalogGate: Promise<void> | null = null;
	await page.route('**/api/v1/scanners', async (route) => {
		const gate = catalogGate;
		if (gate) await gate;
		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify({
				scanners: [
					{
						id: 'axe',
						name: 'Axe Accessibility',
						version: '1.0.0',
						description: 'WCAG checks',
						categories: ['accessibility'],
						aliases: [],
						enabled: true,
						builtIn: true,
						capabilities: {}
					},
					...(optionalScannerAvailable
						? [
								{
									id: 'seo',
									name: 'SEO',
									version: '1.0.0',
									description: 'Meta tags and headings',
									categories: ['seo'],
									aliases: [],
									enabled: true,
									builtIn: true,
									capabilities: {}
								}
							]
						: [])
				],
				categories: optionalScannerAvailable ? ['accessibility', 'seo'] : ['accessibility']
			})
		});
	});
	await page.route(/\/api\/v1\/jobs\/urls(?:\/[^?]*)?(?:\?.*)?$/, (route) =>
		route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify({ job_id: submissionCount++ === 0 ? JOB_ID : SECOND_JOB_ID })
		})
	);
	for (const [jobId, jobReport] of [
		[JOB_ID, report],
		[SECOND_JOB_ID, secondReport]
	] as const) {
		await page.route(`**/api/v1/jobs/${jobId}`, (route) =>
			route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					id: jobId,
					state: 'done',
					violations: jobReport.summary.totalIssues,
					artifacts: {},
					created_at: '2026-01-01T00:00:00Z',
					updated_at: '2026-01-01T00:00:12Z'
				})
			})
		);
		await page.route(`**/api/v1/jobs/${jobId}/results**`, (route) =>
			route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify(jobReport)
			})
		);
		await page.route(`**/api/v1/jobs/${jobId}/stream**`, (route) =>
			route.fulfill({
				status: 200,
				contentType: 'text/event-stream',
				body: 'event: done\ndata: {}\n\n'
			})
		);
	}

	return {
		setOptionalScannerAvailable(available: boolean) {
			optionalScannerAvailable = available;
		},
		holdNextCatalog() {
			if (catalogGate) throw new Error('A scanner catalog request is already held.');
			let release = () => {};
			const gate = new Promise<void>((resolve) => {
				release = resolve;
			});
			catalogGate = gate;
			return () => {
				if (catalogGate === gate) catalogGate = null;
				release();
			};
		}
	};
}

async function navigateWithinApp(page: Page, path: string) {
	await page.evaluate((nextPath) => {
		window.history.pushState({}, '', nextPath);
		window.dispatchEvent(new PopStateEvent('popstate'));
	}, path);
	await expect.poll(() => new URL(page.url()).pathname + new URL(page.url()).search).toBe(path);
}

test('local project creates a run and promotes its report without persisting credentials', async ({
	page
}) => {
	await mockProjectApi(page);
	await page.goto('/projects');

	await page.getByLabel('Project name').fill('Storefront regression');
	await page.getByLabel('Website URL').fill('https://shop.example.com');
	await page.getByRole('button', { name: 'Create and configure' }).click();

	await expect(page).toHaveURL(/\/playground\?project=/);
	await expect(page.getByLabel('Project name')).toHaveValue('Storefront regression');
	await expect(page.getByRole('button', { name: 'ZIP upload' })).toBeDisabled();

	await page.getByRole('button', { name: /Set up/ }).click();
	await page.getByLabel(/Login URL/).fill('https://shop.example.com/login');
	await page.getByLabel(/Username \/ email/).fill('throwaway@example.com');
	await page.getByLabel(/Password/).fill('not-in-indexeddb');

	await page
		.locator('.dock')
		.getByRole('button', { name: /Run scan/ })
		.click();
	await expect(page).toHaveURL(new RegExp(`/scan/${JOB_ID}(?:/report)?\\?project=`));

	const storedJson = await page.evaluate(async () => {
		const database = await new Promise<IDBDatabase>((resolve, reject) => {
			const request = indexedDB.open('stageflow-local-projects');
			request.onsuccess = () => resolve(request.result);
			request.onerror = () => reject(request.error ?? new Error('IndexedDB request failed'));
		});
		const transaction = database.transaction(['projects', 'runs'], 'readonly');
		const readAll = (store: string) =>
			new Promise<unknown[]>((resolve, reject) => {
				const request = transaction.objectStore(store).getAll();
				request.onsuccess = () => resolve(request.result);
				request.onerror = () => reject(request.error ?? new Error('IndexedDB request failed'));
			});
		const snapshot = await Promise.all([readAll('projects'), readAll('runs')]);
		database.close();
		return JSON.stringify(snapshot);
	});
	expect(storedJson).not.toContain('throwaway@example.com');
	expect(storedJson).not.toContain('not-in-indexeddb');

	const projectQuery = new URL(page.url()).search;
	await page.goto('/projects');
	const projectCard = page.getByRole('article').filter({ hasText: 'Storefront regression' });
	const viewRunLink = projectCard.getByRole('link', { name: 'View run' });
	await expect(viewRunLink).toHaveAttribute('href', `/scan/${JOB_ID}${projectQuery}`);
	await viewRunLink.click();
	await expect(page).toHaveURL(new RegExp(`/scan/${JOB_ID}(?:/report)?\\?project=`));

	await page.goto(`/scan/${JOB_ID}/report${projectQuery}`);
	await expect(page.getByRole('heading', { name: 'Storefront regression' })).toBeVisible();
	await expect(page.getByText('No baseline yet')).toBeVisible();
	await page.getByRole('button', { name: 'Promote as baseline' }).click();
	await expect(page.getByRole('button', { name: 'Current baseline' })).toBeDisabled();
	await expect(page.getByText('This report is now the local baseline.')).toBeVisible();

	await navigateWithinApp(page, `/scan/${SECOND_JOB_ID}/report`);
	await expect(
		page.getByRole('tab', { name: `Findings ${secondReport.summary.totalIssues}` })
	).toBeVisible();
	await expect(page.locator('.local-baseline')).toHaveCount(0);
	await expect(page.getByRole('heading', { name: 'Storefront regression' })).toHaveCount(0);

	await navigateWithinApp(page, `/scan/${JOB_ID}/report${projectQuery}`);
	await expect(page.getByRole('heading', { name: 'Storefront regression' })).toBeVisible();

	await navigateWithinApp(page, `/scan/${SECOND_JOB_ID}/report?project=missing-project`);
	await expect(
		page.getByRole('tab', { name: `Findings ${secondReport.summary.totalIssues}` })
	).toBeVisible();
	await expect(page.locator('.local-baseline')).toHaveCount(0);
	await expect(page.getByRole('heading', { name: 'Storefront regression' })).toHaveCount(0);

	await navigateWithinApp(page, `/scan/${JOB_ID}/report${projectQuery}`);
	await expect(page.getByRole('heading', { name: 'Storefront regression' })).toBeVisible();

	await page.getByRole('link', { name: /Run again/ }).click();
	await expect(page.getByLabel('Project name')).toHaveValue('Storefront regression');
	await page
		.locator('.dock')
		.getByRole('button', { name: /Run scan/ })
		.click();
	await expect(page).toHaveURL(new RegExp(`/scan/${SECOND_JOB_ID}(?:/report)?\\?project=`));
	const secondProjectQuery = new URL(page.url()).search;
	await page.goto(`/scan/${SECOND_JOB_ID}/report${secondProjectQuery}`);
	await expect(page.getByLabel('Changes from baseline').getByText('New issues')).toBeVisible();
	await expect(
		page.getByLabel('Changes from baseline').getByText('1', { exact: true })
	).toBeVisible();
	await expect(page.getByText('New regression from the second project run')).toBeVisible();

	await page.goto('/projects');
	await expect(page.getByRole('heading', { name: 'Storefront regression' })).toBeVisible();
	await expect(page.getByRole('link', { name: 'Latest report' })).toBeVisible();
});

test('temporarily unavailable scanners remain stored but are not submitted', async ({ page }) => {
	const api = await mockProjectApi(page, { includeOptionalScanner: true });
	await page.goto('/projects');

	await page.getByLabel('Project name').fill('Catalog evolution');
	await page.getByLabel('Website URL').fill('https://catalog.example.com');
	await page.getByRole('button', { name: 'Create and configure' }).click();
	await expect(page).toHaveURL(/\/playground\?project=/);
	await expect(page.getByLabel('Project name')).toHaveValue('Catalog evolution');
	const projectId = new URL(page.url()).searchParams.get('project');
	expect(projectId).toBeTruthy();
	if (!projectId) throw new Error('Expected a local project id.');

	const optionalScanner = page.getByRole('checkbox', { name: /SEO/ });
	if ((await optionalScanner.getAttribute('aria-checked')) !== 'true') {
		await optionalScanner.click();
	}
	await page.getByRole('button', { name: 'Save project' }).click();
	await expect(page.getByText('Project configuration saved locally.')).toBeVisible();

	api.setOptionalScannerAvailable(false);
	await page.reload();
	await expect(page.getByLabel('Project name')).toHaveValue('Catalog evolution');
	await expect(page.getByRole('checkbox', { name: /SEO/ })).toHaveCount(0);

	const submission = page.waitForRequest(
		(request) => request.method() === 'POST' && request.url().includes('/api/v1/jobs/urls/')
	);
	await page
		.locator('.dock')
		.getByRole('button', { name: /Run scan/ })
		.click();
	const submittedPayload = (await submission).postDataJSON() as { modules: string[] };
	expect(submittedPayload.modules).toEqual(['axe']);

	const storedScannerConfig = await page.evaluate(async (storedProjectId) => {
		const database = await new Promise<IDBDatabase>((resolve, reject) => {
			const request = indexedDB.open('stageflow-local-projects');
			request.onsuccess = () => resolve(request.result);
			request.onerror = () => reject(request.error ?? new Error('IndexedDB request failed'));
		});
		type StoredProject = {
			configuration: { scanners: { id: string; enabled: boolean; config?: unknown }[] };
		};
		const project = await new Promise<StoredProject>((resolve, reject) => {
			const request = database
				.transaction('projects', 'readonly')
				.objectStore('projects')
				.get(storedProjectId);
			// IDBRequest.result is typed `any`; narrow it once, here at the boundary.
			request.onsuccess = () => resolve(request.result as StoredProject);
			request.onerror = () => reject(request.error ?? new Error('IndexedDB request failed'));
		});
		database.close();
		return project.configuration.scanners.find((scanner) => scanner.id === 'seo');
	}, projectId);
	expect(storedScannerConfig).toMatchObject({ id: 'seo', enabled: true });

	api.setOptionalScannerAvailable(true);
	await page.goto(`/playground?project=${projectId}`);
	await expect(page.getByRole('checkbox', { name: /SEO/ })).toHaveAttribute('aria-checked', 'true');
});

test('SPA project query changes discard project association and execution-only state', async ({
	page
}) => {
	const api = await mockProjectApi(page, { includeOptionalScanner: true });
	await page.goto('/projects');

	await page.getByLabel('Project name').fill('Alpha project');
	await page.getByLabel('Website URL').fill('https://alpha.example.com');
	await page.getByRole('button', { name: 'Create and configure' }).click();
	await expect(page).toHaveURL(/\/playground\?project=/);
	await expect(page.getByLabel('Project name')).toHaveValue('Alpha project');
	const alphaProjectId = new URL(page.url()).searchParams.get('project');
	expect(alphaProjectId).toBeTruthy();

	await page.getByRole('link', { name: 'All projects' }).click();
	await expect(page).toHaveURL(/\/projects$/);
	await page.getByLabel('Project name').fill('Beta project');
	await page.getByLabel('Website URL').fill('https://beta.example.com');
	await page.getByRole('button', { name: 'Create and configure' }).click();
	await expect(page).toHaveURL(/\/playground\?project=/);
	await expect(page.getByLabel('Project name')).toHaveValue('Beta project');
	const betaProjectId = new URL(page.url()).searchParams.get('project');
	expect(betaProjectId).toBeTruthy();

	await page.getByRole('link', { name: 'All projects' }).click();
	await expect(page).toHaveURL(/\/projects$/);
	await page
		.getByRole('article')
		.filter({ hasText: 'Alpha project' })
		.getByRole('link', { name: 'Configure a scan' })
		.click();
	await expect(page.getByLabel('Project name')).toHaveValue('Alpha project');

	await page
		.getByRole('textbox', { name: 'URL 1', exact: true })
		.fill('https://alpha-secret.example.com/private');
	await page.getByRole('button', { name: /Set up/ }).click();
	await page.getByLabel(/Login URL/).fill('https://alpha.example.com/login');
	await page.getByLabel(/Username \/ email/).fill('alpha-secret@example.com');
	await page.getByLabel(/Password/).fill('alpha-project-password');

	const releaseBetaCatalog = api.holdNextCatalog();
	await navigateWithinApp(page, `/playground?project=${betaProjectId}`);
	await expect(page.getByRole('heading', { name: 'Configure a scan' })).toBeVisible();
	releaseBetaCatalog();
	await expect(page.getByLabel('Project name')).toHaveValue('Beta project');
	await expect(page.getByRole('button', { name: /Set up/ })).toBeVisible();
	await expect(page.getByRole('textbox', { name: 'URL 1', exact: true })).toHaveValue(
		'https://beta.example.com'
	);

	await navigateWithinApp(page, '/playground');
	await expect(page.getByRole('heading', { name: 'Configure a scan' })).toBeVisible();
	await expect(page.getByLabel('Project name')).toHaveCount(0);
	await expect(page.getByRole('button', { name: 'ZIP upload' })).toBeEnabled();
	await expect(page.getByRole('textbox', { name: 'URL 1', exact: true })).toHaveValue(
		'https://example.com'
	);
	await expect(page.getByRole('button', { name: /Set up/ })).toBeVisible();

	await page
		.getByRole('textbox', { name: 'URL 1', exact: true })
		.fill('https://standalone-secret.example.com/private');
	await page.getByRole('button', { name: /Set up/ }).click();
	await page.getByLabel(/Login URL/).fill('https://standalone.example.com/login');
	await page.getByLabel(/Username \/ email/).fill('standalone-secret@example.com');
	await page.getByLabel(/Password/).fill('standalone-project-password');
	await page.getByRole('button', { name: 'ZIP upload' }).click();
	await page.locator('input[type="file"]').setInputFiles({
		name: 'private-build.zip',
		mimeType: 'application/zip',
		buffer: Buffer.from('not-a-real-zip')
	});
	await expect(page.getByText('private-build.zip')).toBeVisible();

	const releaseMissingCatalog = api.holdNextCatalog();
	await navigateWithinApp(page, '/playground?project=missing-project');
	await expect(page.getByRole('button', { name: 'URL', exact: true })).toHaveAttribute(
		'aria-pressed',
		'true'
	);
	await expect(page.getByText('private-build.zip')).toHaveCount(0);
	releaseMissingCatalog();
	await expect(page.getByText(/That local project was not found/)).toBeVisible();
	await expect(page.getByLabel('Project name')).toHaveCount(0);
	await expect(page.getByRole('button', { name: 'ZIP upload' })).toBeEnabled();
	await expect(page.getByRole('textbox', { name: 'URL 1', exact: true })).toHaveValue(
		'https://example.com'
	);
	await expect(page.getByRole('button', { name: /Set up/ })).toBeVisible();
	await expect(page.locator('input[type="file"]')).toHaveCount(0);
	await page.getByRole('button', { name: 'ZIP upload' }).click();
	await expect(page.locator('.drop__file')).toHaveText('Choose a ZIP archive');
	await expect(page.locator('input[type="file"]')).toHaveValue('');

	const formValues = await page
		.locator('input, textarea')
		.evaluateAll((fields) =>
			fields.map((field) => (field as HTMLInputElement | HTMLTextAreaElement).value)
		);
	expect(formValues).not.toContain('alpha-secret@example.com');
	expect(formValues).not.toContain('alpha-project-password');
	expect(formValues).not.toContain('standalone-secret@example.com');
	expect(formValues).not.toContain('standalone-project-password');
});

test('local projects can be exported and imported as JSON', async ({ page }) => {
	await mockProjectApi(page);
	await page.goto('/projects');

	await page.getByLabel('Project name').fill('Portable project');
	await page.getByLabel('Website URL').fill('https://portable.example.com');
	await page.getByRole('button', { name: 'Create and configure' }).click();
	await expect(page).toHaveURL(/\/playground\?project=/);

	await page.goto('/projects');
	const card = page.getByRole('article').filter({ hasText: 'Portable project' });
	await expect(card).toBeVisible();

	const downloadPromise = page.waitForEvent('download');
	await card.getByRole('button', { name: 'Export' }).click();
	const download = await downloadPromise;
	expect(download.suggestedFilename()).toMatch(/portable-project/i);
	const exportPath = await download.path();
	expect(exportPath).toBeTruthy();

	await card.getByRole('button', { name: 'Delete Portable project' }).click();
	await page.getByRole('dialog').getByRole('button', { name: 'Delete project' }).click();
	await expect(page.getByText('No projects yet')).toBeVisible();

	await page.locator('.project-import input[type="file"]').setInputFiles(exportPath as string);
	await expect(page.getByRole('article').filter({ hasText: 'Portable project' })).toBeVisible();
});
	await mockProjectApi(page);
	await page.goto('/projects');

	await page.getByLabel('Project name').fill('Doomed project');
	await page.getByLabel('Website URL').fill('https://doomed.example.com');
	await page.getByRole('button', { name: 'Create and configure' }).click();
	await expect(page).toHaveURL(/\/playground\?project=/);

	await page.goto('/projects');
	const card = page.getByRole('article').filter({ hasText: 'Doomed project' });
	await expect(card).toBeVisible();

	// Cancelling keeps the project (Cancel holds initial focus, Escape works too).
	await card.getByRole('button', { name: 'Delete Doomed project' }).click();
	const dialog = page.getByRole('dialog');
	await expect(dialog.getByRole('heading', { name: 'Delete “Doomed project”?' })).toBeVisible();
	await expect(dialog.getByRole('button', { name: 'Cancel' })).toBeFocused();
	await dialog.getByRole('button', { name: 'Cancel' }).click();
	await expect(page.getByRole('dialog')).toHaveCount(0);
	await expect(card).toBeVisible();

	// Confirming removes the card and its stored data.
	await card.getByRole('button', { name: 'Delete Doomed project' }).click();
	await page.getByRole('dialog').getByRole('button', { name: 'Delete project' }).click();
	await expect(page.getByRole('dialog')).toHaveCount(0);
	await expect(page.getByRole('article').filter({ hasText: 'Doomed project' })).toHaveCount(0);
	await expect(page.getByText('No projects yet')).toBeVisible();
});
