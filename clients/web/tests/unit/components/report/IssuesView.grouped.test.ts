import type { IssueDetail, UnifiedReport } from '$lib/types/unified-report';

import IssuesView from '$lib/components/report/IssuesView.svelte';
import { cleanup, render } from '@testing-library/svelte';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it } from 'vitest';

function makeIssue(i: number, ruleId = 'color-contrast'): IssueDetail {
	return {
		id: `issue-${i}`,
		scanner: 'axe',
		ruleId,
		severity: 'moderate',
		title: 'Low contrast text',
		description: 'Text has low contrast',
		pageId: 'page-1',
		pageUrl: 'http://example.com',
		elementCount: 1
	};
}

function buildReport(issues: IssueDetail[]): UnifiedReport {
	return {
		version: '2.0.0',
		meta: { jobId: 'job-1' },
		summary: {
			totalIssues: issues.length,
			bySeverity: { critical: 0, serious: 0, moderate: issues.length, minor: 0, info: 0 },
			pagesScanned: 1,
			pagesWithIssues: 1
		},
		scanners: [{ id: 'axe', status: 'success', issueCount: issues.length }],
		pages: [
			{ id: 'page-1', url: 'http://example.com', issueCount: issues.length, durationMs: 100 }
		],
		issues
	};
}

const defaultProps = {
	screenshots: [],
	activeScanners: [],
	activePage: null,
	activeSeverities: [],
	activeCategory: null,
	searchTerm: '',
	issueSort: 'severity',
	selectedIssueId: null,
	onScannersChange: () => undefined,
	onPageChange: () => undefined,
	onSeveritiesChange: () => undefined,
	onCategoryChange: () => undefined,
	onSearchChange: () => undefined,
	onSortChange: () => undefined,
	onIssueSelect: () => undefined,
	onClearFilters: () => undefined
};

describe('IssuesView grouped rendering', () => {
	afterEach(() => cleanup());

	it('aggregates 10 occurrences of one rule into a single group row by default', () => {
		const issues = Array.from({ length: 10 }, (_, i) => makeIssue(i));
		const report = buildReport(issues);
		const { container } = render(IssuesView, { props: { report, ...defaultProps } });

		const rows = container.querySelectorAll("[data-testid='issue-group-row']");
		expect(rows).toHaveLength(1);
		expect(container.textContent).toContain('10');
		expect(container.textContent?.toLowerCase()).toContain('occurrences');
		expect(container.textContent?.toLowerCase()).toContain('fix once → closes 10 instances');
	});

	it('expands a group to show occurrences when clicked', async () => {
		const user = userEvent.setup();
		const issues = Array.from({ length: 3 }, (_, i) => makeIssue(i));
		const report = buildReport(issues);
		const { container } = render(IssuesView, { props: { report, ...defaultProps } });

		const groupButton = container.querySelector("[data-testid='issue-group-row'] button");
		expect(groupButton).toBeTruthy();
		expect(container.querySelector("[data-testid='group-occurrences']")).toBeNull();

		await user.click(groupButton as HTMLElement);
		expect(container.querySelector("[data-testid='group-occurrences']")).toBeTruthy();
	});

	it('displays the scanner attribution on each group row', () => {
		const issues = [makeIssue(0)];
		const report = buildReport(issues);
		const { container } = render(IssuesView, { props: { report, ...defaultProps } });

		const scannerLabel = container.querySelector("[data-testid='group-scanner']");
		expect(scannerLabel?.textContent?.trim()).toBe('axe');
	});

	it('toggles to flat view via the view-mode tab', async () => {
		const user = userEvent.setup();
		const issues = Array.from({ length: 3 }, (_, i) => makeIssue(i));
		const report = buildReport(issues);
		const { container, getByRole } = render(IssuesView, { props: { report, ...defaultProps } });

		await user.click(getByRole('tab', { name: /flat list/i }));
		expect(container.querySelector("[data-testid='issue-list']")).toBeTruthy();
		expect(container.querySelector("[data-testid='issue-group-row']")).toBeNull();
	});
});
