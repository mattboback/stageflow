import type { IssueDetail, PageSummary } from '$lib/types/unified-report';
import type { Meta, StoryObj } from '@storybook/svelte';

import { expect, userEvent, within } from '@storybook/test';

import VerifyContrastTab from './VerifyContrastTab.svelte';

interface VerifyContrastTabStoryArgs {
	issue: IssueDetail;
	page: PageSummary | null;
	pageOverviewUrl: string | null;
	jobId: string;
}

const screenshot = `data:image/svg+xml;utf8,${encodeURIComponent(
	`<svg xmlns="http://www.w3.org/2000/svg" width="800" height="400">
		<rect width="800" height="400" fill="#3a6ea5"/>
		<text x="60" y="210" font-family="sans-serif" font-size="40" fill="#ffffff">Hero headline over an image</text>
	</svg>`
)}`;

const baseIssue: IssueDetail = {
	id: 'issue-contrast-1',
	scanner: 'axe',
	ruleId: 'color-contrast',
	severity: 'serious',
	title: 'Color contrast needs manual verification',
	description: 'axe-core could not determine the background color for this text.',
	pageId: 'page-1',
	pageUrl: 'https://example.com/',
	elementCount: 1,
	occurrences: [{ selector: '.hero h1', html: '<h1>Hero headline over an image</h1>' }],
	scannerData: {
		axeIncomplete: true,
		contrastData: {
			fgColor: '#ffffff',
			fontSize: '30.0pt (40.0px)',
			fontWeight: 'normal',
			messageKey: 'bgImage'
		}
	}
};

const basePage: PageSummary = {
	id: 'page-1',
	url: 'https://example.com/',
	issueCount: 1,
	durationMs: 4200,
	pageOverview: {
		screenshotFilename: 'page-overview-page-1.png',
		pageWidth: 800,
		pageHeight: 400,
		elements: [
			{
				issueId: 'issue-contrast-1',
				ruleId: 'color-contrast',
				severity: 'serious',
				selector: '.hero h1',
				nodeIndex: 0,
				x: 60,
				y: 178,
				width: 520,
				height: 44,
				xPercent: 7.5,
				yPercent: 44.5,
				widthPercent: 65,
				heightPercent: 11
			}
		]
	}
};

const meta = {
	title: 'Report/Verify/Verify Contrast Tab',
	component: VerifyContrastTab,
	tags: ['autodocs'],
	args: {
		issue: baseIssue,
		page: basePage,
		pageOverviewUrl: screenshot,
		jobId: 'job-sb-needs-review'
	},
	parameters: {
		layout: 'padded'
	}
} satisfies Meta<VerifyContrastTabStoryArgs>;

export default meta;
type Story = StoryObj<typeof meta>;

export const NeedsReview: Story = {
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);

		await expect(canvas.getByText('Why this needs a human')).toBeVisible();
		await expect(canvas.getByText(/background image/)).toBeVisible();
		await expect(canvas.getByRole('button', { name: 'Text' })).toBeVisible();
		// fgColor prefilled from axe; large text auto-detected from the 40px font.
		await expect(canvas.getByLabelText('Text color')).toHaveValue('#ffffff');
		await expect(canvas.getByRole('checkbox')).toBeChecked();
	}
};

export const PrefilledViolation: Story = {
	args: {
		jobId: 'job-sb-violation',
		issue: {
			...baseIssue,
			id: 'issue-contrast-2',
			title: 'Elements must meet minimum color contrast ratio thresholds',
			scannerData: {
				contrastData: {
					fgColor: '#999999',
					bgColor: '#ffffff',
					contrastRatio: 2.85,
					expectedContrastRatio: '4.5:1',
					fontSize: '10.0pt (13.3333px)',
					fontWeight: 'normal'
				}
			}
		},
		page: {
			...basePage,
			pageOverview: {
				...basePage.pageOverview!,
				elements: [{ ...basePage.pageOverview!.elements[0], issueId: 'issue-contrast-2' }]
			}
		}
	},
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);

		await expect(canvas.getByText(/axe measured/)).toBeVisible();
		await expect(canvas.getByText('2.85')).toBeVisible();
		await expect(canvas.getAllByText('Fail')).toHaveLength(2);
	}
};

export const NoScreenshotVerdictFlow: Story = {
	args: {
		jobId: 'job-sb-flow',
		issue: { ...baseIssue, id: 'issue-contrast-3' },
		page: null,
		pageOverviewUrl: null
	},
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);

		await expect(canvas.getByText(/No screenshot is available/)).toBeVisible();

		await userEvent.click(canvas.getByRole('button', { name: 'Mark as fail' }));
		await expect(canvas.getByText('Verified · fail')).toBeVisible();

		await userEvent.click(canvas.getByRole('button', { name: 'Clear verdict' }));
		await expect(canvas.getByRole('button', { name: 'Mark as pass' })).toBeVisible();
	}
};
