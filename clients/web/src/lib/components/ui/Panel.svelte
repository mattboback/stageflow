<script lang="ts">
import type { Snippet } from "svelte";
import type { HTMLAttributes } from "svelte/elements";

import { cn } from "$lib/utils";

import {
	type PanelInteractive,
	type PanelPadding,
	type PanelRounded,
	type PanelVariant,
	panelVariants,
} from "./panel";

interface Props extends HTMLAttributes<HTMLDivElement> {
	variant?: PanelVariant;
	padding?: PanelPadding;
	rounded?: PanelRounded;
	interactive?: PanelInteractive;
	class?: string;
	children: Snippet;
}

const {
	variant,
	padding,
	rounded,
	interactive,
	class: className,
	children,
	...rest
}: Props = $props();
</script>

<div class={cn(panelVariants({ variant, padding, rounded, interactive }), className)} {...rest}>
	{@render children()}
</div>
