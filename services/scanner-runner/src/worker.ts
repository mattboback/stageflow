/**
 * Scanner Worker
 *
 * Entry point for running scanners in worker mode.
 */

import type { ManifestConfigSchema } from '@stageflow/contracts-scanner-manifest';

import { type ScannerBase, loadConfigFromEnv, validateConfig } from './core';
import { type BuiltinScannerRegistry, loadBuiltinScannerRegistry } from './scanners/registry';
import { createLogger } from './utils/logger';
import {
	assertScannerIdMatchesManifest,
	assertScannerOptionsMatchSchema
} from './worker/worker-validation';

const logger = createLogger('Worker');

function getScanner(
	scannerType: string,
	registry: BuiltinScannerRegistry
): {
	scanner: ScannerBase;
	manifestId: string;
	manifestConfigSchema?: ManifestConfigSchema;
	manifestMaxConcurrency?: number;
} {
	const resolved = registry.resolve(scannerType);

	logger.info('Using built-in scanner', {
		id: resolved.manifestId,
		requested: scannerType
	});

	assertScannerIdMatchesManifest({
		manifestId: resolved.manifestId,
		scannerId: resolved.scanner.metadata.name,
		strict: registry.strictValidation,
		logger
	});

	return resolved;
}

export async function runWorkerMode(): Promise<void> {
	const scannerType = process.env.SCANNER_TYPE?.trim() ?? 'axe';

	logger.info('Starting worker', {
		scannerType,
		nodeEnv: process.env.NODE_ENV
	});

	let scannerRegistry: BuiltinScannerRegistry;
	try {
		scannerRegistry = await loadBuiltinScannerRegistry({
			strictValidation: process.env.NODE_ENV === 'production',
			logger
		});
		logger.info('Built-in scanner registry loaded', {
			scanners: scannerRegistry.listIds()
		});
	} catch (err) {
		logger.error('Failed to initialize scanner registry', {
			error: err instanceof Error ? err.message : String(err)
		});
		process.exit(1);
	}

	let scanner: ScannerBase;
	let manifestId = '';
	let manifestConfigSchema: ManifestConfigSchema | undefined;
	let manifestMaxConcurrency: number | undefined;
	try {
		const resolved = getScanner(scannerType, scannerRegistry);
		scanner = resolved.scanner;
		manifestId = resolved.manifestId;
		manifestConfigSchema = resolved.manifestConfigSchema;
		manifestMaxConcurrency = resolved.manifestMaxConcurrency;
	} catch (err) {
		logger.error('Failed to get scanner', {
			type: scannerType,
			error: err instanceof Error ? err.message : String(err)
		});
		process.exit(1);
	}

	const config = loadConfigFromEnv({
		scannerName: scanner.metadata.name,
		...(manifestMaxConcurrency !== undefined
			? { defaults: { concurrency: manifestMaxConcurrency } }
			: {})
	});

	try {
		validateConfig(config);
	} catch (err) {
		logger.error('Invalid configuration', {
			error: err instanceof Error ? err.message : String(err)
		});
		process.exit(1);
	}

	if (manifestConfigSchema !== undefined) {
		try {
			assertScannerOptionsMatchSchema({
				manifestId,
				schema: manifestConfigSchema,
				options: config.options,
				strict: scannerRegistry.strictValidation,
				logger
			});
		} catch (err) {
			logger.error('Failed to validate SCANNER_OPTIONS against manifest configSchema', {
				scanner: manifestId,
				error: err instanceof Error ? err.message : String(err)
			});

			if (scannerRegistry.strictValidation) {
				process.exit(1);
			}
		}
	}

	logger.info('Configuration loaded', {
		jobId: config.jobId,
		scanner: scanner.metadata.name,
		version: scanner.metadata.version,
		concurrency: config.concurrency,
		provenancePath: config.provenancePath,
		resultsDir: config.resultsDir
	});

	try {
		const results = await scanner.run(config);

		logger.info('Scan completed successfully', {
			jobId: config.jobId,
			pagesScanned: results.pages.length,
			pagesFailed: results.summary.pagesFailed,
			totalIssues: results.summary.totalIssues,
			durationMs: results.durationMs,
			bySeverity: results.summary.bySeverity
		});

		process.exit(0);
	} catch (err) {
		logger.error('Scan failed', {
			jobId: config.jobId,
			error: err instanceof Error ? err.message : String(err),
			stack: err instanceof Error ? err.stack : undefined
		});
		process.exit(1);
	}
}
