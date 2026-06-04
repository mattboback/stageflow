import type { Meta, StoryObj } from '@storybook/svelte';

import { expect, within } from '@storybook/test';

import ContainerStoryHarness from './story-harnesses/ContainerStoryHarness.svelte';

const meta = {
	title: 'UI/Container',
	component: ContainerStoryHarness,
	tags: ['autodocs'],
	parameters: {
		layout: 'fullscreen'
	}
} satisfies Meta;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		const container = canvas.getByTestId('container');

		await expect(container).toHaveClass('container-width');
		await expect(canvas.getByTestId('container-child')).toBeVisible();
	}
};
