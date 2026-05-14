import type { IssueDetail, UnifiedReport } from '$lib/types/unified-report';

import IssuesView from '$lib/components/report/IssuesView.svelte';
import { cleanup, render } from '@testing-library/svelte';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';

function buildReport(issueCount: number): UnifiedReport {
	const issues: IssueDetail[] = [];
	for (let i = 0; i < issueCount; i += 1) {
		issues.push({
			id: `issue-${i}`,
			scanner: 'axe',
			ruleId: 'color-contrast',
			severity: 'moderate',
			title: 'Low contrast',
			description: 'Text has low contrast',
			pageId: 'page-1',
			pageUrl: 'http://example.com',
			elementCount: 1
		});
	}

	return {
		version: '2.0.0',
		meta: { jobId: 'job-1' },
		summary: {
			totalIssues: issues.length,
			bySeverity: {
				critical: 0,
				serious: 0,
				moderate: issues.length,
				minor: 0,
				info: 0
			},
			pagesScanned: 1,
			pagesWithIssues: 1
		},
		scanners: [
			{
				id: 'axe',
				status: 'success',
				issueCount: issues.length
			}
		],
		pages: [
			{
				id: 'page-1',
				url: 'http://example.com',
				issueCount: issues.length,
				durationMs: 1000
			}
		],
		issues
	};
}

describe('IssuesView', () => {
	afterEach(() => {
		cleanup();
	});

	it('virtualizes when issue count is high', () => {
		const report = buildReport(260);
		const { container } = render(IssuesView, {
			props: {
				report,
				screenshots: [],
				activeScanner: null,
				activePage: null,
				activeSeverity: null,
				activeCategory: null,
				searchTerm: '',
				issueSort: 'severity',
				selectedIssueId: null,
				onScannerChange: () => undefined,
				onPageChange: () => undefined,
				onSeverityChange: () => undefined,
				onCategoryChange: () => undefined,
				onSearchChange: () => undefined,
				onSortChange: () => undefined,
				onIssueSelect: () => undefined,
				onClearFilters: () => undefined
			}
		});

		const list = container.querySelector("[data-testid='issue-list']");
		expect(list).toBeInTheDocument();
		const rows = container.querySelectorAll("[data-testid='issue-row']");
		expect(rows.length).toBeLessThan(report.issues.length);
	});

	it('updates the input immediately and debounces URL/search updates', async () => {
		vi.useFakeTimers();
		const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
		const report = buildReport(10);
		const onSearchChange = vi.fn();

		const { getByPlaceholderText } = render(IssuesView, {
			props: {
				report,
				screenshots: [],
				activeScanner: null,
				activePage: null,
				activeSeverity: null,
				activeCategory: null,
				searchTerm: '',
				issueSort: 'severity',
				selectedIssueId: null,
				onScannerChange: () => undefined,
				onPageChange: () => undefined,
				onSeverityChange: () => undefined,
				onCategoryChange: () => undefined,
				onSearchChange,
				onSortChange: () => undefined,
				onIssueSelect: () => undefined,
				onClearFilters: () => undefined
			}
		});

		const input = getByPlaceholderText('Search issues') as HTMLInputElement;
		await user.type(input, 'abc');
		expect(input.value).toBe('abc');
		expect(onSearchChange).not.toHaveBeenCalled();

		await vi.advanceTimersByTimeAsync(249);
		expect(onSearchChange).not.toHaveBeenCalled();

		await vi.advanceTimersByTimeAsync(1);
		expect(onSearchChange).toHaveBeenCalledTimes(1);
		expect(onSearchChange).toHaveBeenCalledWith('abc');

		vi.useRealTimers();
	});

	it('renders the issue search input', () => {
		const report = buildReport(4);
		const { getByPlaceholderText } = render(IssuesView, {
			props: {
				report,
				screenshots: [],
				activeScanner: null,
				activePage: null,
				activeSeverity: null,
				activeCategory: null,
				searchTerm: '',
				issueSort: 'severity',
				selectedIssueId: null,
				onScannerChange: () => undefined,
				onPageChange: () => undefined,
				onSeverityChange: () => undefined,
				onCategoryChange: () => undefined,
				onSearchChange: () => undefined,
				onSortChange: () => undefined,
				onIssueSelect: () => undefined,
				onClearFilters: () => undefined
			}
		});

		expect(getByPlaceholderText('Search issues')).toBeInTheDocument();
	});
});
