/**
 * Scanner Worker
 *
 * Entry point for running scanners in worker mode.
 * Uses the plugin system (manifests) for scanner resolution.
 */

import type { ManifestConfigSchema } from "@stageflow/contracts-scanner-manifest";

import { type ScannerBase, loadConfigFromEnv, validateConfig } from "./core";
import { type PluginLoader, createPluginLoader } from "./core/plugins";
import { createLogger } from "./utils/logger";
import {
	assertScannerIdMatchesManifest,
	assertScannerOptionsMatchSchema,
} from "./worker/worker-validation";

const logger = createLogger("Worker");

async function initializePlugins(): Promise<PluginLoader> {
	const loader = createPluginLoader();
	logger.info("Initializing plugin loader", {
		searchPaths: loader.getConfig().searchPaths,
	});

	const discovery = await loader.discover();

	if (discovery.errors.length > 0) {
		logger.warn("Plugin discovery errors", {
			errorCount: discovery.errors.length,
			errors: discovery.errors.map((e) => ({ path: e.path, error: e.error })),
		});
	}

	if (discovery.plugins.length > 0) {
		logger.info("Plugins discovered", {
			count: discovery.plugins.length,
			plugins: discovery.plugins.map((p) => ({
				id: p.manifest.id,
				version: p.manifest.version,
			})),
		});
	}

	return loader;
}

async function getScanner(
	scannerType: string,
	pluginLoader: PluginLoader,
): Promise<{
	scanner: ScannerBase;
	manifestId: string;
	manifestConfigSchema?: ManifestConfigSchema;
}> {
	const loadResult = await pluginLoader.load(scannerType);

	if (loadResult.success && loadResult.plugin) {
		const strict = pluginLoader.getConfig().strictValidation;

		logger.info("Using plugin scanner", {
			id: loadResult.plugin.manifest.id,
			version: loadResult.plugin.manifest.version,
			path: loadResult.plugin.path,
		});

		const scanner = loadResult.plugin.factory();
		const manifestId = loadResult.plugin.manifest.id;

		assertScannerIdMatchesManifest({
			manifestId,
			scannerId: scanner.metadata.name,
			strict,
			logger,
		});

		return {
			scanner,
			manifestId,
			manifestConfigSchema: loadResult.plugin.manifest.configSchema,
		};
	}

	const availableFromPlugins = pluginLoader.listDiscovered();
	const loadError = loadResult.error?.trim();

	if (loadError) {
		throw new Error(
			`Failed to load scanner type "${scannerType}": ${loadError}. Available scanners: ${
				availableFromPlugins.join(", ") || "none"
			}`,
		);
	}

	throw new Error(
		`Unknown scanner type: "${scannerType}". Available scanners: ${
			availableFromPlugins.join(", ") || "none"
		}`,
	);
}

export async function runWorkerMode(): Promise<void> {
	const scannerType = process.env.SCANNER_TYPE?.trim() ?? "axe";

	logger.info("Starting worker", {
		scannerType,
		nodeEnv: process.env.NODE_ENV,
	});

	let pluginLoader: PluginLoader;
	try {
		pluginLoader = await initializePlugins();
	} catch (err) {
		logger.error("Failed to initialize plugin system", {
			error: err instanceof Error ? err.message : String(err),
		});
		process.exit(1);
	}

	let scanner: ScannerBase;
	let manifestId = "";
	let manifestConfigSchema: ManifestConfigSchema | undefined;
	try {
		const resolved = await getScanner(scannerType, pluginLoader);
		scanner = resolved.scanner;
		manifestId = resolved.manifestId;
		manifestConfigSchema = resolved.manifestConfigSchema;
	} catch (err) {
		logger.error("Failed to get scanner", {
			type: scannerType,
			error: err instanceof Error ? err.message : String(err),
		});
		process.exit(1);
	}

	const config = loadConfigFromEnv({
		scannerName: scanner.metadata.name,
	});

	try {
		validateConfig(config);
	} catch (err) {
		logger.error("Invalid configuration", {
			error: err instanceof Error ? err.message : String(err),
		});
		process.exit(1);
	}

	if (manifestConfigSchema !== undefined) {
		try {
			const strict = pluginLoader.getConfig().strictValidation;
			assertScannerOptionsMatchSchema({
				manifestId,
				schema: manifestConfigSchema,
				options: config.options,
				strict,
				logger,
			});
		} catch (err) {
			logger.error(
				"Failed to validate SCANNER_OPTIONS against manifest configSchema",
				{
					scanner: manifestId,
					error: err instanceof Error ? err.message : String(err),
				},
			);

			if (pluginLoader.getConfig().strictValidation) {
				process.exit(1);
			}
		}
	}

	logger.info("Configuration loaded", {
		jobId: config.jobId,
		scanner: scanner.metadata.name,
		version: scanner.metadata.version,
		concurrency: config.concurrency,
		provenancePath: config.provenancePath,
		resultsDir: config.resultsDir,
	});

	try {
		const results = await scanner.run(config);

		logger.info("Scan completed successfully", {
			jobId: config.jobId,
			pagesScanned: results.pages.length,
			pagesFailed: results.summary.pagesFailed,
			totalIssues: results.summary.totalIssues,
			durationMs: results.durationMs,
			bySeverity: results.summary.bySeverity,
		});

		process.exit(0);
	} catch (err) {
		logger.error("Scan failed", {
			jobId: config.jobId,
			error: err instanceof Error ? err.message : String(err),
			stack: err instanceof Error ? err.stack : undefined,
		});
		process.exit(1);
	}
}
