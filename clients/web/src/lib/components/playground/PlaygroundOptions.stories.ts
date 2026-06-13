import type { BrowserEngine } from '$lib/api/client';
import type { Meta, StoryObj } from '@storybook/svelte';

import { expect, fn, userEvent, within } from '@storybook/test';

import PlaygroundOptionsStoryHarness from './story-harnesses/PlaygroundOptionsStoryHarness.svelte';

interface PlaygroundOptionsStoryArgs {
	initialScreenshot: boolean;
	initialHighlightStyle: 'solid' | 'dashed';
	initialEngine: BrowserEngine;
	onScreenshotChange: (value: boolean) => void;
	onHighlightStyleChange: (value: 'solid' | 'dashed') => void;
	onEngineChange: (value: BrowserEngine) => void;
}

const meta = {
	title: 'Playground/Options',
	component: PlaygroundOptionsStoryHarness,
	tags: ['autodocs'],
	args: {
		initialScreenshot: true,
		initialHighlightStyle: 'solid',
		initialEngine: 'chromium',
		onScreenshotChange: fn(),
		onHighlightStyleChange: fn(),
		onEngineChange: fn()
	},
	parameters: {
		layout: 'padded'
	}
} satisfies Meta<PlaygroundOptionsStoryArgs>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const SelectsFirefoxEngine: Story = {
	play: async ({ args, canvasElement }) => {
		const canvas = within(canvasElement);
		const engineSelect = canvas.getByLabelText('Browser Engine');

		await userEvent.selectOptions(engineSelect, 'firefox');

		await expect(args.onEngineChange).toHaveBeenCalledWith('firefox');
	}
};
