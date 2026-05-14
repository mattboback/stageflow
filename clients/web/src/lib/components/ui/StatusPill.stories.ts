import type { Meta, StoryObj } from '@storybook/svelte';

import { expect, within } from '@storybook/test';

import StatusPill from './StatusPill.svelte';

const meta = {
	title: 'UI/StatusPill',
	component: StatusPill,
	tags: ['autodocs'],
	args: {
		tone: 'strong',
		size: 'md'
	},
	argTypes: {
		tone: {
			control: 'select',
			options: ['strong', 'watch', 'needs-work', 'high-risk', 'failing', 'neutral']
		},
		size: { control: 'select', options: ['sm', 'md'] },
		label: { control: 'text' }
	},
	parameters: {
		layout: 'centered'
	}
} satisfies Meta<typeof StatusPill>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Strong: Story = {
	args: { tone: 'strong' },
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getByText('Strong')).toBeVisible();
		await expect(canvas.getByRole('status')).toHaveAttribute('data-tone', 'strong');
	}
};

export const Watch: Story = {
	args: { tone: 'watch' },
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getByText('Watch')).toBeVisible();
	}
};

export const NeedsWork: Story = {
	args: { tone: 'needs-work' },
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getByText('Needs work')).toBeVisible();
	}
};

export const HighRisk: Story = {
	args: { tone: 'high-risk' },
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getByText('High risk')).toBeVisible();
	}
};

export const Failing: Story = {
	args: { tone: 'failing' },
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getByText('Failing')).toBeVisible();
	}
};

export const Neutral: Story = {
	args: { tone: 'neutral' },
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getByText('Unknown')).toBeVisible();
	}
};

export const CustomLabel: Story = {
	args: { tone: 'strong', label: 'Healthy' },
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getByText('Healthy')).toBeVisible();
	}
};

export const Small: Story = {
	args: { tone: 'failing', size: 'sm' },
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getByRole('status')).toHaveClass('text-[10px]');
	}
};
