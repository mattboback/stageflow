import type { BoundingBox, UnifiedReportV2, UserImpact } from '@stageflow/contracts-report';

import path from 'node:path';

import type {
	IssueSeverity,
	PageScanResult,
	Provenance,
	ScanResults,
	ScannerMetadata
} from './types';

import { normalizeSeverity } from '../utils/severity';
import { resultsPath } from './artifact-paths';
import { issueFingerprint, occurrenceFingerprint } from './finding-identity';
import { boundRuleEvaluations } from './rule-evaluations';

type SeverityCounts = Record<IssueSeverity, number>;

interface PageOverview {
	screenshotFilename: string;
	pageWidth: number;
	pageHeight: number;
	elements: {
		issueId: string;
		ruleId: string;
		severity: IssueSeverity;
		selector: string;
		nodeIndex: number;
		xPercent: number;
		yPercent: number;
		widthPercent: number;
		heightPercent: number;
		x: number;
		y: number;
		width: number;
		height: number;
	}[];
}

export class WebServerFormatter {
	format(provenance: Provenance, results: ScanResults, metadata: ScannerMetadata): UnifiedReportV2 {
		const scannerId = metadata.name;
		const pages: UnifiedReportV2['pages'] = [];
		const issues: UnifiedReportV2['issues'] = [];
		const artifacts: NonNullable<UnifiedReportV2['artifacts']> = [];
		const errors: NonNullable<UnifiedReportV2['errors']> = [];

		const severityTotals: SeverityCounts = this.emptySeverityCounts();
		const baseUrl =
			typeof provenance.base_url === 'string' && provenance.base_url.trim().length > 0
				? provenance.base_url
				: undefined;

		for (const page of results.pages) {
			const pageSeverity: SeverityCounts = this.emptySeverityCounts();
			const rawResults = this.extractPageOverview(page);

			if (rawResults?.pageOverview?.screenshotFilename) {
				artifacts.push({
					id: `page-overview-${scannerId}-${page.pageId}`,
					type: 'page-overview',
					path: `screenshots/${rawResults.pageOverview.screenshotFilename}`,
					mime: this.guessScreenshotMime(rawResults.pageOverview.screenshotFilename)
				});
			}

			for (const issue of page.issues) {
				const severity = normalizeSeverity(issue.severity, 'info');
				pageSeverity[severity] += 1;
				severityTotals[severity] += 1;

				const issueMetadata = (issue.metadata ?? {}) as {
					impact?: unknown;
					userImpact?: unknown;
					nodes?: {
						target?: string[];
						html?: string;
						failureSummary?: string;
						selector?: string;
						textSnippet?: string;
						boundingBox?: unknown;
						contextHtml?: string;
						ancestorPath?: string;
					}[];
					nodeCount?: number;
					tags?: string[];
					ruleBehavior?: { tags?: string[] };
				};

				const nodes = Array.isArray(issueMetadata.nodes) ? issueMetadata.nodes : [];
				const nodeCount =
					typeof issueMetadata.nodeCount === 'number'
						? issueMetadata.nodeCount
						: Math.max(nodes.length, 1);
				const tags = this.normalizeStringArray(issueMetadata.tags);
				const wcagTags = this.extractWcagTags(tags, issueMetadata.ruleBehavior);

				const issueId = issueFingerprint(
					scannerId,
					issue.id,
					page.pageId,
					nodes[0]?.selector ?? nodes[0]?.target?.[0] ?? ''
				);

				const scannerData = this.extractScannerData(issue.metadata);

				const occurrences: NonNullable<UnifiedReportV2['issues'][number]['occurrences']> =
					nodes.map((node, index) => {
						const target = node.target?.filter((t) => typeof t === 'string');
						const selector = node.selector ?? node.target?.[0];
						const html = index < 5 ? node.html : undefined;
						const contextHtml = index < 5 ? node.contextHtml : undefined;
						const ancestorPath = node.ancestorPath;
						const failureSummary = node.failureSummary;
						const textSnippet = node.textSnippet;
						const boundingBox = index < 5 ? this.asBoundingBox(node.boundingBox) : undefined;

						return {
							elementId: occurrenceFingerprint(
								scannerId,
								issue.id,
								page.pageId,
								target ?? (selector ? [selector] : [])
							),
							...(target !== undefined ? { target } : {}),
							...(selector !== undefined ? { selector } : {}),
							...(html !== undefined ? { html } : {}),
							...(contextHtml !== undefined ? { contextHtml } : {}),
							...(ancestorPath !== undefined ? { ancestorPath } : {}),
							...(failureSummary !== undefined ? { failureSummary } : {}),
							...(textSnippet !== undefined ? { textSnippet } : {}),
							...(boundingBox !== undefined ? { boundingBox } : {})
						};
					});

				issues.push({
					id: issueId,
					scanner: scannerId,
					ruleId: issue.id,
					severity,
					title: issue.title,
					description: issue.description || issue.title,
					wcagTags,
					pageId: page.pageId,
					pageUrl: page.url,
					elementCount: nodeCount,
					elementsTruncated: nodes.length > 5 || nodes.length < nodeCount,
					occurrences,
					category: issue.category,
					...this.buildIssueOptionalFields({
						issue,
						issueMetadata,
						nodes,
						scannerData
					})
				});
			}

			const pageOverview = rawResults?.pageOverview ?? undefined;
			pages.push({
				id: page.pageId,
				url: page.url,
				issueCount: page.issues.length,
				bySeverity: pageSeverity,
				durationMs: page.durationMs,
				startedAt: page.startedAt,
				finishedAt: page.finishedAt,
				...(page.path !== undefined ? { path: page.path } : {}),
				...(pageOverview !== undefined ? { pageOverview } : {})
			});

			if (!page.success) {
				errors.push({
					scope: 'page',
					scannerId,
					pageId: page.pageId,
					code: 'page_scan_failed',
					message: page.error ?? 'Page scan failed',
					retryable: page.retryable ?? false
				});
			} else if (page.rules?.some((rule) => rule.outcome === 'error')) {
				errors.push({
					scope: 'page',
					scannerId,
					pageId: page.pageId,
					code: 'rule_evaluation_failed',
					message: 'One or more rule checks failed to evaluate.',
					retryable: false
				});
			}
		}

		const scannerStatus =
			results.summary.pagesFailed === 0
				? errors.length > 0
					? 'partial'
					: 'success'
				: results.summary.pagesFailed >= results.pages.length
					? 'failed'
					: 'partial';

		return {
			version: '2.2.0',
			meta: {
				jobId: results.jobId,
				...(results.context ? { scanContext: results.context } : {}),
				profile: process.env.SCAN_PROFILE === 'staging' ? 'staging' : 'production',
				policyVersion: 'indexing-v1',
				scannedAt: results.startedAt,
				completedAt: results.completedAt,
				durationMs: results.durationMs,
				...(baseUrl !== undefined ? { baseUrl } : {})
			},
			summary: {
				totalIssues: results.summary.totalIssues,
				bySeverity: severityTotals,
				byScanner: { [scannerId]: results.summary.totalIssues },
				pagesScanned: results.pages.length,
				pagesWithIssues: results.summary.pagesWithIssues,
				...(results.summary.lighthouseCategories !== undefined
					? { lighthouseCategories: results.summary.lighthouseCategories }
					: {})
			},
			scanners: [
				{
					id: scannerId,
					name: scannerId,
					status: scannerStatus,
					issueCount: results.summary.totalIssues,
					severity: severityTotals,
					toolVersion: results.version,
					startedAt: results.startedAt,
					completedAt: results.completedAt,
					durationMs: results.durationMs,
					resultsPath: resultsPath(results.jobId, scannerId),
					pages: results.pages.map((page) => ({
						pageId: page.pageId,
						status: page.success ? ('success' as const) : ('failed' as const),
						...(page.finalUrl ? { finalUrl: page.finalUrl } : {}),
						...(page.rules ? { rules: boundRuleEvaluations(page.rules) } : {})
					}))
				}
			],
			pages,
			issues,
			artifacts,
			errors
		};
	}

