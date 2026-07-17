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
 * The orchestrator emits this env var with the canonical auth shape (form
 * recipe with from_env references, or storage_state with an artifact_key)
 * derived from JobConfig.Auth. Resolved credential values are never present in
 * this env var; only the recipe shape is.
 *
 * Behavior is byte-identical to today when PROVENANCE_AUTH_JSON is unset: no
 * auth block is attached and the synthesized Provenance matches the pre-auth
 * shape on disk.
 */
export function attachAuthFromEnv(provenance: Provenance, logger: ScannerLogger): void {
	const raw = process.env.PROVENANCE_AUTH_JSON;
	if (!raw) {
		return;
	}

	let parsed: unknown;
	try {
		parsed = JSON.parse(raw);
	} catch (err) {
		logger.warn('PROVENANCE_AUTH_JSON is not valid JSON; ignoring auth env var', {
			error: err instanceof Error ? err.message : String(err)
		});
		return;
	}

	if (!isProvenanceAuth(parsed)) {
		logger.warn('PROVENANCE_AUTH_JSON did not match Provenance.auth shape; ignoring', {
			parsed: typeof parsed === 'object' && parsed !== null ? Object.keys(parsed) : typeof parsed
		});
		return;
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
