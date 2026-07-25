/**
 * Link Checker Scanner
 *
 * Validates all links on the page, checking for broken links, redirect chains,
 * and other link quality issues.
 */

import type { Page } from 'playwright';

import { join } from 'node:path';

import type { Issue, PageScanResult, ScanContext } from '../../core/types';
import type { LinkCheckResult, LinkInfo } from './types';

import { ScannerBase } from '../../core/scanner-base';
import { AxeScreenshotService } from '../../screenshots/axe-screenshot-service';
import { capturePageOverviewFromIssues } from '../../screenshots/page-overview-from-issues';
import { SCANNER_VERSION } from '../version';
import { checkSingleLink, getSeverityForStatus, groupByStatus } from './validation';

/** Located link element: a CSS selector plus a trimmed HTML snippet. */
interface LocatedElement {
	selector: string;
	html: string;
}

/** Builds occurrence nodes (selector + html) for attaching to an issue's metadata. */
function nodesFromSelectors(
	items: { selector?: string | undefined; html?: string | undefined }[]
): {
	target: string[];
	selector: string;
	html?: string;
}[] {
	return items
		.filter((item): item is { selector: string; html?: string } => Boolean(item.selector))
		.slice(0, 5)
		.map((item) => ({
			target: [item.selector],
			selector: item.selector,
			...(item.html !== undefined ? { html: item.html } : {})
		}));
}

export type { LinkCheckResult, LinkInfo } from './types';
// Re-export for backwards compatibility and testing
export { checkSingleLink, getSeverityForStatus, groupByStatus } from './validation';

export class LinkCheckerScanner extends ScannerBase {
	readonly metadata = {
		name: 'link-checker',
		version: SCANNER_VERSION,
		description: 'Link validation and broken link detection'
	};

	private readonly maxConcurrentRequests = 5;
	private readonly screenshotService = new AxeScreenshotService();

