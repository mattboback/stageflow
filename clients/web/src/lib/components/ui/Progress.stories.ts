import type { Meta, StoryObj } from '@storybook/svelte';

import { expect, within } from '@storybook/test';

import Progress from './Progress.svelte';

interface ProgressStoryArgs {
	value: number;
	max: number;
	'data-testid': string;
}

const meta = {
	title: 'UI/Progress',
	component: Progress,
	tags: ['autodocs'],
	args: {
		value: 65,
		max: 100,
		'data-testid': 'progress'
	}
} satisfies Meta<ProgressStoryArgs>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		const progress = canvas.getByTestId('progress');
		const indicator = progress.firstElementChild;

		if (!(indicator instanceof HTMLElement)) {
			throw new Error('Progress indicator not rendered');
		}

		await expect(progress).toBeVisible();
		await expect(indicator.style.transform).toContain('-35%');
	}
};

export const Complete: Story = {
	args: {
		value: 100
	}
};

export const OverflowClamped: Story = {
	args: {
		value: 200
	}
};
