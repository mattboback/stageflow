<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLSelectAttributes } from 'svelte/elements';

	import { cn } from '$lib/utils';

	interface Props extends HTMLSelectAttributes {
		variant?: 'default' | 'prominent';
		class?: string;
		wrapperClass?: string;
		value?: string;
		error?: boolean;
		children: Snippet;
	}

	let {
		variant = 'default',
		class: className,
		wrapperClass,
		value = $bindable(),
		error = false,
		children,
		...rest
	}: Props = $props();

	const selectClass = $derived(
		cn(
			'border-line text-ink w-full cursor-pointer appearance-none pr-10 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-accent focus-visible:ring-offset-paper',
			variant === 'default'
				? 'bg-surface-muted rounded-lg border px-3 py-2 text-sm'
				: 'bg-surface hover:border-ink/20 rounded-xl border-2 px-4 py-3 text-sm font-medium',
			error && 'border-red-500 focus-visible:ring-red-500',
			className
		)
	);
</script>

<div class={cn('relative', wrapperClass)}>
	<select bind:value class={selectClass} aria-invalid={error} {...rest}>
		{@render children()}
	</select>
	<div class="pointer-events-none absolute top-1/2 right-4 -translate-y-1/2">
		<svg
			class="text-ink-muted h-4 w-4"
			fill="none"
			stroke="currentColor"
			viewBox="0 0 24 24"
			aria-hidden="true"
		>
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
		</svg>
	</div>
</div>
