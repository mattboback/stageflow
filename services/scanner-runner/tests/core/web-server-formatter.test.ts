import { describe, expect, it } from 'vitest';

import type { Provenance, ScanResults, ScannerMetadata } from '../../src/core/types';

import { WebServerFormatter } from '../../src/core/web-server-formatter';

describe('WebServerFormatter', () => {
	it('formats scan results into the v2 web-server schema', () => {
		const provenance: Provenance = {
			version: '1.0.0',
			job_id: 'job-123',
			base_url: 'http://localhost:8080',
			pages: [
				{
					id: 'page-1',
					url: 'http://localhost:8080/index.html',
					path: '/index.html'
				}
			]
		};

		const results: ScanResults = {
			jobId: 'job-123',
			scanner: 'axe',
			version: '0.1.0',
			totalPages: 1,
			startedAt: '2025-12-22T00:00:00Z',
			completedAt: '2025-12-22T00:00:01Z',
			durationMs: 1000,
			pages: [
				{
					pageId: 'page-1',
					url: 'http://localhost:8080/index.html',
					path: '/index.html',
					success: true,
					durationMs: 1000,
					startedAt: '2025-12-22T00:00:00Z',
					finishedAt: '2025-12-22T00:00:01Z',
					issues: [
						{
							id: 'color-contrast',
							scanner: 'axe',
							severity: 'critical',
							category: 'accessibility',
							title: 'Elements must have sufficient color contrast',
							description: 'Text must have sufficient contrast against its background.',
							helpUrl: 'https://example.com/rules/color-contrast',
							metadata: {
								impact: 'CRITICAL',
								tags: ['wcag2aa', 'cat.color'],
								nodeCount: 2,
								nodes: [
									{
										target: ['#main a'],
										html: '<a>Link</a>',
										failureSummary: 'Fix contrast',
										selector: '#main a'
									},
									{
										target: ['#footer a'],
										html: '<a>Footer Link</a>',
										failureSummary: 'Fix contrast',
										selector: '#footer a'
									}
								]
							}
						}
					],
					rawResults: {
						pageOverview: {
							screenshotFilename: 'overview.png',
							pageWidth: 1280,
							pageHeight: 720,
							elements: []
						}
					}
				}
			],
			summary: {
				totalIssues: 1,
				bySeverity: { critical: 1, serious: 0, moderate: 0, minor: 0, info: 0 },
				byCategory: { accessibility: 1 },
				pagesScanned: 1,
				pagesFailed: 0,
				pagesWithIssues: 1,
				avgDurationMs: 1000
			}
		};

		const metadata: ScannerMetadata = { name: 'axe', version: '0.1.0' };

		const formatter = new WebServerFormatter();
		const formatted = formatter.format(provenance, results, metadata);

		expect(formatted.version).toBe('2.2.0');
		expect(formatted.meta.jobId).toBe('job-123');
		expect(formatted.meta.baseUrl).toBe('http://localhost:8080');

		expect(formatted.summary.totalIssues).toBe(1);
		expect(formatted.summary.bySeverity.critical).toBe(1);
		expect(formatted.summary.pagesWithIssues).toBe(1);

		expect(formatted.scanners).toHaveLength(1);
		expect(formatted.scanners[0]!.resultsPath).toBe('job-123/axe/results.json');

		expect(formatted.pages).toHaveLength(1);
		expect(formatted.pages[0]!.pageOverview?.screenshotFilename).toBe('overview.png');

		expect(formatted.issues).toHaveLength(1);
		const issue = formatted.issues[0]!;
		expect(issue.severity).toBe('critical');
		expect(issue.severityRaw).toBe('CRITICAL');
		expect(issue.wcagTags).toContain('wcag2aa');
		const occurrences = issue.occurrences ?? [];
		expect(occurrences).toHaveLength(2);

		const artifacts = formatted.artifacts ?? [];
		expect(artifacts).toHaveLength(1);
		expect(occurrences[0]!.artifactIds).toBeUndefined();
		expect(occurrences[1]!.artifactIds).toBeUndefined();

		const overviewArtifact = artifacts.find((entry) => entry.type === 'page-overview');
		expect(overviewArtifact).toBeDefined();
		expect(overviewArtifact?.path).toBe('screenshots/overview.png');
		expect(overviewArtifact?.mime).toBe('image/png');

		const formatted2 = new WebServerFormatter().format(provenance, results, metadata);
		expect(formatted2.issues[0]!.id).toBe(issue.id);

		// Adding a sixth failing element retains every identity without expanding
		// the expensive HTML and screenshot evidence budget.
		const scanIssue = results.pages[0]!.issues[0]!;
		scanIssue.howToFix = 'Increase the foreground/background contrast ratio.';
		scanIssue.metadata = {
			...scanIssue.metadata,
			nodeCount: 6,
			ruleBehavior: { tags: ['Level AA', 'Level AAA', 'Level A'] },
			nodes: Array.from({ length: 6 }, (_, index) => ({
				selector: `#element-${index}`,
				target: [`#element-${index}`],
				html: '<a>Link</a>',
				contextHtml: '<nav><a>Link</a></nav>'
			}))
		};
		const withSix = formatter.format(provenance, results, metadata).issues[0]!;
		expect(withSix.occurrences).toHaveLength(6);
		expect(new Set(withSix.occurrences?.map((node) => node.elementId)).size).toBe(6);
		expect(withSix.occurrences?.[5]?.elementId).toMatch(/^occ-v2-/);
		expect(withSix.occurrences?.[5]?.html).toBeUndefined();
		expect(withSix.occurrences?.[5]?.contextHtml).toBeUndefined();
		expect(withSix.elementsTruncated).toBe(true);
		expect(withSix.howToFix).toBe(scanIssue.howToFix);
		expect(withSix.wcagTags).toEqual(expect.arrayContaining(['Level AA', 'Level AAA', 'Level A']));
	});

	it('marks rule evaluation errors partial without discarding successful page coverage', () => {
		const results: ScanResults = {
			jobId: 'rule-errors',
			scanner: 'seo',
			version: '0.1.0',
			totalPages: 1,
			startedAt: '2026-09-30T00:00:00Z',
			completedAt: '2026-09-30T00:00:01Z',
			durationMs: 1000,
			pages: [
				{
					pageId: 'page-ok',
					url: 'https://sample.invalid/',
					finalUrl: 'https://sample.invalid/',
					success: true,
					durationMs: 1000,
					startedAt: '2026-09-30T00:00:00Z',
					finishedAt: '2026-09-30T00:00:01Z',
					issues: [],
					rules: [
						{ ruleId: 'title', outcome: 'pass' },
						{ ruleId: 'description', outcome: 'error' },
						{ ruleId: 'canonical', outcome: 'error' }
					]
				}
			],
			summary: {
				totalIssues: 0,
				bySeverity: { critical: 0, serious: 0, moderate: 0, minor: 0, info: 0 },
				byCategory: {},
				pagesScanned: 1,
				pagesFailed: 0,
				pagesWithIssues: 0,
				avgDurationMs: 1000
			}
		};
		const provenance: Provenance = { version: '1.0.0', job_id: results.jobId, pages: [] };
		const report = new WebServerFormatter().format(provenance, results, {
			name: 'seo',
			version: '0.1.0'
		});
		expect(report.scanners[0]?.status).toBe('partial');
		expect(report.scanners[0]?.pages?.[0]).toMatchObject({
			status: 'success',
			rules: [
				{ ruleId: 'title', outcome: 'pass' },
				{ ruleId: 'description', outcome: 'error' },
				{ ruleId: 'canonical', outcome: 'error' }
			]
		});
		expect(report.errors).toEqual([
			{
				scope: 'page',
				scannerId: 'seo',
				pageId: 'page-ok',
				code: 'rule_evaluation_failed',
				message: 'One or more rule checks failed to evaluate.',
				retryable: false
			}
		]);
		results.pages[0]!.success = false;
		results.pages[0]!.error = 'Navigation failed';
		results.summary.pagesFailed = 1;
		const failed = new WebServerFormatter().format(provenance, results, {
			name: 'seo',
			version: '0.1.0'
		});
		expect(failed.scanners[0]?.status).toBe('failed');
		expect(failed.errors).toHaveLength(1);
		expect(failed.errors?.[0]?.code).toBe('page_scan_failed');
	});

	it('reports page scan failures and marks mixed scanner results as partial', () => {
		const provenance: Provenance = {
			version: '1.0.0',
			job_id: 'job-partial',
			base_url: 'http://localhost:8080',
			pages: []
		};

		const results: ScanResults = {
			jobId: 'job-partial',
			scanner: 'axe',
			version: '0.1.0',
			totalPages: 2,
			startedAt: '2025-12-22T00:00:00Z',
			completedAt: '2025-12-22T00:00:01Z',
			durationMs: 1000,
			pages: [
				{
					pageId: 'page-ok',
					url: 'http://localhost:8080/ok',
					success: true,
					durationMs: 500,
					startedAt: '2025-12-22T00:00:00Z',
					finishedAt: '2025-12-22T00:00:00.500Z',
					issues: []
				},
				{
					pageId: 'page-failed',
					url: 'http://localhost:8080/failed',
					success: false,
					durationMs: 500,
					startedAt: '2025-12-22T00:00:00.500Z',
					finishedAt: '2025-12-22T00:00:01Z',
					issues: [],
					error: 'Navigation timeout',
					retryable: true
				}
			],
			summary: {
				totalIssues: 0,
				bySeverity: { critical: 0, serious: 0, moderate: 0, minor: 0, info: 0 },
				byCategory: {},
				pagesScanned: 2,
				pagesFailed: 1,
				pagesWithIssues: 0,
				avgDurationMs: 500
			}
		};

		const formatted = new WebServerFormatter().format(provenance, results, {
			name: 'axe',
			version: '0.1.0'
		});

		expect(formatted.scanners[0]!.status).toBe('partial');
		expect(formatted.errors).toEqual([
			{
				scope: 'page',
				scannerId: 'axe',
				pageId: 'page-failed',
				code: 'page_scan_failed',
				message: 'Navigation timeout',
				retryable: true
			}
		]);
	});

	it('sets screenshot mime based on file extension', () => {
		const provenance: Provenance = {
			version: '1.0.0',
			job_id: 'job-123',
			base_url: 'http://localhost:8080',
			pages: [
				{
					id: 'page-1',
					url: 'http://localhost:8080/index.html',
					path: '/index.html'
				}
			]
		};

		const results: ScanResults = {
			jobId: 'job-123',
			scanner: 'axe',
			version: '0.1.0',
			totalPages: 1,
			startedAt: '2025-12-22T00:00:00Z',
			completedAt: '2025-12-22T00:00:01Z',
			durationMs: 1000,
			pages: [
				{
					pageId: 'page-1',
					url: 'http://localhost:8080/index.html',
					success: true,
					durationMs: 1000,
					startedAt: '2025-12-22T00:00:00Z',
					finishedAt: '2025-12-22T00:00:01Z',
					issues: [
						{
							id: 'color-contrast',
							scanner: 'axe',
							severity: 'critical',
							category: 'accessibility',
							title: 'Contrast',
							description: 'Contrast',
							helpUrl: '',
							metadata: { nodes: [{ target: ['#a'], selector: '#a' }] }
						}
					],
					rawResults: {
						pageOverview: {
							screenshotFilename: 'overview-123.webp',
							pageWidth: 1280,
							pageHeight: 720,
							elements: []
						}
					}
				}
			],
			summary: {
				totalIssues: 1,
				bySeverity: { critical: 1, serious: 0, moderate: 0, minor: 0, info: 0 },
				byCategory: { accessibility: 1 },
				pagesScanned: 1,
				pagesFailed: 0,
				pagesWithIssues: 1,
				avgDurationMs: 1000
			}
		};

		const metadata: ScannerMetadata = { name: 'axe', version: '0.1.0' };
		const formatted = new WebServerFormatter().format(provenance, results, metadata);

		const artifacts = formatted.artifacts ?? [];
		expect(artifacts).toHaveLength(1);
		expect(artifacts[0]!.mime).toBe('image/webp');
	});

	it('adds a page overview artifact even when a page has no issues', () => {
		const provenance: Provenance = {
			version: '1.0.0',
			job_id: 'job-789',
			base_url: 'http://localhost:8080',
			pages: [{ id: 'page-1', url: 'http://localhost:8080', path: '/' }]
		};

		const results: ScanResults = {
			jobId: 'job-789',
			scanner: 'axe',
			version: '0.1.0',
			totalPages: 1,
			startedAt: '2025-12-22T00:00:00Z',
			completedAt: '2025-12-22T00:00:01Z',
			durationMs: 1000,
			pages: [
				{
					pageId: 'page-1',
					url: 'http://localhost:8080',
					path: '/',
					success: true,
					durationMs: 1000,
					startedAt: '2025-12-22T00:00:00Z',
					finishedAt: '2025-12-22T00:00:01Z',
					issues: [],
					rawResults: {
						pageOverview: {
							screenshotFilename: 'overview-clean.webp',
							pageWidth: 1200,
							pageHeight: 800,
							elements: []
						}
					}
				}
			],
			summary: {
				totalIssues: 0,
				bySeverity: { critical: 0, serious: 0, moderate: 0, minor: 0, info: 0 },
				byCategory: {},
				pagesScanned: 1,
				pagesFailed: 0,
				pagesWithIssues: 0,
				avgDurationMs: 1000
			}
		};

		const metadata: ScannerMetadata = { name: 'axe', version: '0.1.0' };
		const formatted = new WebServerFormatter().format(provenance, results, metadata);

		expect(formatted.artifacts).toEqual([
			{
				id: 'page-overview-axe-page-1',
				type: 'page-overview',
				path: 'screenshots/overview-clean.webp',
				mime: 'image/webp'
			}
		]);
	});
});
