/**
 * SSRF-safe redirect following.
 *
 * A redirect chain is an SSRF hole if only the first hop is validated: a public
 * URL can 302 to 169.254.169.254. Every hop therefore has to be re-checked
 * against the target policy, and the chain has to be bounded.
 *
 * This lived twice — once in the link-checker scanner over `fetch`, once in the
 * security-headers scanner over Playwright's `page.request.fetch` — and the two
 * copies had already drifted apart in their error handling. Since SECURITY.md
 * names SSRF as an in-scope boundary, the loop, the hop cap, and the location
 * resolution are defined once here, and callers supply only the transport.
 */

import { type TargetValidationPolicy, validateTargetURLForPolicy } from './target-validation';

/** Maximum hops before a chain is treated as a redirect loop. */
export const MAX_VALIDATED_REDIRECTS = 10;

/** Statuses that carry a `Location` and continue the chain. */
const REDIRECT_STATUSES = new Set([301, 302, 303, 307, 308]);

export function isRedirectStatus(status: number): boolean {
	return REDIRECT_STATUSES.has(status);
}

/** One transport response, reduced to what the redirect loop needs. */
export interface RedirectHop<TResponse> {
	response: TResponse;
	status: number;
	/** The raw `Location` header, or null when absent or unreadable. */
	location: string | null;
}

export interface FollowedRedirects<TResponse> {
	response: TResponse;
	finalURL: string;
	redirects: string[];
}

/**
 * Requests `startURL`, following redirects while validating every hop.
 *
 * @param request issues one request. Receives the URL to fetch and the zero-based
 *   hop number, and reports the status plus `Location` back to the loop.
 * @throws BlockedTargetError when any hop violates `policy` — including hops
 *   reached only via redirect, which is the point of this function.
 * @throws Error when the chain exceeds {@link MAX_VALIDATED_REDIRECTS}.
 */
export async function followValidatedRedirects<TResponse>(
	startURL: string,
	policy: TargetValidationPolicy,
	request: (url: string, hop: number) => Promise<RedirectHop<TResponse>>
): Promise<FollowedRedirects<TResponse>> {
	let currentURL = startURL;
	const redirects: string[] = [];

	for (let hop = 0; hop <= MAX_VALIDATED_REDIRECTS; hop++) {
		await validateTargetURLForPolicy(currentURL, policy);

		const { response, status, location } = await request(currentURL, hop);

		// Not a redirect, or a redirect we cannot follow: this is the final response.
		if (!isRedirectStatus(status) || location === null) {
			return { response, finalURL: currentURL, redirects };
		}

		let nextURL: string;
		try {
			nextURL = new URL(location, currentURL).toString();
		} catch {
			// A malformed Location is not followable; return what the server sent.
			return { response, finalURL: currentURL, redirects };
		}

		redirects.push(nextURL);
		currentURL = nextURL;
	}

	throw new Error(`Too many redirects (>${MAX_VALIDATED_REDIRECTS})`);
}
