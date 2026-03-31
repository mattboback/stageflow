import type { Issue, PageScanResult, ScanContext } from '../../core/types';

import { ScannerBase } from '../../core/scanner-base';
import { SCANNER_VERSION } from '../version';

const REQUIRED_TAGS = ['og:title', 'og:description', 'og:image'];
const OPEN_GRAPH_HELP_URL = 'https://ogp.me/';

export class OpenGraphScanner extends ScannerBase {
	readonly metadata = {
		name: 'open-graph',
		version: SCANNER_VERSION,
		description: 'Open Graph and social sharing metadata analysis'
	};

	async scanPage(context: ScanContext): Promise<PageScanResult> {
		const { page, pageEntry, logger } = context;
		const startTime = Date.now();

		try {
			const ogTags = await page.evaluate(() => {
				const tags: Record<string, string> = {};
				const metaTags = document.querySelectorAll('meta[property^="og:"]');

				metaTags.forEach((tag) => {
					const property = tag.getAttribute('property');
					const content = tag.getAttribute('content');
					if (property && content) {
						tags[property] = content;
					}
				});

				return tags;
			});

			const missingTags = REQUIRED_TAGS.filter((tag) => !ogTags[tag]);
			const issues: Issue[] = missingTags.map((tag) => ({
				id: `${this.metadata.name}-missing-${tag.replace(/[^a-z0-9]+/gi, '-')}`,
				scanner: this.metadata.name,
				severity: 'moderate',
				category: 'seo',
				title: `Missing required Open Graph tag: ${tag}`,
				description: `${tag} is missing from the page metadata, which can reduce the quality of social sharing previews.`,
				helpUrl: OPEN_GRAPH_HELP_URL,
				metadata: {
					tag
				}
			}));

			logger.info('Open Graph scan complete', {
				url: pageEntry.url,
				tagCount: Object.keys(ogTags).length,
				missingTags
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
					ogTags,
					missingTags,
					hasRequiredTags: missingTags.length === 0
				}
			};
		} catch (error) {
			logger.error('Open Graph scan failed', {
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
}
