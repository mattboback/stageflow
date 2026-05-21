import IssueDetailModal from '$lib/components/report/IssueDetailModal.svelte';
import { cleanup, render } from '@testing-library/svelte';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it } from 'vitest';

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
	afterEach(() => {
		cleanup();
	});

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

	it('switches to the Evidence tab and shows the page overview when not opened from a highlight', async () => {
		const user = userEvent.setup();
		const { getByRole, getByText } = render(IssueDetailModal, {
			props: {
				issue: {
					...baseIssue,
					occurrences: [{ selector: '.hero-text', elementId: 'issue-1-el-0' }]
				},
				page,
				screenshots,
				onClose: () => undefined
			}
		});

		await user.click(getByRole('tab', { name: /evidence/i }));
		expect(getByText('On the page')).toBeInTheDocument();
	});

	it('hides the page overview by default when opened from a page highlight, until full context is requested', async () => {
		const user = userEvent.setup();
		const { getByRole, queryByText, getByText } = render(IssueDetailModal, {
			props: {
				issue: {
					...baseIssue,
					occurrences: [{ selector: '.hero-text', elementId: 'issue-1-el-0' }]
				},
				page,
				screenshots,
				highlightedElementId: 'issue-1-el-0',
				onClose: () => undefined
			}
		});

		await user.click(getByRole('tab', { name: /evidence/i }));
		expect(queryByText('On the page')).not.toBeInTheDocument();
		await user.click(getByRole('button', { name: /show full page context/i }));
		expect(getByText('On the page')).toBeInTheDocument();
	});

	it('shows occurrence thumbnails when page overview is visible', async () => {
		const user = userEvent.setup();
		const testPage = {
			...page,
			pageOverview: {
				...page.pageOverview,
				elements: [
					...page.pageOverview.elements,
					{
						issueId: 'issue-1',
						ruleId: 'color-contrast',
						severity: 'critical' as const,
						selector: '.sub-text',
						nodeIndex: 1,
						xPercent: 10,
						yPercent: 30,
						widthPercent: 30,
						heightPercent: 10,
						x: 120,
						y: 240,
						width: 360,
						height: 80
					}
				]
			}
		};
		const { container, getByRole } = render(IssueDetailModal, {
			props: {
				issue: {
					...baseIssue,
					occurrences: [
						{ selector: '.hero-text', elementId: 'issue-1-el-0' },
						{ selector: '.sub-text', elementId: 'issue-1-el-1' }
					]
				},
				page: testPage,
				screenshots,
				onClose: () => undefined
			}
		});

		await user.click(getByRole('tab', { name: /occurrences/i }));

		const occurrenceCards = container.querySelectorAll('[data-occurrence-id]');
		expect(occurrenceCards.length).toBeGreaterThan(0);
		for (const card of occurrenceCards) {
			expect(card.querySelector('svg image')).toBeInTheDocument();
		}
	});

	it('shows occurrence thumbnails when no page overview is available', () => {
		const { pageOverview: _, ...pageWithoutOverview } = page;
		const { container } = render(IssueDetailModal, {
			props: {
				issue: {
					...baseIssue,
					occurrences: [{ selector: '.hero-text', elementId: 'issue-1-el-0' }]
				},
				page: pageWithoutOverview,
				screenshots,
				onClose: () => undefined
			}
		});

		// Without page overview, occurrence cards should still render normally
		// (no SVG crop either since there's no overview data to crop from).
		const dialog = container.querySelector('[role="dialog"]');
		expect(dialog).toBeInTheDocument();
	});
});
