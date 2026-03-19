/**
 * Plugin Loader (public surface)
 *
 * Implementation lives in focused modules; this file remains the stable import.
 */

export { createPluginLoader, PluginLoader } from "./plugin-loader";
export type {
  PluginDiscoveryError,
  PluginDiscoveryResult,
  PluginInfo,
  PluginLoaderConfig,
  PluginLoadResult,
  ScannerPlugin,
} from "./plugin-loader-types";
