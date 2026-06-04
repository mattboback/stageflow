import type { Meta, StoryObj } from '@storybook/svelte';

import { expect, within } from '@storybook/test';

import ChipStoryHarness from './story-harnesses/ChipStoryHarness.svelte';

const meta = {
	title: 'UI/Chip',
	component: ChipStoryHarness,
	tags: ['autodocs'],
	args: {
		label: 'Axe',
		tone: 'default',
		size: 'sm',
		caps: false,
		interactive: false,
		as: 'span'
	},
	argTypes: {
		tone: {
			control: 'select',
			options: ['default', 'muted', 'active', 'success', 'warning', 'danger', 'ghost']
		},
		size: {
			control: 'select',
			options: ['xs', 'sm', 'md']
		},
		as: {
			control: 'select',
			options: ['span', 'button', 'a']
		}
	}
} satisfies Meta;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		const chip = canvas.getByTestId('chip');

		await expect(chip).toBeVisible();
		await expect(chip).toHaveTextContent('Axe');
	}
};

export const Interactive: Story = {
	args: {
		as: 'button',
		interactive: true,
		label: 'Run scanner',
		tone: 'active'
	},
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		const chip = canvas.getByRole('button', { name: 'Run scanner' });

		await expect(chip).toBeVisible();
		await expect(chip).toHaveClass('cursor-pointer');
	}
};
