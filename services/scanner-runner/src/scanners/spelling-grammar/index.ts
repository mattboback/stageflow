import type { Issue, PageScanResult, ScanContext } from '../../core/types';

import { ScannerBase } from '../../core/scanner-base';
import { SCANNER_VERSION } from '../version';

const COMMON_MISSPELLINGS = [
	{ pattern: /\bteh\b/gi, replacement: 'the' },
	{ pattern: /\badn\b/gi, replacement: 'and' }
];

export class SpellingGrammarScanner extends ScannerBase {
	readonly metadata = {
		name: 'spelling-grammar',
		version: SCANNER_VERSION,
		description: 'Content quality analysis for simple spelling and grammar issues'
	};

	async scanPage(context: ScanContext): Promise<PageScanResult> {
		const { page, pageEntry, logger } = context;
		const startTime = Date.now();

		try {
			const textContent = await page.evaluate(() => document.body.innerText);
			const issues: Issue[] = [];

			for (const rule of COMMON_MISSPELLINGS) {
				const matches = [...textContent.matchAll(rule.pattern)];
				if (matches.length === 0) {
					continue;
				}

				issues.push({
					id: `${this.metadata.name}-${rule.replacement}`,
					scanner: this.metadata.name,
					severity: 'minor',
					category: 'content-quality',
					title: `Potential spelling issue: ${matches[0]?.[0] ?? rule.replacement}`,
					description: `Found ${matches.length} potential occurrence(s) that may need review. Suggested replacement: ${rule.replacement}.`,
					metadata: {
						matchCount: matches.length,
						suggestion: rule.replacement,
						sample: matches.slice(0, 5).map((match) => match[0])
					}
				});
			}

			logger.info('Spelling and grammar scan complete', {
				url: pageEntry.url,
				issues: issues.length,
				wordCount: textContent.split(/\s+/).filter(Boolean).length
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
					wordCount: textContent.split(/\s+/).filter(Boolean).length,
					issuesFound: issues.length
				}
			};
		} catch (error) {
			logger.error('Spelling and grammar scan failed', {
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
