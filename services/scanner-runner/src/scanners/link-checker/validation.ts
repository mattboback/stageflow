/**
 * Link Validation Utilities
 *
 * Pure functions for link checking and result processing.
 */

import type { IssueSeverity } from '../../core/types';
import type { LinkCheckResult } from './types';

import { followValidatedRedirects } from '../../core/redirect-guard';
import { BlockedTargetError, type TargetValidationPolicy } from '../../core/target-validation';

const REQUEST_TIMEOUT = 10000;
const USER_AGENT = 'Stageflow-LinkChecker/1.0';

async function fetchWithValidatedRedirects(
	url: string,
	method: 'HEAD' | 'GET',
	signal: AbortSignal,
	targetValidationPolicy: TargetValidationPolicy
): Promise<{ response: Response; redirects: string[] }> {
	// A 303 downgrades the method to GET for the remainder of the chain, per
	// RFC 9110. That is specific to this scanner, which may start with HEAD.
	let currentMethod: 'HEAD' | 'GET' = method;

	const { response, redirects } = await followValidatedRedirects<Response>(
		url,
		targetValidationPolicy,
		async (currentURL) => {
			const hopResponse = await fetch(currentURL, {
				method: currentMethod,
				redirect: 'manual',
				signal,
				headers: {
					'User-Agent': USER_AGENT
				}
			});

			if (hopResponse.status === 303) {
				currentMethod = 'GET';
			}

			let location: string | null;
			try {
				location = hopResponse.headers.get('location');
			} catch {
				location = null;
			}

			return { response: hopResponse, status: hopResponse.status, location };
		}
	);

	return { response, redirects };
}

/**
 * Groups link check results by HTTP status code.
 */
export function groupByStatus(links: LinkCheckResult[]): Record<string, LinkCheckResult[]> {
	const grouped: Record<string, LinkCheckResult[]> = {};
	for (const link of links) {
		const status = String(link.status ?? 0);
		grouped[status] ??= [];
		grouped[status].push(link);
	}
	return grouped;
}

/**
 * Maps HTTP status code to issue severity.
 */
export function getSeverityForStatus(status: number): IssueSeverity {
	if (status === 0) {
		return 'serious';
	}
	if (status === 404) {
		return 'serious';
	}
	if (status >= 500) {
		return 'critical';
	}
	if (status >= 400) {
		return 'moderate';
	}
	return 'minor';
}

/**
 * Statuses a HEAD request gets from servers that simply don't serve HEAD (or
 * gate it behind a bot wall); only a GET says whether the link works.
 */
const HEAD_RETRY_STATUSES = new Set([403, 405, 501]);

/**
 * True when the response says "I won't tell a scanner", not "this is broken":
 * auth walls, rate limits, and non-standard bot-wall codes such as LinkedIn's
 * 999. A person clicking the link usually gets the page. A 401/403 from the
 * scanned site itself is not excused: the scan already carries that site's
 * session, so its own link refusing it is a real finding.
 */
export function isUnverifiableStatus(status: number, isInternal = false): boolean {
	if (status === 401 || status === 403) {
		return !isInternal;
	}
	return status === 429 || status >= 600;
}

async function requestLink(
	url: string,
	method: 'HEAD' | 'GET',
	targetValidationPolicy: TargetValidationPolicy
): Promise<{ status: number; redirects: string[] }> {
	const controller = new AbortController();
	const timeoutId = setTimeout(() => {
		controller.abort();
	}, REQUEST_TIMEOUT);
	try {
		const { response, redirects } = await fetchWithValidatedRedirects(
			url,
			method,
			controller.signal,
			targetValidationPolicy
		);
		// Only the status matters; don't download GET bodies.
		void response.body?.cancel();
		return { status: response.status, redirects };
	} finally {
		clearTimeout(timeoutId);
	}
}

/**
 * Checks a single URL for availability, using HEAD with GET fallback.
 */
export async function checkSingleLink(
	url: string,
	targetValidationPolicy: TargetValidationPolicy = { allowedOrigins: [] }
): Promise<LinkCheckResult> {
	const startTime = Date.now();
	const done = (
		status: number | null,
		error: string | null,
		redirects: string[]
	): LinkCheckResult => ({
		url,
		status,
		error,
		redirects,
		responseTime: Date.now() - startTime
	});

	let refusedHead: { status: number; redirects: string[] } | null = null;
	try {
		const head = await requestLink(url, 'HEAD', targetValidationPolicy);
		if (!HEAD_RETRY_STATUSES.has(head.status)) {
			return done(head.status, null, head.redirects);
		}
		refusedHead = head;
	} catch (headError) {
		if (headError instanceof BlockedTargetError) {
			return done(null, headError.message, []);
		}
		// HEAD failed outright; some servers only answer GET.
	}

	try {
		const get = await requestLink(url, 'GET', targetValidationPolicy);
		return done(get.status, null, get.redirects);
	} catch (getError) {
		// Bot walls often answer HEAD with 403 and then stall the GET. The server
		// did respond, so report its status rather than a connection error.
		if (refusedHead) {
			return done(refusedHead.status, null, refusedHead.redirects);
		}
		return done(null, getError instanceof Error ? getError.message : 'Connection failed', []);
	}
}
