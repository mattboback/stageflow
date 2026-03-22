<script lang="ts">
import type { Snippet } from "svelte";
import type { HTMLAttributes } from "svelte/elements";

import { cn } from "$lib/utils";

interface Props extends HTMLAttributes<HTMLDivElement> {
	class?: string;
	children: Snippet;
}

let { class: className, children, ...rest }: Props = $props();
</script>

<div class={cn('container-width', className)} {...rest}>
	{@render children()}
</div>
