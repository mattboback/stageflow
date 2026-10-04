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
	await page.route(
		(url) => url.pathname === '/api/v1/jobs/urls',
		(route) =>
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
		const transaction = database.transaction(['projects', 'projectRuns'], 'readonly');
		const readAll = (store: string) =>
			new Promise<unknown[]>((resolve, reject) => {
				const request = transaction.objectStore(store).getAll();
				request.onsuccess = () => resolve(request.result);
				request.onerror = () => reject(request.error ?? new Error('IndexedDB request failed'));
			});
		const snapshot = await Promise.all([readAll('projects'), readAll('projectRuns')]);
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
	await expect(page.locator('.baseline-diff').getByRole('heading')).toContainText(/\b1 new\b/);
	await expect(page.getByText('New regression from the second project run')).toBeVisible();

	await page.goto('/projects');
	await expect(page.getByRole('heading', { name: 'Storefront regression' })).toBeVisible();
	await expect(page.getByRole('link', { name: 'Latest report' })).toBeVisible();
});

test('removed scanners are dropped from the project and not submitted', async ({ page }) => {
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
		(request) =>
			request.method() === 'POST' && new URL(request.url()).pathname === '/api/v1/jobs/urls'
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
	// Running the scan saved the project, which forgot the removed scanner.
	expect(storedScannerConfig).toBeUndefined();

	api.setOptionalScannerAvailable(true);
	await page.goto(`/playground?project=${projectId}`);
	await expect(page.getByRole('checkbox', { name: /SEO/ })).toHaveAttribute(
		'aria-checked',
		'false'
	);
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

	await page.locator('.project-import input[type="file"]').setInputFiles(exportPath);
	await expect(page.getByRole('article').filter({ hasText: 'Portable project' })).toBeVisible();
});

