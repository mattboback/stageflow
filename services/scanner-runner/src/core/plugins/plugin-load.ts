import fs from 'fs-extra';
import path from 'node:path';

import type { ScannerBase } from '../scanner-base';
import type { ScannerLogger } from '../types';
import type {
	PluginInfo,
	PluginLoadResult,
	PluginLoaderConfig,
	ScannerPlugin
} from './plugin-loader-types';

import { resolveEntryPath } from '../manifest';

export async function loadPluginFromManifest(
	info: PluginInfo,
	config: PluginLoaderConfig,
	logger: ScannerLogger
): Promise<PluginLoadResult> {
	try {
		const manifestDir = path.dirname(info.manifestPath);

		// Security: verify the manifest lives inside a configured search path.
		// This prevents loading plugins from arbitrary filesystem locations.
		const resolvedManifestDir = path.resolve(manifestDir);
		const insideSearchPath = config.searchPaths.some((sp) => {
			const resolvedSp = path.resolve(sp);
			return (
				resolvedManifestDir === resolvedSp || resolvedManifestDir.startsWith(resolvedSp + path.sep)
			);
		});
		if (!insideSearchPath) {
			return {
				success: false,
				error: `Plugin manifest is outside configured search paths: ${info.manifestPath}`
			};
		}

		// Security: resolveEntryPath rejects absolute paths and paths that escape
		// the plugin directory via ".." traversal, so the import() below is
		// constrained to files within the plugin's own directory tree.
		const entryPath = resolveEntryPath(info.manifest, manifestDir);

		if (!(await fs.pathExists(entryPath))) {
			return {
				success: false,
				error: `Entry point not found: ${entryPath}`
			};
		}

		const module: Record<string, unknown> = (await import(entryPath)) as Record<string, unknown>;
		const factoryResult = resolveFactory(module, info);
		if (!factoryResult.ok) {
			return {
				success: false,
				error: factoryResult.error
			};
		}

		const instanceResult = instantiateScanner(factoryResult.factory);
		if (!instanceResult.ok) {
			return {
				success: false,
				error: instanceResult.error
			};
		}

		const scanner = instanceResult.scanner;
		const plugin: ScannerPlugin = {
			manifest: info.manifest,
			factory: () => scanner,
			path: manifestDir
		};

		if (config.verbose) {
			logger.info('Plugin loaded', {
				id: info.manifest.id,
				version: info.manifest.version,
				path: manifestDir
			});
		}

		return { success: true, plugin };
	} catch (err) {
		return {
			success: false,
			error: err instanceof Error ? err.message : String(err)
		};
	}
}

type FactoryResolution = { ok: true; factory: () => ScannerBase } | { ok: false; error: string };

function resolveFactory(module: Record<string, unknown>, info: PluginInfo): FactoryResolution {
	if (info.manifest.entry.factoryName) {
		const factoryFn = module[info.manifest.entry.factoryName];
		if (typeof factoryFn !== 'function') {
			return {
				ok: false,
				error: `Factory function not found: ${info.manifest.entry.factoryName}`
			};
		}

		return { ok: true, factory: factoryFn as () => ScannerBase };
	}

	const exportName = info.manifest.entry.exportName?.trim() ?? 'default';
	const ScannerClass = exportName === 'default' ? module.default : module[exportName];

	if (!ScannerClass) {
		return {
			ok: false,
			error: `Export not found: ${exportName}`
		};
	}

	if (typeof ScannerClass !== 'function') {
		return {
			ok: false,
			error: 'Export is not a class or constructor'
		};
	}

	type ScannerConstructor = new () => ScannerBase;
	return {
		ok: true,
		factory: () => new (ScannerClass as ScannerConstructor)()
	};
}

type ScannerInstantiation =
	| { ok: true; scanner: ScannerBase }
	| { ok: false; error: string };

function instantiateScanner(factory: () => ScannerBase): ScannerInstantiation {
	try {
		const scanner = factory();
		// metadata is abstract on ScannerBase so must be implemented by subclasses
		if (typeof scanner.scanPage !== 'function') {
			return { ok: false, error: 'Scanner does not implement required interface (scanPage method)' };
		}

		return { ok: true, scanner };
	} catch (err) {
		return {
			ok: false,
			error: `Failed to instantiate scanner: ${err instanceof Error ? err.message : String(err)}`
		};
	}
}
