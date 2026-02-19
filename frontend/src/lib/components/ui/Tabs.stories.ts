import type { Meta, StoryObj } from '@storybook/svelte';

import { expect, fn, userEvent, within } from '@storybook/test';

import TabsStoryHarness from './story-harnesses/TabsStoryHarness.svelte';

interface TabsStoryArgs {
	defaultTab: string;
	onValueChange: (tabId: string) => void;
}

const meta = {
	title: 'UI/Tabs',
	component: TabsStoryHarness,
	tags: ['autodocs', 'skip-test'],
	args: {
		defaultTab: 'overview',
		onValueChange: fn()
	}
} satisfies Meta<TabsStoryArgs>;

export default meta;
type Story = StoryObj<typeof meta>;

export const SwitchTabs: Story = {
	play: async ({ args, canvasElement }) => {
		const canvas = within(canvasElement);

		await expect(canvas.getByTestId('active-panel')).toHaveTextContent('Active panel: overview');

		await userEvent.click(canvas.getByRole('button', { name: 'Security' }));
		await expect(canvas.getByTestId('active-panel')).toHaveTextContent('Active panel: security');
		await expect(canvas.getByTestId('last-selection')).toHaveTextContent('Last selection: security');
		await expect(args.onValueChange).toHaveBeenCalledWith('security');
	}
};

export const StartsOnAccessibility: Story = {
	args: {
		defaultTab: 'accessibility'
	},
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);

		await expect(canvas.getByTestId('active-panel')).toHaveTextContent('Active panel: accessibility');
	}
};
