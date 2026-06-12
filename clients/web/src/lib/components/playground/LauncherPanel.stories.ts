import type { Meta, StoryObj } from '@storybook/svelte';

import { expect, fn, userEvent, within } from '@storybook/test';

import LauncherPanel from './LauncherPanel.svelte';

interface LauncherPanelStoryArgs {
	mode: 'url' | 'zip';
	urls: string;
	file: File | null;
	preset: 'coverage' | 'quick' | 'custom';
	hasPresets: boolean;
	enabledScannerCount: number;
	canSubmit: boolean;
	isSubmitting: boolean;
	missingRequirements: string[];
	advancedOpen: boolean;
	onModeChange: (mode: 'url' | 'zip') => void;
	onUrlsChange: (urls: string) => void;
	onNormalize: () => void;
	onFileChange: (file: File | null) => void;
	onFileError: (message: string) => void;
	onPresetChange: (preset: 'coverage' | 'quick' | 'custom') => void;
	onToggleAdvanced: () => void;
}

const meta = {
	title: 'Playground/Launcher Panel',
	component: LauncherPanel,
	tags: ['autodocs'],
	args: {
		mode: 'url' as const,
		urls: '',
		file: null,
		preset: 'coverage' as const,
		hasPresets: true,
		enabledScannerCount: 7,
		canSubmit: false,
		isSubmitting: false,
		missingRequirements: ['Add at least one URL or switch to ZIP mode.'],
		advancedOpen: false,
		onModeChange: fn(),
		onUrlsChange: fn(),
		onNormalize: fn(),
		onFileChange: fn(),
		onFileError: fn(),
		onPresetChange: fn(),
		onToggleAdvanced: fn()
	},
	parameters: {
		layout: 'padded'
	}
} satisfies Meta<LauncherPanelStoryArgs>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
	play: async ({ args, canvasElement }) => {
		const canvas = within(canvasElement);

		await expect(canvas.getByRole('textbox', { name: 'URLs to Scan' })).toBeInTheDocument();
		await expect(canvas.getByRole('button', { name: /start scan/i })).toBeDisabled();
		await expect(canvas.getByRole('button', { name: 'Coverage' })).toHaveAttribute(
			'aria-pressed',
			'true'
		);
		await expect(
			canvas.getByText('Add at least one URL or switch to ZIP mode.')
		).toBeVisible();

		await userEvent.click(canvas.getByRole('button', { name: /advanced options/i }));
		await expect(args.onToggleAdvanced).toHaveBeenCalledTimes(1);

		await userEvent.click(canvas.getByRole('button', { name: 'Quick' }));
		await expect(args.onPresetChange).toHaveBeenCalledWith('quick');
	}
};

export const ReadyToSubmit: Story = {
	args: {
		urls: 'https://example.com',
		canSubmit: true,
		missingRequirements: []
	},
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);

		await expect(canvas.getByRole('button', { name: /start scan/i })).toBeEnabled();
	}
};
