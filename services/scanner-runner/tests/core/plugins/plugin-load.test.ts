import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import { afterEach, describe, expect, it, vi } from 'vitest';

import type { ScannerManifest } from '../../../src/core/manifest';
import type {
	PluginInfo,
	PluginLoaderConfig,
	ScannerLogger
} from '../../../src/core/plugins/plugin-loader-types';

import { loadPluginFromManifest } from '../../../src/core/plugins/plugin-load';

function baseManifest(overrides: Partial<ScannerManifest> = {}): ScannerManifest {
	return {
		id: 'test-plugin',
		name: 'Test Plugin',
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
			module: './scanner.mjs'
		},
		...overrides
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

function loggerMock(): ScannerLogger {
	return {
		info: vi.fn(),
		warn: vi.fn(),
		error: vi.fn(),
		debug: vi.fn()
	};
}

async function writeModule(filePath: string, content: string): Promise<void> {
	await mkdir(path.dirname(filePath), { recursive: true });
	await writeFile(filePath, content, 'utf8');
}

function pluginInfo(manifestPath: string, manifest: ScannerManifest): PluginInfo {
	return { manifestPath, manifest };
}

describe('loadPluginFromManifest', () => {
	let tempRoot = '';

	afterEach(async () => {
		vi.restoreAllMocks();
		if (tempRoot) {
			await rm(tempRoot, { recursive: true, force: true });
			tempRoot = '';
		}
	});

	it('returns an error when entry point does not exist', async () => {
		tempRoot = await mkdtemp(path.join(os.tmpdir(), 'stageflow-plugin-load-missing-'));
		const manifestPath = path.join(tempRoot, 'manifest.json');
		const info = pluginInfo(
			manifestPath,
			baseManifest({
				entry: { module: './missing.mjs' }
			})
		);

		const result = await loadPluginFromManifest(
			info,
			config({ searchPaths: [tempRoot] }),
			loggerMock()
		);

		expect(result.success).toBe(false);
		expect(result.error).toContain('Entry point not found:');
	});

	it('returns an error when entry path escapes plugin directory', async () => {
		tempRoot = await mkdtemp(path.join(os.tmpdir(), 'stageflow-plugin-load-path-escape-'));
		const manifestPath = path.join(tempRoot, 'manifest.json');
		const outsideModule = path.join(os.tmpdir(), 'outside-scanner.mjs');
		await writeModule(
			outsideModule,
			[
				'export default class ScannerImpl {',
				'  async scanPage() {',
				'    return {};',
				'  }',
				'}',
				''
			].join('\n')
		);
		const info = pluginInfo(
			manifestPath,
			baseManifest({
				entry: { module: '../outside-scanner.mjs' }
			})
		);

		const result = await loadPluginFromManifest(
			info,
			config({ searchPaths: [tempRoot] }),
			loggerMock()
		);

		expect(result.success).toBe(false);
		expect(result.error).toBe('Plugin entry path escapes plugin directory: ../outside-scanner.mjs');
	});

	it('returns an error when configured factory function is missing', async () => {
		tempRoot = await mkdtemp(path.join(os.tmpdir(), 'stageflow-plugin-load-factory-missing-'));
		const manifestPath = path.join(tempRoot, 'manifest.json');
		const modulePath = path.join(tempRoot, 'scanner.mjs');

		await writeModule(
			modulePath,
			['export class ScannerImpl {', '  async scanPage() {', '    return {};', '  }', '}', ''].join(
				'\n'
			)
		);

		const info = pluginInfo(
			manifestPath,
			baseManifest({
				entry: {
					module: './scanner.mjs',
					factoryName: 'createScanner'
				}
			})
		);

		const result = await loadPluginFromManifest(
			info,
			config({ searchPaths: [tempRoot] }),
			loggerMock()
		);

		expect(result.success).toBe(false);
		expect(result.error).toBe('Factory function not found: createScanner');
	});

	it('returns an error when factory output does not implement scanPage', async () => {
		tempRoot = await mkdtemp(path.join(os.tmpdir(), 'stageflow-plugin-load-invalid-factory-'));
		const manifestPath = path.join(tempRoot, 'manifest.json');
		const modulePath = path.join(tempRoot, 'scanner.mjs');

		await writeModule(
			modulePath,
			[
				'export function createScanner() {',
				"  return { metadata: { name: 'x', version: '1.0.0' } };",
				'}',
				''
			].join('\n')
		);

		const info = pluginInfo(
			manifestPath,
			baseManifest({
				entry: {
					module: './scanner.mjs',
					factoryName: 'createScanner'
				}
			})
		);

		const result = await loadPluginFromManifest(
			info,
			config({ searchPaths: [tempRoot] }),
			loggerMock()
		);

		expect(result.success).toBe(false);
		expect(result.error).toBe('Scanner does not implement required interface (scanPage method)');
	});

	it('returns an instantiation error when factory throws', async () => {
		tempRoot = await mkdtemp(path.join(os.tmpdir(), 'stageflow-plugin-load-factory-throws-'));
		const manifestPath = path.join(tempRoot, 'manifest.json');
		const modulePath = path.join(tempRoot, 'scanner.mjs');

		await writeModule(
			modulePath,
			['export function createScanner() {', "  throw new Error('factory boom');", '}', ''].join(
				'\n'
			)
		);

		const info = pluginInfo(
			manifestPath,
			baseManifest({
				entry: {
					module: './scanner.mjs',
					factoryName: 'createScanner'
				}
			})
		);

		const result = await loadPluginFromManifest(
			info,
			config({ searchPaths: [tempRoot] }),
			loggerMock()
		);

		expect(result.success).toBe(false);
		expect(result.error).toBe('Failed to instantiate scanner: factory boom');
	});

	it('returns an error when exportName is missing from module exports', async () => {
		tempRoot = await mkdtemp(path.join(os.tmpdir(), 'stageflow-plugin-load-export-missing-'));
		const manifestPath = path.join(tempRoot, 'manifest.json');
		const modulePath = path.join(tempRoot, 'scanner.mjs');

		await writeModule(
			modulePath,
			[
				'export class ExistingScanner {',
				'  async scanPage() {',
				'    return {};',
				'  }',
				'}',
				''
			].join('\n')
		);

		const info = pluginInfo(
			manifestPath,
			baseManifest({
				entry: {
					module: './scanner.mjs',
					exportName: 'MissingScanner'
				}
			})
		);

		const result = await loadPluginFromManifest(
			info,
			config({ searchPaths: [tempRoot] }),
			loggerMock()
		);

		expect(result.success).toBe(false);
		expect(result.error).toBe('Export not found: MissingScanner');
	});

	it('returns an error when export is not a constructor', async () => {
		tempRoot = await mkdtemp(
			path.join(os.tmpdir(), 'stageflow-plugin-load-export-not-constructor-')
		);
		const manifestPath = path.join(tempRoot, 'manifest.json');
		const modulePath = path.join(tempRoot, 'scanner.mjs');

		await writeModule(modulePath, ['export const NotAClass = { nope: true };', ''].join('\n'));

		const info = pluginInfo(
			manifestPath,
			baseManifest({
				entry: {
					module: './scanner.mjs',
					exportName: 'NotAClass'
				}
			})
		);

		const result = await loadPluginFromManifest(
			info,
			config({ searchPaths: [tempRoot] }),
			loggerMock()
		);

		expect(result.success).toBe(false);
		expect(result.error).toBe('Export is not a class or constructor');
	});

	it('loads a plugin from default export and logs when verbose mode is enabled', async () => {
		tempRoot = await mkdtemp(path.join(os.tmpdir(), 'stageflow-plugin-load-default-success-'));
		const manifestPath = path.join(tempRoot, 'manifest.json');
		const modulePath = path.join(tempRoot, 'scanner.mjs');
		const logger = loggerMock();

		await writeModule(
			modulePath,
			[
				'export default class DefaultScanner {',
				'  constructor() {',
				"    this.metadata = { name: 'default-scanner', version: '1.0.0' };",
				'  }',
				'  async scanPage() {',
				'    return { success: true };',
				'  }',
				'}',
				''
			].join('\n')
		);

		const info = pluginInfo(manifestPath, baseManifest());

		const result = await loadPluginFromManifest(
			info,
			config({ searchPaths: [tempRoot], verbose: true }),
			logger
		);

		expect(result.success).toBe(true);
		expect(result.plugin?.manifest.id).toBe('test-plugin');
		expect(result.plugin?.path).toBe(tempRoot);
		expect(typeof result.plugin?.factory).toBe('function');
		expect(logger.info).toHaveBeenCalledWith(
			'Plugin loaded',
			expect.objectContaining({
				id: 'test-plugin',
				version: '1.0.0',
				path: tempRoot
			})
		);
	});

	it('returns import error details when module parsing fails', async () => {
		tempRoot = await mkdtemp(path.join(os.tmpdir(), 'stageflow-plugin-load-import-error-'));
		const manifestPath = path.join(tempRoot, 'manifest.json');
		const modulePath = path.join(tempRoot, 'scanner.mjs');

		await writeModule(modulePath, 'export default class BrokenScanner {');

		const info = pluginInfo(manifestPath, baseManifest());
		const result = await loadPluginFromManifest(
			info,
			config({ searchPaths: [tempRoot] }),
			loggerMock()
		);

		expect(result.success).toBe(false);
		expect(result.error).toBeDefined();
		expect(result.error).toMatch(/parse|syntax|expected/i);
	});
});
