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

	let { value = 0, max = 100, ariaLabel = 'Progress', indicatorClass, class: className, ...rest }: Props = $props();

	const percentage = $derived(Math.min(Math.max(0, ((value || 0) / max) * 100), 100));
</script>

<div
	class={cn('bg-surface-muted relative h-2 w-full overflow-hidden rounded-full', className)}
	role="progressbar"
	aria-label={ariaLabel}
	aria-valuemin={0}
	aria-valuemax={max}
	aria-valuenow={Math.min(Math.max(value || 0, 0), max)}
	{...rest}
>
	<div
		class={cn(
			'bg-accent h-full w-full flex-1 transition-all duration-500 ease-in-out',
			indicatorClass
		)}
		style="transform: translateX(-{100 - percentage}%)"
	></div>
</div>
