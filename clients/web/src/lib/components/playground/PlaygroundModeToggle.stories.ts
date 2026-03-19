import type { Meta, StoryObj } from "@storybook/svelte";

import { expect, fn, userEvent, within } from "@storybook/test";

import PlaygroundModeToggleStoryHarness from "./story-harnesses/PlaygroundModeToggleStoryHarness.svelte";

interface PlaygroundModeToggleStoryArgs {
	initialMode: "url" | "zip";
	onModeChange: (mode: "url" | "zip") => void;
}

const meta = {
	title: "Playground/Mode Toggle",
	component: PlaygroundModeToggleStoryHarness,
	tags: ["autodocs"],
	args: {
		initialMode: "url",
		onModeChange: fn(),
	},
	argTypes: {
		initialMode: {
			control: "inline-radio",
			options: ["url", "zip"],
		},
	},
	parameters: {
		layout: "padded",
	},
} satisfies Meta<PlaygroundModeToggleStoryArgs>;

export default meta;
type Story = StoryObj<typeof meta>;

export const ToggleModes: Story = {
	play: async ({ args, canvasElement }) => {
		const canvas = within(canvasElement);
		const zipButton = canvas.getByRole("button", { name: /zip archive/i });
		const urlButton = canvas.getByRole("button", { name: /live urls/i });

		await userEvent.click(zipButton);
		await expect(args.onModeChange).toHaveBeenCalledWith("zip");
		await expect(canvas.getByTestId("selected-mode")).toHaveTextContent(
			"Selected mode: zip",
		);

		await userEvent.click(urlButton);
		await expect(args.onModeChange).toHaveBeenCalledWith("url");
		await expect(canvas.getByTestId("selected-mode")).toHaveTextContent(
			"Selected mode: url",
		);
	},
};
