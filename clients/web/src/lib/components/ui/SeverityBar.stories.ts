import type { Meta, StoryObj } from '@storybook/svelte';

import { expect, within } from '@storybook/test';

import SeverityBar from './SeverityBar.svelte';

interface SeverityBarStoryArgs {
	counts: { critical: number; serious: number; moderate: number; minor: number; info: number };
	showLabels: boolean;
	height: 'sm' | 'md';
}

const meta = {
	title: 'UI/SeverityBar',
	component: SeverityBar,
	tags: ['autodocs'],
	args: {
		counts: { critical: 3, serious: 8, moderate: 14, minor: 5, info: 2 },
		showLabels: true,
		height: 'md'
	},
	argTypes: {
		showLabels: { control: 'boolean' },
		height: { control: 'select', options: ['sm', 'md'] }
	},
	parameters: {
		layout: 'padded'
	}
} satisfies Meta<SeverityBarStoryArgs>;

export default meta;
type Story = StoryObj<typeof meta>;

export const FullDistribution: Story = {
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getByTestId('severity-bar')).toBeVisible();
		await expect(canvas.getByText('3')).toBeVisible();
		await expect(canvas.getByText('Critical')).toBeVisible();
	}
};

export const CriticalOnly: Story = {
	args: {
		counts: { critical: 5, serious: 0, moderate: 0, minor: 0, info: 0 }
	},
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getByText('Critical')).toBeVisible();
		await expect(canvas.queryByText('Serious')).not.toBeInTheDocument();
	}
};

export const NoIssues: Story = {
	args: {
		counts: { critical: 0, serious: 0, moderate: 0, minor: 0, info: 0 },
		showLabels: false
	},
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getByTestId('severity-bar')).toBeVisible();
	}
};

export const NoLabels: Story = {
	args: { showLabels: false },
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.queryByText('Critical')).not.toBeInTheDocument();
	}
};

export const SmallHeight: Story = {
	args: { height: 'sm' }
};
