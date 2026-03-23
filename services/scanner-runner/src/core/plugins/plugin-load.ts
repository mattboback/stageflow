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

		const factory = factoryResult.factory;
		const validationError = validateFactory(factory);
		if (validationError) {
			return {
				success: false,
				error: validationError
			};
		}

		const plugin: ScannerPlugin = {
			manifest: info.manifest,
			factory,
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

function validateFactory(factory: () => ScannerBase): string | undefined {
	try {
		const testInstance = factory();
		// metadata is abstract on ScannerBase so must be implemented by subclasses
		if (typeof testInstance.scanPage !== 'function') {
			return 'Scanner does not implement required interface (scanPage method)';
		}
	} catch (err) {
		return `Failed to instantiate scanner: ${err instanceof Error ? err.message : String(err)}`;
	}

	return undefined;
}
