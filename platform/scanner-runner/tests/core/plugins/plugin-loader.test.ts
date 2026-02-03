import path from "node:path";
import { afterAll, beforeEach, describe, expect, it, vi } from "vitest";

import type { ScannerManifest } from "../../../src/core/manifest";
import type {
  PluginDiscoveryError,
  PluginDiscoveryResult,
  PluginInfo,
  PluginLoaderConfig,
  ScannerPlugin,
} from "../../../src/core/plugins/plugin-loader-types";

import { discoverPluginManifests } from "../../../src/core/plugins/plugin-discovery";
import { loadPluginFromManifest } from "../../../src/core/plugins/plugin-load";
import {
  createPluginLoader,
  PluginLoader,
} from "../../../src/core/plugins/plugin-loader";

type PluginDiscoveryState = PluginDiscoveryResult & {
  manifestsById: Map<string, PluginInfo>;
  aliasByToken: Map<string, string>;
};

vi.mock("../../../src/core/plugins/plugin-discovery", () => ({
  discoverPluginManifests: vi.fn(),
}));

vi.mock("../../../src/core/plugins/plugin-load", () => ({
  loadPluginFromManifest: vi.fn(),
}));

const discoverPluginManifestsMock = vi.mocked(discoverPluginManifests);
const loadPluginFromManifestMock = vi.mocked(loadPluginFromManifest);

function makeManifest(id: string, aliases: string[] = []): ScannerManifest {
  return {
    id,
    aliases,
    name: id,
    version: "1.0.0",
    capabilities: {
      categories: ["accessibility"],
      outputFormats: ["json"],
      supportsScreenshots: false,
      supportsConcurrency: false,
      requiresBrowser: false,
    },
    entry: { module: "index.js" },
  };
}

function makePluginInfo(id: string, aliases: string[] = []): PluginInfo {
  const manifest = makeManifest(id, aliases);
  return {
    manifestPath: `/plugins/${id}/manifest.json`,
    manifest,
  };
}

function makeDiscoveryState(
  infos: PluginInfo[],
  errors: PluginDiscoveryError[] = [],
): PluginDiscoveryState {
  return {
    plugins: infos,
    errors,
    manifestsById: new Map(infos.map((info) => [info.manifest.id, info])),
    aliasByToken: new Map(
      infos.flatMap((info) =>
        (info.manifest.aliases ?? []).map((a) => [a.toLowerCase(), info.manifest.id]),
      ),
    ),
  };
}

function makePlugin(info: PluginInfo): ScannerPlugin {
  return {
    manifest: info.manifest,
    path: info.manifestPath,
    // We don’t execute the scanner in these tests; a minimal stub keeps typing happy.
    factory: () =>
      ({
        metadata: { name: info.manifest.id, version: info.manifest.version },
      }) as unknown as import("../../../src/core/scanner-base").ScannerBase,
  };
}

