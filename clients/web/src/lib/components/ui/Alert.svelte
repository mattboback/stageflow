<script lang="ts">
import type { Snippet } from "svelte";
import type { HTMLAttributes } from "svelte/elements";

import { cn } from "$lib/utils";

import { type AlertVariant, alertVariants } from "./alert";

interface Props extends HTMLAttributes<HTMLDivElement> {
	variant?: AlertVariant;
	class?: string;
	children: Snippet;
}

const {
	variant = "info",
	class: className,
	children,
	...rest
}: Props = $props();

const role = $derived(
	variant === "error" || variant === "warning" ? "alert" : "status",
);
const ariaLive = $derived<"assertive" | "polite">(
	variant === "error" ? "assertive" : "polite",
);
</script>

<div {role} aria-live={ariaLive} class={cn(alertVariants({ variant }), className)} {...rest}>
	{@render children()}
</div>
