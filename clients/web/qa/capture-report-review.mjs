#!/usr/bin/env node
// Captures docs/images/report-review.png: the Review workspace as a visitor
// first sees it at /demo.
//
// /demo renders the report committed under public/demo, so no API, mock, or
// network access is required — the image is whatever the built client shows.
//
// Requires: Bun and Playwright Chromium.
//
// Usage: node clients/web/qa/capture-report-review.mjs [output.png] [--no-build]

import { chromium } from '@playwright/test';
import { spawn, execFileSync } from 'node:child_process';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const webRoot = path.resolve(__dirname, '..');
const repoRoot = path.resolve(webRoot, '..', '..');

const args = process.argv.slice(2);
const skipBuild = args.includes('--no-build');
const outArg = args.find((arg) => !arg.startsWith('--'));
const OUT = outArg ? path.resolve(outArg) : path.join(repoRoot, 'docs/images/report-review.png');

const PORT = 4175;
const VIEWPORT = { width: 1600, height: 1266 };

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
	if (!skipBuild) {
		console.log('==> Building clients/web...');
		execFileSync('bun', ['run', 'build'], { cwd: webRoot, stdio: 'inherit' });
	}

	console.log('==> Starting the static server...');
	const server = spawn('node', ['qa/serve-build.mjs'], {
		cwd: webRoot,
		env: { ...process.env, PORT: String(PORT) },
		stdio: 'ignore'
	});
	const killServer = () => {
		if (!server.killed) server.kill();
	};
	process.on('exit', killServer);

	try {
		await waitForServer(`http://127.0.0.1:${PORT}/`, 30_000);

		const browser = await chromium.launch();
		// Pinned to light so the image does not depend on the host's theme.
		const context = await browser.newContext({ viewport: VIEWPORT, colorScheme: 'light' });
		const page = await context.newPage();

		console.log('==> Capturing the Review workspace...');
		await page.goto(`http://127.0.0.1:${PORT}/demo`, { waitUntil: 'networkidle' });
		await page.locator('.vrev__svg rect').first().waitFor();
		await page.screenshot({ path: OUT });

		await browser.close();
		console.log(`==> Wrote ${OUT}`);
	} finally {
		killServer();
	}
}

main().catch((error) => {
	console.error(error);
	process.exit(1);
});
