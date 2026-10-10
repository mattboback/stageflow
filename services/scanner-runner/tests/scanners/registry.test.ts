import { readFile, stat } from 'node:fs/promises';
import path from 'node:path';
import { describe, expect, it } from 'vitest';

import { validateOptionsAgainstSchema } from '../../src/core/manifest';
import { DEFAULT_DYNAMIC_CONTENT_WAIT_MS } from '../../src/scanners/axe/options';
import { BUILTIN_SCANNER_IDS, loadBuiltinScannerRegistry } from '../../src/scanners/registry';

function escapeRegExp(value: string): string {
	return value.replaceAll(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

describe('built-in scanner registry', () => {
	it('loads and validates every shared built-in manifest', async () => {
		const registry = await loadBuiltinScannerRegistry({ strictValidation: true });

		expect(registry.listIds()).toEqual([...BUILTIN_SCANNER_IDS].sort());
	});

	it('advertises supported axe options while accepting historical unknown settings', async () => {
		const registry = await loadBuiltinScannerRegistry({ strictValidation: true });
		const schema = registry.resolve('axe').manifestConfigSchema;
		if (!schema || typeof schema !== 'object') {
			throw new Error('Expected axe config schema');
		}
		const properties = schema.properties as Record<string, unknown> | undefined;
		expect(Object.keys(properties ?? {}).sort()).toEqual([
			'disabledRules',
			'dynamicContentWaitMs',
			'runOnlyTags'
		]);
		expect(properties?.dynamicContentWaitMs).toMatchObject({
			default: DEFAULT_DYNAMIC_CONTENT_WAIT_MS
		});
		for (const options of [
			{},
			{ disabledRules: [], runOnlyTags: [] },
			{ disabledRules: ['image-alt'], runOnlyTags: ['wcag2a'], dynamicContentWaitMs: 0 },
			{ dynamicContentWaitMs: 125.5 },
			{ rules: ['image-alt'], standard: 'wcag21aa', includeBestPractices: false },
			{ futureOption: { enabled: true } }
		]) {
			expect(validateOptionsAgainstSchema(schema, options)).toEqual({ valid: true, errors: [] });
		}
		for (const options of [
			{ disabledRules: 'image-alt' },
			{ disabledRules: [42] },
			{ runOnlyTags: 'wcag2a' },
			{ runOnlyTags: [false] },
			{ dynamicContentWaitMs: -1 },
			{ dynamicContentWaitMs: '50' }
		]) {
			expect(validateOptionsAgainstSchema(schema, options).valid).toBe(false);
		}
	});

	it('keeps manifest entrypoints aligned with scanner source exports', async () => {
		const repoRoot = path.resolve(process.cwd(), '../..');
		const manifestsRoot = path.join(repoRoot, 'libs/go/scannercatalog/manifests');

		for (const id of BUILTIN_SCANNER_IDS) {
			const manifestPath = path.join(manifestsRoot, id, 'manifest.json');
			await expect(stat(manifestPath)).resolves.toBeTruthy();

			const manifest = JSON.parse(await readFile(manifestPath, 'utf8')) as {
				entry: { module: string; exportName?: string };
			};
			expect(manifest.entry.module).toBe('./index.js');

			const exportName = manifest.entry.exportName;
			if (typeof exportName !== 'string' || exportName.trim().length === 0) {
				throw new Error(`Expected manifest entry exportName to be set for ${id}`);
			}

			const sourcePath = path.join(repoRoot, `services/scanner-runner/src/scanners/${id}/index.ts`);
			const content = await readFile(sourcePath, 'utf8');
			const exportRegex = new RegExp(`export\\s+class\\s+${escapeRegExp(exportName)}\\b`, 'm');
			expect(content).toMatch(exportRegex);
		}
	});

	it('resolves every manifest alias to its built-in scanner', async () => {
		const registry = await loadBuiltinScannerRegistry({ strictValidation: true });
		const repoRoot = path.resolve(process.cwd(), '../..');
		const manifestsRoot = path.join(repoRoot, 'libs/go/scannercatalog/manifests');

		for (const id of BUILTIN_SCANNER_IDS) {
			const manifest = JSON.parse(
				await readFile(path.join(manifestsRoot, id, 'manifest.json'), 'utf8')
			) as { aliases?: string[] };

			for (const token of [id, ...(manifest.aliases ?? [])]) {
				const resolved = registry.resolve(token);
				expect(resolved.manifestId).toBe(id);
				expect(resolved.scanner.metadata.name).toBe(id);
			}
		}
	});
});
