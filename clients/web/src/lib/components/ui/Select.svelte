<script lang="ts">
import type { Snippet } from "svelte";
import type { HTMLSelectAttributes } from "svelte/elements";

import { cn } from "$lib/utils";

import { type SelectSize, selectVariants } from "./select";

interface Props extends HTMLSelectAttributes {
	value?: string | number;
	uiSize?: SelectSize;
	class?: string;
	error?: boolean;
	children: Snippet;
}

let {
	value = $bindable(""),
	uiSize,
	class: className,
	error = false,
	children,
	...rest
}: Props = $props();
</script>

<select
	bind:value
	class={cn(
		'border-line',
		error && 'border-red-500 focus-visible:ring-red-500',
		selectVariants({ size: uiSize }),
		className
	)}
	aria-invalid={error}
	{...rest}
>
	{@render children()}
</select>
