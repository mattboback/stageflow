/**
 * Security Headers Scanner
 *
 * Analyzes HTTP security headers and detects common security misconfigurations.
 */

import type { APIResponse, Page } from 'playwright';

import type { Issue, IssueSeverity, PageScanResult, ScanContext } from '../../core/types';

import { ScannerBase } from '../../core/scanner-base';
import { followValidatedRedirects } from '../../core/redirect-guard';
import type { TargetValidationPolicy } from '../../core/target-validation';
import { SCANNER_VERSION } from '../version';

interface SecurityHeader {
	name: string;
	severity: IssueSeverity;
	title: string;
	description: string;
	helpUrl: string;
	validator?: (value: string) => boolean;
}

const HEADER_FIXES: Record<string, string> = {
	'content-security-policy':
		"Start with Content-Security-Policy: default-src 'self'; object-src 'none'; base-uri 'self'; frame-ancestors 'none'. Test in report-only mode and allow the sources your application needs before enforcing it.",
	'strict-transport-security':
		'After verifying HTTPS works for every route, send Strict-Transport-Security: max-age=31536000. Add includeSubDomains only when all subdomains support HTTPS.',
	'x-frame-options':
		"Send Content-Security-Policy: frame-ancestors 'self' (or 'none' when framing is unnecessary), or X-Frame-Options: SAMEORIGIN.",
	'x-content-type-options':
		'Send X-Content-Type-Options: nosniff and serve resources with their correct Content-Type.',
	'referrer-policy': 'Send Referrer-Policy: strict-origin-when-cross-origin.',
	'permissions-policy':
		'Send Permissions-Policy: camera=(), microphone=(), geolocation=(). Enable only the features and origins your application needs.'
};

const SECURITY_HEADERS: SecurityHeader[] = [
	{
		name: 'content-security-policy',
		severity: 'serious',
		title: 'Missing Content Security Policy',
		description:
			'CSP helps prevent XSS attacks by specifying allowed content sources. Without it, your site is more vulnerable to cross-site scripting attacks.',
		helpUrl: 'https://developer.mozilla.org/en-US/docs/Web/HTTP/CSP'
	},
	{
		name: 'strict-transport-security',
		severity: 'serious',
		title: 'Missing HSTS Header',
		description:
			'HSTS ensures browsers only connect via HTTPS, protecting against protocol downgrade attacks and cookie hijacking.',
		helpUrl: 'https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Strict-Transport-Security'
	},
	{
		name: 'x-frame-options',
		severity: 'moderate',
		title: 'Missing X-Frame-Options',
		description:
			'Prevents clickjacking by controlling whether the page can be embedded in iframes on other sites.',
		helpUrl: 'https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/X-Frame-Options'
	},
	{
		name: 'x-content-type-options',
		severity: 'moderate',
		title: 'Missing X-Content-Type-Options',
		description:
			'Prevents MIME type sniffing attacks by ensuring browsers respect the declared Content-Type.',
		helpUrl: 'https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/X-Content-Type-Options',
		validator: (value) => value.toLowerCase() === 'nosniff'
	},
	{
		name: 'referrer-policy',
		severity: 'minor',
		title: 'Missing Referrer Policy',
		description:
			'Controls how much referrer information is sent with requests, helping protect user privacy.',
		helpUrl: 'https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Referrer-Policy'
	},
	{
		name: 'permissions-policy',
		severity: 'minor',
		title: 'Missing Permissions Policy',
		description:
			'Controls which browser features (camera, microphone, geolocation, etc.) can be used by the page.',
		helpUrl: 'https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Permissions-Policy'
	}
];

async function fetchWithValidatedRedirects(
	page: Page,
	url: string,
	targetValidationPolicy: TargetValidationPolicy
): Promise<{ response: APIResponse; finalURL: string; redirects: string[] }> {
	return followValidatedRedirects<APIResponse>(url, targetValidationPolicy, async (currentURL) => {
		// maxRedirects: 0 keeps Playwright from following hops itself, which would
		// bypass the per-hop policy check.
		const response = await page.request.fetch(currentURL, {
			method: 'GET',
			timeout: 30_000,
			maxRedirects: 0
		});

		return {
			response,
			status: response.status(),
			location: response.headers().location ?? null
		};
	});
}

