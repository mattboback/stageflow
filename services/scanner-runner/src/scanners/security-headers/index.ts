/**
 * Security Headers Scanner
 *
 * Analyzes HTTP security headers and detects common security misconfigurations.
 */

import type { Page } from "playwright";

import type {
	Issue,
	IssueSeverity,
	PageScanResult,
	ScanContext,
} from "../../core/types";

import { ScannerBase } from "../../core/scanner-base";

interface SecurityHeader {
	name: string;
	severity: IssueSeverity;
	title: string;
	description: string;
	helpUrl: string;
	validator?: (value: string) => boolean;
}

const SECURITY_HEADERS: SecurityHeader[] = [
	{
		name: "content-security-policy",
		severity: "serious",
		title: "Missing Content Security Policy",
		description:
			"CSP helps prevent XSS attacks by specifying allowed content sources. Without it, your site is more vulnerable to cross-site scripting attacks.",
		helpUrl: "https://developer.mozilla.org/en-US/docs/Web/HTTP/CSP",
	},
	{
		name: "strict-transport-security",
		severity: "serious",
		title: "Missing HSTS Header",
		description:
			"HSTS ensures browsers only connect via HTTPS, protecting against protocol downgrade attacks and cookie hijacking.",
		helpUrl:
			"https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Strict-Transport-Security",
	},
	{
		name: "x-frame-options",
		severity: "moderate",
		title: "Missing X-Frame-Options",
		description:
			"Prevents clickjacking by controlling whether the page can be embedded in iframes on other sites.",
		helpUrl:
			"https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/X-Frame-Options",
	},
	{
		name: "x-content-type-options",
		severity: "moderate",
		title: "Missing X-Content-Type-Options",
		description:
			"Prevents MIME type sniffing attacks by ensuring browsers respect the declared Content-Type.",
		helpUrl:
			"https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/X-Content-Type-Options",
		validator: (value) => value.toLowerCase() === "nosniff",
	},
	{
		name: "referrer-policy",
		severity: "minor",
		title: "Missing Referrer Policy",
		description:
			"Controls how much referrer information is sent with requests, helping protect user privacy.",
		helpUrl:
			"https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Referrer-Policy",
	},
	{
		name: "permissions-policy",
		severity: "minor",
		title: "Missing Permissions Policy",
		description:
			"Controls which browser features (camera, microphone, geolocation, etc.) can be used by the page.",
		helpUrl:
			"https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Permissions-Policy",
	},
	{
		name: "x-xss-protection",
		severity: "info",
		title: "Missing X-XSS-Protection",
		description:
			"Legacy header that enabled browser XSS filtering. Modern browsers have deprecated this in favor of CSP.",
		helpUrl:
			"https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/X-XSS-Protection",
	},
];

export class SecurityHeadersScanner extends ScannerBase {
	readonly metadata = {
		name: "security-headers",
		version: "1.0.0",
		description: "HTTP security headers analysis",
	};