test('project deletion is confirmed in a dialog instead of window.confirm', async ({ page }) => {
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

test('legacy local runs migrate without loss and repeated imports do not overwrite a project', async ({
	page
}) => {
	await mockProjectApi(page);
	await page.goto('/');
	const imported = await page.evaluate(async (storedReport) => {
		const now = '2026-09-30T00:00:00.000Z';
		const project = {
			id: 'legacy-project',
			schemaVersion: 1,
			name: 'Legacy portable history',
			configuration: {
				urls: ['https://example.com'],
				scanners: [{ id: 'axe', enabled: true }],
				browser: 'chromium',
				highlightStyle: 'solid'
			},
			createdAt: now,
			updatedAt: now
		};
		const run = {
			projectId: project.id,
			jobId: storedReport.meta.jobId,
			configFingerprint: 'legacy',
			status: 'complete',
			createdAt: now,
			report: storedReport
		};
		await new Promise<void>((resolve, reject) => {
			const request = indexedDB.open('stageflow-local-projects', 1);
			request.onupgradeneeded = () => {
				const database = request.result;
				database
					.createObjectStore('projects', { keyPath: 'id' })
					.createIndex('updatedAt', 'updatedAt');
				database.createObjectStore('baselines', { keyPath: 'projectId' });
				const runs = database.createObjectStore('runs', { keyPath: 'jobId' });
				runs.createIndex('projectId', 'projectId');
				runs.createIndex('createdAt', 'createdAt');
				request.transaction?.objectStore('projects').put(project);
				request.transaction?.objectStore('runs').put(run);
			};
			request.onsuccess = () => {
				request.result.close();
				resolve();
			};
			request.onerror = () => reject(request.error ?? new Error('IndexedDB request failed'));
		});
		return {
			schema: 'stageflow/local-project@v1',
			exportedAt: now,
			project,
			baseline: null,
			runs: [run]
		};
	}, report);
	await page.goto('/projects');
	await expect(
		page.getByRole('article').filter({ hasText: 'Legacy portable history' })
	).toHaveCount(1);
	for (let count = 2; count <= 3; count++) {
		await page.locator('.project-import input[type="file"]').setInputFiles({
			name: 'legacy.json',
			mimeType: 'application/json',
			buffer: Buffer.from(JSON.stringify(imported))
		});
		await expect(
			page.getByRole('article').filter({ hasText: 'Legacy portable history' })
		).toHaveCount(count);
	}
	const snapshot = await page.evaluate(async () => {
		const database = await new Promise<IDBDatabase>((resolve, reject) => {
			const request = indexedDB.open('stageflow-local-projects');
			request.onsuccess = () => resolve(request.result);
			request.onerror = () => reject(request.error ?? new Error('IndexedDB request failed'));
		});
		const request = database
			.transaction('projectRuns', 'readonly')
			.objectStore('projectRuns')
			.getAll();
		const records = await new Promise<Array<{ projectId: string; jobId: string }>>(
			(resolve, reject) => {
				request.onsuccess = () =>
					resolve(request.result as Array<{ projectId: string; jobId: string }>);
				request.onerror = () => reject(request.error ?? new Error('IndexedDB request failed'));
			}
		);
		database.close();
		return records;
	});
	expect(snapshot).toHaveLength(3);
	expect(new Set(snapshot.map((run) => run.projectId)).size).toBe(3);
	expect(snapshot.find((run) => run.projectId === 'legacy-project')?.jobId).toBe(JOB_ID);
});

test('portable audit JSON imports into a new project with review notes and archived screenshots', async ({
	page
}) => {
	await mockProjectApi(page);
	await page.goto('/projects');
	const audit = {
		schema: 'stageflow/portable-audit@v1',
		jobId: JOB_ID,
		createdAt: '2026-09-30',
		report,
		baseline: { ...report, meta: { ...report.meta, jobId: 'prior-audit-baseline' } },
		verification: {
			jobId: JOB_ID,
			sourceJobId: 'original-audit-source',
			createdAt: 'today',
			issue: templateIssue,
			sourceScannerVersion: '1.0',
			expectedFinalUrl: templateIssue.pageUrl,
			sourceContext: {
				expectedPages: [templateIssue.pageId],
				identityScope: 'path',
				scannerConfigFingerprints: { [templateIssue.scanner]: 'scanner-config' },
				pageConfigFingerprints: { [templateIssue.pageId]: 'page-config' },
				purpose: 'full',
				authenticated: false,
				inputType: 'urls'
			}
		},
		screenshots: [
			{
				artifact_id: 'saved-overview',
				kind: 'page_overview',
				scanner_id: 'axe',
				page_id: report.pages[0]?.id ?? 'page',
				url: 'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+a1XkAAAAASUVORK5CYII='
			}
		],
		missingArtifacts: [],
		reviews: {
			[JOB_ID]: { [templateIssue.id]: { at: '2026-09-30', note: 'Reviewed in the portable audit' } }
		}
	};
	await page.locator('.project-import input[type="file"]').setInputFiles({
		name: 'audit.json',
		mimeType: 'application/json',
		buffer: Buffer.from(JSON.stringify(audit))
	});
	const card = page.getByRole('article').filter({ hasText: 'Imported audit:' });
	await expect(card).toHaveCount(1);
	const importedSnapshot = await page.evaluate(async () => {
		const database = await new Promise<IDBDatabase>((resolve, reject) => {
			const request = indexedDB.open('stageflow-local-projects');
			request.onsuccess = () => resolve(request.result);
			request.onerror = () => reject(request.error ?? new Error('IndexedDB failed'));
		});
		const transaction = database.transaction(['projectRuns', 'reviews'], 'readonly');
		const read = (name: string) =>
			new Promise<unknown[]>((resolve, reject) => {
				const request = transaction.objectStore(name).getAll();
				request.onsuccess = () => resolve(request.result);
				request.onerror = () => reject(request.error ?? new Error('IndexedDB failed'));
			});
		const result = await Promise.all([read('projectRuns'), read('reviews')]);
		database.close();
		return JSON.stringify(result);
	});
	expect(importedSnapshot).toContain('saved-overview');
	expect(importedSnapshot).toContain('prior-audit-baseline');
	expect(importedSnapshot).toContain('original-audit-source');
	expect(importedSnapshot).toContain('Reviewed in the portable audit');
	expect(importedSnapshot).toContain('"archived":true');
	const downloadPromise = page.waitForEvent('download');
	await card.getByRole('button', { name: 'Export' }).click();
	const download = await downloadPromise;
	const path = await download.path();
	expect(path).toBeTruthy();
	// Downloaded project JSON must preserve reviews and raw reports for reuse.
	await page.locator('.project-import input[type="file"]').setInputFiles(path);
	await expect(page.getByRole('article').filter({ hasText: 'Imported audit:' })).toHaveCount(2);
});

test('project review edits from two tabs preserve both notes', async ({ page, context }) => {
	await mockProjectApi(page);
	await page.goto('/projects');
	const secondIssue = report.issues.find((issue) => issue.id !== templateIssue.id);
	if (!secondIssue) throw new Error('Fixture requires two distinct findings');
	const audit = {
		schema: 'stageflow/portable-audit@v1',
		jobId: JOB_ID,
		createdAt: '2026-09-30',
		report,
		screenshots: [],
		missingArtifacts: [],
		reviews: {}
	};
	await page.locator('.project-import input[type="file"]').setInputFiles({
		name: 'two-tabs.json',
		mimeType: 'application/json',
		buffer: Buffer.from(JSON.stringify(audit))
	});
	const card = page.getByRole('article').filter({ hasText: 'Imported audit:' });
	const reportPath = await card.getByRole('link', { name: 'Latest report' }).getAttribute('href');
	if (!reportPath) throw new Error('Expected imported report link');
	const other = await context.newPage();
	await mockProjectApi(other);
	await page.goto(`${reportPath}&section=issues&issue=${encodeURIComponent(templateIssue.id)}`);
	await other.goto(`${reportPath}&section=issues&issue=${encodeURIComponent(secondIssue.id)}`);
	await expect(page.getByLabel('Review note (optional)')).toBeVisible();
	await expect(other.getByLabel('Review note (optional)')).toBeVisible();
	await page.getByLabel('Review note (optional)').fill('First tab note');
	await page.getByRole('button', { name: 'Save note', exact: true }).click();
	await other.getByLabel('Review note (optional)').fill('Second tab note');
	await other.getByRole('button', { name: 'Save note', exact: true }).click();
	const persistedNotes = () =>
		page.evaluate(async () => {
			const database = await new Promise<IDBDatabase>((resolve, reject) => {
				const request = indexedDB.open('stageflow-local-projects');
				request.onsuccess = () => resolve(request.result);
				request.onerror = () => reject(request.error ?? new Error('IndexedDB failed'));
			});
			const request = database.transaction('reviews', 'readonly').objectStore('reviews').getAll();
			const values = await new Promise<unknown[]>((resolve, reject) => {
				request.onsuccess = () => resolve(request.result);
				request.onerror = () => reject(request.error ?? new Error('IndexedDB failed'));
			});
			database.close();
			return JSON.stringify(values);
		});
	await expect.poll(persistedNotes).toContain('First tab note');
	await expect.poll(persistedNotes).toContain('Second tab note');
	await navigateWithinApp(
		page,
		`${reportPath}&section=issues&issue=${encodeURIComponent(secondIssue.id)}`
	);
	await expect(page.getByLabel('Review note (optional)')).toHaveValue('Second tab note');
	await other.close();
});
