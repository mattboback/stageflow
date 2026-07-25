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
 * Checks a single URL for availability, using HEAD with GET fallback.
 */
export async function checkSingleLink(
	url: string,
	targetValidationPolicy: TargetValidationPolicy = { allowedOrigins: [] }
): Promise<LinkCheckResult> {
	const startTime = Date.now();

	try {
		const controller = new AbortController();
		const timeoutId = setTimeout(() => {
			controller.abort();
		}, REQUEST_TIMEOUT);
		try {
			const { response, redirects } = await fetchWithValidatedRedirects(
				url,
				'HEAD',
				controller.signal,
				targetValidationPolicy
			);

			return {
				url,
				status: response.status,
				error: null,
				redirects,
				responseTime: Date.now() - startTime
			};
		} finally {
			clearTimeout(timeoutId);
		}
	} catch (headError) {
		if (headError instanceof BlockedTargetError) {
			return {
				url,
				status: null,
				error: headError.message,
				redirects: [],
				responseTime: Date.now() - startTime
			};
		}

		// If HEAD fails, try GET (some servers don't support HEAD)
		try {
			const controller = new AbortController();
			const timeoutId = setTimeout(() => {
				controller.abort();
			}, REQUEST_TIMEOUT);
			try {
				const { response, redirects } = await fetchWithValidatedRedirects(
					url,
					'GET',
					controller.signal,
					targetValidationPolicy
				);

				return {
					url,
					status: response.status,
					error: null,
					redirects,
					responseTime: Date.now() - startTime
				};
			} finally {
				clearTimeout(timeoutId);
			}
		} catch (getError) {
			return {
				url,
				status: null,
				error: getError instanceof Error ? getError.message : 'Connection failed',
				redirects: [],
				responseTime: Date.now() - startTime
			};
		}
	}
}
