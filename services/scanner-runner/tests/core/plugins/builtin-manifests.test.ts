import { copyFile, mkdir, mkdtemp, readFile, rm, stat, writeFile } from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import { describe, expect, it } from 'vitest';

import { PluginLoader } from '../../../src/core/plugins/loader';

const BUILTIN_MANIFEST_IDS = [
	'ai-navigator',
	'axe',
	'lighthouse',
	'link-checker',
	'open-graph',
	'security-headers',
	'seo',
	'spelling-grammar'
] as const;

function escapeRegExp(value: string): string {
	return value.replaceAll(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

describe('Built-in manifests (drift guard)', () => {
	it('discovers and validates the shared built-in manifests', async () => {
		const manifestsRoot = path.resolve(process.cwd(), '../../libs/go/scannercatalog/manifests');

		await expect(stat(manifestsRoot)).resolves.toBeTruthy();

		const loader = new PluginLoader({
			searchPaths: [manifestsRoot],
			manifestPatterns: ['manifest.json'],
			strictValidation: true,
			verbose: false
		});

		const discovery = await loader.discover();
		expect(discovery.errors).toEqual([]);

		const discoveredIds = loader.listDiscovered().sort();
		expect(discoveredIds).toEqual([...BUILTIN_MANIFEST_IDS].sort());
	});

	it('keeps manifest entrypoints aligned with scanner-runner source exports', async () => {
		const repoRoot = path.resolve(process.cwd(), '../..');
		const manifestsRoot = path.join(repoRoot, 'libs/go/scannercatalog/manifests');

		const loader = new PluginLoader({
			searchPaths: [manifestsRoot],
			manifestPatterns: ['manifest.json'],
			strictValidation: true,
			verbose: false
		});

		const discovery = await loader.discover();
		expect(discovery.errors).toEqual([]);

		const byId = new Map(discovery.plugins.map((p) => [p.manifest.id, p]));

		for (const id of BUILTIN_MANIFEST_IDS) {
			const plugin = byId.get(id);
			expect(plugin).toBeDefined();

			const { manifest } = plugin!;

			expect(manifest.entry.module).toBe('./index.js');
			const expectedSourcePath = path.join(
				repoRoot,
				`services/scanner-runner/src/scanners/${id}/index.ts`
			);

			await expect(stat(expectedSourcePath)).resolves.toBeTruthy();

			const exportName = manifest.entry.exportName;
			if (typeof exportName !== 'string' || exportName.trim().length === 0) {
				throw new Error(`Expected manifest entry exportName to be set for ${id}`);
			}

			const content = await readFile(expectedSourcePath, 'utf8');
			const exportRegex = new RegExp(`export\\s+class\\s+${escapeRegExp(exportName)}\\b`, 'm');
			expect(content).toMatch(exportRegex);
		}
	});

	it('loads the built-in manifests and resolves every alias (stub runtime)', async () => {
		const repoRoot = path.resolve(process.cwd(), '../..');
		const manifestsRoot = path.join(repoRoot, 'libs/go/scannercatalog/manifests');

		const sourceLoader = new PluginLoader({
			searchPaths: [manifestsRoot],
			manifestPatterns: ['manifest.json'],
			strictValidation: true,
			verbose: false
		});

		const sourceDiscovery = await sourceLoader.discover();
		expect(sourceDiscovery.errors).toEqual([]);

		const tempRoot = await mkdtemp(path.join(os.tmpdir(), 'stageflow-builtin-'));

		try {
			for (const plugin of sourceDiscovery.plugins) {
				const relativeManifestPath = path.relative(manifestsRoot, plugin.manifestPath);
				const destManifestPath = path.join(tempRoot, relativeManifestPath);

				await mkdir(path.dirname(destManifestPath), { recursive: true });
				await copyFile(plugin.manifestPath, destManifestPath);

				const exportName = plugin.manifest.entry.exportName;
				expect(exportName).toBeTruthy();

				const entryPath = path.resolve(
					path.dirname(destManifestPath),
					plugin.manifest.entry.module
				);

				await mkdir(path.dirname(entryPath), { recursive: true });
				await writeFile(
					entryPath,
					[
						`class ${exportName!} {`,
						'  constructor() {',
						`    this.metadata = { name: ${JSON.stringify(plugin.manifest.id)}, version: ${JSON.stringify(
							plugin.manifest.version
						)} };`,
						'  }',
						'  scanPage() {',
						'    return {};',
						'  }',
						'}',
						`exports.${exportName!} = ${exportName!};`,
						''
					].join('\n'),
					'utf8'
				);
			}

			const loader = new PluginLoader({
				searchPaths: [tempRoot],
				manifestPatterns: ['manifest.json'],
				strictValidation: true,
				verbose: false
			});

			const discovery = await loader.discover();
			expect(discovery.errors).toEqual([]);

			const discoveredIds = loader.listDiscovered().sort();
			expect(discoveredIds).toEqual([...BUILTIN_MANIFEST_IDS].sort());

			for (const plugin of discovery.plugins) {
				const id = plugin.manifest.id;
				const loadById = await loader.load(id);
				expect(loadById.success).toBe(true);
				expect(loadById.plugin?.manifest.id).toBe(id);

				for (const alias of plugin.manifest.aliases ?? []) {
					const loadByAlias = await loader.load(alias);
					expect(loadByAlias.success).toBe(true);
					expect(loadByAlias.plugin?.manifest.id).toBe(id);
				}
			}
		} finally {
			await rm(tempRoot, { recursive: true, force: true });
		}
	});
});
