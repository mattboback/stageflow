import type { Meta, StoryObj } from "@storybook/svelte";

import { expect, fn, userEvent, within } from "@storybook/test";

import SelectFieldStoryHarness from "./story-harnesses/SelectFieldStoryHarness.svelte";

interface SelectFieldStoryArgs {
	label: string;
	value: string;
	variant: "default" | "prominent";
	onChange: (value: string) => void;
}

const meta = {
	title: "UI/SelectField",
	component: SelectFieldStoryHarness,
	tags: ["autodocs"],
	args: {
		label: "Scanner",
		value: "axe",
		variant: "default",
		onChange: fn(),
	},
	argTypes: {
		variant: {
			control: "select",
			options: ["default", "prominent"],
		},
	},
} satisfies Meta<SelectFieldStoryArgs>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
	play: async ({ args, canvasElement }) => {
		const canvas = within(canvasElement);
		const select = canvas.getByRole("combobox", { name: "Scanner" });

		await userEvent.selectOptions(select, "lighthouse");
		await expect(select).toHaveValue("lighthouse");
		await expect(canvas.getByTestId("select-field-value")).toHaveTextContent(
			"Selected: lighthouse",
		);
		await expect(args.onChange).toHaveBeenCalledWith("lighthouse");
	},
};

export const Prominent: Story = {
	args: {
		variant: "prominent",
	},
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		const select = canvas.getByRole("combobox", { name: "Scanner" });

		await expect(select).toHaveClass("border-2");
	},
};
