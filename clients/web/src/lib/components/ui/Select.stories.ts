import type { Meta, StoryObj } from '@storybook/svelte';

import { expect, fn, userEvent, within } from '@storybook/test';

import type { SelectSize } from './select';

import SelectStoryHarness from './story-harnesses/SelectStoryHarness.svelte';

interface SelectStoryArgs {
	label: string;
	value: string;
	uiSize: SelectSize;
	error: boolean;
	disabled: boolean;
	onChange: (value: string) => void;
}

const meta = {
	title: 'UI/Select',
	component: SelectStoryHarness,
	tags: ['autodocs'],
	args: {
		label: 'Scanner',
		value: 'axe',
		uiSize: 'md',
		error: false,
		disabled: false,
		onChange: fn()
	},
	argTypes: {
		uiSize: {
			control: 'select',
			options: ['sm', 'md']
		}
	}
} satisfies Meta<SelectStoryArgs>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
	play: async ({ args, canvasElement }) => {
		const canvas = within(canvasElement);
		const select = canvas.getByRole('combobox', { name: 'Scanner' });

		await expect(select).toBeVisible();
		await userEvent.selectOptions(select, 'seo');

		await expect(select).toHaveValue('seo');
		await expect(canvas.getByTestId('selected-value')).toHaveTextContent('Selected: seo');
		await expect(args.onChange).toHaveBeenCalledWith('seo');
	}
};

export const ErrorState: Story = {
	args: {
		error: true
	},
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		const select = canvas.getByRole('combobox', { name: 'Scanner' });

		await expect(select).toHaveClass('border-red-500');
	}
};
