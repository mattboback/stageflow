import type { Meta, StoryObj } from '@storybook/svelte';

import { expect, userEvent, within } from '@storybook/test';

import Textarea from './Textarea.svelte';

interface TextareaStoryArgs {
	placeholder: string;
	disabled: boolean;
	error: boolean;
	'aria-label': string;
	value: string;
}

const meta = {
	title: 'UI/Textarea',
	component: Textarea,
	tags: ['autodocs'],
	args: {
		placeholder: 'Describe the scan context',
		disabled: false,
		error: false,
		'aria-label': 'Scan notes',
		value: ''
	}
} satisfies Meta<TextareaStoryArgs>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		const textarea = canvas.getByRole('textbox', { name: 'Scan notes' });

		await expect(textarea).toBeVisible();
		await userEvent.type(textarea, 'Run crawler with default configuration.');
		await expect(textarea).toHaveValue('Run crawler with default configuration.');
	}
};

export const ErrorState: Story = {
	args: {
		error: true
	},
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		const textarea = canvas.getByRole('textbox', { name: 'Scan notes' });

		await expect(textarea).toHaveAttribute('aria-invalid', 'true');
		await expect(textarea).toHaveClass('border-red-500');
	}
};

export const Disabled: Story = {
	args: {
		disabled: true
	},
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		const textarea = canvas.getByRole('textbox', { name: 'Scan notes' });

		await expect(textarea).toBeDisabled();
	}
};
