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

<div class="playground-options-grid-container">
	<div class="playground-options-grid">
		<div>
			<Label class="mb-3 block text-sm font-bold">Screenshot Options</Label>
			<button
				type="button"
				onclick={() => onScreenshotChange(!screenshot)}
				class={selectableSurfaceClass(
					'group flex w-full items-center gap-3 rounded-2xl border p-4 text-left transition-[background-color,border-color,box-shadow,transform] duration-200',
					screenshot
				)}
			>
				<div
					class={cn(
						'flex h-10 w-10 shrink-0 items-center justify-center rounded-xl transition-colors',
						screenshot
							? 'bg-accent text-white'
							: 'bg-surface-muted text-ink-muted group-hover:bg-accent/10 group-hover:text-accent'
					)}
				>
					<Camera class="h-5 w-5" />
				</div>
				<div class="min-w-0 flex-1">
					<span class="text-ink block text-sm font-bold">Capture Screenshots</span>
					<span class="text-ink-muted text-xs">Visual evidence of issues</span>
				</div>
				<div
					class={cn(
						'flex h-5 w-5 shrink-0 items-center justify-center rounded-full border transition-[background-color,border-color,color] duration-200',
						screenshot ? 'border-accent bg-accent text-white' : 'border-line bg-surface'
					)}
				>
					{#if screenshot}
						{@render checkMarkSvg()}
					{/if}
				</div>
			</button>
			<p class="text-ink-muted mt-2 text-xs leading-relaxed">
				Artifacts, logs, and screenshots expire automatically within 24 hours of a completed scan.
			</p>
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
</div>
