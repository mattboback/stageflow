import type { Page } from 'playwright';

import type { HeadingData, ImageData, LinkData, PageSEOData } from './types';

export async function extractSEOData(page: Page, url: string): Promise<PageSEOData> {
	return page.evaluate((pageUrl) => {
		const getMeta = (name: string): string | null => {
			const el =
				document.querySelector(`meta[name="${name}"]`) ??
				document.querySelector(`meta[property="${name}"]`);
			return el?.getAttribute('content') ?? null;
		};

		const headings: HeadingData[] = [];
		for (const el of document.querySelectorAll('h1, h2, h3, h4, h5, h6')) {
			headings.push({
				level: Number.parseInt(el.tagName.substring(1), 10),
				text: (el.textContent || '').trim().slice(0, 100)
			});
		}

		const images: ImageData[] = [];
		for (const el of document.querySelectorAll('img')) {
			images.push({
				src: el.src,
				alt: el.alt || null,
				width: el.naturalWidth || null,
				height: el.naturalHeight || null
			});
		}

		const links: LinkData[] = [];
		const currentHost = new URL(pageUrl).host;
		for (const el of document.querySelectorAll('a[href]')) {
			const href = el.getAttribute('href') ?? '';
			let isInternal: boolean;
			try {
				const linkUrl = new URL(href, pageUrl);
				isInternal = linkUrl.host === currentHost;
			} catch {
				isInternal = href.startsWith('/') || href.startsWith('#');
			}

			links.push({
				href,
				text: (el.textContent || '').trim().slice(0, 100),
				isInternal,
				hasNofollow: (el.getAttribute('rel') ?? '').trim().includes('nofollow')
			});
		}

		const ogTags: Record<string, string> = {};
		for (const el of document.querySelectorAll('meta[property^="og:"]')) {
			const property = el.getAttribute('property');
			const content = el.getAttribute('content');
			if (property && content) {
				ogTags[property] = content;
			}
		}

		const twitterTags: Record<string, string> = {};
		for (const el of document.querySelectorAll('meta[name^="twitter:"]')) {
			const name = el.getAttribute('name');
			const content = el.getAttribute('content');
			if (name && content) {
				twitterTags[name] = content;
			}
		}

		const structuredData: unknown[] = [];
		for (const el of document.querySelectorAll('script[type="application/ld+json"]')) {
			try {
				structuredData.push(JSON.parse(el.textContent || ''));
			} catch {
				// Invalid JSON, skip
			}
		}

		const bodyText = document.body.innerText || '';
		const wordCount = bodyText.split(/\s+/).filter((word) => word.length > 0).length;

		return {
			title: document.title || null,
			description: getMeta('description'),
			canonical: document.querySelector('link[rel="canonical"]')?.getAttribute('href') ?? null,
			robots: getMeta('robots'),
			viewport: getMeta('viewport'),
			charset:
				document.characterSet ||
				(document.querySelector('meta[charset]')?.getAttribute('charset') ?? null),
			language: document.documentElement.lang || null,
			headings,
			images,
			links,
			ogTags,
			twitterTags,
			structuredData,
			wordCount,
			url: pageUrl
		};
	}, url);
}
