import { mkdir, readdir, readFile, stat, writeFile } from 'node:fs/promises';
import path from 'node:path';

export const REQUIRED_PATH_PARTS = ['dashboard', 'applications', 'profile'];
export const STANDARD_SCANNERS = [
	'axe',
	'lighthouse',
	'link-checker',
	'open-graph',
	'security-headers',
	'seo',
	'spelling-grammar'
];

export function trimTrailingSlash(value) {
	return value.endsWith('/') ? value.slice(0, -1) : value;
}

export function resolveConfig(env = process.env) {
	const site = trimTrailingSlash(env.QA_SITE || 'https://alchemizecv.com');
	const stageflowSite = trimTrailingSlash(env.STAGEFLOW_QA_SITE || 'https://stageflow.org');

	return {
		artifactRoot:
			env.ARTIFACT_ROOT ||
			path.join(
				'output',
				'alchemizecv-prod-qa',
				new Date().toISOString().replace(/[-:]/g, '').replace(/\.\d{3}Z$/, 'Z')
			),
		chromePath: env.STAGEFLOW_QA_CHROME || '/usr/bin/google-chrome',
		email: env.QA_LOGIN_EMAIL || '',
		password: env.QA_LOGIN_PASS || '',
		port: Number.parseInt(env.STAGEFLOW_QA_CDP_PORT || '9223', 10),
		site,
		stageflowSite,
		targetUrls: [`${site}/dashboard`, `${site}/applications`, `${site}/profile`],
		timeoutMs: Number.parseInt(env.STAGEFLOW_QA_TIMEOUT_MS || String(12 * 60 * 1000), 10)
	};
}

export function validateConfig(config) {
	const missing = [];
	if (!config.email) missing.push('QA_LOGIN_EMAIL');
	if (!config.password) missing.push('QA_LOGIN_PASS');
	if (!Number.isInteger(config.port) || config.port <= 0) missing.push('STAGEFLOW_QA_CDP_PORT');
	if (!Number.isInteger(config.timeoutMs) || config.timeoutMs <= 0) {
		missing.push('STAGEFLOW_QA_TIMEOUT_MS');
	}

	return missing;
}

export function redactText(value, secrets = []) {
	let text = String(value ?? '');
	for (const secret of secrets) {
		if (secret) {
			text = text.split(secret).join('<redacted:secret>');
		}
	}

	return text
		.replace(/Bearer\s+[A-Za-z0-9._~+/-]+=*/g, 'Bearer <redacted:token>')
		.replace(/([?&]X-Amz-Signature=)[^&\s]+/g, '$1<redacted:signature>')
		.replace(/([?&]X-Amz-Credential=)[^&\s]+/g, '$1<redacted:credential>');
}

export function collectPageUrls(report, status) {
	const reportPages = Array.isArray(report?.pages)
		? report.pages.map((page) => page.url || page.finalUrl || page.path || page.id).filter(Boolean)
		: [];
	const artifactPages = Array.isArray(status?.artifacts?.screenshots)
		? status.artifacts.screenshots.map((shot) => shot.page_url).filter(Boolean)
		: [];

	return {
		artifactPages,
		pages: reportPages,
		allPages: [...new Set([...reportPages, ...artifactPages])]
	};
}

export function verifyAuthenticatedCoverage(report, status, requiredParts = REQUIRED_PATH_PARTS) {
	const { allPages, artifactPages, pages } = collectPageUrls(report, status);
	const missing = requiredParts.filter(
		(part) => !allPages.some((url) => String(url).includes(part))
	);
	const serialized = JSON.stringify({ report, status }).toLowerCase();
	const authIssue =
		serialized.includes('auth-hydration-failed') ||
		serialized.includes('form auth hydration failed') ||
		serialized.includes('did not leave the login page');

	return {
		artifactPages,
		authIssue,
		missing,
		pages
	};
}

export function summarizeScanners(report) {
	return Array.isArray(report?.scanners)
		? report.scanners.map((scanner) => ({
				id: scanner.id,
				issueCount: scanner.issueCount ?? 0,
				status: scanner.status
			}))
		: [];
}

export function verifyScannerSuccess(scanners, requiredScanners = STANDARD_SCANNERS) {
	const byId = new Map(scanners.map((scanner) => [scanner.id, scanner]));
	const missing = requiredScanners.filter((id) => !byId.has(id));
	const failed = requiredScanners.filter((id) => {
		const scanner = byId.get(id);
		return scanner && scanner.status !== 'success';
	});

	return { failed, missing };
}

export async function writeJson(filePath, value) {
	await mkdir(path.dirname(filePath), { recursive: true });
	await writeFile(filePath, `${JSON.stringify(value, null, 2)}\n`);
}

export async function findSecretLeaks(paths, secrets) {
	const needles = secrets.filter(Boolean).map((secret) => Buffer.from(secret));
	const leaks = [];
	if (needles.length === 0) {
		return leaks;
	}

	async function scanFile(filePath) {
		const data = await readFile(filePath);
		if (needles.some((needle) => data.includes(needle))) {
			leaks.push(filePath);
		}
	}

	async function scanPath(entryPath) {
		let info;
		try {
			info = await stat(entryPath);
		} catch {
			return;
		}

		if (info.isDirectory()) {
			const entries = await readdir(entryPath);
			await Promise.all(entries.map((entry) => scanPath(path.join(entryPath, entry))));
			return;
		}

		if (info.isFile()) {
			await scanFile(entryPath);
		}
	}

	for (const entryPath of paths) {
		await scanPath(entryPath);
	}

	return leaks.sort();
}

export function buildMarkdownReport({ config, directLogin, verification }) {
	const lines = [
		'# AlchemizeCV Authenticated StageFlow Dogfooding QA',
		'',
		`Run ID: \`${path.basename(config.artifactRoot)}\`  `,
		'Tester: Chrome DevTools Protocol (`connectOverCDP`) attached to Xvfb-hosted system Chrome  ',
		`Target: \`${config.site}\`  `,
		`StageFlow surface: \`${config.stageflowSite}/playground\`  `,
		`Job ID: \`${verification.jobId}\``,
		'',
		'## Top Signals',
		'',
		`1. Auth probe status: ${directLogin?.me?.status ?? 'not-run'} for the QA account.`,
		`2. Authenticated pages verified: ${verification.pages.join(', ') || '(none)'}.`,
		`3. Standard scanners: ${verification.scanners.every((s) => s.status === 'success') ? 'all succeeded' : 'one or more failed'}.`,
		'',
		'## Result',
		'',
		`- State: \`${verification.state}\``,
		`- Report schema version: \`${verification.reportVersion}\``,
		`- Missing required pages: ${verification.missing.length ? verification.missing.join(', ') : 'none'}`,
		`- Auth failure signals: ${verification.authIssue ? 'found' : 'none found'}`,
		'',
		'## Artifacts',
		'',
		'- `console.json`',
		'- `network.json`',
		'- `scan-verification.json`',
		'- `screenshots/`'
	];

	return `${lines.join('\n')}\n`;
}
