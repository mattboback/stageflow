<script lang="ts">
import type { Snippet } from "svelte";
import type { HTMLLabelAttributes } from "svelte/elements";

import { cn } from "$lib/utils";

interface Props extends HTMLLabelAttributes {
	class?: string;
	required?: boolean;
	children: Snippet;
}

let {
	class: className,
	required = false,
	children,
	...rest
}: Props = $props();
</script>

<label class={cn('text-ink-muted mb-1 block text-sm font-medium', className)} {...rest}>
	{@render children()}
	{#if required}
		<span class="ml-1 text-red-500" aria-hidden="true">*</span>
		<span class="sr-only">required</span>
	{/if}
</label>
