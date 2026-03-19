import type { ScannerManifest } from "../manifest";
import type { ScannerBase } from "../scanner-base";

export interface ScannerPlugin {
	manifest: ScannerManifest;
	factory: () => ScannerBase;
	path: string;
}

export interface PluginLoaderConfig {
	searchPaths: string[];
	manifestPatterns: string[];
	strictValidation: boolean;
	verbose: boolean;
}

export interface PluginInfo {
	manifestPath: string;
	manifest: ScannerManifest;
}

export interface PluginDiscoveryResult {
	plugins: PluginInfo[];
	errors: PluginDiscoveryError[];
}

export interface PluginDiscoveryError {
	path: string;
	error: string;
	details?: string;
}

export interface PluginLoadResult {
	success: boolean;
	plugin?: ScannerPlugin;
	error?: string;
}

export const DEFAULT_PLUGIN_LOADER_CONFIG: PluginLoaderConfig = {
	searchPaths: [],
	manifestPatterns: ["manifest.json", "scanner.json"],
	strictValidation: true,
	verbose: false,
};
