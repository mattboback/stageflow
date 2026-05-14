import type { Meta, StoryObj } from '@storybook/svelte';

import { expect, within } from '@storybook/test';

import Score from './Score.svelte';

const meta = {
	title: 'UI/Score',
	component: Score,
	tags: ['autodocs'],
	args: {
		score: 94,
		size: 'md',
		showPill: true
	},
	argTypes: {
		score: { control: { type: 'number', min: 0, max: 100, step: 1 } },
		size: { control: 'select', options: ['sm', 'md', 'lg'] },
		showPill: { control: 'boolean' }
	},
	parameters: {
		layout: 'padded'
	}
} satisfies Meta<typeof Score>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Strong: Story = {
	args: { score: 94 },
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getByTestId('score-number')).toHaveTextContent('94');
		await expect(canvas.getByText('Strong')).toBeVisible();
	}
};

export const Watch: Story = {
	args: { score: 84 },
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getByTestId('score-number')).toHaveTextContent('84');
		await expect(canvas.getByText('Watch')).toBeVisible();
	}
};

export const NeedsWork: Story = {
	args: { score: 73 },
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getByText('Needs work')).toBeVisible();
	}
};

export const HighRisk: Story = {
	args: { score: 64 },
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getByText('High risk')).toBeVisible();
	}
};

export const Failing: Story = {
	args: { score: 42 },
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getByTestId('score-number')).toHaveTextContent('42');
		await expect(canvas.getByText('Failing')).toBeVisible();
	}
};

export const NoScore: Story = {
	args: { score: null },
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getByTestId('score-number')).toHaveTextContent('—');
	}
};

export const LargeNumberOnly: Story = {
	args: { score: 94, size: 'lg', showPill: false },
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getByTestId('score-number')).toHaveTextContent('94');
		await expect(canvas.queryByText('Strong')).not.toBeInTheDocument();
	}
};
