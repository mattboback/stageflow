import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import { describe, expect, it } from 'vitest';

import { PluginLoader } from '../../../src/core/plugins/loader';

describe('PluginLoader', () => {
	it('resolves aliases to plugin IDs', async () => {
		const tempDir = await mkdtemp(path.join(os.tmpdir(), 'stageflow-plugin-'));

		try {
			const pluginDir = path.join(tempDir, 'dummy');
			await mkdir(pluginDir, { recursive: true });

			await writeFile(
				path.join(pluginDir, 'dummy.js'),
				[
					'class DummyScanner {',
					'  constructor() {',
					'    this.metadata = { name: "dummy", version: "1.0.0" };',
					'  }',
					'  scanPage() {',
					'    return {};',
					'  }',
					'}',
					'exports.DummyScanner = DummyScanner;',
					''
				].join('\n'),
				'utf8'
			);

			await writeFile(
				path.join(pluginDir, 'manifest.json'),
				JSON.stringify(
					{
						id: 'dummy',
						aliases: ['a11y', 'accessibility'],
						name: 'Dummy Scanner',
						version: '1.0.0',
						capabilities: {
							categories: ['custom'],
							outputFormats: ['json'],
							supportsScreenshots: false,
							supportsConcurrency: true,
							requiresBrowser: false
						},
						entry: {
							module: './dummy.js',
							exportName: 'DummyScanner'
						}
					},
					null,
					2
				),
				'utf8'
			);

			const loader = new PluginLoader({
				searchPaths: [tempDir],
				manifestPatterns: ['manifest.json'],
				strictValidation: true,
				verbose: false
			});

			const discovery = await loader.discover();
			expect(discovery.errors).toEqual([]);
			expect(loader.listDiscovered()).toEqual(['dummy']);

			const loadResult = await loader.load('a11y');
			expect(loadResult.success).toBe(true);
			expect(loadResult.plugin?.manifest.id).toBe('dummy');
		} finally {
			await rm(tempDir, { recursive: true, force: true });
		}
	});
});
