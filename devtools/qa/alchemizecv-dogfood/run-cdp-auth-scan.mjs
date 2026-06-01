#!/usr/bin/env node

import { access, mkdir, writeFile } from 'node:fs/promises';
import { spawn, spawnSync } from 'node:child_process';
import http from 'node:http';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

import {
	buildMarkdownReport,
	findSecretLeaks,
	redactText,
	resolveConfig,
	summarizeScanners,
	validateConfig,
	verifyAuthenticatedCoverage,
	verifyScannerSuccess,
	writeJson
} from './lib.mjs';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(__dirname, '../../..');

async function importPlaywright() {
	try {
		return await import('playwright');
	} catch {
		const candidates = [
			path.join(repoRoot, 'services/scanner-runner/node_modules/playwright/index.js'),
			path.join(repoRoot, 'clients/web/node_modules/playwright/index.js')
		];
		for (const candidate of candidates) {
			try {
				await access(candidate);
				return await import(pathToFileURL(candidate).href);
			} catch {
				// Try the next workspace dependency location.
			}
		}
		throw new Error('Playwright is not installed in a discoverable workspace node_modules');
	}
}

function ensureXvfb() {
	if (process.env.DISPLAY || process.env.STAGEFLOW_QA_XVFB_CHILD === '1') {
		return;
	}

	const probe = spawnSync('command', ['-v', 'xvfb-run'], { shell: true, stdio: 'ignore' });
	if (probe.status !== 0) {
		return;
	}

	const result = spawnSync(
		'xvfb-run',
		[
			'-a',
			'--server-args=-screen 0 1440x900x24',
			process.execPath,
			...process.argv.slice(1)
		],
		{
			env: { ...process.env, STAGEFLOW_QA_XVFB_CHILD: '1' },
			stdio: 'inherit'
		}
	);
	process.exit(result.status ?? 1);
}

function fetchJson(url) {
	return new Promise((resolve, reject) => {
		http
			.get(url, (response) => {
				let data = '';
				response.on('data', (chunk) => {
					data += chunk;
				});
				response.on('end', () => {
					try {
						resolve(JSON.parse(data));
					} catch (error) {
						reject(error);
					}
				});
			})
			.on('error', reject);
	});
}

async function waitForCDP(port) {
	for (let attempt = 0; attempt < 80; attempt += 1) {
		try {
			return await fetchJson(`http://127.0.0.1:${port}/json/version`);
		} catch {
			await new Promise((resolve) => setTimeout(resolve, 250));
		}
	}

	throw new Error(`Chrome DevTools endpoint did not start on port ${port}`);
}

async function screenshot(page, artifactRoot, name, observations, note) {
	const target = path.join(artifactRoot, 'screenshots', name);
	await page.screenshot({ path: target, fullPage: true });
	observations.push({
		name,
		note,
		screenshot: target,
		ts: new Date().toISOString(),
		url: page.url(),
		viewport: page.viewportSize()
	});
}

