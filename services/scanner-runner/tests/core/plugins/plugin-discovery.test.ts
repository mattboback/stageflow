import fs from 'fs-extra';
import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import { afterEach, describe, expect, it, vi } from 'vitest';

import type { ScannerManifest } from '../../../src/core/manifest';
import type {
	PluginLoaderConfig,
	ScannerLogger
} from '../../../src/core/plugins/plugin-loader-types';

import { discoverPluginManifests } from '../../../src/core/plugins/plugin-discovery';

function baseManifest(id: string): ScannerManifest {
	return {
		id,
		name: `Plugin ${id}`,
		version: '1.0.0',
		description: 'test plugin',
		capabilities: {
			categories: ['accessibility'],
			outputFormats: ['json'],
			supportsScreenshots: false,
			supportsConcurrency: false,
			requiresBrowser: false
		},
		entry: {
			module: './index.js',
			exportName: 'TestScanner'
		}
	};
}

async function writeManifest(manifestPath: string, manifest: ScannerManifest): Promise<void> {
	await mkdir(path.dirname(manifestPath), { recursive: true });
	await writeFile(manifestPath, JSON.stringify(manifest, null, 2), 'utf8');
}

function loggerMock(): ScannerLogger {
	return {
		info: vi.fn(),
		warn: vi.fn(),
		error: vi.fn(),
		debug: vi.fn()
	};
}

function config(overrides: Partial<PluginLoaderConfig> = {}): PluginLoaderConfig {
	return {
		searchPaths: [],
		manifestPatterns: ['manifest.json'],
		strictValidation: true,
		verbose: false,
		...overrides
	};
}

