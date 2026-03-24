<script lang="ts">
	import { Label, SelectField } from '$lib/components/ui';
	import { cn } from '$lib/utils';
	import { Camera } from 'lucide-svelte';

	interface Props {
		screenshot: boolean;
		highlightStyle: 'solid' | 'dashed';
		onScreenshotChange: (value: boolean) => void;
		onHighlightStyleChange: (value: 'solid' | 'dashed') => void;
	}

	let { screenshot, highlightStyle, onScreenshotChange, onHighlightStyleChange }: Props = $props();

	function selectableSurfaceClass(base: string, isSelected: boolean) {
		return cn(
			base,
			isSelected
				? 'border-accent/60 bg-accent/5 shadow-[0_0_0_1px_rgba(13,92,99,0.18)]'
				: 'border-line bg-surface hover:border-accent/30 hover:bg-surface-muted'
		);
	}
</script>

{#snippet checkMarkSvg()}
	<svg class="h-3 w-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
		<path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
	</svg>
{/snippet}

<div class="grid gap-4 sm:grid-cols-2">
	<div>
		<Label class="mb-3 block text-sm font-bold">Screenshot Options</Label>
		<button
			type="button"
			onclick={() => onScreenshotChange(!screenshot)}
			class={selectableSurfaceClass(
				'group flex w-full items-center gap-3 rounded-2xl border p-4 text-left transition-all duration-200',
				screenshot
			)}
		>
			<div
				class={cn(
					'flex h-10 w-10 items-center justify-center rounded-xl transition-colors',
					screenshot
						? 'bg-accent text-white'
						: 'bg-surface-muted text-ink-muted group-hover:bg-accent/10 group-hover:text-accent'
				)}
			>
				<Camera class="h-5 w-5" />
			</div>
			<div class="flex-1">
				<span class="text-ink block text-sm font-bold">Capture Screenshots</span>
				<span class="text-ink-muted text-xs">Visual evidence of issues</span>
			</div>
			<div
				class={cn(
					'flex h-5 w-5 items-center justify-center rounded-full border transition-all',
					screenshot ? 'border-accent bg-accent text-white' : 'border-line bg-surface'
				)}
			>
				{#if screenshot}
					{@render checkMarkSvg()}
				{/if}
			</div>
		</button>
	</div>

	<div>
		<Label for="highlight-style" class="mb-3 block text-sm font-bold">Highlight Style</Label>
		<SelectField
			id="highlight-style"
			variant="prominent"
			value={highlightStyle}
			onchange={(e) => onHighlightStyleChange(e.currentTarget.value as 'solid' | 'dashed')}
			class="py-4"
		>
			<option value="solid">Solid Border</option>
			<option value="dashed">Dashed Border</option>
		</SelectField>
	</div>
</div>
