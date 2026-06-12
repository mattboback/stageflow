<script lang="ts">
	import type { HTMLAttributes } from 'svelte/elements';

	import { cn } from '$lib/utils';

	interface Props extends HTMLAttributes<HTMLDivElement> {
		value?: number;
		max?: number;
		ariaLabel?: string;
		indicatorClass?: string;
		class?: string;
	}

	let {
		value = 0,
		max = 100,
		ariaLabel = 'Progress',
		indicatorClass,
		class: className,
		...rest
	}: Props = $props();

	const percentage = $derived(Math.min(Math.max(0, ((value || 0) / max) * 100), 100));
</script>

<div
	class={cn('bg-paper-deep relative h-1 w-full overflow-hidden', className)}
	role="progressbar"
	aria-label={ariaLabel}
	aria-valuemin={0}
	aria-valuemax={max}
	aria-valuenow={Math.min(Math.max(value || 0, 0), max)}
	{...rest}
>
	<div
		class={cn(
			'bg-accent h-full w-full flex-1 transition-transform duration-500 ease-in-out',
			indicatorClass
		)}
		style="transform: translateX(-{100 - percentage}%)"
	></div>
</div>