describe("PluginLoader", () => {
  const baseConfig: PluginLoaderConfig = {
    searchPaths: [],
    manifestPatterns: ["manifest.json"],
    strictValidation: true,
    verbose: false,
  };

  beforeEach(() => {
    vi.clearAllMocks();
    // Provide a default empty discovery so tests can override per-case.
    discoverPluginManifestsMock.mockResolvedValue(makeDiscoveryState([]));
  });

  it("discovers plugins, resolves aliases, and caches loaded plugins", async () => {
    const info = makePluginInfo("scanner-1", ["alpha"]);
    const plugin = makePlugin(info);

    discoverPluginManifestsMock.mockResolvedValue(makeDiscoveryState([info]));
    loadPluginFromManifestMock.mockResolvedValue({ success: true, plugin });

    const loader = new PluginLoader(baseConfig);
    const discovery = await loader.discover();
    expect(discovery.plugins).toHaveLength(1);

    const first = await loader.load("alpha");
    expect(first.success).toBe(true);
    expect(first.plugin).toBe(plugin);
    expect(loadPluginFromManifestMock).toHaveBeenCalledTimes(1);

    const second = await loader.load("scanner-1");
    expect(second.success).toBe(true);
    // second call should hit the cache, not the loader
    expect(loadPluginFromManifestMock).toHaveBeenCalledTimes(1);
    expect(loader.has("scanner-1")).toBe(true);
    expect(loader.listLoaded()).toEqual(["scanner-1"]);
  });

  it("returns errors for missing ids, undiscovered plugins, and failed loads", async () => {
    const info = makePluginInfo("scanner-2", ["beta"]);
    discoverPluginManifestsMock.mockResolvedValue(makeDiscoveryState([info]));
    loadPluginFromManifestMock.mockResolvedValue({ success: false, error: "boom" });

    const loader = new PluginLoader(baseConfig);
    await loader.discover();

    const emptyId = await loader.load("   ");
    expect(emptyId.success).toBe(false);
    expect(emptyId.error).toMatch(/required/);

    const missing = await loader.load("missing");
    expect(missing.success).toBe(false);
    expect(missing.error).toMatch(/Plugin not found/);

    const failed = await loader.load("beta");
    expect(failed.success).toBe(false);
    expect(failed.error).toBe("boom");
    expect(loader.get("scanner-2")).toBeUndefined();
  });

  it("loads all discovered manifests and exposes helper accessors", async () => {
    const infoA = makePluginInfo("scanner-a");
    const infoB = makePluginInfo("scanner-b", ["bee"]);
    const pluginA = makePlugin(infoA);
    const pluginB = makePlugin(infoB);

    discoverPluginManifestsMock.mockResolvedValue(makeDiscoveryState([infoA, infoB]));
    loadPluginFromManifestMock
      .mockResolvedValueOnce({ success: true, plugin: pluginA })
      .mockResolvedValueOnce({ success: true, plugin: pluginB });

    const loader = new PluginLoader(baseConfig);
    await loader.discover();
    const results = await loader.loadAll();

    expect(results.size).toBe(2);
    expect(results.get("scanner-a")?.plugin).toBe(pluginA);
    expect(results.get("scanner-b")?.plugin).toBe(pluginB);
    expect(loader.listDiscovered()).toEqual(["scanner-a", "scanner-b"]);

    // state helpers
    expect(loader.getDiscoveredInfo("scanner-b")).toEqual(infoB);
    expect(loader.get("scanner-a")).toBe(pluginA);
    expect(loader.unload("scanner-a")).toBe(true);
    expect(loader.has("scanner-a")).toBe(false);

    loader.unloadAll();
    expect(loader.listLoaded()).toEqual([]);

    loader.clearDiscovered();
    expect(loader.listDiscovered()).toEqual([]);
  });

  it("adds search paths without duplicates and exposes config view", () => {
    const loader = new PluginLoader({
      ...baseConfig,
      searchPaths: ["/existing"],
    });

    loader.addSearchPath("/existing");
    loader.addSearchPath("/new");

    const cfg = loader.getConfig();
    expect(cfg.searchPaths).toEqual(["/existing", "/new"]);
    // shallow copy: external mutation should reflect current implementation
    cfg.searchPaths.push("/mutated");
    expect(loader.getConfig().searchPaths).toContain("/mutated");
  });
});

describe("createPluginLoader", () => {
  const originalEnv = { ...process.env };

  beforeEach(() => {
    vi.resetModules();
    process.env = { ...originalEnv };
  });

  it("builds default and environment-driven search paths", () => {
    process.env.HOME = "/home/tester";
    process.env.PLUGIN_PATHS = "/opt/extra:/mnt/plugins";
    process.env.NODE_ENV = "production";
    process.env.PLUGIN_VERBOSE = "true";

    const loader = createPluginLoader(["/addon"], {
      manifestPatterns: ["manifest.json"],
    });
    const cfg = loader.getConfig();

    expect(cfg.strictValidation).toBe(true); // NODE_ENV production
    expect(cfg.verbose).toBe(true); // PLUGIN_VERBOSE true
    expect(cfg.searchPaths).toEqual(
      expect.arrayContaining([
        path.join(process.cwd(), "src", "scanners"),
        "/plugins",
        "/home/tester/.stageflow/plugins",
        "/opt/extra",
        "/mnt/plugins",
        "/addon",
      ]),
    );
  });

  afterAll(() => {
    process.env = originalEnv;
  });
});
