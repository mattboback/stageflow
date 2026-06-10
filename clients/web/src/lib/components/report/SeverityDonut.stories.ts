import type { Meta, StoryObj } from '@storybook/svelte';

import { expect, within } from '@storybook/test';

import SeverityDonut from './SeverityDonut.svelte';

const meta = {
	title: 'Report/SeverityDonut',
	component: SeverityDonut,
	tags: ['autodocs'],
	args: {
		counts: { critical: 6, serious: 10, moderate: 6, minor: 0, info: 0 }
	},
	parameters: {
		layout: 'padded'
	}
} satisfies Meta<typeof SeverityDonut>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Mixed: Story = {
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getByTestId('severity-donut')).toHaveTextContent('22');
	}
};

export const SingleSeverity: Story = {
	args: { counts: { critical: 0, serious: 0, moderate: 4, minor: 0, info: 0 } },
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getByTestId('severity-donut')).toHaveTextContent('4');
	}
};

export const Empty: Story = {
	args: { counts: { critical: 0, serious: 0, moderate: 0, minor: 0, info: 0 } },
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getByTestId('severity-donut')).toHaveTextContent('0');
	}
};

export const CustomCenter: Story = {
	args: {
		counts: { critical: 2, serious: 3, moderate: 0, minor: 1, info: 0 },
		centerValue: 4,
		centerLabel: 'distinct'
	},
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getByTestId('severity-donut')).toHaveTextContent('distinct');
	}
};