async function main() {
	ensureXvfb();

	const config = resolveConfig();
	const missingConfig = validateConfig(config);
	if (missingConfig.length > 0) {
		throw new Error(`Missing or invalid live QA env var(s): ${missingConfig.join(', ')}`);
	}

	await mkdir(path.join(config.artifactRoot, 'screenshots'), { recursive: true });
	await mkdir(path.join(config.artifactRoot, 'downloads'), { recursive: true });
	await mkdir(path.join(config.artifactRoot, 'har'), { recursive: true });

	const { chromium } = await importPlaywright();
	const userDataDir = await import('node:fs/promises').then((fs) =>
		fs.mkdtemp(path.join(tmpdir(), 'stageflow-cdp-chrome-'))
	);
	const secrets = [config.password];
	const consoleLog = [];
	const networkLog = [];
	const observations = [];
	const chrome = spawn(
		config.chromePath,
		[
			`--remote-debugging-port=${config.port}`,
			`--user-data-dir=${userDataDir}`,
			'--no-first-run',
			'--no-default-browser-check',
			'--disable-features=Translate',
			'--no-sandbox',
			'--window-size=1440,900',
			'about:blank'
		],
		{ stdio: ['ignore', 'ignore', 'pipe'] }
	);
	let chromeStderr = '';
	chrome.stderr.on('data', (chunk) => {
		chromeStderr += chunk.toString();
	});

	try {
		const cdpVersion = await waitForCDP(config.port);
		const browser = await chromium.connectOverCDP(`http://127.0.0.1:${config.port}`);
		const context = browser.contexts()[0] || (await browser.newContext());
		const page = context.pages()[0] || (await context.newPage());
		await page.setViewportSize({ width: 1440, height: 900 });

		page.on('console', (message) => {
			consoleLog.push({
				location: message.location(),
				text: redactText(message.text(), secrets),
				ts: Date.now(),
				type: message.type()
			});
		});
		page.on('pageerror', (error) => {
			consoleLog.push({
				stack: redactText(error.stack, secrets),
				text: redactText(error.message, secrets),
				ts: Date.now(),
				type: 'pageerror'
			});
		});
		page.on('request', (request) => {
			networkLog.push({
				method: request.method(),
				phase: 'request',
				resourceType: request.resourceType(),
				ts: Date.now(),
				url: redactText(request.url(), secrets)
			});
		});
		page.on('response', (response) => {
			networkLog.push({
				method: response.request().method(),
				phase: 'response',
				resourceType: response.request().resourceType(),
				status: response.status(),
				ts: Date.now(),
				url: redactText(response.url(), secrets)
			});
		});
		page.on('requestfailed', (request) => {
			networkLog.push({
				failure: request.failure()?.errorText,
				method: request.method(),
				phase: 'requestfailed',
				resourceType: request.resourceType(),
				ts: Date.now(),
				url: redactText(request.url(), secrets)
			});
		});

		await page.goto(`${config.stageflowSite}/playground`, { waitUntil: 'networkidle' });
		await page.locator('#urls').fill(config.targetUrls.join('\n'));
		await page.getByRole('switch', { name: /Enable authentication/i }).click();
		await page.locator('#auth-login-url').fill(`${config.site}/login`);
		await page.locator('#auth-username').fill(config.email);
		await page.locator('#auth-password').fill(config.password);
		// Exercise the recommended default: login URL + credentials only, with no
		// success selector. This dogfoods the auto-detect login path (success:
		// networkidle + post-login grace poll) that the hosted Playground leads with.
		await screenshot(
			page,
			config.artifactRoot,
			'cdp-01-playground-auth-configured.png',
			observations,
			'Configured authenticated AlchemizeCV scan'
		);

		await page.getByRole('button', { name: /Start Scan/i }).last().click();
		await page.waitForURL(/\/scan\/[^/]+$/, { timeout: 30_000 });
		const jobId = page.url().match(/\/scan\/([^/?#]+)/)?.[1];
		if (!jobId) {
			throw new Error(`Could not parse job id from ${page.url()}`);
		}
		await writeFile(path.join(config.artifactRoot, 'job-id.txt'), `${jobId}\n`);
		await screenshot(
			page,
			config.artifactRoot,
			'cdp-02-scan-created.png',
			observations,
			`Scan status page opened for ${jobId}`
		);

		const started = Date.now();
		let finalStatus = null;
		while (Date.now() - started < config.timeoutMs) {
			finalStatus = await page.evaluate(async (id) => {
				const response = await fetch(`/api/v1/jobs/${id}`);
				return { json: await response.json(), status: response.status };
			}, jobId);
			await writeJson(path.join(config.artifactRoot, 'latest-status.json'), finalStatus);
			const state = String(finalStatus.json?.state || '').toLowerCase();
			if (state === 'done' || state === 'failed') {
				break;
			}
			await new Promise((resolve) => setTimeout(resolve, 15_000));
		}

		await page.reload({ waitUntil: 'networkidle' }).catch(() => undefined);
		await screenshot(
			page,
			config.artifactRoot,
			'cdp-03-scan-terminal.png',
			observations,
			'Terminal scan status page'
		);
		if (String(finalStatus?.json?.state || '').toLowerCase() !== 'done') {
			throw new Error(`Scan did not finish DONE: ${finalStatus?.json?.state || 'unknown'}`);
		}

		await page.goto(`${config.stageflowSite}/scan/${jobId}/report`, { waitUntil: 'networkidle' });
		await page.waitForTimeout(3_000);
		await screenshot(
			page,
			config.artifactRoot,
			'cdp-04-report-overview.png',
			observations,
			'Unified report overview'
		);

		const reportResult = await page.evaluate(async (id) => {
			const response = await fetch(`/api/v1/jobs/${id}/results`, { redirect: 'follow' });
			return { json: await response.json(), status: response.status };
		}, jobId);
		const report = reportResult.json;
		const coverage = verifyAuthenticatedCoverage(report, finalStatus.json);
		const scanners = summarizeScanners(report);
		const scannerVerification = verifyScannerSuccess(scanners);
		const verification = {
			artifactPages: coverage.artifactPages,
			authIssue: coverage.authIssue,
			browserWSEndpoint: cdpVersion.webSocketDebuggerUrl ? 'available-redacted' : 'missing',
			jobId,
			missing: coverage.missing,
			pages: coverage.pages,
			reportStatus: reportResult.status,
			reportVersion: report?.version || null,
			scannerFailures: scannerVerification.failed,
			scannerMissing: scannerVerification.missing,
			scanners,
			state: finalStatus.json?.state,
			tester: 'Chrome DevTools Protocol connectOverCDP + Xvfb headful'
		};

		await writeJson(path.join(config.artifactRoot, 'report-results-fetch.json'), reportResult);
		await writeJson(path.join(config.artifactRoot, 'scan-verification.json'), verification);
		if (
			reportResult.status !== 200 ||
			!report?.version ||
			coverage.authIssue ||
			coverage.missing.length > 0 ||
			scannerVerification.failed.length > 0 ||
			scannerVerification.missing.length > 0
		) {
			throw new Error(`Verification failed: ${JSON.stringify(verification)}`);
		}

		await writeFile(
			path.join(config.artifactRoot, 'report.md'),
			buildMarkdownReport({ config, directLogin: null, verification })
		);
		await writeJson(path.join(config.artifactRoot, 'observations.json'), observations);
		const leaks = await findSecretLeaks(
			[config.artifactRoot, path.join(repoRoot, 'README.md'), path.join(repoRoot, 'docs/images')],
			secrets
		);
		if (leaks.length > 0) {
			throw new Error(`Secret value found in generated/public artifact(s): ${leaks.join(', ')}`);
		}
		await browser.close();
		console.log(`AlchemizeCV dogfood scan complete: ${config.artifactRoot}`);
	} finally {
		await writeJson(path.join(config.artifactRoot, 'console.json'), consoleLog);
		await writeJson(path.join(config.artifactRoot, 'network.json'), networkLog);
		await writeFile(
			path.join(config.artifactRoot, 'chrome-stderr.log'),
			redactText(chromeStderr.slice(-4000), secrets)
		);
		chrome.kill('SIGTERM');
	}
}

main().catch((error) => {
	console.error(error instanceof Error ? error.message : String(error));
	process.exit(1);
});
