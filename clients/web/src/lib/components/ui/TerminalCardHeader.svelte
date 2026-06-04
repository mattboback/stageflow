<script lang="ts">
	import type { HTMLAttributes } from 'svelte/elements';

	import { cn } from '$lib/utils';

	import type { BadgeVariant } from './badge';

	import Badge from './Badge.svelte';

	interface TerminalBadge {
		label: string;
		variant?: BadgeVariant;
	}

	interface Props extends HTMLAttributes<HTMLDivElement> {
		path: string;
		badges?: TerminalBadge[];
		class?: string;
	}

	let { path, badges = [], class: className, ...rest }: Props = $props();
</script>

<div
	class={cn(
		'border-line bg-surface-muted/30 flex items-center justify-between border-b px-4 py-2',
		className
	)}
	{...rest}
>
	<div class="flex items-center gap-2">
		<div class="flex gap-1.5">
			<div class="h-2.5 w-2.5 rounded-full bg-red-400"></div>
			<div class="h-2.5 w-2.5 rounded-full bg-amber-400"></div>
			<div class="h-2.5 w-2.5 rounded-full bg-emerald-400"></div>
		</div>
		<span class="text-ink-muted font-mono text-[10px]">{path}</span>
	</div>

	{#if badges.length > 0}
		<div class="flex items-center gap-2">
			{#each badges as badge (badge.label)}
				<Badge variant={badge.variant ?? 'terminal'}>{badge.label}</Badge>
			{/each}
		</div>
	{/if}
</div>
