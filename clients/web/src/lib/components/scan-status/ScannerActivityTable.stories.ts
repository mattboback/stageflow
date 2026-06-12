import type { Meta, StoryObj } from '@storybook/svelte';

import { expect, within } from '@storybook/test';

import ScannerActivityTable from './ScannerActivityTable.svelte';

interface ScannerActivityTableStoryArgs {
	expected: string[];
	completed: string[];
	remaining: string[];
}

const EXPECTED = ['axe', 'lighthouse', 'security-headers', 'seo', 'ai-navigator'];

const meta = {
	title: 'Scan Status/Scanner Activity Table',
	component: ScannerActivityTable,
	tags: ['autodocs'],
	args: {
		expected: EXPECTED,
		completed: ['axe', 'security-headers'],
		remaining: ['lighthouse', 'seo', 'ai-navigator']
	},
	parameters: {
		layout: 'padded'
	}
} satisfies Meta<ScannerActivityTableStoryArgs>;

export default meta;
type Story = StoryObj<typeof meta>;

export const MidScan: Story = {
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);

		await expect(canvas.getByText('2 of 5 finished')).toBeVisible();
		await expect(canvas.getByText('Ai Navigator')).toBeVisible();
		await expect(canvas.getAllByText('done')).toHaveLength(2);
		await expect(canvas.getAllByText('running')).toHaveLength(3);
	}
};

export const AllFinished: Story = {
	args: {
		completed: EXPECTED,
		remaining: []
	},
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);

		await expect(canvas.getByText('5 of 5 finished')).toBeVisible();
		await expect(canvas.getAllByText('done')).toHaveLength(5);
	}
};
