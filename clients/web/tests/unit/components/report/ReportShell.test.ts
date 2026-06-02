import type { ScreenshotArtifact } from '$lib/types/scan';
import type { UnifiedReport } from '$lib/types/unified-report';

import ReportShell from '$lib/components/report/ReportShell.svelte';
import { render } from '@testing-library/svelte';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const { mockGoto, mockPage } = vi.hoisted(() => ({
	mockGoto: vi.fn(),
	mockPage: { url: new URL('http://localhost/scan/job-1/report') }
}));

vi.mock('$app/navigation', () => ({
	goto: mockGoto
}));

vi.mock('$app/state', () => ({
	page: mockPage
}));

function createBaseReport(): UnifiedReport {
	return {
		version: '2.0.0',
		meta: { jobId: 'job-1' },
		summary: {
			totalIssues: 1,
			bySeverity: { critical: 0, serious: 0, moderate: 1, minor: 0, info: 0 },
			pagesScanned: 1,
			pagesWithIssues: 1
		},
		scanners: [{ id: 'axe', status: 'success', issueCount: 1 }],
		pages: [
			{
				id: 'page-1',
				url: 'http://example.com',
				issueCount: 1,
				durationMs: 1000
			}
		],
		issues: [
			{
				id: 'issue-1',
				scanner: 'axe',
				ruleId: 'color-contrast',
				severity: 'moderate',
				title: 'Low contrast',
				description: 'Text has low contrast',
				pageId: 'page-1',
				pageUrl: 'http://example.com',
				elementCount: 1
			}
		]
	};
}

function createOccurrenceReport(): UnifiedReport {
	return {
		version: '2.0.0',
		meta: { jobId: 'job-1' },
		summary: {
			totalIssues: 1,
			bySeverity: { critical: 0, serious: 0, moderate: 1, minor: 0, info: 0 },
			pagesScanned: 1,
			pagesWithIssues: 1
		},
		scanners: [{ id: 'axe', status: 'success', issueCount: 1 }],
		pages: [
			{
				id: 'page-1',
				url: 'http://example.com',
				issueCount: 1,
				durationMs: 1000,
				pageOverview: {
					screenshotFilename: 'overview.png',
					pageWidth: 1200,
					pageHeight: 900,
					elements: [
						{
							issueId: 'issue-1',
							ruleId: 'heading-order',
							severity: 'moderate',
							selector: '.first',
							nodeIndex: 0,
							xPercent: 10,
							yPercent: 10,
							widthPercent: 10,
							heightPercent: 10,
							x: 100,
							y: 100,
							width: 120,
							height: 80
						},
						{
							issueId: 'issue-1',
							ruleId: 'heading-order',
							severity: 'moderate',
							selector: '.second',
							nodeIndex: 1,
							xPercent: 30,
							yPercent: 30,
							widthPercent: 10,
							heightPercent: 10,
							x: 300,
							y: 300,
							width: 120,
							height: 80
						}
					]
				}
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
				pageUrl: 'http://example.com',
				elementCount: 2,
				occurrences: [
					{ selector: '.first', elementId: 'issue-1-el-0' },
					{ selector: '.second', elementId: 'issue-1-el-1' }
				]
			}
		]
	};
}

const screenshots: ScreenshotArtifact[] = [
	{
		kind: 'page_overview',
		artifact_id: 'page-overview:axe:page-1',
		scanner_id: 'axe',
		page_id: 'page-1',
		url: 'http://example.com/overview.png'
	}
];

describe('ReportShell', () => {
	beforeEach(() => {
		mockGoto.mockReset();
		mockPage.url = new URL('http://localhost/scan/job-1/report');
	});

	it('switches sections via nav', async () => {
		const user = userEvent.setup();
		const { getByText } = render(ReportShell, {
			props: {
				jobId: 'job-1',
				status: 'complete',
				report: createBaseReport(),
				job: null,
				logs: [],
				screenshots: [],
				error: null
			}
		});

		const issuesTab = getByText('Issues');
		await user.click(issuesTab);

		expect(mockGoto).toHaveBeenCalledWith('/scan/job-1/report?section=issues', {
			replaceState: false,
			noScroll: true,
			keepFocus: true
		});
	});

	it('uses triage actions in the report header to jump to issues and pages', async () => {
		const user = userEvent.setup();
		const { getAllByRole, getByRole } = render(ReportShell, {
			props: {
				jobId: 'job-1',
				status: 'complete',
				report: createBaseReport(),
				job: null,
				logs: [],
				screenshots: [],
				error: null
			}
		});

		await user.click(getAllByRole('button', { name: /^review issues$/i })[0]);
		expect(mockGoto).toHaveBeenCalledWith('/scan/job-1/report?section=issues', {
			replaceState: false,
			noScroll: true,
			keepFocus: true
		});

		await user.click(getAllByRole('button', { name: /check pages/i })[0]);
		expect(mockGoto).toHaveBeenCalledWith('/scan/job-1/report?section=pages', {
			replaceState: false,
			noScroll: true,
			keepFocus: true
		});
	});

	it('renders the score band label in the report header', () => {
		const report = createBaseReport();
		const { getByText } = render(ReportShell, {
			props: {
				jobId: 'job-1',
				status: 'complete',
				report: {
					...report,
					summary: {
						...report.summary,
						score: 77
					}
				},
				job: null,
				logs: [],
				screenshots: [],
				error: null
			}
		});

		expect(getByText('Needs work')).toBeInTheDocument();
	});

	it('opens the derived occurrence issue when clicking a non-first overlay marker', async () => {
		mockPage.url = new URL('http://localhost/scan/job-1/report?section=pages&page=page-1');

		const user = userEvent.setup();
		const { container } = render(ReportShell, {
			props: {
				jobId: 'job-1',
				status: 'complete',
				report: createOccurrenceReport(),
				job: null,
				logs: [],
				screenshots,
				error: null
			}
		});

		const markers = container.querySelectorAll('rect.overlay-rect');
		expect(markers).toHaveLength(2);

		await user.click(markers[1]);

		expect(mockGoto).toHaveBeenCalledWith(
			expect.stringContaining('issueId=issue-1--occ-2'),
			expect.objectContaining({
				replaceState: false,
				noScroll: true,
				keepFocus: true
			})
		);
		expect(mockGoto).toHaveBeenCalledWith(
			expect.stringContaining('elementId=issue-1--occ-2-el-0'),
			expect.objectContaining({
				replaceState: false,
				noScroll: true,
				keepFocus: true
			})
		);
	});
});
