import type { ManifestConfigSchema, ScannerManifest } from '@stageflow/contracts-scanner-manifest';

import { readFile } from 'node:fs/promises';
import path from 'node:path';

import type { ScannerBase } from '../core/scanner-base';

import { validateManifest } from '../core/manifest';
import {
	AiNavigatorScanner,
	AxeScanner,
	LighthouseScanner,
	LinkCheckerScanner,
	OpenGraphScanner,
	SecurityHeadersScanner,
	SEOScanner,
	SpellingGrammarScanner
} from './index';

export const BUILTIN_SCANNER_IDS = [
	'ai-navigator',
	'axe',
	'lighthouse',
	'link-checker',
	'open-graph',
	'security-headers',
	'seo',
	'spelling-grammar'
] as const;

type BuiltinScannerId = (typeof BUILTIN_SCANNER_IDS)[number];

interface BuiltinScannerDefinition {
	id: BuiltinScannerId;
	create: () => ScannerBase;
}

interface BuiltinScannerEntry extends BuiltinScannerDefinition {
	manifest: ScannerManifest;
}

export interface ResolvedBuiltinScanner {
	scanner: ScannerBase;
	manifestId: string;
	manifestConfigSchema?: ManifestConfigSchema;
	manifestMaxConcurrency?: number;
}

const BUILTIN_SCANNERS: readonly BuiltinScannerDefinition[] = [
	{ id: 'ai-navigator', create: () => new AiNavigatorScanner() },
	{ id: 'axe', create: () => new AxeScanner() },
	{ id: 'lighthouse', create: () => new LighthouseScanner() },
	{ id: 'link-checker', create: () => new LinkCheckerScanner() },
	{ id: 'open-graph', create: () => new OpenGraphScanner() },
	{ id: 'security-headers', create: () => new SecurityHeadersScanner() },
	{ id: 'seo', create: () => new SEOScanner() },
	{ id: 'spelling-grammar', create: () => new SpellingGrammarScanner() }
];

export class BuiltinScannerRegistry {
	private readonly entriesById: ReadonlyMap<string, BuiltinScannerEntry>;
	private readonly aliasesByToken: ReadonlyMap<string, string>;
	readonly strictValidation: boolean;

	constructor(entries: BuiltinScannerEntry[], strictValidation: boolean) {
		this.entriesById = new Map(entries.map((entry) => [entry.id, entry]));
		this.aliasesByToken = new Map(
			entries.flatMap((entry) =>
				(entry.manifest.aliases ?? []).map((alias) => [alias.trim().toLowerCase(), entry.id])
			)
		);
		this.strictValidation = strictValidation;
	}

	resolve(scannerType: string): ResolvedBuiltinScanner {
		const token = scannerType.trim().toLowerCase();
		const id = this.aliasesByToken.get(token) ?? token;
		const entry = this.entriesById.get(id);

		if (!entry) {
			throw new Error(
				`Unknown scanner type: "${scannerType}". Available scanners: ${this.listIds().join(', ')}`
			);
		}

		const scanner = entry.create();
		const manifestId = entry.manifest.id;

		return {
			scanner,
			manifestId,
			...(entry.manifest.configSchema !== undefined
				? { manifestConfigSchema: entry.manifest.configSchema }
				: {}),
			...(entry.manifest.capabilities.maxConcurrency !== undefined
				? { manifestMaxConcurrency: entry.manifest.capabilities.maxConcurrency }
				: {})
		};
	}

	listIds(): string[] {
		return [...this.entriesById.keys()].sort();
	}
}

export async function loadBuiltinScannerRegistry(options: {
	strictValidation: boolean;
	logger?: { warn(message: string, meta?: Record<string, unknown>): void };
}): Promise<BuiltinScannerRegistry> {
	const entries: BuiltinScannerEntry[] = [];

	for (const definition of BUILTIN_SCANNERS) {
		const manifest = await loadBuiltinManifest(definition.id);
		const validation = validateManifest(manifest);

		if (!validation.valid) {
			throw new Error(
				`Invalid built-in scanner manifest "${definition.id}": ${validation.errors
					.map((error) => `${error.path} ${error.message}`)
					.join('; ')}`
			);
		}

		if (validation.warnings.length > 0) {
			options.logger?.warn('Built-in scanner manifest warnings', {
				id: definition.id,
				warnings: validation.warnings
			});
		}

		if (manifest.id !== definition.id) {
			throw new Error(
				`Built-in scanner manifest id mismatch: expected "${definition.id}", got "${manifest.id}"`
			);
		}

		entries.push({ ...definition, manifest });
	}

	return new BuiltinScannerRegistry(entries, options.strictValidation);
}

async function loadBuiltinManifest(id: BuiltinScannerId): Promise<ScannerManifest> {
	const candidates = [
		path.join(__dirname, id, 'manifest.json'),
		path.resolve(process.cwd(), 'libs/go/scannercatalog/manifests', id, 'manifest.json'),
		path.resolve(process.cwd(), '../../libs/go/scannercatalog/manifests', id, 'manifest.json'),
		path.resolve(__dirname, '../../../..', 'libs/go/scannercatalog/manifests', id, 'manifest.json')
	];

	const errors: string[] = [];
	for (const candidate of candidates) {
		try {
			return JSON.parse(await readFile(candidate, 'utf8')) as ScannerManifest;
		} catch (err) {
			if ((err as NodeJS.ErrnoException).code === 'ENOENT') {
				errors.push(candidate);
				continue;
			}
			throw err;
		}
	}

	throw new Error(`Built-in scanner manifest "${id}" not found. Tried: ${errors.join(', ')}`);
}
