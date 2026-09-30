/**
 * SEO Scanner
 *
 * Analyzes pages for SEO best practices including meta tags, headings,
 * structured data, and other ranking factors.
 */

import type { Issue, PageScanResult, ScanContext } from '../../core/types';

import { waitForPageSettled } from '../../core/page-settle';
import { ScannerBase } from '../../core/scanner-base';
import { SCANNER_VERSION } from '../version';
import { SEO_CHECKS } from './checks';
import { extractSEOData } from './extract';
import { SEO_FIXES } from './fix-guidance';

export class SEOScanner extends ScannerBase {
	readonly metadata = {
		name: 'seo',
		version: SCANNER_VERSION,
		description: 'SEO best practices analysis'
	};

	async scanPage(context: ScanContext): Promise<PageScanResult> {
		const { page, pageEntry, logger } = context;
		const startTime = Date.now();
		const issues: Issue[] = [];
		const rules: NonNullable<PageScanResult['rules']> = [];

		try {
			await waitForPageSettled(page);
			const seoData = await extractSEOData(page, pageEntry.url);

			for (const check of SEO_CHECKS) {
				try {
					const result = check.check(seoData);
					rules.push({
						ruleId: `${this.metadata.name}-${check.id}`,
						outcome: result ? (result.passed ? 'pass' : 'fail') : 'inapplicable'
					});
					if (result && !result.passed) {
						issues.push({
							id: `${this.metadata.name}-${check.id}`,
							scanner: this.metadata.name,
							severity: check.severity,
							category: check.category,
							title: check.title,
							description: result.message,
							...(check.helpUrl !== undefined ? { helpUrl: check.helpUrl } : {}),
							...(SEO_FIXES[check.id] ? { howToFix: SEO_FIXES[check.id] } : {}),
							...(result.details !== undefined ? { metadata: result.details } : {})
						});
					}
				} catch (err) {
					rules.push({ ruleId: `${this.metadata.name}-${check.id}`, outcome: 'error' });
					logger.warn(`SEO check failed: ${check.id}`, {
						error: err instanceof Error ? err.message : String(err)
					});
				}
			}

			logger.info('SEO scan complete', {
				url: pageEntry.url,
				issues: issues.length
			});

			return {
				pageId: pageEntry.id,
				url: pageEntry.url,
				path: pageEntry.path,
				success: true,
				finalUrl: page.url(),
				rules,
				issues,
				durationMs: Date.now() - startTime,
				startedAt: new Date(startTime).toISOString(),
				finishedAt: new Date().toISOString()
			};
		} catch (error) {
			logger.error('SEO scan failed', {
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
