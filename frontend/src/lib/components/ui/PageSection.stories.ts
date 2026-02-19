import type { Meta, StoryObj } from '@storybook/svelte';

import { expect, within } from '@storybook/test';

import PageSectionStoryHarness from './story-harnesses/PageSectionStoryHarness.svelte';

interface PageSectionStoryArgs {
	padding: 'default' | 'page' | 'none';
	disableContainer: boolean;
	containerClass?: string;
}

const meta = {
	title: 'UI/PageSection',
	component: PageSectionStoryHarness,
	tags: ['autodocs'],
	parameters: {
		layout: 'fullscreen'
	},
	args: {
		padding: 'default',
		disableContainer: false
	},
	argTypes: {
		padding: {
			control: 'select',
			options: ['default', 'page', 'none']
		}
	}
} satisfies Meta<PageSectionStoryArgs>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		const section = canvas.getByTestId('page-section');

		await expect(section).toHaveClass('pt-24');
		await expect(section).toHaveClass('pb-20');
		await expect(canvas.getByTestId('page-section-child')).toBeVisible();
		await expect(canvasElement.querySelector('.container-width')).not.toBeNull();
	}
};

export const WithoutContainer: Story = {
	args: {
		disableContainer: true,
		padding: 'page'
	},
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		const section = canvas.getByTestId('page-section');

		await expect(section).toHaveClass('pt-28');
		await expect(section).toHaveClass('pb-24');
		await expect(canvasElement.querySelector('.container-width')).toBeNull();
	}
};
