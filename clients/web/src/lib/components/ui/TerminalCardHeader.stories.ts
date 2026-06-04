import type { Meta, StoryObj } from '@storybook/svelte';

import { expect, within } from '@storybook/test';

import TerminalCardHeaderStoryHarness from './story-harnesses/TerminalCardHeaderStoryHarness.svelte';

const meta = {
	title: 'UI/TerminalCardHeader',
	component: TerminalCardHeaderStoryHarness,
	tags: ['autodocs'],
	args: {
		path: 'stageflow scan https://example.com',
		showBadges: true
	}
} satisfies Meta;

export default meta;
type Story = StoryObj<typeof meta>;

export const WithBadges: Story = {
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);

		await expect(canvas.getByTestId('terminal-header')).toBeVisible();
		await expect(canvas.getByText('stageflow scan https://example.com')).toBeVisible();
		await expect(canvas.getByText('live')).toBeVisible();
	}
};

export const PathOnly: Story = {
	args: {
		showBadges: false
	},
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);

		await expect(canvas.getByTestId('terminal-header')).toBeVisible();
		await expect(canvas.queryByText('live')).not.toBeInTheDocument();
	}
};
