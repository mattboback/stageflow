<script lang="ts">
import type { HTMLTextareaAttributes } from "svelte/elements";

import { cn } from "$lib/utils";

interface Props extends HTMLTextareaAttributes {
	class?: string;
	error?: boolean;
}

let { class: className, error = false, ...rest }: Props = $props();
</script>

<textarea
	class={cn(
		'border-line bg-surface-muted text-ink placeholder:text-ink-faint focus-visible:ring-accent focus-visible:ring-offset-paper flex min-h-[120px] w-full rounded-md border px-3 py-2 text-sm transition-colors focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50',
		error && 'border-red-500 focus-visible:ring-red-500',
		className
	)}
	aria-invalid={error}
	{...rest}
></textarea>
