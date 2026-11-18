import type { Page } from 'playwright';

import type { Issue, ProvenanceAuth } from './types';
import { sameOriginAndPath } from '../utils/url-compare';

const LOGIN_PATH_PATTERNS: RegExp[] = [
	/^\/login\/?$/i,
	/^\/log-in\/?$/i,
	/^\/sign[_-]?in\/?$/i,
	/^\/auth(?:\/|$)/i,
	/^\/sso(?:\/|$)/i,
	/^\/oauth(?:\/|$)/i,
	/^\/openid(?:\/|$)/i,
	/^\/accounts?\/login\/?$/i,
	/^\/users\/sign_in\/?$/i,
	/^\/identity\/account\/login/i,
	/^\/admin\/login\/?$/i
];

type AuthWallSignal =
	| { type: 'login_redirect'; finalUrl: string; matchedPattern: string }
	| { type: 'login_form'; selector: string };

interface DetectAuthWallOptions {
	page: Page;
	requestedUrl: string;
	scannerName: string;
	authConfigured: boolean;
	authHydrated: boolean;
	authMode?: ProvenanceAuth['mode'];
}

export async function detectAuthWall(options: DetectAuthWallOptions): Promise<Issue | null> {
	const finalUrl = safePageUrl(options.page);
	const signals: AuthWallSignal[] = [];
	const loginRedirect = detectLoginRedirect(options.requestedUrl, finalUrl);
	if (loginRedirect) {
		signals.push(loginRedirect);
	}

	const requestedPathIsLogin = isLoginPath(options.requestedUrl);
	if (!requestedPathIsLogin) {
		const formSignal = await detectLoginForm(options.page);
		if (formSignal) {
			signals.push(formSignal);
		}
	}

	if (signals.length === 0) {
		return null;
	}

	const severity = options.authConfigured && options.authHydrated ? 'serious' : 'info';
	const metadata: Record<string, unknown> = {
		signals,
		requested_url: options.requestedUrl,
		final_url: finalUrl,
		auth_configured: options.authConfigured
	};
	if (options.authMode !== undefined) {
		metadata.auth_mode = options.authMode;
	}

	return {
		id: 'auth-wall-detected',
		scanner: options.scannerName,
		severity,
		category: 'auth',
		title: options.authConfigured
			? 'Authenticated scan landed on a login page'
			: 'Scan landed on a login page',
		description: buildDescription(options.authConfigured),
		metadata
	};
}

function detectLoginRedirect(requestedUrl: string, finalUrl: string): AuthWallSignal | null {
	if (sameOriginAndPath(requestedUrl, finalUrl)) {
		return null;
	}

	const matchedPattern = matchingLoginPattern(finalUrl);
	if (!matchedPattern) {
		return null;
	}

	return { type: 'login_redirect', finalUrl, matchedPattern };
}

async function detectLoginForm(page: Page): Promise<AuthWallSignal | null> {
	try {
		const selector = await page.evaluate(() => {
			const passwordInputs = [
				...document.querySelectorAll<HTMLInputElement>('input[type="password"]')
			];
			for (const passwordInput of passwordInputs) {
				const form = passwordInput.closest('form');
				if (!form) {
					continue;
				}

				const hasIdentityInput = [
					...form.querySelectorAll<HTMLInputElement>('input[name], input[id], input[autocomplete]')
				].some((input) => {
					const haystack = [
						input.name,
						input.id,
						input.autocomplete,
						input.getAttribute('aria-label') ?? ''
					]
						.join(' ')
						.toLowerCase();
					return /\b(email|user|login|account|username)\b/.test(haystack);
				});

				if (hasIdentityInput) {
					return selectorFor(form);
				}
			}

			return null;

			function selectorFor(element: Element): string {
				if (element.id) {
					return `#${CSS.escape(element.id)}`;
				}

				const tag = element.tagName.toLowerCase();
				const parent = element.parentElement;
				if (!parent) {
					return tag;
				}

				const siblings = [...parent.children].filter(
					(sibling) => sibling.tagName === element.tagName
				);
				if (siblings.length <= 1) {
					return tag;
				}

				return `${tag}:nth-of-type(${siblings.indexOf(element) + 1})`;
			}
		});

		return selector ? { type: 'login_form', selector } : null;
	} catch {
		return null;
	}
}

function buildDescription(authConfigured: boolean): string {
	if (authConfigured) {
		return (
			'Authentication was configured, but the scan still landed on a login page. ' +
			'Refresh the stored session or tighten the form recipe success selector.'
		);
	}

	return (
		'The scan landed on a login page instead of protected content. ' +
		'Configure authenticated scanning to scan the post-login surface.'
	);
}

function matchingLoginPattern(rawUrl: string): string | null {
	let pathname: string;
	try {
		pathname = new URL(rawUrl).pathname;
	} catch {
		pathname = rawUrl;
	}

	for (const pattern of LOGIN_PATH_PATTERNS) {
		if (pattern.test(pathname)) {
			return pattern.source;
		}
	}

	return null;
}

function isLoginPath(rawUrl: string): boolean {
	return matchingLoginPattern(rawUrl) !== null;
}

function safePageUrl(page: Page): string {
	try {
		return page.url();
	} catch {
		return '';
	}
}
