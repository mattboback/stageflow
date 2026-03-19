/**
 * Plugin System Module
 *
 * Provides plugin discovery and loading capabilities for scanner extensions.
 */

export {
	createPluginLoader,
	type PluginDiscoveryError,
	type PluginDiscoveryResult,
	type PluginInfo,
	PluginLoader,
	type PluginLoaderConfig,
	type PluginLoadResult,
	type ScannerPlugin,
} from "./loader";
