<script lang="ts">
	import type { SeverityCounts } from '$lib/types/unified-report';

	import { Chip } from '$lib/components/ui';
	import { cn } from '$lib/utils';

	interface Props {
		bySeverity: SeverityCounts;
	}

	let { bySeverity }: Props = $props();

	const total = $derived(
		bySeverity.critical +
			bySeverity.serious +
			bySeverity.moderate +
			bySeverity.minor +
			(bySeverity.info ?? 0)
	);

	const severityItems = $derived([
		{
			id: 'critical',
			label: 'Critical',
			count: bySeverity.critical,
			class: 'bg-red-500 shadow-[0_0_10px_rgba(239,68,68,0.2)]'
		},
		{
			id: 'serious',
			label: 'Serious',
			count: bySeverity.serious,
			class: 'bg-orange-500 shadow-[0_0_10px_rgba(249,115,22,0.2)]'
		},
		{
			id: 'moderate',
			label: 'Moderate',
			count: bySeverity.moderate,
			class: 'bg-amber-500 shadow-[0_0_10px_rgba(245,158,11,0.2)]'
		},
		{
			id: 'minor',
			label: 'Minor',
			count: bySeverity.minor,
			class: 'bg-blue-500 shadow-[0_0_10px_rgba(59,130,246,0.2)]'
		},
		{
			id: 'info',
			label: 'Info',
			count: bySeverity.info ?? 0,
			class: 'bg-purple-500 shadow-[0_0_10px_rgba(168,85,247,0.2)]'
		}
	]);
</script>

<div class="space-y-3">
	<div class="flex flex-wrap gap-2 text-xs">
		{#each severityItems as item (item.id)}
			{#if item.count > 0}
				<Chip class="transition-all duration-300 hover:scale-105">
					<span class={cn('mr-2 inline-block h-2 w-2 rounded-full', item.class)}></span>
					<span class="mr-1 font-bold">{item.count}</span>
					<span class="text-ink-muted">{item.label.toLowerCase()}</span>
				</Chip>
			{/if}
		{/each}
	</div>
	{#if total > 0}
		<div
			class="bg-surface-muted border-line/30 flex h-2.5 w-full overflow-hidden rounded-full border shadow-inner"
		>
			{#each severityItems.filter((i) => i.count > 0) as item, index (item.id)}
				<div
					class={cn(item.class, 'animate-grow h-full first:rounded-l-full last:rounded-r-full')}
					style={`width: ${(item.count / total) * 100}%; animation-delay: ${index * 80}ms;`}
				></div>
			{/each}
		</div>
	{/if}
</div>

<style>
	@keyframes slide-grow {
		from {
			width: 0%;
			opacity: 0;
		}
	}

	.animate-grow {
		animation: slide-grow 1.2s cubic-bezier(0.16, 1, 0.3, 1) both;
	}
</style>