	private emptySeverityCounts(): SeverityCounts {
		return {
			critical: 0,
			serious: 0,
			moderate: 0,
			minor: 0,
			info: 0
		};
	}

	private extractWcagTags(tags: string[], ruleBehavior?: { tags?: string[] }): string[] {
		const allTags = [...tags, ...(ruleBehavior?.tags ?? [])];
		return allTags.filter((tag) => /^wcag/i.test(tag) || /^level\s+(aaa|aa|a)$/i.test(tag));
	}

	private normalizeStringArray(value: unknown): string[] {
		if (!Array.isArray(value)) {
			return [];
		}
		return value.filter(
			(item): item is string => typeof item === 'string' && item.trim().length > 0
		);
	}

	private extractScannerData(metadata: unknown): Record<string, unknown> | undefined {
		if (typeof metadata !== 'object' || metadata === null || Array.isArray(metadata)) {
			return undefined;
		}

		const omit = new Set(['impact', 'userImpact', 'nodes', 'nodeCount', 'tags', 'ruleBehavior']);

		const data: Record<string, unknown> = {};
		for (const [key, value] of Object.entries(metadata)) {
			if (omit.has(key)) {
				continue;
			}
			if (value === undefined) {
				continue;
			}
			data[key] = value;
		}

		return Object.keys(data).length > 0 ? data : undefined;
	}

