import type { ScanResult } from '$lib/types/scan';
import type { UnifiedReport } from '$lib/types/unified-report';
import type { Meta, StoryObj } from '@storybook/svelte';

import { expect, fn, userEvent, within } from '@storybook/test';

import ScannersViewStoryHarness from './story-harnesses/ScannersViewStoryHarness.svelte';

interface ScannersViewStoryArgs {
	report: UnifiedReport;
	job: ScanResult;
	initialScanner: string;
	onSelectScanner: (scannerId: string) => void;
}

const report: UnifiedReport = {
	version: '2.0.0',
	meta: { jobId: 'job-storybook' },
	summary: {
		totalIssues: 4,
		bySeverity: { critical: 1, serious: 1, moderate: 2, minor: 0, info: 0 },
		pagesScanned: 2,
		pagesWithIssues: 2,
		lighthouseCategories: [
			{ id: 'performance', title: 'Performance', avgScore: 0.72 },
			{ id: 'accessibility', title: 'Accessibility', avgScore: 0.81 }
		]
	},
	scanners: [
		{
			id: 'lighthouse',
			name: 'Lighthouse',
			status: 'success',
			issueCount: 2,
			severity: { critical: 0, serious: 1, moderate: 1, minor: 0, info: 0 }
		},
		{
			id: 'security-headers',
			name: 'Security Headers',
			status: 'failed',
			issueCount: 2
		}
	],
	pages: [
		{
			id: 'page-home',
			url: 'https://example.com',
			issueCount: 2,
			durationMs: 6500
		},
		{
			id: 'page-contact',
			url: 'https://example.com/contact',
			issueCount: 2,
			durationMs: 5500
		}
	],
	issues: [
		{
			id: 'issue-lh-1',
			scanner: 'lighthouse',
			ruleId: 'largest-contentful-paint',
			severity: 'serious',
			title: 'Largest Contentful Paint is high',
			description: 'LCP exceeds recommended threshold.',
			pageId: 'page-home',
			pageUrl: 'https://example.com',
			elementCount: 1
		},
		{
			id: 'issue-lh-2',
			scanner: 'lighthouse',
			ruleId: 'total-blocking-time',
			severity: 'moderate',
			title: 'Total blocking time is elevated',
			description: 'Main thread blocking time is above target.',
			pageId: 'page-contact',
			pageUrl: 'https://example.com/contact',
			elementCount: 1
		},
		{
			id: 'issue-sh-1',
			scanner: 'security-headers',
			ruleId: 'missing-content-security-policy',
			severity: 'critical',
			title: 'Missing Content-Security-Policy header',
			description: 'Responses do not include CSP.',
			pageId: 'page-home',
			pageUrl: 'https://example.com',
			elementCount: 1
		},
		{
			id: 'issue-sh-2',
			scanner: 'security-headers',
			ruleId: 'missing-strict-transport-security',
			severity: 'moderate',
			title: 'Missing Strict-Transport-Security header',
			description: 'Responses do not include HSTS.',
			pageId: 'page-contact',
			pageUrl: 'https://example.com/contact',
			elementCount: 1
		}
	]
};

const job: ScanResult = {
	id: 'job-storybook',
	state: 'DONE',
	created_at: '2026-01-01T00:00:00.000Z',
	updated_at: '2026-01-01T00:01:05.000Z',
	artifacts: {
		report_json: 'https://example.com/report.json',
		report_html: 'https://example.com/report.html',
		scanner_artifacts: {
			lighthouse: {
				scanner_type: 'lighthouse',
				results_json: 'https://example.com/lighthouse.json',
				report_html: 'https://example.com/lighthouse.html'
			}
		}
	}
};

const meta = {
	title: 'Report/Scanners View',
	component: ScannersViewStoryHarness,
	tags: ['autodocs'],
	args: {
		report,
		job,
		initialScanner: 'lighthouse',
		onSelectScanner: fn()
	},
	parameters: {
		layout: 'fullscreen'
	}
} satisfies Meta<ScannersViewStoryArgs>;

export default meta;
type Story = StoryObj<typeof meta>;

export const SwitchScannerDetails: Story = {
	play: async ({ args, canvasElement }) => {
		const canvas = within(canvasElement);
		const securityHeadersCard = canvas.getByRole('button', {
			name: /security headers/i
		});

		await userEvent.click(securityHeadersCard);
		await expect(args.onSelectScanner).toHaveBeenCalledWith('security-headers');
		await expect(canvas.getByText('Header gaps')).toBeVisible();
	}
};
