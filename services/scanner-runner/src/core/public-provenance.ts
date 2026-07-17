import type { Provenance } from './types';

/**
 * Build the provenance document that may be exposed through the job API.
 *
 * The execution document can contain literal form values, environment-variable
 * references, and storage-state object keys. None of that recipe belongs in a
 * downloadable artifact. Keep only enough metadata to explain that an
 * authenticated scan was requested and which hydration mode was used.
 */
export function buildPublicProvenance(provenance: Provenance): Provenance {
	const { auth, ...publicProvenance } = provenance;
	const publicPages = provenance.pages.map((page) => {
		const { pre_scan_actions: _preScanActions, ...publicPage } = page;
		return publicPage;
	});

	return {
		...publicProvenance,
		pages: publicPages,
		metadata: {
			...(provenance.metadata ?? {}),
			auth_configured: auth !== undefined,
			...(auth !== undefined ? { auth_mode: auth.mode } : {})
		}
	};
}
