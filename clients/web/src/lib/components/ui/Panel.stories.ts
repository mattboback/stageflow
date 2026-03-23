import type { Meta, StoryObj } from '@storybook/svelte';

import { expect, within } from '@storybook/test';

import type { PanelPadding, PanelRounded, PanelVariant } from './panel';

import PanelStoryHarness from './story-harnesses/PanelStoryHarness.svelte';

interface PanelStoryArgs {
	variant: PanelVariant;
	padding: PanelPadding;
	rounded: PanelRounded;
	interactive: boolean;
	title: string;
	body: string;
}

const meta = {
	title: 'UI/Panel',
	component: PanelStoryHarness,
	tags: ['autodocs'],
	args: {
		variant: 'default',
		padding: 'md',
		rounded: 'lg',
		interactive: false,
		title: 'Panel title',
		body: 'Panel body content'
	},
	argTypes: {
		variant: {
			control: 'select',
			options: ['default', 'muted', 'ghost']
		},
		padding: {
			control: 'select',
			options: ['none', 'xs', 'sm', 'md', 'lg', 'xl']
		},
		rounded: {
			control: 'select',
			options: ['sm', 'md', 'lg', 'xl', '2xl', '3xl']
		}
	}
} satisfies Meta<PanelStoryArgs>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		const panel = canvas.getByTestId('panel');
		const title = canvas.getByTestId('panel-title');

		await expect(panel).toBeVisible();
		await expect(title).toHaveTextContent('Panel title');
	}
};

export const InteractiveMuted: Story = {
	args: {
		variant: 'muted',
		interactive: true
	},
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		const panel = canvas.getByTestId('panel');

		await expect(panel).toHaveClass('bg-surface-muted');
		await expect(panel).toHaveClass('hover-glow');
	}
};