	private buildIssueOptionalFields(input: {
		issue: PageScanResult['issues'][number];
		issueMetadata: {
			impact?: unknown;
			userImpact?: unknown;
		};
		nodes: {
			failureSummary?: string;
		}[];
		scannerData: Record<string, unknown> | undefined;
	}): Partial<UnifiedReportV2['issues'][number]> {
		const severityRaw =
			typeof input.issueMetadata.impact === 'string' ? input.issueMetadata.impact : undefined;
		const userImpact =
			typeof input.issueMetadata.userImpact === 'object' && input.issueMetadata.userImpact !== null
				? (input.issueMetadata.userImpact as UserImpact)
				: undefined;
		const helpUrl =
			typeof input.issue.helpUrl === 'string' && input.issue.helpUrl.trim().length > 0
				? input.issue.helpUrl
				: undefined;
		const howToFix = input.issue.howToFix ?? input.nodes[0]?.failureSummary;

		return {
			...(severityRaw !== undefined ? { severityRaw } : {}),
			...(helpUrl !== undefined ? { helpUrl } : {}),
			...(userImpact !== undefined ? { userImpact } : {}),
			...(howToFix !== undefined ? { howToFix } : {}),
			...(input.scannerData !== undefined ? { scannerData: input.scannerData } : {})
		};
	}

	private extractPageOverview(page: PageScanResult): { pageOverview?: PageOverview } | null {
		const rawResults = page.rawResults as {
			pageOverview?: PageOverview;
		} | null;
		return rawResults;
	}

	private guessScreenshotMime(filename: string): string {
		const ext = path.extname(filename).toLowerCase();
		if (ext === '.webp') {
			return 'image/webp';
		}

		return 'image/png';
	}

	private asBoundingBox(value: unknown): BoundingBox | undefined {
		if (typeof value !== 'object' || value === null) {
			return undefined;
		}

		const maybe = value as Partial<BoundingBox>;
		if (
			typeof maybe.x === 'number' &&
			typeof maybe.y === 'number' &&
			typeof maybe.width === 'number' &&
			typeof maybe.height === 'number'
		) {
			return {
				x: maybe.x,
				y: maybe.y,
				width: maybe.width,
				height: maybe.height
			};
		}

		return undefined;
	}
}
