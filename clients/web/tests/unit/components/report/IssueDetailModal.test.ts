import IssueDetailModal from '$lib/components/report/IssueDetailModal.svelte';
import { render } from '@testing-library/svelte';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';

const baseIssue = {
	id: 'issue-1',
	scanner: 'axe',
	ruleId: 'color-contrast',
	severity: 'critical' as const,
	title: 'Low contrast',
	description: 'Text has low contrast',
	pageId: 'page-1',
	pageUrl: 'http://example.com',
	elementCount: 1,
	helpUrl: 'https://example.com/help'
};

const page = {
	id: 'page-1',
	url: 'http://example.com',
	path: '/',
	issueCount: 1,
	durationMs: 1000,
	pageOverview: {
		screenshotFilename: 'overview.png',
		pageWidth: 1200,
		pageHeight: 800,
		elements: [
			{
				issueId: 'issue-1',
				ruleId: 'color-contrast',
				severity: 'critical' as const,
				selector: '.hero-text',
				nodeIndex: 0,
				xPercent: 10,
				yPercent: 20,
				widthPercent: 30,
				heightPercent: 10,
				x: 120,
				y: 160,
				width: 360,
				height: 80
			}
		]
	}
};

const screenshots = [
	{
		kind: 'violation' as const,
		issue_id: 'issue-1',
		occurrence_index: 0,
		artifact_id: 'ss-issue-1',
		scanner_id: 'axe',
		page_id: 'page-1',
		url: 'http://example.com/issue.png'
	},
	{
		kind: 'page_overview' as const,
		artifact_id: 'page-overview:axe:page-1',
		scanner_id: 'axe',
		page_id: 'page-1',
		url: 'http://example.com/overview.png'
	}
];

describe('IssueDetailModal', () => {
	it('traps focus within modal', async () => {
		const user = userEvent.setup();
		render(IssueDetailModal, {
			props: {
				issue: baseIssue,
				screenshots: [],
				onClose: () => undefined
			}
		});

		const dialog = document.querySelector('div[role="dialog"][aria-label="Issue details"]');
		expect(dialog).toBeInTheDocument();

		const closeButton = document.querySelector("button[aria-label='Close modal']");
		expect(closeButton).toBeInTheDocument();
		expect(document.activeElement).toBe(closeButton);

		for (let i = 0; i < 12; i++) {
			await user.tab();
			expect(dialog?.contains(document.activeElement)).toBe(true);
		}

		for (let i = 0; i < 12; i++) {
			await user.keyboard('{Shift>}{Tab}{/Shift}');
			expect(dialog?.contains(document.activeElement)).toBe(true);
		}
	});

	it('hides page overview evidence by default when opened from a page highlight', async () => {
		const user = userEvent.setup();
		const { getByRole, queryByText, getByText } = render(IssueDetailModal, {
			props: {
				issue: {
					...baseIssue,
					occurrences: [{ selector: '.hero-text', elementId: 'issue-1-el-0' }]
				},
				page,
				audience: 'engineer',
				screenshots,
				highlightedElementId: 'issue-1-el-0',
				onClose: () => undefined
			}
		});

		expect(queryByText('On the page')).not.toBeInTheDocument();
		expect(getByText('Scanner screenshot')).toBeInTheDocument();

		await user.click(getByRole('button', { name: /show full page context/i }));
		expect(getByText('On the page')).toBeInTheDocument();
		expect(getByRole('button', { name: /hide full page context/i })).toBeInTheDocument();
	});

	it('keeps technical evidence collapsed by default for PM audience', async () => {
		const user = userEvent.setup();
		const { getByText } = render(IssueDetailModal, {
			props: {
				issue: {
					...baseIssue,
					occurrences: [{ selector: '.hero-text', elementId: 'issue-1-el-0' }]
				},
				page,
				audience: 'pm',
				screenshots,
				onClose: () => undefined
			}
		});

		const summary = getByText('Technical evidence (optional)');
		const details = summary.closest('details');
		expect(summary).toBeInTheDocument();
		expect(details).toBeInTheDocument();
		expect(details).not.toHaveAttribute('open');

		await user.click(summary);
		expect(details).toHaveAttribute('open');
	});
});