	async scanPage(context: ScanContext): Promise<PageScanResult> {
		const { page, pageEntry, logger, resultsDir } = context;
		const startTime = Date.now();
		const issues: Issue[] = [];

		try {
			const links = await this.extractLinks(page, pageEntry.url);
			logger.info('Extracted links', {
				count: links.length,
				url: pageEntry.url
			});

			const results = await this.checkLinks(
				links,
				context.targetValidationPolicy ?? { allowedOrigins: [] }
			);

			const brokenLinks: LinkCheckResult[] = [];
			const redirectChains: LinkCheckResult[] = [];
			const slowLinks: LinkCheckResult[] = [];

			for (const result of results) {
				if (result.error || (result.status && result.status >= 400)) {
					brokenLinks.push(result);
				} else if (result.redirects.length > 2) {
					redirectChains.push(result);
				} else if (result.responseTime > 3000) {
					slowLinks.push(result);
				}
			}

			this.addBrokenLinkIssues(issues, brokenLinks);
			this.addRedirectChainIssue(issues, redirectChains);
			this.addSlowLinkIssue(issues, slowLinks);

			const emptyLinks = await this.checkEmptyLinks(page);
			if (emptyLinks.length > 0) {
				issues.push({
					id: `${this.metadata.name}-empty-links`,
					scanner: this.metadata.name,
					severity: 'moderate',
					category: 'links',
					title: 'Empty or Placeholder Links',
					description: `Found ${emptyLinks.length} link(s) with empty href, javascript:void(0), or # placeholders. These provide poor user experience and accessibility.`,
					metadata: {
						links: emptyLinks.slice(0, 10).map((l) => l.html),
						nodes: nodesFromSelectors(emptyLinks)
					}
				});
			}

			const noTextLinks = await this.checkLinksWithoutText(page);
			if (noTextLinks.length > 0) {
				issues.push({
					id: `${this.metadata.name}-no-text-links`,
					scanner: this.metadata.name,
					severity: 'serious',
					category: 'accessibility',
					title: 'Links Without Accessible Text',
					description: `Found ${noTextLinks.length} link(s) without accessible text content. Screen reader users won't know where these links lead.`,
					helpUrl: 'https://www.w3.org/WAI/WCAG21/Understanding/link-purpose-in-context.html',
					metadata: {
						links: noTextLinks.slice(0, 10).map((l) => l.html),
						nodes: nodesFromSelectors(noTextLinks)
					}
				});
			}

			logger.info('Link check complete', {
				url: pageEntry.url,
				totalLinks: links.length,
				brokenCount: brokenLinks.length,
				issues: issues.length
			});

			const pageOverview = await capturePageOverviewFromIssues({
				service: this.screenshotService,
				page,
				issues,
				screenshotsDir: join(resultsDir, 'screenshots'),
				pageId: pageEntry.id,
				scannerId: this.metadata.name,
				logger
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
				artifacts: pageOverview ? [pageOverview.screenshotPath] : [],
				rawResults: {
					totalLinks: links.length,
					internalLinks: links.filter((l) => l.isInternal).length,
					externalLinks: links.filter((l) => !l.isInternal).length,
					brokenCount: brokenLinks.length,
					redirectChainCount: redirectChains.length,
					averageResponseTime:
						results.length > 0
							? Math.round(results.reduce((sum, r) => sum + r.responseTime, 0) / results.length)
							: 0,
					pageOverview: pageOverview
						? {
								screenshotFilename: pageOverview.screenshotFilename,
								pageWidth: pageOverview.pageWidth,
								pageHeight: pageOverview.pageHeight,
								elements: pageOverview.elements
							}
						: null
				}
			};
		} catch (error) {
			logger.error('Link check failed', {
				url: pageEntry.url,
				error: error instanceof Error ? error.message : String(error)
			});
			return {
				pageId: pageEntry.id,
				url: pageEntry.url,
				path: pageEntry.path,
				success: false,
				issues: [],
				durationMs: Date.now() - startTime,
				startedAt: new Date(startTime).toISOString(),
				finishedAt: new Date().toISOString(),
				error: error instanceof Error ? error.message : String(error)
			};
		}
	}

	private addBrokenLinkIssues(issues: Issue[], brokenLinks: LinkCheckResult[]): void {
		if (brokenLinks.length === 0) {
			return;
		}

		const grouped = groupByStatus(brokenLinks);
		for (const [status, links] of Object.entries(grouped)) {
			const severity = getSeverityForStatus(Number.parseInt(status, 10));
			issues.push({
				id: `${this.metadata.name}-broken-${status}`,
				scanner: this.metadata.name,
				severity,
				category: 'links',
				title: `Broken Links (${status === '0' ? 'Connection Error' : `HTTP ${status}`})`,
				description: `Found ${links.length} link(s) returning ${status === '0' ? 'connection errors' : `HTTP ${status}`}. Broken links hurt user experience and SEO.`,
				helpUrl: 'https://developer.mozilla.org/en-US/docs/Web/HTTP/Status',
				metadata: {
					links: links.slice(0, 10).map((l) => ({ url: l.url, error: l.error })),
					totalCount: links.length,
					nodes: nodesFromSelectors(links.map((l) => ({ selector: l.selector, html: l.url })))
				}
			});
		}
	}

	private addRedirectChainIssue(issues: Issue[], redirectChains: LinkCheckResult[]): void {
		if (redirectChains.length === 0) {
			return;
		}

		issues.push({
			id: `${this.metadata.name}-redirect-chains`,
			scanner: this.metadata.name,
			severity: 'moderate',
			category: 'links',
			title: 'Excessive Redirect Chains',
			description: `Found ${redirectChains.length} link(s) with more than 2 redirects. Long redirect chains slow down page loading and can hurt SEO.`,
			helpUrl: 'https://developers.google.com/search/docs/crawling-indexing/301-redirects',
			metadata: {
				links: redirectChains.slice(0, 5).map((l) => ({
					url: l.url,
					redirectCount: l.redirects.length,
					chain: l.redirects
				}))
			}
		});
	}

	private addSlowLinkIssue(issues: Issue[], slowLinks: LinkCheckResult[]): void {
		if (slowLinks.length === 0) {
			return;
		}

		issues.push({
			id: `${this.metadata.name}-slow-responses`,
			scanner: this.metadata.name,
			severity: 'minor',
			category: 'performance',
			title: 'Slow Link Responses',
			description: `Found ${slowLinks.length} link(s) with response times over 3 seconds. Slow external resources can impact page performance.`,
			metadata: {
				links: slowLinks.slice(0, 5).map((l) => ({
					url: l.url,
					responseTime: l.responseTime
				}))
			}
		});
	}

	private async extractLinks(page: Page, baseUrl: string): Promise<LinkInfo[]> {
		return page.evaluate((base) => {
			const cssPath = (el: Element): string => {
				const parts: string[] = [];
				let node: Element | null = el;
				while (node?.nodeType === 1 && node !== document.body && parts.length < 6) {
					if (node.id) {
						parts.unshift(`#${CSS.escape(node.id)}`);
						break;
					}
					let part = node.tagName.toLowerCase();
					const parent: Element | null = node.parentElement;
					if (parent) {
						const nodeTag = node.tagName;
						const sameTag = Array.from(parent.children).filter((c) => c.tagName === nodeTag);
						if (sameTag.length > 1) {
							part += `:nth-of-type(${sameTag.indexOf(node) + 1})`;
						}
					}
					parts.unshift(part);
					node = node.parentElement;
				}
				return parts.join(' > ');
			};

			const links: {
				href: string;
				text: string;
				isInternal: boolean;
				element: string;
				selector: string;
			}[] = [];
			const currentHost = new URL(base).host;

			for (const el of document.querySelectorAll('a[href]')) {
				const href = el.getAttribute('href') ?? '';

				if (
					!href ||
					href === '#' ||
					href.startsWith('javascript:') ||
					href.startsWith('data:') ||
					href.startsWith('vbscript:') ||
					href.startsWith('mailto:') ||
					href.startsWith('tel:')
				) {
					continue;
				}

				let absoluteUrl: string;
				let isInternal: boolean;

				try {
					const parsed = new URL(href, base);
					absoluteUrl = parsed.href;
					isInternal = parsed.host === currentHost;
				} catch {
					continue;
				}

				links.push({
					href: absoluteUrl,
					text: (el.textContent || '').trim().slice(0, 100),
					isInternal,
					element: el.tagName.toLowerCase(),
					selector: cssPath(el)
				});
			}

			const seen = new Set<string>();
			return links.filter((link) => {
				if (seen.has(link.href)) {
					return false;
				}
				seen.add(link.href);
				return true;
			});
		}, baseUrl);
	}

	private async checkLinks(
		links: LinkInfo[],
		targetValidationPolicy: NonNullable<ScanContext['targetValidationPolicy']>
	): Promise<LinkCheckResult[]> {
		const results: LinkCheckResult[] = [];

		for (let i = 0; i < links.length; i += this.maxConcurrentRequests) {
			const batch = links.slice(i, i + this.maxConcurrentRequests);
			const batchResults = await Promise.all(
				batch.map(async (link) => {
					const result = await checkSingleLink(link.href, targetValidationPolicy);
					return link.selector !== undefined ? { ...result, selector: link.selector } : result;
				})
			);
			results.push(...batchResults);

			if (i + this.maxConcurrentRequests < links.length) {
				await new Promise((resolve) => setTimeout(resolve, 100));
			}
		}

		return results;
	}

	private async checkEmptyLinks(page: Page): Promise<LocatedElement[]> {
		return page.evaluate(() => {
			const cssPath = (el: Element): string => {
				const parts: string[] = [];
				let node: Element | null = el;
				while (node?.nodeType === 1 && node !== document.body && parts.length < 6) {
					if (node.id) {
						parts.unshift(`#${CSS.escape(node.id)}`);
						break;
					}
					let part = node.tagName.toLowerCase();
					const parent: Element | null = node.parentElement;
					if (parent) {
						const nodeTag = node.tagName;
						const sameTag = Array.from(parent.children).filter((c) => c.tagName === nodeTag);
						if (sameTag.length > 1) {
							part += `:nth-of-type(${sameTag.indexOf(node) + 1})`;
						}
					}
					parts.unshift(part);
					node = node.parentElement;
				}
				return parts.join(' > ');
			};

			const emptyLinks: { selector: string; html: string }[] = [];
			for (const el of document.querySelectorAll('a')) {
				const href = el.getAttribute('href');
				if (
					!href ||
					href === '#' ||
					href === 'javascript:void(0)' ||
					href === 'javascript:;' ||
					href === 'javascript:void(0);'
				) {
					emptyLinks.push({ selector: cssPath(el), html: el.outerHTML.slice(0, 200) });
				}
			}
			return emptyLinks;
		});
	}

	private async checkLinksWithoutText(page: Page): Promise<LocatedElement[]> {
		return page.evaluate(() => {
			const cssPath = (el: Element): string => {
				const parts: string[] = [];
				let node: Element | null = el;
				while (node?.nodeType === 1 && node !== document.body && parts.length < 6) {
					if (node.id) {
						parts.unshift(`#${CSS.escape(node.id)}`);
						break;
					}
					let part = node.tagName.toLowerCase();
					const parent: Element | null = node.parentElement;
					if (parent) {
						const nodeTag = node.tagName;
						const sameTag = Array.from(parent.children).filter((c) => c.tagName === nodeTag);
						if (sameTag.length > 1) {
							part += `:nth-of-type(${sameTag.indexOf(node) + 1})`;
						}
					}
					parts.unshift(part);
					node = node.parentElement;
				}
				return parts.join(' > ');
			};

			const noTextLinks: { selector: string; html: string }[] = [];
			for (const el of document.querySelectorAll('a[href]')) {
				const text = (el.textContent || '').trim();
				const ariaLabel = el.getAttribute('aria-label')?.trim() ?? '';
				const title = el.getAttribute('title')?.trim() ?? '';
				const hasImage = el.querySelector('img[alt]') !== null;

				if (!text && !ariaLabel && !title && !hasImage) {
					noTextLinks.push({ selector: cssPath(el), html: el.outerHTML.slice(0, 200) });
				}
			}
			return noTextLinks;
		});
	}
}
