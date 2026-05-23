/**
 * Redaction test.
 *
 * Build-fails if the resolved value of any allow-listed env var leaks into
 * persisted artifacts: stored Provenance, the auth_hydrated audit event,
 * the serialized scan stage log, the recipe, or the synthetic
 * auth-hydration-failed issue.
 */

import fs from 'fs-extra';
import { mkdtemp, readFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';

import type { Provenance, ScannerConfig, StorageProvider } from '../../src/core/types';

import { ScanStageLogger } from '../../src/core/scan-stage-logger';
import { createSecretsResolver, collectFromEnvReferences } from '../../src/core/secrets-resolver';

class InMemoryStorageProvider implements StorageProvider {
	uploads: { bucket: string; key: string; path: string; body: string }[] = [];

	ensureBucket = (): Promise<void> => Promise.resolve();
	upload = async (bucket: string, key: string, filePath: string): Promise<void> => {
		const body = await readFile(filePath, 'utf8');
		this.uploads.push({ bucket, key, path: filePath, body });
	};
	uploadBuffer = (): Promise<void> => Promise.resolve();
	uploadDirectory = (): Promise<number> => Promise.resolve(0);
	download = (): Promise<void> => Promise.resolve();
	exists = (): Promise<boolean> => Promise.resolve(false);
}

const SECRET_USER = 'auth-test-user-aljkfhqwouihasdf';
const SECRET_PASSWORD = 'auth-test-password-pq3i48hsdfkj2';

function makeProvenance(): Provenance {
	return {
		version: '1.0.0',
		job_id: 'job-redact',
		base_url: 'https://app.example.com',
		pages: [{ id: 'profile', path: '/profile', url: 'https://app.example.com/profile' }],
		auth: {
			mode: 'form',
			login_url: 'https://app.example.com/login',
			steps: [
				{
					type: 'fill',
					selector: 'input[name=email]',
					value: { from_env: 'STAGEFLOW_AUTH_USER' }
				},
				{
					type: 'fill',
					selector: 'input[name=password]',
					value: { from_env: 'STAGEFLOW_AUTH_PASSWORD' }
				},
				{ type: 'click', selector: 'button[type=submit]' }
			],
			success: { type: 'selector', selector: '[data-test=signed-in]' }
		}
	};
}

function expectNoSecrets(haystack: string): void {
	expect(haystack).not.toContain(SECRET_USER);
	expect(haystack).not.toContain(SECRET_PASSWORD);
}

describe('Auth redaction', () => {
	let tmpDir: string;

	beforeEach(async () => {
		tmpDir = await mkdtemp(join(tmpdir(), 'stageflow-redaction-'));
	});

	afterEach(async () => {
		await rm(tmpDir, { recursive: true, force: true });
	});

	it('keeps resolved env values out of persisted Provenance', () => {
		const provenance = makeProvenance();
		const serialized = JSON.stringify(provenance);
		expect(serialized).toContain('STAGEFLOW_AUTH_USER');
		expect(serialized).toContain('STAGEFLOW_AUTH_PASSWORD');
		expectNoSecrets(serialized);
	});

	it('keeps resolved env values out of the SecretsResolver allow-list view', () => {
		const allowList = collectFromEnvReferences(makeProvenance());
		const resolver = createSecretsResolver({
			allowList,
			env: {
				STAGEFLOW_AUTH_USER: SECRET_USER,
				STAGEFLOW_AUTH_PASSWORD: SECRET_PASSWORD
			}
		});

		expect(resolver.resolve({ from_env: 'STAGEFLOW_AUTH_USER' })).toBe(SECRET_USER);

		expectNoSecrets(JSON.stringify(resolver.allowList));
		expectNoSecrets(JSON.stringify({ allowList: resolver.allowList }));
	});

	it('keeps resolved env values out of the serialized scan stage log and recipe', async () => {
		const config: ScannerConfig = {
			jobId: 'job-redact',
			provenancePath: join(tmpDir, 'provenance.json'),
			resultsDir: tmpDir,
			scannerName: 'axe',
			concurrency: 1,
			maxRetries: 1,
			browser: {
				headless: true,
				args: [],
				defaultViewport: { width: 1280, height: 720 },
				deviceScaleFactor: 1,
				defaultTimeout: 30_000,
				pageLoadTimeout: 15_000
			},
			storage: {
				endpoint: 'localhost:9000',
				accessKey: 'k',
				secretKey: 's',
				useSSL: false,
				bucket: 'test-bucket'
			},
			messaging: {
				url: '',
				subjects: { pageCompleted: 'a', scanCompleted: 'b', scanFailed: 'c' }
			}
		};

		const storage = new InMemoryStorageProvider();
		const stageLogger = new ScanStageLogger(config, storage);
		await stageLogger.start();

		// Same auth_hydrated event detail that PageIterator emits on success.
		stageLogger.recordEvent('auth_hydrated', {
			mode: 'form',
			login_url: 'https://app.example.com/login',
			post_login_url: 'https://app.example.com/profile'
		});

		stageLogger.setMetrics({ pages_total: 1, pages_scanned: 1, total_issues: 0 });
		stageLogger.setArtifacts({ results_key: 'job-redact/axe/results.json' });
		await stageLogger.finalizeSuccess();

		const stageLogPath = join(tmpDir, 'stages', 'scan.log.json');
		const recipePath = join(tmpDir, 'recipes', 'scan.json');

		const stageLogText = await readFile(stageLogPath, 'utf8');
		const recipeText = await readFile(recipePath, 'utf8');

		expectNoSecrets(stageLogText);
		expectNoSecrets(recipeText);
		expect(stageLogText).toContain('auth_hydrated');
		expect(stageLogText).toContain('https://app.example.com/profile');

		// Anything we uploaded to MinIO must also be free of secrets.
		for (const upload of storage.uploads) {
			expectNoSecrets(upload.body);
		}

		// Defensive: write the Provenance as the orchestrator would and check.
		const persistedProvenancePath = join(tmpDir, 'provenance.json');
		await fs.writeJSON(persistedProvenancePath, makeProvenance());
		const persistedProvenanceText = await readFile(persistedProvenancePath, 'utf8');
		expectNoSecrets(persistedProvenanceText);
	});
});
