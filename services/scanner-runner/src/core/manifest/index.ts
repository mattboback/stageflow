/**
 * Scanner Manifest Utilities
 *
 * Loader and validation for scanner plugin manifests.
 */

import type { ManifestConfigSchema, ScannerManifest } from '@stageflow/contracts-scanner-manifest';

import manifestSchema from '@stageflow/contracts-scanner-manifest/schema';
import Ajv from 'ajv';
import addFormats from 'ajv-formats';
import fs from 'fs-extra';
import path from 'node:path';

export type { ScannerManifest } from '@stageflow/contracts-scanner-manifest';

const ajv = new Ajv({ allErrors: true });
addFormats(ajv);
const validateSchema = ajv.compile(manifestSchema);

interface ManifestValidationResult {
	valid: boolean;
	errors: ManifestValidationError[];
	warnings: ManifestValidationWarning[];
}

interface ManifestValidationError {
	path: string;
	message: string;
	value?: unknown;
}

interface ManifestValidationWarning {
	path: string;
	message: string;
	suggestion?: string;
}

export async function loadManifest(manifestPath: string): Promise<ScannerManifest> {
	const content = await fs.readFile(manifestPath, 'utf-8');
	const manifest = JSON.parse(content) as ScannerManifest;
	return manifest;
}

export function validateManifest(manifest: ScannerManifest): ManifestValidationResult {
	const errors: ManifestValidationError[] = [];
	const warnings: ManifestValidationWarning[] = [];

	const valid = validateSchema(manifest);
	if (!valid && validateSchema.errors) {
		for (const error of validateSchema.errors) {
			errors.push({
				path: error.instancePath || '/',
				message: error.message ?? 'Validation failed',
				value: error.data
			});
		}
	}

	if (manifest.id && manifest.id.length < 3) {
		warnings.push({
			path: '/id',
			message: 'Scanner ID is very short',
			suggestion: 'Consider using a more descriptive ID'
		});
	}

	if (!manifest.description) {
		warnings.push({
			path: '/description',
			message: 'Missing description',
			suggestion: 'Add a description to help users understand the scanner purpose'
		});
	}

	if (manifest.capabilities.supportsScreenshots && !manifest.capabilities.requiresBrowser) {
		errors.push({
			path: '/capabilities',
			message: 'Screenshots require browser capability'
		});
	}

	if (manifest.configSchema !== undefined) {
		if (typeof manifest.configSchema !== 'object' && typeof manifest.configSchema !== 'boolean') {
			errors.push({
				path: '/configSchema',
				message: 'configSchema must be an object or boolean'
			});
		} else {
			try {
				ajv.compile(manifest.configSchema);
			} catch (err) {
				errors.push({
					path: '/configSchema',
					message: `Invalid configSchema: ${err instanceof Error ? err.message : String(err)}`
				});
			}
		}
	}

	return {
		valid: errors.length === 0,
		errors,
		warnings
	};
}

export function resolveEntryPath(manifest: ScannerManifest, manifestDir: string): string {
	const entryModule = manifest.entry.module.trim();
	if (!entryModule) {
		throw new Error('Manifest entry.module must not be empty');
	}
	if (path.isAbsolute(entryModule)) {
		throw new Error(`Absolute plugin entry paths are not allowed: ${entryModule}`);
	}

	const resolvedPath = path.resolve(manifestDir, entryModule);
	const relativePath = path.relative(manifestDir, resolvedPath);
	if (relativePath.startsWith('..') || path.isAbsolute(relativePath)) {
		throw new Error(`Plugin entry path escapes plugin directory: ${entryModule}`);
	}

	return resolvedPath;
}

export function validateOptionsAgainstSchema(
	schema: ManifestConfigSchema,
	value: unknown
): { valid: boolean; errors: string[] } {
	const validate = ajv.compile(schema);
	const valid = validate(value);

	if (valid) {
		return { valid: true, errors: [] };
	}

	const errors = validate.errors?.map(
		(e) => `${e.instancePath || '/'} ${e.message ?? 'invalid'}`
	) ?? ['Invalid SCANNER_OPTIONS'];

	return { valid: false, errors };
}
