<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';

	import { cn } from '$lib/utils';

	import Container from './Container.svelte';

	interface Props extends HTMLAttributes<HTMLElement> {
		class?: string;
		containerClass?: string;
		disableContainer?: boolean;
		padding?: 'default' | 'page' | 'none';
		children: Snippet;
	}

	let {
		class: className,
		containerClass,
		disableContainer = false,
		padding = 'default',
		children,
		...rest
	}: Props = $props();

	const paddingClasses = $derived.by(() => {
		if (padding === 'page') return 'pt-28 pb-24';
		if (padding === 'none') return '';
		return 'pt-24 pb-20';
	});

	const baseSectionClasses = $derived.by(() => cn('min-h-screen bg-paper', paddingClasses));
</script>

<section class={cn(baseSectionClasses, className)} {...rest}>
	{#if disableContainer}
		{@render children()}
	{:else}
		<Container class={containerClass ?? ''}>
			{@render children()}
		</Container>
	{/if}
</section>
