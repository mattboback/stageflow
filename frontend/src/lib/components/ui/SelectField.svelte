<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLSelectAttributes } from 'svelte/elements';

	import { cn } from '$lib/utils';

	interface Props extends HTMLSelectAttributes {
		variant?: 'default' | 'prominent';
		class?: string;
		wrapperClass?: string;
		value?: string;
		children: Snippet;
	}

	let {
		variant = 'default',
		class: className,
		wrapperClass,
		value = $bindable(),
		children,
		...rest
	}: Props = $props();

	const selectClass = $derived(
		cn(
			'border-line text-ink w-full cursor-pointer appearance-none pr-10 transition-colors focus-visible:outline-none',
			variant === 'default'
				? 'bg-surface-muted focus-visible:ring-accent focus-visible:ring-offset-paper rounded-lg border px-3 py-2 text-sm focus-visible:ring-2 focus-visible:ring-offset-2'
				: 'bg-surface hover:border-ink/20 focus:border-accent rounded-xl border-2 px-4 py-3 text-sm font-medium focus:outline-none',
			className
		)
	);
</script>

<div class={cn('relative', wrapperClass)}>
	<select bind:value class={selectClass} {...rest}>
		{@render children()}
	</select>
	<div class="pointer-events-none absolute top-1/2 right-4 -translate-y-1/2">
		<svg class="text-ink-muted h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
		</svg>
	</div>
</div>