	async scanPage(context: ScanContext): Promise<PageScanResult> {
		const { page, pageEntry, logger } = context;
		const startTime = Date.now();
		const issues: Issue[] = [];

		try {
			const currentPageURL = page.url();
			const targetURL =
				currentPageURL.startsWith("http://") ||
				currentPageURL.startsWith("https://")
					? currentPageURL
					: pageEntry.url;

			const response = await page.request.fetch(targetURL, {
				method: "GET",
				timeout: 30_000,
			});

			const headers = response.headers();
			const headerKeys = new Set(
				Object.keys(headers).map((k) => k.toLowerCase()),
			);
			const statusCode = response.status();

			for (const check of SECURITY_HEADERS) {
				const headerName = check.name.toLowerCase();
				const hasHeader = headerKeys.has(headerName);
				const headerValue = headers[headerName];

				if (!hasHeader) {
					issues.push(this.createIssue(check, "missing"));
				} else if (check.validator && !check.validator(headerValue ?? "")) {
					issues.push(this.createIssue(check, "invalid", headerValue));
				}
			}

			const mixedContent = await this.checkMixedContent(page);
			if (mixedContent.length > 0) {
				issues.push({
					id: `${this.metadata.name}-mixed-content`,
					scanner: this.metadata.name,
					severity: "serious",
					category: "security",
					title: "Mixed Content Detected",
					description: `Page loads ${mixedContent.length} insecure resource(s) over HTTP. This can compromise the security of HTTPS pages.`,
					helpUrl:
						"https://developer.mozilla.org/en-US/docs/Web/Security/Mixed_content",
					metadata: {
						resources: mixedContent.slice(0, 10),
						totalCount: mixedContent.length,
					},
				});
			}

			const insecureCookies = this.checkCookieSecurity(headers);
			if (insecureCookies.length > 0) {
				issues.push({
					id: `${this.metadata.name}-insecure-cookies`,
					scanner: this.metadata.name,
					severity: "moderate",
					category: "security",
					title: "Insecure Cookie Configuration",
					description: `Found ${insecureCookies.length} cookie(s) without recommended security flags (Secure, HttpOnly, SameSite).`,
					helpUrl:
						"https://developer.mozilla.org/en-US/docs/Web/HTTP/Cookies#security",
					metadata: {
						cookies: insecureCookies,
					},
				});
			}

			logger.info("Security scan complete", {
				url: pageEntry.url,
				issues: issues.length,
				statusCode,
			});

			return {
				pageId: pageEntry.id,
				url: pageEntry.url,
				path: pageEntry.path,
				success: true,
				issues,
				durationMs: Date.now() - startTime,
				startedAt: new Date(startTime).toISOString(),
				finishedAt: new Date().toISOString(),
				rawResults: {
					headers: Object.fromEntries(
						Object.entries(headers).filter(([k]) =>
							SECURITY_HEADERS.some((h) => h.name === k.toLowerCase()),
						),
					),
					statusCode,
					mixedContentCount: mixedContent.length,
				},
			};
		} catch (error) {
			logger.error("Security scan failed", {
				url: pageEntry.url,
				error: error instanceof Error ? error.message : String(error),
			});
			return this.createErrorResult(
				pageEntry,
				startTime,
				error instanceof Error ? error.message : String(error),
			);
		}
	}

	private createIssue(
		check: SecurityHeader,
		type: "missing" | "invalid",
		value?: string,
	): Issue {
		const id =
			type === "missing"
				? `${this.metadata.name}-missing-${check.name}`
				: `${this.metadata.name}-invalid-${check.name}`;

		const title =
			type === "missing" ? check.title : `Invalid ${check.name} Header`;

		const description =
			type === "missing"
				? check.description
				: `Header present but has invalid value: "${value}". ${check.description}`;

		return {
			id,
			scanner: this.metadata.name,
			severity: check.severity,
			category: "security-headers",
			title,
			description,
			helpUrl: check.helpUrl,
			metadata: {
				headerName: check.name,
				...(value && { headerValue: value }),
			},
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

			for (const el of document.querySelectorAll(
				'link[rel="stylesheet"][href^="http:"]',
			)) {
				insecureResources.push((el as HTMLLinkElement).href);
			}

			for (const el of document.querySelectorAll('iframe[src^="http:"]')) {
				insecureResources.push((el as HTMLIFrameElement).src);
			}

			for (const el of document.querySelectorAll(
				'video source[src^="http:"], audio source[src^="http:"]',
			)) {
				insecureResources.push((el as HTMLSourceElement).src);
			}

			return insecureResources;
		});
	}

	private checkCookieSecurity(
		headers: Record<string, string>,
	): { name: string; issues: string[] }[] {
		const setCookieHeaders = Object.entries(headers)
			.filter(([k]) => k.toLowerCase() === "set-cookie")
			.map(([, v]) => v);

		const insecureCookies: { name: string; issues: string[] }[] = [];

		for (const cookie of setCookieHeaders) {
			if (!cookie) {
				continue;
			}
			const parts = cookie.split(";").map((p) => p.trim().toLowerCase());
			const [rawName] = cookie.split("=");
			const cookieName = (rawName ?? "").trim();
			const issues: string[] = [];

			if (!parts.some((p) => p === "secure")) {
				issues.push("Missing Secure flag");
			}
			if (!parts.some((p) => p === "httponly")) {
				issues.push("Missing HttpOnly flag");
			}
			if (!parts.some((p) => p.startsWith("samesite"))) {
				issues.push("Missing SameSite attribute");
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
		error: string,
	): PageScanResult {
		return {
			pageId: pageEntry.id,
			url: pageEntry.url,
			path: pageEntry.path,
			success: false,
			issues: [],
			durationMs: Date.now() - startTime,
			startedAt: new Date(startTime).toISOString(),
			finishedAt: new Date().toISOString(),
			error,
		};
	}
}
