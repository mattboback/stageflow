import type { Meta, StoryObj } from "@storybook/svelte";

import { expect, fn, userEvent, within } from "@storybook/test";

import PlaygroundUrlInputStoryHarness from "./story-harnesses/PlaygroundUrlInputStoryHarness.svelte";

interface PlaygroundUrlInputStoryArgs {
	initialUrls: string;
	onUrlsChange: (urls: string) => void;
	onNormalize: () => void;
}

const meta = {
	title: "Playground/URL Input",
	component: PlaygroundUrlInputStoryHarness,
	tags: ["autodocs"],
	args: {
		initialUrls: "",
		onUrlsChange: fn(),
		onNormalize: fn(),
	},
	parameters: {
		layout: "padded",
	},
} satisfies Meta<PlaygroundUrlInputStoryArgs>;

export default meta;
type Story = StoryObj<typeof meta>;

export const NormalizesUrlBatch: Story = {
	play: async ({ args, canvasElement }) => {
		const canvas = within(canvasElement);
		const input = canvas.getByRole("textbox", { name: "URLs to Scan" });

		await userEvent.type(input, "example.com{enter}https://example.com/about");
		await userEvent.tab();

		await expect(args.onUrlsChange).toHaveBeenCalled();
		await expect(args.onNormalize).toHaveBeenCalledTimes(1);
	},
};
