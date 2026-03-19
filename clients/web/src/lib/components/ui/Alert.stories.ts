import type { Meta, StoryObj } from "@storybook/svelte";

import { expect, within } from "@storybook/test";

import type { AlertVariant } from "./alert";

import AlertStoryHarness from "./story-harnesses/AlertStoryHarness.svelte";

interface AlertStoryArgs {
	variant: AlertVariant;
	message: string;
}

const meta = {
	title: "UI/Alert",
	component: AlertStoryHarness,
	tags: ["autodocs"],
	args: {
		variant: "info",
		message: "Scan started successfully.",
	},
	argTypes: {
		variant: {
			control: "select",
			options: ["info", "success", "warning", "error"],
		},
	},
} satisfies Meta<AlertStoryArgs>;

export default meta;
type Story = StoryObj<typeof meta>;

export const InfoStatus: Story = {
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		const alert = canvas.getByRole("status");

		await expect(alert).toBeVisible();
		await expect(alert).toHaveAttribute("aria-live", "polite");
	},
};

export const ErrorAlert: Story = {
	args: {
		variant: "error",
		message: "Accessibility scan failed for one page.",
	},
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		const alert = canvas.getByRole("alert");

		await expect(alert).toBeVisible();
		await expect(alert).toHaveAttribute("aria-live", "assertive");
	},
};
