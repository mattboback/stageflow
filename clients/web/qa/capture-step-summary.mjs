#!/usr/bin/env node
// Captures assets/step-summary.png: the markdown digest `stageflow diff`
// writes to a GitHub step summary, rendered by GitHub's own markdown API.
//
// The diff compares the committed golden reports (sample site v1 -> v2), so
// the figures match `just scan-diff-golden`. Rendering calls the GitHub API
// through `gh` and loads GitHub's markdown stylesheet from jsDelivr.
//
// Requires: Go, an authenticated `gh`, network access and Playwright Chromium.
//
// Usage: node clients/web/qa/capture-step-summary.mjs [output.png]

import { chromium } from '@playwright/test';
import { spawnSync } from 'node:child_process';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(__dirname, '..', '..', '..');
const OUT = process.argv[2]
	? path.resolve(process.argv[2])
	: path.join(repoRoot, 'assets/step-summary.png');
const golden = path.join(repoRoot, 'qa/fixtures/scan-diff-golden');

function run(command, args, input) {
	const result = spawnSync(command, args, { cwd: repoRoot, input, encoding: 'utf8' });
	if (result.error) throw result.error;
	return result;
}

// The gate trips on the new findings, so the diff exits 1 by design.
const diff = run('go', [
	'run',
	'./clients/cli/cmd/stageflow',
	'diff',
	path.join(golden, 'golden-baseline-report.json'),
	path.join(golden, 'golden-regression-report.json'),
	'--format',
	'markdown',
	'--fail-on-new=any'
]);
if (diff.status !== 1 || !diff.stdout.includes('## Regression Diff')) {
	throw new Error(`stageflow diff exited ${diff.status}: ${diff.stderr}`);
}

const rendered = run(
	'gh',
	['api', 'markdown', '--input', '-'],
	JSON.stringify({ text: diff.stdout, mode: 'gfm' })
);
if (rendered.status !== 0) throw new Error(`gh api markdown failed: ${rendered.stderr}`);

const html = `<!doctype html>
<html><head><meta charset="utf-8">
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/github-markdown-css@5.8.1/github-markdown-light.css">
<style>body { margin: 0; background: #fff; } .markdown-body { box-sizing: border-box; width: 820px; padding: 24px 32px; }</style>
</head><body><article class="markdown-body">${rendered.stdout}</article></body></html>`;

const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width: 820, height: 600 }, deviceScaleFactor: 2 });
await page.setContent(html, { waitUntil: 'networkidle' });
await page.locator('.markdown-body').screenshot({ path: OUT });
await browser.close();
console.log(`==> Wrote ${OUT}`);
