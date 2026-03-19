import type { Meta, StoryObj } from "@storybook/svelte";

import { expect, within } from "@storybook/test";

import type { BadgeVariant } from "./badge";

import BadgeStoryHarness from "./story-harnesses/BadgeStoryHarness.svelte";

interface BadgeStoryArgs {
	variant: BadgeVariant;
	label: string;
}

const meta = {
	title: "UI/Badge",
	component: BadgeStoryHarness,
	tags: ["autodocs"],
	args: {
		variant: "default",
		label: "Live",
	},
	argTypes: {
		variant: {
			control: "select",
			options: [
				"default",
				"secondary",
				"destructive",
				"outline",
				"status",
				"terminal",
				"live",
			],
		},
	},
} satisfies Meta<BadgeStoryArgs>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		const badge = canvas.getByTestId("badge");

		await expect(badge).toBeVisible();
		await expect(badge).toHaveTextContent("Live");
	},
};

export const TerminalVariant: Story = {
	args: {
		variant: "terminal",
		label: "terminal",
	},
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		const badge = canvas.getByTestId("badge");

		await expect(badge).toHaveClass("font-mono");
		await expect(badge).toHaveClass("text-accent-ink");
	},
};
