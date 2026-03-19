<script lang="ts">
import type { HTMLAttributes } from "svelte/elements";

import { cn } from "$lib/utils";

interface Props extends HTMLAttributes<HTMLDivElement> {
	orientation?: "horizontal" | "vertical";
	class?: string;
}

const {
	orientation = "horizontal",
	class: className,
	...rest
}: Props = $props();
</script>

<div
	role="separator"
	aria-orientation={orientation}
	class={cn(
		'bg-line shrink-0',
		orientation === 'horizontal' ? 'h-[1px] w-full' : 'h-full w-[1px]',
		className
	)}
	{...rest}
></div>
