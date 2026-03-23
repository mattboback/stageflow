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
			class: 'bg-red-500'
		},
		{
			id: 'serious',
			label: 'Serious',
			count: bySeverity.serious,
			class: 'bg-orange-500'
		},
		{
			id: 'moderate',
			label: 'Moderate',
			count: bySeverity.moderate,
			class: 'bg-amber-500'
		},
		{
			id: 'minor',
			label: 'Minor',
			count: bySeverity.minor,
			class: 'bg-blue-500'
		},
		{
			id: 'info',
			label: 'Info',
			count: bySeverity.info ?? 0,
			class: 'bg-purple-500'
		}
	]);
</script>

<div class="space-y-3">
	<div class="flex flex-wrap gap-2 text-xs">
		{#each severityItems as item (item.id)}
			{#if item.count > 0}
				<Chip>
					<span class={cn('mr-2 inline-block h-2 w-2 rounded-full', item.class)}></span>
					{item.count}
					{item.label.toLowerCase()}
				</Chip>
			{/if}
		{/each}
	</div>
	{#if total > 0}
		<div class="bg-surface-muted flex h-2 w-full overflow-hidden rounded-full">
			{#each severityItems as item (item.id)}
				{#if item.count > 0}
					<div class={item.class} style={`width: ${(item.count / total) * 100}%`}></div>
				{/if}
			{/each}
		</div>
	{/if}
</div>
