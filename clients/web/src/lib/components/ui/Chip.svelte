<script lang="ts">
import type { Snippet } from "svelte";
import type { HTMLAttributes } from "svelte/elements";

import { cn } from "$lib/utils";

import {
	type ChipCaps,
	type ChipInteractive,
	type ChipSize,
	type ChipTone,
	chipVariants,
} from "./chip";

type ChipTag = "span" | "button" | "a";

interface Props extends HTMLAttributes<HTMLElement> {
	as?: ChipTag;
	tone?: ChipTone;
	size?: ChipSize;
	caps?: ChipCaps;
	interactive?: ChipInteractive;
	class?: string;
	href?: string;
	target?: string;
	rel?: string;
	type?: "button" | "submit" | "reset";
	children: Snippet;
}

const {
	as = "span",
	tone,
	size,
	caps,
	interactive,
	class: className,
	href,
	target,
	rel,
	type,
	children,
	...rest
}: Props = $props();
</script>

<svelte:element
	this={as}
	class={cn(chipVariants({ tone, size, caps, interactive }), className)}
	href={as === 'a' ? href : undefined}
	target={as === 'a' ? target : undefined}
	rel={as === 'a' ? rel : undefined}
	type={as === 'button' ? type : undefined}
	{...rest}
>
	{@render children()}
</svelte:element>
