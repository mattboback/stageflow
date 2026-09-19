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

import { waitForPageSettled } from '../../core/page-settle';
import { ScannerBase } from '../../core/scanner-base';
import { AxeScreenshotService } from '../../screenshots/axe-screenshot-service';
import { capturePageOverviewFromIssues } from '../../screenshots/page-overview-from-issues';
import { SCANNER_VERSION } from '../version';
import { checkSingleLink, getSeverityForStatus, isUnverifiableStatus } from './validation';

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
export {
	checkSingleLink,
	getSeverityForStatus,
	groupByStatus,
	isUnverifiableStatus
} from './validation';

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
			await waitForPageSettled(page);
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
			const unverifiedLinks: LinkCheckResult[] = [];
			const redirectChains: LinkCheckResult[] = [];
			const slowLinks: LinkCheckResult[] = [];

			for (const result of results) {
				if (result.status && isUnverifiableStatus(result.status, result.isInternal)) {
					unverifiedLinks.push(result);
				} else if (result.error || (result.status && result.status >= 400)) {
					brokenLinks.push(result);
				} else if (result.redirects.length > 2) {
					redirectChains.push(result);
				} else if (result.responseTime > 3000) {
					slowLinks.push(result);
				}
			}

			this.addBrokenLinkIssues(issues, brokenLinks);
			this.addUnverifiedLinkIssue(issues, unverifiedLinks);
			this.addRedirectChainIssue(issues, redirectChains);

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
					// Not a finding: a link target's latency says nothing about this page and
					// varies run to run, which made baselines churn.
					slowLinkCount: slowLinks.length,
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

	// One issue per broken URL, all under one rule ID. The fingerprint is built from
	// the rule ID and the link's selector, so fixing one link resolves exactly one
	// issue, and a status that flaps between runs (404 -> 503) is not a new issue.
	private addBrokenLinkIssues(issues: Issue[], brokenLinks: LinkCheckResult[]): void {
		for (const link of brokenLinks) {
			const outcome = link.status
				? `returned HTTP ${link.status}`
				: `could not be reached (connection error${link.error ? `: ${link.error}` : ''})`;
			issues.push({
				id: `${this.metadata.name}-broken`,
				scanner: this.metadata.name,
				severity: getSeverityForStatus(link.status ?? 0),
				category: 'links',
				title: `Broken link (${link.status ? `HTTP ${link.status}` : 'connection error'})`,
				description: `${link.url} ${outcome}. Broken links hurt user experience and SEO.`,
				helpUrl: 'https://developer.mozilla.org/en-US/docs/Web/HTTP/Status',
				metadata: {
					links: [{ url: link.url, status: link.status, error: link.error }],
					totalCount: 1,
					// The formatter surfaces the first node's failureSummary as the fix guidance.
					nodes: nodesFromSelectors([{ selector: link.selector, html: link.url }]).map((node) => ({
						...node,
						failureSummary:
							'Update the href to a working URL, restore or redirect the missing destination, or remove the link.'
					}))
				}
			});
		}
	}

	private addUnverifiedLinkIssue(issues: Issue[], unverifiedLinks: LinkCheckResult[]): void {
		if (unverifiedLinks.length === 0) {
			return;
		}

		issues.push({
			id: `${this.metadata.name}-unverified`,
			scanner: this.metadata.name,
			severity: 'info',
			category: 'links',
			title: 'Links that could not be verified',
			description: `${unverifiedLinks.length} link(s) refused an automated check (login wall, rate limit, or bot protection). They usually work in a browser — open each one to confirm.`,
			helpUrl: 'https://developer.mozilla.org/en-US/docs/Web/HTTP/Status',
			metadata: {
				links: unverifiedLinks.slice(0, 10).map((l) => ({ url: l.url, status: l.status })),
				totalCount: unverifiedLinks.length,
				nodes: nodesFromSelectors(
					unverifiedLinks.map((l) => ({ selector: l.selector, html: l.url }))
				)
			}
		});
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
					const result = {
						...(await checkSingleLink(link.href, targetValidationPolicy)),
						isInternal: link.isInternal
					};
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