describe('discoverPluginManifests', () => {
	let tempRoot = '';

	afterEach(async () => {
		vi.restoreAllMocks();
		if (tempRoot) {
			await rm(tempRoot, { recursive: true, force: true });
			tempRoot = '';
		}
	});

	it('returns empty results for a missing path and logs debug when verbose', async () => {
		const logger = loggerMock();
		const missingPath = path.join(os.tmpdir(), 'stageflow-plugin-missing-never-created');

		const result = await discoverPluginManifests(
			config({ searchPaths: [missingPath], verbose: true }),
			logger
		);

		expect(result.plugins).toEqual([]);
		expect(result.errors).toEqual([]);
		expect(logger.debug).toHaveBeenCalledWith('Search path does not exist', {
			path: missingPath
		});
	});

	it('discovers manifests from both nested plugin directories and root files', async () => {
		tempRoot = await mkdtemp(path.join(os.tmpdir(), 'stageflow-plugin-discovery-'));
		const pluginDirManifest = path.join(tempRoot, 'alpha', 'manifest.json');
		const rootManifest = path.join(tempRoot, 'scanner.json');

		await writeManifest(pluginDirManifest, {
			...baseManifest('alpha'),
			aliases: ['a11y-alpha']
		});
		await writeManifest(rootManifest, {
			...baseManifest('beta'),
			aliases: ['a11y-beta']
		});

		const result = await discoverPluginManifests(
			config({
				searchPaths: [tempRoot],
				manifestPatterns: ['manifest.json', 'scanner.json']
			}),
			loggerMock()
		);

		expect(result.errors).toEqual([]);
		expect(result.plugins.map((p) => p.manifest.id).sort()).toEqual(['alpha', 'beta']);
		expect(result.manifestsById.get('alpha')?.manifestPath).toBe(pluginDirManifest);
		expect(result.aliasByToken.get('a11y-alpha')).toBe('alpha');
		expect(result.aliasByToken.get('beta')).toBe('beta');
	});

	it('reports validation errors when strictValidation is enabled', async () => {
		tempRoot = await mkdtemp(path.join(os.tmpdir(), 'stageflow-plugin-strict-'));
		const invalidManifestPath = path.join(tempRoot, 'broken', 'manifest.json');

		await writeManifest(invalidManifestPath, {
			...baseManifest('broken'),
			capabilities: {
				categories: ['accessibility'],
				outputFormats: ['json'],
				supportsScreenshots: true,
				supportsConcurrency: false,
				requiresBrowser: false
			}
		});

		const result = await discoverPluginManifests(
			config({ searchPaths: [tempRoot], strictValidation: true }),
			loggerMock()
		);

		expect(result.plugins).toEqual([]);
		expect(result.errors).toEqual([
			expect.objectContaining({
				path: invalidManifestPath,
				error: expect.stringContaining('Validation failed')
			})
		]);
	});

	it('accepts warning-only manifests when strictValidation is disabled and logs warnings', async () => {
		tempRoot = await mkdtemp(path.join(os.tmpdir(), 'stageflow-plugin-warnings-'));
		const manifestPath = path.join(tempRoot, 'warn', 'manifest.json');
		const logger = loggerMock();

		await writeManifest(manifestPath, {
			...baseManifest('ab'),
			description: ''
		});

		const result = await discoverPluginManifests(
			config({
				searchPaths: [tempRoot],
				strictValidation: false,
				verbose: true
			}),
			logger
		);

		expect(result.plugins.map((p) => p.manifest.id)).toEqual(['ab']);
		expect(result.errors).toEqual([]);
		expect(logger.warn).toHaveBeenCalledWith(
			'Manifest warnings',
			expect.objectContaining({ path: manifestPath })
		);
	});

	it('records alias collisions and empty aliases', async () => {
		tempRoot = await mkdtemp(path.join(os.tmpdir(), 'stageflow-plugin-aliases-'));

		const firstManifest = path.join(tempRoot, 'one', 'manifest.json');
		const secondManifest = path.join(tempRoot, 'two', 'manifest.json');

		await writeManifest(firstManifest, {
			...baseManifest('one'),
			aliases: ['shared', '   ']
		});
		await writeManifest(secondManifest, {
			...baseManifest('two'),
			aliases: ['shared']
		});

		const result = await discoverPluginManifests(config({ searchPaths: [tempRoot] }), loggerMock());

		expect(result.errors).toEqual(
			expect.arrayContaining([
				expect.objectContaining({
					path: firstManifest,
					error: 'Empty alias declared for plugin id: one'
				}),
				expect.objectContaining({
					path: secondManifest,
					error: "Alias 'shared' already used by plugin 'one'"
				})
			])
		);
	});

	it('records duplicate plugin ids', async () => {
		tempRoot = await mkdtemp(path.join(os.tmpdir(), 'stageflow-plugin-duplicate-'));

		const firstManifest = path.join(tempRoot, 'first', 'manifest.json');
		const duplicateManifest = path.join(tempRoot, 'second', 'manifest.json');

		await writeManifest(firstManifest, baseManifest('duplicate'));
		await writeManifest(duplicateManifest, baseManifest('duplicate'));

		const result = await discoverPluginManifests(config({ searchPaths: [tempRoot] }), loggerMock());

		expect(result.errors).toEqual(
			expect.arrayContaining([
				expect.objectContaining({
					path: duplicateManifest,
					error: 'Duplicate plugin id: duplicate'
				})
			])
		);
	});

	it('captures search path failures when fs.stat throws', async () => {
		tempRoot = await mkdtemp(path.join(os.tmpdir(), 'stageflow-plugin-stat-'));
		const logger = loggerMock();

		const statSpy = vi.spyOn(fs, 'stat').mockRejectedValueOnce(new Error('stat exploded'));

		const result = await discoverPluginManifests(config({ searchPaths: [tempRoot] }), logger);

		expect(statSpy).toHaveBeenCalled();
		expect(result.plugins).toEqual([]);
		expect(result.errors).toEqual([
			{
				path: tempRoot,
				error: 'Failed to search path',
				details: 'stat exploded'
			}
		]);
	});
});
