import assert from 'node:assert/strict';
import { mkdtemp, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import test from 'node:test';

import {
	findSecretLeaks,
	redactText,
	resolveConfig,
	summarizeScanners,
	validateConfig,
	verifyAuthenticatedCoverage,
	verifyScannerSuccess
} from './lib.mjs';

test('redactText removes passwords, bearer tokens, and presigned signatures', () => {
	const input =
		'password=secret-value Authorization: Bearer abc.def https://s3.local/file?X-Amz-Credential=stageflow/2026&X-Amz-Signature=abcdef';

	const redacted = redactText(input, ['secret-value']);

	assert(!redacted.includes('secret-value'));
	assert(!redacted.includes('abc.def'));
	assert(!redacted.includes('abcdef'));
	assert(redacted.includes('<redacted:secret>'));
	assert(redacted.includes('Bearer <redacted:token>'));
	assert(redacted.includes('X-Amz-Signature=<redacted:signature>'));
});

test('verifyAuthenticatedCoverage accepts report pages and artifact screenshots', () => {
	const report = {
		pages: [
			{ url: 'https://alchemizecv.com/dashboard' },
			{ url: 'https://alchemizecv.com/applications' }
		]
	};
	const status = {
		artifacts: {
			screenshots: [{ page_url: 'https://alchemizecv.com/profile' }]
		}
	};

	const result = verifyAuthenticatedCoverage(report, status);

	assert.deepEqual(result.missing, []);
	assert.equal(result.authIssue, false);
	assert.deepEqual(result.pages, [
		'https://alchemizecv.com/dashboard',
		'https://alchemizecv.com/applications'
	]);
	assert.deepEqual(result.artifactPages, ['https://alchemizecv.com/profile']);
});

test('verifyAuthenticatedCoverage reports missing pages and auth failure signals', () => {
	const result = verifyAuthenticatedCoverage(
		{
			issues: [{ ruleId: 'auth-hydration-failed' }],
			pages: [{ url: 'https://alchemizecv.com/dashboard' }]
		},
		{}
	);

	assert.deepEqual(result.missing, ['applications', 'profile']);
	assert.equal(result.authIssue, true);
});

test('verifyScannerSuccess requires all standard scanners to succeed', () => {
	const scanners = [
		{ id: 'axe', status: 'success' },
		{ id: 'lighthouse', status: 'failed' },
		{ id: 'link-checker', status: 'success' },
		{ id: 'open-graph', status: 'success' },
		{ id: 'security-headers', status: 'success' },
		{ id: 'seo', status: 'success' }
	];

	const result = verifyScannerSuccess(scanners);

	assert.deepEqual(result.failed, ['lighthouse']);
	assert.deepEqual(result.missing, ['spelling-grammar']);
});

test('summarizeScanners normalizes report scanner output', () => {
	const result = summarizeScanners({
		scanners: [
			{ id: 'axe', status: 'success', issueCount: 2 },
			{ id: 'seo', status: 'success' }
		]
	});

	assert.deepEqual(result, [
		{ id: 'axe', issueCount: 2, status: 'success' },
		{ id: 'seo', issueCount: 0, status: 'success' }
	]);
});

test('resolveConfig and validateConfig keep live credentials env-gated', () => {
	const config = resolveConfig({
		QA_LOGIN_EMAIL: 'qa@example.com',
		QA_LOGIN_PASS: 'pw',
		QA_SITE: 'https://target.example/',
		STAGEFLOW_QA_CDP_PORT: '9333',
		STAGEFLOW_QA_SITE: 'https://stageflow.example/'
	});

	assert.equal(config.site, 'https://target.example');
	assert.equal(config.stageflowSite, 'https://stageflow.example');
	assert.equal(config.port, 9333);
	assert.deepEqual(validateConfig(config), []);
	assert.deepEqual(validateConfig(resolveConfig({})), ['QA_LOGIN_EMAIL', 'QA_LOGIN_PASS']);
});

test('findSecretLeaks reports files containing configured secrets', async () => {
	const dir = await mkdtemp(path.join(tmpdir(), 'stageflow-secret-check-'));
	try {
		await writeFile(path.join(dir, 'safe.txt'), 'nothing to see');
		await writeFile(path.join(dir, 'leak.txt'), 'contains super-secret');

		const leaks = await findSecretLeaks([dir], ['super-secret']);

		assert.deepEqual(leaks, [path.join(dir, 'leak.txt')]);
	} finally {
		await rm(dir, { force: true, recursive: true });
	}
});