export class SecurityHeadersScanner extends ScannerBase {
	readonly metadata = {
		name: 'security-headers',
		version: SCANNER_VERSION,
		description: 'HTTP security headers analysis'
	};

	async scanPage(context: ScanContext): Promise<PageScanResult> {
		const { page, pageEntry, logger } = context;
		const startTime = Date.now();
		const issues: Issue[] = [];
		const rules: NonNullable<PageScanResult['rules']> = [];

		try {
			const currentPageURL = page.url();
			const targetURL =
				currentPageURL.startsWith('http://') || currentPageURL.startsWith('https://')
					? currentPageURL
					: pageEntry.url;

			const { response, finalURL, redirects } = await fetchWithValidatedRedirects(
				page,
				targetURL,
				context.targetValidationPolicy ?? { allowedOrigins: [] }
			);

			const headers = response.headers();
			const headerKeys = new Set(Object.keys(headers).map((k) => k.toLowerCase()));
			const statusCode = response.status();

			for (const check of SECURITY_HEADERS) {
				const headerName = check.name.toLowerCase();
				const hasHeader = headerKeys.has(headerName);
				const headerValue = headers[headerName];

				if (
					headerName === 'x-frame-options' &&
					/(?:^|;)\s*frame-ancestors\s+[^;]+/i.test(headers['content-security-policy'] ?? '')
				) {
					rules.push(
						{ ruleId: `${this.metadata.name}-missing-${check.name}`, outcome: 'pass' },
						{ ruleId: `${this.metadata.name}-invalid-${check.name}`, outcome: 'pass' }
					);
					continue;
				}

				rules.push(
					{
						ruleId: `${this.metadata.name}-missing-${check.name}`,
						outcome: hasHeader ? 'pass' : 'fail'
					},
					{
						ruleId: `${this.metadata.name}-invalid-${check.name}`,
						outcome: !hasHeader
							? 'inapplicable'
							: check.validator && !check.validator(headerValue ?? '')
								? 'fail'
								: 'pass'
					}
				);
				if (!hasHeader) {
					issues.push(this.createIssue(check, 'missing'));
				} else if (check.validator && !check.validator(headerValue ?? '')) {
					issues.push(this.createIssue(check, 'invalid', headerValue));
				}
			}

			const mixedContent =
				new URL(finalURL).protocol === 'https:' ? await this.checkMixedContent(page) : [];
			if (mixedContent.length > 0) {
				issues.push({
					id: `${this.metadata.name}-mixed-content`,
					scanner: this.metadata.name,
					severity: 'serious',
					category: 'security',
					title: 'Mixed Content Detected',
					description: `Page loads ${mixedContent.length} insecure resource(s) over HTTP. This can compromise the security of HTTPS pages.`,
					helpUrl: 'https://developer.mozilla.org/en-US/docs/Web/Security/Mixed_content',
					howToFix:
						'Replace each http:// resource URL with a working https:// URL, or serve the resource from your HTTPS origin.',
					metadata: {
						resources: mixedContent.slice(0, 10),
						totalCount: mixedContent.length
					}
				});
			}

			const insecureCookies = this.checkCookieSecurity(headers);
			if (insecureCookies.length > 0) {
				issues.push({
					id: `${this.metadata.name}-insecure-cookies`,
					scanner: this.metadata.name,
					severity: 'moderate',
					category: 'security',
					title: 'Insecure Cookie Configuration',
					description: `Found ${insecureCookies.length} cookie(s) without recommended security flags (Secure, SameSite).`,
					helpUrl: 'https://developer.mozilla.org/en-US/docs/Web/HTTP/Cookies#security',
					howToFix:
						'For cookies on HTTPS, send Set-Cookie: name=value; Secure; SameSite=Lax; Path=/. Add HttpOnly for session cookies that JavaScript does not need. Cross-site cookies need SameSite=None; Secure.',
					metadata: {
						cookies: insecureCookies
					}
				});
			}

			rules.push(
				{
					ruleId: `${this.metadata.name}-mixed-content`,
					outcome:
						new URL(finalURL).protocol !== 'https:'
							? 'inapplicable'
							: mixedContent.length
								? 'fail'
								: 'pass'
				},
				{
					ruleId: `${this.metadata.name}-insecure-cookies`,
					outcome: insecureCookies.length ? 'fail' : 'pass'
				}
			);
			logger.info('Security scan complete', {
				url: pageEntry.url,
				finalURL,
				redirects: redirects.length,
				issues: issues.length,
				statusCode
			});

			return {
				pageId: pageEntry.id,
				url: pageEntry.url,
				path: pageEntry.path,
				success: true,
				finalUrl: finalURL,
				rules:
					statusCode >= 400 ? rules.map((rule) => ({ ...rule, outcome: 'error' as const })) : rules,
				issues,
				durationMs: Date.now() - startTime,
				startedAt: new Date(startTime).toISOString(),
				finishedAt: new Date().toISOString(),
				rawResults: {
					headers: Object.fromEntries(
						Object.entries(headers).filter(([k]) =>
							SECURITY_HEADERS.some((h) => h.name === k.toLowerCase())
						)
					),
					statusCode,
					mixedContentCount: mixedContent.length
				}
			};
		} catch (error) {
			logger.error('Security scan failed', {
				url: pageEntry.url,
				error: error instanceof Error ? error.message : String(error)
			});
			return this.createErrorResult(
				pageEntry,
				startTime,
				error instanceof Error ? error.message : String(error)
			);
		}
	}

