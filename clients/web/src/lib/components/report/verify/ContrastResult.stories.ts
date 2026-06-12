import type { ContrastVerdict } from '$lib/stores/contrast-verdicts.svelte';
import type { Meta, StoryObj } from '@storybook/svelte';

import { expect, fn, within } from '@storybook/test';

import ContrastResult from './ContrastResult.svelte';

interface ContrastResultStoryArgs {
	fg: string;
	bg: string;
	ruleId: string;
	largeText: boolean;
	verdict: ContrastVerdict | null;
	onFgChange: (value: string) => void;
	onBgChange: (value: string) => void;
	onSwap: () => void;
	onLargeTextChange: (value: boolean) => void;
	onRecord: (verdict: 'pass' | 'fail', ratio: number | null) => void;
	onClear: () => void;
}

const meta = {
	title: 'Report/Verify/Contrast Result',
	component: ContrastResult,
	tags: ['autodocs'],
	args: {
		fg: '',
		bg: '',
		ruleId: 'color-contrast',
		largeText: false,
		verdict: null,
		onFgChange: fn(),
		onBgChange: fn(),
		onSwap: fn(),
		onLargeTextChange: fn(),
		onRecord: fn(),
		onClear: fn()
	},
	parameters: {
		layout: 'padded'
	}
} satisfies Meta<ContrastResultStoryArgs>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Empty: Story = {
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);

		await expect(canvas.getByText('—')).toBeVisible();
		await expect(canvas.getAllByText('No colors')).toHaveLength(2);
		await expect(canvas.getByRole('button', { name: 'Mark as pass' })).toBeEnabled();
	}
};

export const Computed: Story = {
	args: {
		fg: '#767676',
		bg: '#ffffff'
	},
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);

		// #767676 on white is the canonical 4.54:1 minimum-AA grey.
		await expect(canvas.getByText('4.54')).toBeVisible();
		await expect(canvas.getByText('Pass')).toBeVisible();
		await expect(canvas.getByText('Fail')).toBeVisible();
	}
};

export const VerdictRecorded: Story = {
	args: {
		fg: '#767676',
		bg: '#ffffff',
		verdict: {
			verdict: 'fail',
			fg: '#767676',
			bg: '#ffffff',
			ratio: 4.54,
			at: '2026-06-12T10:00:00.000Z'
		}
	},
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);

		await expect(canvas.getByText('Verified · fail')).toBeVisible();
		await expect(canvas.getByRole('button', { name: 'Clear verdict' })).toBeEnabled();
		await expect(canvas.queryByRole('button', { name: 'Mark as pass' })).not.toBeInTheDocument();
	}
};
