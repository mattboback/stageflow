import type { Meta, StoryObj } from '@storybook/svelte';

import { expect, within } from '@storybook/test';

import Skeleton from './Skeleton.svelte';

const meta = {
	title: 'UI/Skeleton',
	component: Skeleton,
	tags: ['autodocs'],
	args: {
		lines: 3
	},
	argTypes: {
		lines: { control: { type: 'number', min: 1, max: 10, step: 1 } }
	},
	parameters: {
		layout: 'padded'
	}
} satisfies Meta<typeof Skeleton>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		const skeleton = canvas.getByTestId('skeleton');
		await expect(skeleton).toBeInTheDocument();
		await expect(skeleton.children).toHaveLength(3);
	}
};

export const SingleLine: Story = {
	args: { lines: 1 },
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getByTestId('skeleton').children).toHaveLength(1);
	}
};

export const Paragraph: Story = {
	args: { lines: 6 },
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getByTestId('skeleton').children).toHaveLength(6);
	}
};
