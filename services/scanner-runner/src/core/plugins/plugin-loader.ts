/**
 * Plugin Loader
 *
 * Discovers, validates, and loads scanner plugins from multiple sources.
 * Supports built-in scanners, filesystem plugins, and volume-mounted plugins.
 */

import path from "node:path";

import type { ScannerLogger } from "../types";

import { createLogger } from "../../utils/logger";
import { discoverPluginManifests } from "./plugin-discovery";
import { loadPluginFromManifest } from "./plugin-load";
import {
	DEFAULT_PLUGIN_LOADER_CONFIG,
	type PluginDiscoveryResult,
	type PluginInfo,
	type PluginLoadResult,
	type PluginLoaderConfig,
	type ScannerPlugin,
} from "./plugin-loader-types";

const logger = createLogger("PluginLoader");

export class PluginLoader {
	private config: PluginLoaderConfig;
	private loadedPlugins = new Map<string, ScannerPlugin>();
	private discoveredManifests = new Map<string, PluginInfo>();
	private aliasMap = new Map<string, string>();
	private logger: ScannerLogger;

	constructor(config: Partial<PluginLoaderConfig> = {}) {
		this.config = { ...DEFAULT_PLUGIN_LOADER_CONFIG, ...config };
		this.logger = logger;
	}

	async discover(): Promise<PluginDiscoveryResult> {
		const { plugins, errors, manifestsById, aliasByToken } =
			await discoverPluginManifests(this.config, this.logger);

		this.discoveredManifests = manifestsById;
		this.aliasMap = aliasByToken;

		if (this.config.verbose) {
			this.logger.info("Plugin discovery complete", {
				found: plugins.length,
				errors: errors.length,
				plugins: plugins.map((p) => p.manifest.id),
			});
		}

		return { plugins, errors };
	}

	async load(pluginIdOrAlias: string): Promise<PluginLoadResult> {
		const cleaned = pluginIdOrAlias.trim();
		if (!cleaned) {
			return {
				success: false,
				error: "Plugin ID is required",
			};
		}

		const pluginId =
			this.aliasMap.get(cleaned.toLowerCase()) ??
			(this.discoveredManifests.has(cleaned) ? cleaned : undefined);

		if (!pluginId) {
			return {
				success: false,
				error: `Plugin not found: ${cleaned}. Run discover() first or check plugin ID.`,
			};
		}

		if (this.loadedPlugins.has(pluginId)) {
			return {
				success: true,
				plugin: this.loadedPlugins.get(pluginId),
			};
		}

		const info = this.discoveredManifests.get(pluginId);
		if (!info) {
			return {
				success: false,
				error: `Plugin not found: ${pluginId}. Run discover() first or check plugin ID.`,
			};
		}

		return this.loadFromManifest(info);
	}

	async loadFromManifest(info: PluginInfo): Promise<PluginLoadResult> {
		const result = await loadPluginFromManifest(info, this.config, this.logger);
		if (!result.success || !result.plugin) {
			return result;
		}

		this.loadedPlugins.set(info.manifest.id, result.plugin);
		return result;
	}

	async loadAll(): Promise<Map<string, PluginLoadResult>> {
		const results = new Map<string, PluginLoadResult>();

		for (const [id, info] of this.discoveredManifests) {
			const result = await this.loadFromManifest(info);
			results.set(id, result);
		}

		return results;
	}

	get(pluginId: string): ScannerPlugin | undefined {
		return this.loadedPlugins.get(pluginId);
	}

	getAll(): Map<string, ScannerPlugin> {
		return new Map(this.loadedPlugins);
	}

	has(pluginId: string): boolean {
		return this.loadedPlugins.has(pluginId);
	}

	listDiscovered(): string[] {
		return [...this.discoveredManifests.keys()];
	}

	listLoaded(): string[] {
		return [...this.loadedPlugins.keys()];
	}

	getDiscoveredInfo(pluginId: string): PluginInfo | undefined {
		return this.discoveredManifests.get(pluginId);
	}

	unload(pluginId: string): boolean {
		return this.loadedPlugins.delete(pluginId);
	}

	unloadAll(): void {
		this.loadedPlugins.clear();
	}

	clearDiscovered(): void {
		this.discoveredManifests.clear();
		this.aliasMap.clear();
	}

	reset(): void {
		this.loadedPlugins.clear();
		this.discoveredManifests.clear();
		this.aliasMap.clear();
	}

	addSearchPath(searchPath: string): void {
		if (!this.config.searchPaths.includes(searchPath)) {
			this.config.searchPaths.push(searchPath);
		}
	}

	getConfig(): Readonly<PluginLoaderConfig> {
		return { ...this.config };
	}
}

export function createPluginLoader(
	additionalPaths: string[] = [],
	config: Partial<PluginLoaderConfig> = {},
): PluginLoader {
	const defaultPaths = [
		// Built-in scanners
		path.join(__dirname, "..", "..", "scanners"),
		// Volume-mounted plugins
		"/plugins",
	];

	// Add home directory plugins for development
	if (process.env.HOME) {
		defaultPaths.push(path.join(process.env.HOME, ".stageflow", "plugins"));
	}

	// Add environment-specified paths
	if (process.env.PLUGIN_PATHS) {
		defaultPaths.push(...process.env.PLUGIN_PATHS.split(":").filter(Boolean));
	}

	return new PluginLoader({
		searchPaths: [...defaultPaths, ...additionalPaths],
		strictValidation: process.env.NODE_ENV === "production",
		verbose: process.env.PLUGIN_VERBOSE === "true",
		...config,
	});
}