	private createIssue(check: SecurityHeader, type: 'missing' | 'invalid', value?: string): Issue {
		const id =
			type === 'missing'
				? `${this.metadata.name}-missing-${check.name}`
				: `${this.metadata.name}-invalid-${check.name}`;

		const title = type === 'missing' ? check.title : `Invalid ${check.name} Header`;

		const description =
			type === 'missing'
				? check.description
				: `Header present but has invalid value: "${value}". ${check.description}`;

		return {
			id,
			scanner: this.metadata.name,
			severity: check.severity,
			category: 'security-headers',
			title,
			description,
			helpUrl: check.helpUrl,
			...(HEADER_FIXES[check.name] ? { howToFix: HEADER_FIXES[check.name] } : {}),
			metadata: {
				headerName: check.name,
				...(value && { headerValue: value })
			}
		};
	}

	private async checkMixedContent(page: Page): Promise<string[]> {
		return page.evaluate(() => {
			const insecureResources: string[] = [];

			for (const el of document.querySelectorAll('img[src^="http:"]')) {
				insecureResources.push((el as HTMLImageElement).src);
			}

			for (const el of document.querySelectorAll('script[src^="http:"]')) {
				insecureResources.push((el as HTMLScriptElement).src);
			}

			for (const el of document.querySelectorAll('link[rel="stylesheet"][href^="http:"]')) {
				insecureResources.push((el as HTMLLinkElement).href);
			}

			for (const el of document.querySelectorAll('iframe[src^="http:"]')) {
				insecureResources.push((el as HTMLIFrameElement).src);
			}

			for (const el of document.querySelectorAll(
				'video source[src^="http:"], audio source[src^="http:"]'
			)) {
				insecureResources.push((el as HTMLSourceElement).src);
			}

			return insecureResources;
		});
	}

	private checkCookieSecurity(
		headers: Record<string, string>
	): { name: string; issues: string[] }[] {
		const setCookieHeaders = Object.entries(headers)
			.filter(([k]) => k.toLowerCase() === 'set-cookie')
			.flatMap(([, v]) => v.split('\n'));

		const insecureCookies: { name: string; issues: string[] }[] = [];

		for (const cookie of setCookieHeaders) {
			if (!cookie) {
				continue;
			}
			const parts = cookie.split(';').map((p) => p.trim().toLowerCase());
			const [rawName] = cookie.split('=');
			const cookieName = (rawName ?? '').trim();
			const issues: string[] = [];

			if (!parts.some((p) => p === 'secure')) {
				issues.push('Missing Secure flag');
			}
			if (!parts.some((p) => p.startsWith('samesite'))) {
				issues.push('Missing SameSite attribute');
			}

			if (issues.length > 0) {
				insecureCookies.push({ name: cookieName, issues });
			}
		}

		return insecureCookies;
	}

	private createErrorResult(
		pageEntry: { id: string; url: string; path?: string },
		startTime: number,
		error: string
	): PageScanResult {
		return {
			pageId: pageEntry.id,
			url: pageEntry.url,
			success: false,
			issues: [],
			durationMs: Date.now() - startTime,
			startedAt: new Date(startTime).toISOString(),
			finishedAt: new Date().toISOString(),
			error,
			...(pageEntry.path !== undefined ? { path: pageEntry.path } : {})
		};
	}
}
