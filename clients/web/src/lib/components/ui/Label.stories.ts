import type { Meta, StoryObj } from '@storybook/svelte';

import { expect, within } from '@storybook/test';

import LabelStoryHarness from './story-harnesses/LabelStoryHarness.svelte';

const meta = {
	title: 'UI/Label',
	component: LabelStoryHarness,
	tags: ['autodocs'],
	args: {
		required: false
	}
} satisfies Meta;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		const label = canvas.getByText('Project name');

		await expect(label).toBeVisible();
	}
};

export const Required: Story = {
	args: {
		required: true
	},
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);

		await expect(canvas.getByText('required')).toBeInTheDocument();
	}
};
