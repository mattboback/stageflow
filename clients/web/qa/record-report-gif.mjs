#!/usr/bin/env node
// Records docs/images/report-bounding-box.gif: clicking a bounding-box
// overlay in Review opens the finding detail card for that element.
//
// The API is fully mocked from the committed report fixture (the same
// pattern as e2e/report.spec.ts) so no backend is required. The page-overview
// screenshot is a real Playwright capture of https://example.com, and the
// bounding boxes are the real DOM coordinates of elements on that page, so
// the overlay lines up with genuine content rather than a fabricated image.
//
// Requires: `bun run build` in clients/web, and ffmpeg on PATH.
//
// Usage: node clients/web/qa/record-report-gif.mjs [output.gif]

import { chromium } from '@playwright/test';
import { spawn, execFileSync } from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const webRoot = path.resolve(__dirname, '..');
const repoRoot = path.resolve(webRoot, '..', '..');
const OUT = process.argv[2] ? path.resolve(process.argv[2]) : path.join(repoRoot, 'docs/images/report-bounding-box.gif');

const PORT = 4174;
const VIEWPORT = { width: 1280, height: 860 };

function waitForServer(url, timeoutMs) {
	const deadline = Date.now() + timeoutMs;
	const attempt = async () => {
		try {
			const res = await fetch(url);
			if (res.ok) return;
		} catch {
			// server not ready yet
		}
		if (Date.now() > deadline) throw new Error(`timed out waiting for ${url}`);
		await new Promise((r) => setTimeout(r, 300));
		await attempt();
	};
	return attempt();
}

async function main() {
	console.log('==> Building clients/web...');
	execFileSync('bun', ['run', 'build'], { cwd: webRoot, stdio: 'inherit' });

	const browser = await chromium.launch();

	console.log('==> Capturing a real page-overview screenshot from example.com...');
	const exampleContext = await browser.newContext({ viewport: VIEWPORT });
	const examplePage = await exampleContext.newPage();
	await examplePage.goto('https://example.com', { waitUntil: 'networkidle' });

	const h1Box = await examplePage.locator('h1').boundingBox();
	const pBox = await examplePage.locator('p').first().boundingBox();
	const aBox = await examplePage.locator('a').first().boundingBox();
	const pageDims = await examplePage.evaluate(() => ({
		width: document.documentElement.scrollWidth,
		height: document.documentElement.scrollHeight
	}));
	const screenshotBuffer = await examplePage.screenshot();
	await exampleContext.close();

	console.log('==> Building the mocked report...');
	const fixturePath = path.join(
		repoRoot,
		'libs/contracts/report/fixtures/unified-report.v2.all-scans.json'
	);
	const report = JSON.parse(fs.readFileSync(fixturePath, 'utf8'));
	const jobId = report.meta.jobId;

	const page1 = report.pages.find((p) => p.id === 'page-1');
	const page1Issues = report.issues.filter((i) => i.pageId === 'page-1');

	const boxes = [h1Box, pBox, aBox];
	const elements = boxes.map((box, i) => {
		const issue = page1Issues[i];
		return {
			issueId: issue.id,
			ruleId: issue.ruleId,
			severity: issue.severity,
			selector: '',
			nodeIndex: 0,
			x: box.x,
			y: box.y,
			width: box.width,
			height: box.height,
			xPercent: (box.x / pageDims.width) * 100,
			yPercent: (box.y / pageDims.height) * 100,
			widthPercent: (box.width / pageDims.width) * 100,
			heightPercent: (box.height / pageDims.height) * 100
		};
	});

	page1.pageOverview = {
		screenshotFilename: 'page-overview.png',
		pageWidth: pageDims.width,
		pageHeight: pageDims.height,
		elements
	};

	const jobStatus = {
		id: jobId,
		state: 'done',
		violations: report.summary.totalIssues,
		artifacts: {
			screenshots: [
				{
					artifact_id: 'shot-page-1-overview',
					scanner_id: 'axe',
					page_id: 'page-1',
					kind: 'page_overview',
					url: '/mock/page-overview.png'
				}
			]
		},
		created_at: '2026-01-01T00:00:00Z',
		updated_at: '2026-01-01T00:00:12Z'
	};

	console.log('==> Starting the preview server...');
	const preview = spawn(
		'bun',
		['run', 'preview', '--', '--host', '127.0.0.1', '--port', String(PORT), '--strictPort'],
		{ cwd: webRoot, stdio: 'ignore' }
	);
	const killPreview = () => {
		if (!preview.killed) preview.kill();
	};
	process.on('exit', killPreview);

	try {
		await waitForServer(`http://127.0.0.1:${PORT}/`, 30_000);

		const videoDir = fs.mkdtempSync(path.join(os.tmpdir(), 'stageflow-report-gif-'));
		const context = await browser.newContext({
			viewport: VIEWPORT,
			recordVideo: { dir: videoDir, size: VIEWPORT }
		});
		const page = await context.newPage();

		await page.route(`**/api/v1/jobs/${jobId}`, (route) =>
			route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(jobStatus) })
		);
		await page.route(`**/api/v1/jobs/${jobId}/results**`, (route) =>
			route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(report) })
		);
		await page.route(`**/api/v1/jobs/${jobId}/stream**`, (route) =>
			route.fulfill({ status: 404, body: 'no stream' })
		);
		await page.route('**/mock/page-overview.png', (route) =>
			route.fulfill({ status: 200, contentType: 'image/png', body: screenshotBuffer })
		);

		console.log('==> Recording the interaction...');
		await page.goto(`http://127.0.0.1:${PORT}/scan/${jobId}/report`);
		await page.getByRole('heading', { name: 'Scan report' }).waitFor();
		await page.waitForTimeout(500);

		await page.locator('#report-tab-pages').click();
		await page.locator('.vrev__svg rect').first().waitFor();
		await page.waitForTimeout(900);

		await page.locator('.vrev__svg rect').first().click();
		await page.getByRole('dialog', { name: 'Issue details' }).waitFor();
		await page.waitForTimeout(1400);

		await context.close();
		await browser.close();

		const [webmFile] = fs.readdirSync(videoDir).filter((f) => f.endsWith('.webm'));
		const webmPath = path.join(videoDir, webmFile);

		console.log('==> Converting to GIF...');
		execFileSync('ffmpeg', [
			'-y',
			'-i',
			webmPath,
			'-vf',
			'fps=12,scale=960:-1:flags=lanczos,split[s0][s1];[s0]palettegen[p];[s1][p]paletteuse',
			'-loop',
			'0',
			OUT
		]);
		fs.rmSync(videoDir, { recursive: true, force: true });

		const { size } = fs.statSync(OUT);
		console.log(`==> Wrote ${OUT} (${(size / 1024).toFixed(0)} KiB)`);
	} finally {
		killPreview();
	}
}

main().catch((err) => {
	console.error(err);
	process.exit(1);
});
