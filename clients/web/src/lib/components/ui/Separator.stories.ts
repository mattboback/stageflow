import type { Meta, StoryObj } from '@storybook/svelte';

import { expect, within } from '@storybook/test';

import SeparatorStoryHarness from './story-harnesses/SeparatorStoryHarness.svelte';

const meta = {
	title: 'UI/Separator',
	component: SeparatorStoryHarness,
	tags: ['autodocs'],
	args: {
		orientation: 'horizontal'
	},
	argTypes: {
		orientation: {
			control: 'select',
			options: ['horizontal', 'vertical']
		}
	}
} satisfies Meta;

export default meta;
type Story = StoryObj<typeof meta>;

export const Horizontal: Story = {
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		const separator = canvas.getByRole('separator');

		await expect(separator).toHaveAttribute('aria-orientation', 'horizontal');
		await expect(separator).toHaveClass('h-[1px]');
	}
};

export const Vertical: Story = {
	args: {
		orientation: 'vertical'
	},
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		const separator = canvas.getByRole('separator');

		await expect(separator).toHaveAttribute('aria-orientation', 'vertical');
		await expect(separator).toHaveClass('w-[1px]');
	}
};
