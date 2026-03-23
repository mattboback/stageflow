import type { UnifiedReport } from '$lib/types/unified-report';

import { buildOccurrenceModeReport } from '$lib/report/occurrence-mode';
import { describe, expect, it } from 'vitest';

function createReport(): UnifiedReport {
	return {
		version: '2.0.0',
		meta: { jobId: 'job-1' },
		summary: {
			totalIssues: 2,
			bySeverity: { critical: 0, serious: 1, moderate: 1, minor: 0, info: 0 },
			byScanner: { axe: 2 },
			pagesScanned: 2,
			pagesWithIssues: 2
		},
		scanners: [
			{
				id: 'axe',
				status: 'success',
				issueCount: 2,
				severity: { critical: 0, serious: 1, moderate: 1, minor: 0, info: 0 }
			}
		],
		pages: [
			{
				id: 'page-1',
				url: 'https://example.com/page-1',
				path: '/page-1',
				issueCount: 1,
				durationMs: 10,
				bySeverity: { critical: 0, serious: 0, moderate: 1, minor: 0, info: 0 },
				pageOverview: {
					screenshotFilename: 'page-1.png',
					pageWidth: 1200,
					pageHeight: 800,
					elements: [
						{
							issueId: 'issue-1',
							ruleId: 'heading-order',
							severity: 'moderate',
							selector: '.first',
							nodeIndex: 0,
							xPercent: 10,
							yPercent: 10,
							widthPercent: 20,
							heightPercent: 5,
							x: 120,
							y: 80,
							width: 240,
							height: 40
						},
						{
							issueId: 'issue-1',
							ruleId: 'heading-order',
							severity: 'moderate',
							selector: '.second',
							nodeIndex: 1,
							xPercent: 20,
							yPercent: 20,
							widthPercent: 20,
							heightPercent: 5,
							x: 240,
							y: 160,
							width: 240,
							height: 40
						}
					]
				}
			},
			{
				id: 'page-2',
				url: 'https://example.com/page-2',
				path: '/page-2',
				issueCount: 1,
				durationMs: 10,
				bySeverity: { critical: 0, serious: 1, moderate: 0, minor: 0, info: 0 }
			}
		],
		issues: [
			{
				id: 'issue-1',
				scanner: 'axe',
				ruleId: 'heading-order',
				severity: 'moderate',
				title: 'Heading levels should only increase by one',
				description: 'Heading levels should increase by one.',
				pageId: 'page-1',
				pageUrl: 'https://example.com/page-1',
				elementCount: 2,
				occurrences: [
					{
						selector: '.first',
						elementId: 'issue-1-el-0',
						failureSummary: 'Fix first occurrence.'
					},
					{
						selector: '.second',
						elementId: 'issue-1-el-1',
						failureSummary: 'Fix second occurrence.'
					}
				]
			},
			{
				id: 'issue-2',
				scanner: 'axe',
				ruleId: 'landmark-one-main',
				severity: 'serious',
				title: 'Page should have one main landmark',
				description: 'Ensure one main landmark exists.',
				pageId: 'page-2',
				pageUrl: 'https://example.com/page-2',
				elementCount: 3
			}
		]
	};
}

describe('buildOccurrenceModeReport', () => {
	it('splits grouped issues by occurrence and keeps first issue ID unchanged', () => {
		const result = buildOccurrenceModeReport(createReport());

		expect(result.issues).toHaveLength(3);
		expect(result.issues[0]?.id).toBe('issue-1');
		expect(result.issues[1]?.id).toBe('issue-1--occ-2');
		expect(result.issues[2]?.id).toBe('issue-2');
	});

	it('normalizes each derived issue occurrence to a single element ID', () => {
		const result = buildOccurrenceModeReport(createReport());

		const firstOccurrence = result.issues[0]?.occurrences?.[0];
		const secondOccurrence = result.issues[1]?.occurrences?.[0];

		expect(result.issues[0]?.elementCount).toBe(1);
		expect(result.issues[1]?.elementCount).toBe(1);
		expect(firstOccurrence?.elementId).toBe('issue-1-el-0');
		expect(secondOccurrence?.elementId).toBe('issue-1--occ-2-el-0');
	});

	it('remaps page overview overlays to derived issue IDs', () => {
		const result = buildOccurrenceModeReport(createReport());
		const elements = result.pages[0]?.pageOverview?.elements ?? [];

		expect(elements).toHaveLength(2);
		expect(elements[0]).toMatchObject({ issueId: 'issue-1', nodeIndex: 0 });
		expect(elements[1]).toMatchObject({
			issueId: 'issue-1--occ-2',
			nodeIndex: 0
		});
	});

	it('recomputes summary, page, and scanner counts from derived issues', () => {
		const result = buildOccurrenceModeReport(createReport());

		expect(result.summary.totalIssues).toBe(3);
		expect(result.summary.bySeverity).toEqual({
			critical: 0,
			serious: 1,
			moderate: 2,
			minor: 0,
			info: 0
		});
		expect(result.summary.byScanner).toEqual({ axe: 3 });
		expect(result.summary.pagesScanned).toBe(2);
		expect(result.summary.pagesWithIssues).toBe(2);

		expect(result.pages[0]?.issueCount).toBe(2);
		expect(result.pages[1]?.issueCount).toBe(1);
		expect(result.scanners[0]?.issueCount).toBe(3);
		expect(result.scanners[0]?.severity).toEqual({
			critical: 0,
			serious: 1,
			moderate: 2,
			minor: 0,
			info: 0
		});
	});

	it('leaves issues without occurrences unchanged', () => {
		const result = buildOccurrenceModeReport(createReport());
		const noOccurrenceIssue = result.issues.find((issue) => issue.id === 'issue-2');

		expect(noOccurrenceIssue).toMatchObject({
			id: 'issue-2',
			elementCount: 3,
			pageId: 'page-2'
		});
		expect(noOccurrenceIssue?.occurrences).toBeUndefined();
	});
});
