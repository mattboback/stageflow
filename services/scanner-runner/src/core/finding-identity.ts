import { createHash } from 'node:crypto';

function digest(value: string): string {
	return createHash('sha256').update(value).digest('hex').slice(0, 16);
}

/** Keep path, query and selector case: they can identify different resources. */
export function stablePageId(rawUrl: string, includeOrigin = false): string {
	let identity = rawUrl.split('#')[0] ?? rawUrl;
	try {
		const url = new URL(rawUrl);
		identity = `${includeOrigin ? url.origin : ''}${url.pathname}${url.search}`;
	} catch {
		// Relative archive paths retain the same case and query semantics.
	}
	return `page-v2-${digest(identity)}`;
}

export function hasMultipleOrigins(urls: string[]): boolean {
	return (
		new Set(
			urls.flatMap((raw) => {
				try {
					return [new URL(raw).origin];
				} catch {
					return [];
				}
			})
		).size > 1
	);
}

export function issueFingerprint(
	scanner: string,
	ruleId: string,
	pageId: string,
	selector = ''
): string {
	return digest(JSON.stringify([scanner, ruleId, pageId, selector]));
}

export function occurrenceFingerprint(
	scanner: string,
	ruleId: string,
	pageId: string,
	target: string[]
): string {
	return `occ-v2-${digest(JSON.stringify([scanner, ruleId, pageId, target]))}`;
}
