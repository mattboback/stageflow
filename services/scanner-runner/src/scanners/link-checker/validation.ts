/**
 * Link Validation Utilities
 *
 * Pure functions for link checking and result processing.
 */

import type { IssueSeverity } from '../../core/types';
import type { LinkCheckResult } from './types';

import {
	BlockedTargetError,
	shouldEnforceRuntimeTargetValidation,
	validateRuntimeTargetURL
} from '../../core/target-validation';

const REQUEST_TIMEOUT = 10000;
const USER_AGENT = 'Stageflow-LinkChecker/1.0';
const MAX_REDIRECTS = 10;
const REDIRECT_STATUSES = new Set([301, 302, 303, 307, 308]);

async function fetchWithValidatedRedirects(
	url: string,
	method: 'HEAD' | 'GET',
	signal: AbortSignal
): Promise<{ response: Response; redirects: string[] }> {
	let currentURL = url;
	let currentMethod: 'HEAD' | 'GET' = method;
	const redirects: string[] = [];

	for (let redirectCount = 0; redirectCount <= MAX_REDIRECTS; redirectCount++) {
		if (shouldEnforceRuntimeTargetValidation()) {
			await validateRuntimeTargetURL(currentURL);
		}

		const response = await fetch(currentURL, {
			method: currentMethod,
			redirect: 'manual',
			signal,
			headers: {
				'User-Agent': USER_AGENT
			}
		});

		if (!REDIRECT_STATUSES.has(response.status)) {
			return { response, redirects };
		}

		let locationHeader: string | null = null;
		try {
			locationHeader = response.headers.get('location');
		} catch {
			locationHeader = null;
		}
		if (!locationHeader) {
			return { response, redirects };
		}

		const nextURL = new URL(locationHeader, currentURL).toString();
		redirects.push(nextURL);
		currentURL = nextURL;

		if (response.status === 303 && currentMethod !== 'GET') {
			currentMethod = 'GET';
		}
	}

	throw new Error(`Too many redirects (>${MAX_REDIRECTS})`);
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
export async function checkSingleLink(url: string): Promise<LinkCheckResult> {
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
				controller.signal
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
					controller.signal
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
