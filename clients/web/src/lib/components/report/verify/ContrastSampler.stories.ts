import type { Rect, SampleSlot, ViewBox } from '$lib/report';
import type { Meta, StoryObj } from '@storybook/svelte';

import { expect, fn, userEvent, within } from '@storybook/test';

import ContrastSampler from './ContrastSampler.svelte';

interface ContrastSamplerStoryArgs {
	imageUrl: string;
	pageWidth: number;
	pageHeight: number;
	viewBox: ViewBox;
	element: Rect | null;
	onPick: (slot: SampleSlot, hex: string) => void;
}

const sampleImage = `data:image/svg+xml;utf8,${encodeURIComponent(
	`<svg xmlns="http://www.w3.org/2000/svg" width="800" height="400">
		<defs>
			<linearGradient id="g" x1="0" y1="0" x2="1" y2="1">
				<stop offset="0" stop-color="#3a6ea5"/>
				<stop offset="1" stop-color="#c0d6e8"/>
			</linearGradient>
		</defs>
		<rect width="800" height="400" fill="url(#g)"/>
		<text x="60" y="210" font-family="sans-serif" font-size="40" fill="#ffffff">Hero headline over an image</text>
	</svg>`
)}`;

const meta = {
	title: 'Report/Verify/Contrast Sampler',
	component: ContrastSampler,
	tags: ['autodocs'],
	args: {
		imageUrl: sampleImage,
		pageWidth: 800,
		pageHeight: 400,
		viewBox: { x: 0, y: 60, width: 700, height: 300 },
		element: { x: 60, y: 178, width: 520, height: 44 },
		onPick: fn()
	},
	parameters: {
		layout: 'padded'
	}
} satisfies Meta<ContrastSamplerStoryArgs>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
	play: async ({ canvasElement }) => {
		const canvas = within(canvasElement);

		await expect(canvas.getByRole('button', { name: 'Text' })).toHaveAttribute(
			'aria-pressed',
			'true'
		);
		await expect(canvas.getByRole('button', { name: 'Background' })).toHaveAttribute(
			'aria-pressed',
			'false'
		);
		// The hex readout fills in once the screenshot is decoded and sampled.
		await expect(await canvas.findByText(/^#[0-9a-f]{6}$/)).toBeVisible();

		await userEvent.click(canvas.getByRole('button', { name: 'Background' }));
		await expect(canvas.getByRole('button', { name: 'Background' })).toHaveAttribute(
			'aria-pressed',
			'true'
		);
	}
};

export const KeyboardSampling: Story = {
	play: async ({ args, canvasElement }) => {
		const canvas = within(canvasElement);

		await canvas.findByText(/^#[0-9a-f]{6}$/);
		const application = canvas.getByRole('application');
		application.focus();
		await userEvent.keyboard('{ArrowRight}{Enter}');

		await expect(args.onPick).toHaveBeenCalledWith('fg', expect.stringMatching(/^#[0-9a-f]{6}$/));
	}
};
