/**
 * SEO Scanner
 *
 * Analyzes pages for SEO best practices including meta tags, headings,
 * structured data, and other ranking factors.
 */

import type { Issue, PageScanResult, ScanContext } from "../../core/types";

import { ScannerBase } from "../../core/scanner-base";
import { SEO_CHECKS } from "./checks";
import { extractSEOData } from "./extract";

export class SEOScanner extends ScannerBase {
  readonly metadata = {
    name: "seo",
    version: "1.0.0",
    description: "SEO best practices analysis",
  };

  async scanPage(context: ScanContext): Promise<PageScanResult> {
    const { page, pageEntry, logger } = context;
    const startTime = Date.now();
    const issues: Issue[] = [];

    try {
      const seoData = await extractSEOData(page, pageEntry.url);

      for (const check of SEO_CHECKS) {
        try {
          const result = check.check(seoData);
          if (result && !result.passed) {
            issues.push({
              id: `${this.metadata.name}-${check.id}`,
              scanner: this.metadata.name,
              severity: check.severity,
              category: check.category,
              title: check.title,
              description: result.message,
              helpUrl: check.helpUrl,
              metadata: result.details,
            });
          }
        } catch (err) {
          logger.warn(`SEO check failed: ${check.id}`, {
            error: err instanceof Error ? err.message : String(err),
          });
        }
      }

      logger.info("SEO scan complete", {
        url: pageEntry.url,
        issues: issues.length,
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
          title: seoData.title,
          description: seoData.description,
          canonical: seoData.canonical,
          h1Count: seoData.headings.filter((h) => h.level === 1).length,
          imageCount: seoData.images.length,
          imagesWithoutAlt: seoData.images.filter((i) => !i.alt).length,
          wordCount: seoData.wordCount,
          hasStructuredData: seoData.structuredData.length > 0,
          ogTags: Object.keys(seoData.ogTags),
        },
      };
    } catch (error) {
      logger.error("SEO scan failed", {
        url: pageEntry.url,
        error: error instanceof Error ? error.message : String(error),
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
        error: error instanceof Error ? error.message : String(error),
      };
    }
  }
}
