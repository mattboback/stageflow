import type { Meta, StoryObj } from '@storybook/svelte';

import { expect, fn, userEvent, within } from '@storybook/test';

import ModalStoryHarness from './story-harnesses/ModalStoryHarness.svelte';

interface ModalStoryArgs {
	title: string;
	openLabel: string;
	closeLabel: string;
	closeOnBackdrop: boolean;
	closeOnEscape: boolean;
	trapFocus: boolean;
	onClose: () => void;
}

const meta = {
	title: 'UI/Modal',
	component: ModalStoryHarness,
	tags: ['autodocs'],
	args: {
		title: 'Scanner details',
		openLabel: 'Open modal',
		closeLabel: 'Close modal',
		closeOnBackdrop: true,
		closeOnEscape: true,
		trapFocus: true,
		onClose: fn()
	}
} satisfies Meta<ModalStoryArgs>;

export default meta;
type Story = StoryObj<typeof meta>;

export const KeyboardDismiss: Story = {
	play: async ({ args, canvasElement }) => {
		const canvas = within(canvasElement);
		const openButton = canvas.getByRole('button', { name: 'Open modal' });

		openButton.focus();
		await userEvent.click(openButton);

		const dialog = await canvas.findByRole('dialog');
		const closeButton = canvas.getByRole('button', { name: 'Close modal' });
		const helpLink = canvas.getByRole('link', { name: 'Help docs' });
		const lastButton = canvas.getByRole('button', { name: 'Last action' });

		await expect(dialog).toBeVisible();
		await expect(closeButton).toHaveFocus();

		await userEvent.tab();
		await expect(helpLink).toHaveFocus();

		await userEvent.tab();
		await expect(lastButton).toHaveFocus();

		await userEvent.keyboard('{Escape}');
		await expect(args.onClose).toHaveBeenCalledTimes(1);
		await expect(canvas.queryByRole('dialog')).not.toBeInTheDocument();
		await expect(openButton).toHaveFocus();
	}
};

export const BackdropDismiss: Story = {
	play: async ({ args, canvasElement }) => {
		const canvas = within(canvasElement);
		const openButton = canvas.getByRole('button', { name: 'Open modal' });

		await userEvent.click(openButton);
		const dialog = await canvas.findByRole('dialog');

		await expect(dialog).toBeVisible();
		await userEvent.click(dialog);
		await expect(args.onClose).toHaveBeenCalledTimes(1);
		await expect(canvas.queryByRole('dialog')).not.toBeInTheDocument();
	}
};
