import type { Meta, StoryObj } from '@storybook/svelte';

import { expect, within } from '@storybook/test';

import LandingDataPreview from './LandingDataPreview.svelte';

const meta = {
	title: 'Landing/Data Preview',
	component: LandingDataPreview,
	tags: ['autodocs'],
	parameters: {
		layout: 'padded'
	}
} satisfies Meta<LandingDataPreview>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);

		await expect(canvas.getByRole('table')).toBeVisible();
		await expect(canvas.getByText('Axe')).toBeVisible();
		await expect(canvas.getByText('AI Navigator')).toBeVisible();
		await expect(canvas.getByTestId('severity-bar')).toBeVisible();
	}
};
