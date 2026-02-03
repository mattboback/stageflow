import fs from "fs-extra";
import path from "node:path";

import type { ScannerLogger } from "../types";
import type {
  PluginDiscoveryError,
  PluginDiscoveryResult,
  PluginInfo,
  PluginLoaderConfig,
} from "./plugin-loader-types";

import { loadManifest, type ScannerManifest, validateManifest } from "../manifest";

export async function discoverPluginManifests(
  config: PluginLoaderConfig,
  logger: ScannerLogger,
): Promise<
  PluginDiscoveryResult & {
    manifestsById: Map<string, PluginInfo>;
    aliasByToken: Map<string, string>;
  }
> {
  const plugins: PluginInfo[] = [];
  const errors: PluginDiscoveryError[] = [];

  for (const searchPath of config.searchPaths) {
    try {
      const found = await discoverInPath(searchPath, config, logger);
      plugins.push(...found.plugins);
      errors.push(...found.errors);
    } catch (err) {
      errors.push({
        path: searchPath,
        error: "Failed to search path",
        details: err instanceof Error ? err.message : String(err),
      });
    }
  }

  const manifestsById = new Map<string, PluginInfo>();
  const aliasByToken = new Map<string, string>();

  for (const plugin of plugins) {
    const id = plugin.manifest.id;

    if (manifestsById.has(id)) {
      errors.push({
        path: plugin.manifestPath,
        error: `Duplicate plugin id: ${id}`,
      });
      continue;
    }

    manifestsById.set(id, plugin);
    addAlias(aliasByToken, errors, plugin.manifestPath, id, id);

    for (const alias of plugin.manifest.aliases ?? []) {
      const cleaned = alias.trim();
      if (!cleaned) {
        errors.push({
          path: plugin.manifestPath,
          error: `Empty alias declared for plugin id: ${id}`,
        });
        continue;
      }

      addAlias(aliasByToken, errors, plugin.manifestPath, cleaned, id);
    }
  }

  return { plugins, errors, manifestsById, aliasByToken };
}

async function discoverInPath(
  searchPath: string,
  config: PluginLoaderConfig,
  logger: ScannerLogger,
): Promise<PluginDiscoveryResult> {
  const plugins: PluginInfo[] = [];
  const errors: PluginDiscoveryError[] = [];

  if (!(await fs.pathExists(searchPath))) {
    if (config.verbose) {
      logger.debug("Search path does not exist", { path: searchPath });
    }

    return { plugins, errors };
  }

  const stat = await fs.stat(searchPath);

  if (stat.isFile()) {
    const result = await tryLoadManifest(searchPath, config, logger);
    if (result.success && result.info) {
      plugins.push(result.info);
    } else if (result.error) {
      errors.push({ path: searchPath, error: result.error });
    }

    return { plugins, errors };
  }

  if (!stat.isDirectory()) {
    return { plugins, errors };
  }

  const entries = await fs.readdir(searchPath, { withFileTypes: true });

  for (const entry of entries) {
    const entryPath = path.join(searchPath, entry.name);

    if (entry.isDirectory()) {
      for (const pattern of config.manifestPatterns) {
        const manifestPath = path.join(entryPath, pattern);
        if (await fs.pathExists(manifestPath)) {
          const result = await tryLoadManifest(manifestPath, config, logger);
          if (result.success && result.info) {
            plugins.push(result.info);
            break;
          } else if (result.error) {
            errors.push({ path: manifestPath, error: result.error });
          }
        }
      }

      continue;
    }

    if (entry.isFile() && config.manifestPatterns.includes(entry.name)) {
      const result = await tryLoadManifest(entryPath, config, logger);
      if (result.success && result.info) {
        plugins.push(result.info);
      } else if (result.error) {
        errors.push({ path: entryPath, error: result.error });
      }
    }
  }

  return { plugins, errors };
}

async function tryLoadManifest(
  manifestPath: string,
  config: PluginLoaderConfig,
  logger: ScannerLogger,
): Promise<{ success: boolean; info?: PluginInfo; error?: string }> {
  try {
    const manifest = await loadManifest(manifestPath);
    return validateAndReturn(manifestPath, manifest, config, logger);
  } catch (err) {
    return {
      success: false,
      error: err instanceof Error ? err.message : String(err),
    };
  }
}

function validateAndReturn(
  manifestPath: string,
  manifest: ScannerManifest,
  config: PluginLoaderConfig,
  logger: ScannerLogger,
): { success: boolean; info?: PluginInfo; error?: string } {
  const validation = validateManifest(manifest);

  if (!validation.valid && config.strictValidation) {
    return {
      success: false,
      error: `Validation failed: ${validation.errors.map((e) => e.message).join(", ")}`,
    };
  }

  if (validation.warnings.length > 0 && config.verbose) {
    logger.warn("Manifest warnings", {
      path: manifestPath,
      warnings: validation.warnings,
    });
  }

  return {
    success: true,
    info: { manifestPath, manifest },
  };
}

function addAlias(
  aliasByToken: Map<string, string>,
  errors: PluginDiscoveryError[],
  manifestPath: string,
  token: string,
  pluginId: string,
): void {
  const key = token.toLowerCase();
  const existing = aliasByToken.get(key);
  if (existing && existing !== pluginId) {
    errors.push({
      path: manifestPath,
      error: `Alias '${token.trim()}' already used by plugin '${existing}'`,
    });
    return;
  }

  aliasByToken.set(key, pluginId);
}
