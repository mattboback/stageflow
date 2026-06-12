<script lang="ts">
	import { Chip } from '$lib/components/ui';
	import { scanHistoryStore } from '$lib/stores/scan-history.svelte';
	import { formatTimestamp } from '$lib/utils/date';

	const items = $derived(scanHistoryStore.history);

	const statusTone = {
		complete: 'success',
		failed: 'danger',
		pending: 'warning'
	} as const;

	const statusLabel = {
		complete: 'Complete',
		failed: 'Failed',
		pending: 'Running'
	} as const;
</script>

{#if items.length > 0}
	<section aria-labelledby="recent-scans-heading">
		<div class="flex items-baseline justify-between gap-4">
			<h2 id="recent-scans-heading" class="font-display text-ink-strong text-xl font-semibold">
				Recent scans
			</h2>
			<button
				type="button"
				class="text-ink-faint hover:text-ink text-xs font-medium transition-colors"
				onclick={() => scanHistoryStore.clearHistory()}
			>
				Clear history
			</button>
		</div>

		<table class="border-line mt-4 w-full border-t text-sm">
			<thead>
				<tr class="border-line text-ink-faint border-b text-left font-mono text-[11px] uppercase">
					<th scope="col" class="py-2 pr-4 font-medium">Target</th>
					<th scope="col" class="hidden px-4 py-2 font-medium sm:table-cell">Started</th>
					<th scope="col" class="px-4 py-2 font-medium">Status</th>
					<th scope="col" class="py-2 pl-4">
						<span class="sr-only">Open scan</span>
					</th>
				</tr>
			</thead>
			<tbody>
				{#each items as item (item.id)}
					<tr class="border-line/70 border-b">
						<td class="max-w-0 truncate py-2.5 pr-4 font-mono text-xs" title={item.label}>
							{item.label}
						</td>
						<td class="text-ink-muted hidden px-4 py-2.5 text-xs whitespace-nowrap sm:table-cell">
							{formatTimestamp(item.createdAt) ?? '—'}
						</td>
						<td class="px-4 py-2.5">
							<Chip tone={statusTone[item.status]} size="xs" caps={true}>
								{statusLabel[item.status]}
							</Chip>
						</td>
						<td class="py-2.5 pl-4 text-right whitespace-nowrap">
							<a
								href={`/scan/${item.id}`}
								class="text-accent text-xs font-semibold hover:underline"
							>
								View
							</a>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</section>
{/if}
