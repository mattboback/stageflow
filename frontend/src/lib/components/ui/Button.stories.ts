import type { Meta, StoryObj } from '@storybook/svelte';

import { expect, fn, userEvent, within } from '@storybook/test';

import type { ButtonSize, ButtonVariant } from './button';

import ButtonStoryHarness from './story-harnesses/ButtonStoryHarness.svelte';

interface ButtonStoryArgs {
	label: string;
	variant: ButtonVariant;
	size: ButtonSize;
	disabled: boolean;
	onClick: () => void;
}

const meta = {
	title: 'UI/Button',
	component: ButtonStoryHarness,
	tags: ['autodocs'],
	args: {
		label: 'Run scan',
		variant: 'default',
		size: 'default',
		disabled: false,
		onClick: fn()
	},
	argTypes: {
		variant: {
			control: 'select',
			options: ['default', 'destructive', 'outline', 'secondary', 'ghost', 'link', 'glow']
		},
		size: {
			control: 'select',
			options: ['default', 'sm', 'lg', 'icon']
		}
	}
} satisfies Meta<ButtonStoryArgs>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Primary: Story = {
	play: async ({ args, canvasElement }) => {
		const canvas = within(canvasElement);
		const button = canvas.getByRole('button', { name: 'Run scan' });

		await expect(button).toBeVisible();
		await expect(button).toBeEnabled();
		await userEvent.click(button);
		await expect(args.onClick).toHaveBeenCalledTimes(1);
	}
};

export const Disabled: Story = {
	args: {
		disabled: true,
		label: 'Scan unavailable'
	},
	play: async ({ args, canvasElement }) => {
		const canvas = within(canvasElement);
		const button = canvas.getByRole('button', { name: 'Scan unavailable' });

		await expect(button).toBeDisabled();
		await expect(args.onClick).not.toHaveBeenCalled();
	}
};
