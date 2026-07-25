/**
 * HTTP URL normalization and validation.
 *
 * These are general utilities: the landing page, the projects form, and the
 * playground all validate user-entered URLs. They previously lived inside
 * app/lib/components/playground/, a directory that contained no components and
 * implied a playground-only scope they never had.
 */

export function normalizeUrlInput(input: string): string | null {
	const trimmed = input.trim();
	if (!trimmed) return null;

	if (/^[a-zA-Z][a-zA-Z0-9+.-]*:/.test(trimmed)) {
		return trimmed;
	}

	if (trimmed.startsWith('//')) {
		return `https:${trimmed}`;
	}

	return `https://${trimmed}`;
}

export function validateHttpUrls(urls: string[]): {
	valid: string[];
	invalid: { url: string; reason: string }[];
} {
	const valid: string[] = [];
	const invalid: { url: string; reason: string }[] = [];

	for (const url of urls) {
		try {
			const parsed = new URL(url);
			const protocol = parsed.protocol.toLowerCase();
			if (protocol !== 'http:' && protocol !== 'https:') {
				invalid.push({
					url,
					reason: 'URL must start with http:// or https://.'
				});
				continue;
			}
			const hostname = parsed.hostname;
			if (!hostname) {
				invalid.push({ url, reason: 'Missing hostname.' });
				continue;
			}
			const hasDot = hostname.includes('.');
			const isLocalhost = hostname.toLowerCase() === 'localhost';
			// Node may keep brackets; browsers expose bare IPv6 (colons, no dots).
			const isIpv6 =
				(hostname.startsWith('[') && hostname.endsWith(']')) || (!hasDot && hostname.includes(':'));
			if (!hasDot && !isLocalhost && !isIpv6) {
				invalid.push({
					url,
					reason: 'Hostname must contain a dot or be localhost.'
				});
				continue;
			}
			valid.push(url);
		} catch {
			invalid.push({ url, reason: 'Invalid URL.' });
		}
	}

	return { valid, invalid };
}

export type HttpUrlCheck = { ok: true; url: string } | { ok: false; reason: string };

/**
 * Validates exactly one URL.
 *
 * Prefer this over `validateHttpUrls([one])`: the batch result forces every
 * caller to index into an array it knows has one element, which reads as an
 * unhandled empty case and cannot be expressed safely under
 * noUncheckedIndexedAccess.
 */
export function validateHttpUrl(url: string): HttpUrlCheck {
	const { valid, invalid } = validateHttpUrls([url]);

	const rejected = invalid[0];
	if (rejected) {
		return { ok: false, reason: rejected.reason };
	}

	const accepted = valid[0];
	if (accepted !== undefined) {
		return { ok: true, url: accepted };
	}

	// validateHttpUrls always classifies every input, so this is unreachable.
	return { ok: false, reason: 'Invalid URL.' };
}
