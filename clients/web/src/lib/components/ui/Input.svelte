<script lang="ts">
	import type { HTMLInputAttributes } from 'svelte/elements';

	import { cn } from '$lib/utils';

	interface Props extends HTMLInputAttributes {
		value?: string | number;
		class?: string;
		error?: boolean;
	}

	let { value = $bindable(''), class: className, error = false, ...rest }: Props = $props();
</script>

<input
	bind:value
	class={cn(
		'bg-surface-muted ring-offset-paper placeholder:text-ink-faint flex h-10 w-full rounded-md border px-3 py-2 text-sm transition-colors file:border-0 file:bg-transparent file:text-sm file:font-medium disabled:cursor-not-allowed disabled:opacity-50',
		'border-line',
		'focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-none',
		error
			? 'border-red-500 focus-visible:ring-red-500'
			: 'focus-visible:ring-accent focus:border-accent',
		className
	)}
	aria-invalid={error}
	{...rest}
/>
