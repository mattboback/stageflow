import type { ScanResult, ScanStatus } from '$lib/types/scan';
import type { Meta, StoryObj } from '@storybook/svelte';

import { expect, within } from '@storybook/test';

import ScanStatusHeader from './ScanStatusHeader.svelte';

interface ScanStatusHeaderStoryArgs {
	id: string;
	status: ScanStatus;
	elapsed: number;
	result: ScanResult | null;
}

const scanningResult: ScanResult = {
	id: 'job-storybook',
	state: 'RUNNING',
	progress: {
		current_page: 2,
		total_pages: 5,
		percentage: 40
	},
	created_at: '2026-01-01T00:00:00.000Z',
	updated_at: '2026-01-01T00:00:45.000Z'
};

const meta = {
	title: 'Scan Status/Header',
	component: ScanStatusHeader,
	tags: ['autodocs'],
	args: {
		id: 'job-storybook',
		status: 'scanning',
		elapsed: 45,
		result: scanningResult
	},
	parameters: {
		layout: 'padded'
	}
} satisfies Meta<ScanStatusHeaderStoryArgs>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Processing: Story = {
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);

		await expect(canvas.getByText('SCANNING')).toBeVisible();
		await expect(canvas.getByText('Scanning page 2 of 5')).toBeVisible();
	}
};

export const Complete: Story = {
	args: {
		status: 'complete',
		result: null,
		elapsed: 133
	},
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);

		await expect(canvas.getByText('COMPLETE')).toBeVisible();
		await expect(canvas.queryByText('Initializing scan...')).not.toBeInTheDocument();
	}
};
