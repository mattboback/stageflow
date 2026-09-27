/**
 * Provenance synthesis helpers for page iteration: turning SCAN_URLS and
 * PROVENANCE_AUTH_JSON environment input into a well-formed Provenance.
 */

import type { Provenance, ProvenanceAuth, ScannerLogger } from './types';

export function normalizeUrl(url: string): string {
	const trimmed = url.trim();
	if (!trimmed) {
		return trimmed;
	}

	if (/^https?:\/\//i.test(trimmed)) {
		return trimmed;
	}

	return `https://${trimmed}`;
}

export function parseScanUrls(raw: string): string[] {
	let parsed: unknown;
	try {
		parsed = JSON.parse(raw);
	} catch (err) {
		const message = err instanceof Error ? err.message : String(err);
		throw new Error(`Failed to create provenance from SCAN_URLS: ${message}`, { cause: err });
	}

	if (!Array.isArray(parsed) || parsed.some((value) => typeof value !== 'string')) {
		throw new Error('SCAN_URLS must be a JSON array of URLs (strings)');
	}

	return parsed as string[];
}

/**
 * Attaches a Provenance.auth block from PROVENANCE_AUTH_JSON when present.
 *
 * The orchestrator transports canonical literal auth. Keep it in memory;
 * synthesized execution provenance and public artifacts omit this block.
 */
export function attachAuthFromEnv(provenance: Provenance, logger: ScannerLogger): void {
	const raw = process.env.PROVENANCE_AUTH_JSON;
	if (raw === undefined) {
		return;
	}

	let parsed: unknown;
	try {
		parsed = JSON.parse(raw);
	} catch {
		throw new Error('PROVENANCE_AUTH_JSON is not valid JSON');
	}

	if (!isProvenanceAuth(parsed)) {
		throw new Error('PROVENANCE_AUTH_JSON did not match Provenance.auth shape');
	}

	provenance.auth = parsed;
	logger.info('Attached auth block from PROVENANCE_AUTH_JSON', {
		mode: parsed.mode
	});
}

function isProvenanceAuth(value: unknown): value is ProvenanceAuth {
	if (typeof value !== 'object' || value === null) {
		return false;
	}

	const v = value as Record<string, unknown>;
	if (v.mode === 'storage_state') {
		return typeof v.artifact_key === 'string' && v.artifact_key.length > 0;
	}

	if (v.mode === 'form') {
		return (
			typeof v.login_url === 'string' &&
			Array.isArray(v.steps) &&
			typeof v.success === 'object' &&
			v.success !== null
		);
	}

	return false;
}
