import type { Meta, StoryObj } from "@storybook/svelte";

import { expect, fn, userEvent, within } from "@storybook/test";

import InputStoryHarness from "./story-harnesses/InputStoryHarness.svelte";

interface InputStoryArgs {
	label: string;
	placeholder: string;
	value: string;
	error: boolean;
	disabled: boolean;
	onInput: (value: string) => void;
}

const meta = {
	title: "UI/Input",
	component: InputStoryHarness,
	tags: ["autodocs"],
	args: {
		label: "Email",
		placeholder: "name@example.com",
		value: "",
		error: false,
		disabled: false,
		onInput: fn(),
	},
} satisfies Meta<InputStoryArgs>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Typing: Story = {
	play: async ({ args, canvasElement }) => {
		const canvas = within(canvasElement);
		const input = canvas.getByRole("textbox", { name: "Email" });

		await userEvent.type(input, "owner@stageflow.dev");
		await expect(input).toHaveValue("owner@stageflow.dev");
		await expect(canvas.getByTestId("input-value")).toHaveTextContent(
			"Value: owner@stageflow.dev",
		);
		await expect(args.onInput).toHaveBeenLastCalledWith("owner@stageflow.dev");
	},
};

export const ErrorState: Story = {
	args: {
		error: true,
		value: "invalid",
	},
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		const input = canvas.getByRole("textbox", { name: "Email" });

		await expect(input).toHaveClass("border-red-500");
	},
};
