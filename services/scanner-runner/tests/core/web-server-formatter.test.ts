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
							screenshot: 'issue-1.png',
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

		expect(formatted.version).toBe('2.0.0');
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
		expect(artifacts).toHaveLength(2);
		const artifact = artifacts.find((entry) => entry.type === 'screenshot');
		expect(artifact).toBeDefined();
		if (!artifact) {
			throw new Error('expected screenshot artifact');
		}
		expect(artifact.type).toBe('screenshot');
		expect(artifact.path).toBe('screenshots/issue-1.png');
		expect(artifact.mime).toBe('image/png');
		expect(occurrences[0]!.artifactIds).toEqual([artifact.id]);
		expect(occurrences[1]!.artifactIds).toEqual([artifact.id]);

		const overviewArtifact = artifacts.find((entry) => entry.type === 'page-overview');
		expect(overviewArtifact).toBeDefined();
		expect(overviewArtifact?.path).toBe('screenshots/overview.png');

		const formatted2 = new WebServerFormatter().format(provenance, results, metadata);
		expect(formatted2.issues[0]!.id).toBe(issue.id);
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
							screenshot: 'violation-color-contrast-123.webp',
							metadata: { nodes: [{ target: ['#a'], selector: '#a' }] }
						}
					]
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
