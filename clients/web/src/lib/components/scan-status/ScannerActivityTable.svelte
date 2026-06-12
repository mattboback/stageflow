<script lang="ts">
	import { cn } from '$lib/utils';

	interface Props {
		expected: string[];
		completed: string[];
		remaining: string[];
	}

	let { expected, completed, remaining }: Props = $props();

	// Scanner names are derived from ids, not hardcoded labels, so new
	// scanners render correctly without a client release.
	function formatScannerName(scannerType: string): string {
		return scannerType
			.split('-')
			.map((part) => part.charAt(0).toUpperCase() + part.slice(1))
			.join(' ');
	}

	const rows = $derived.by(() => {
		const order = expected.length > 0 ? expected : [...completed, ...remaining];
		const done = new Set(completed);
		return order.map((id) => ({ id, done: done.has(id) }));
	});

	const finishedCount = $derived(rows.filter((r) => r.done).length);
</script>

<div>
	<div class="mb-2 flex items-baseline justify-between gap-3">
		<h4 class="text-ink text-sm font-semibold">Scanner activity</h4>
		<span class="text-ink-faint font-mono text-[11px] tabular-nums">
			{finishedCount} of {rows.length} finished
		</span>
	</div>
	<table class="border-line w-full border-t text-sm">
		<caption class="sr-only">Per-scanner progress for this run</caption>
		<tbody>
			{#each rows as row (row.id)}
				<tr class="border-line/70 border-b">
					<td class="flex h-9 items-center gap-2.5 pr-4">
						{#if row.done}
							<svg
								class="text-accent h-3.5 w-3.5 shrink-0"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
								aria-hidden="true"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="3"
									d="M5 13l4 4L19 7"
								/>
							</svg>
						{:else}
							<span
								class="bg-accent inline-block h-2 w-2 shrink-0 animate-pulse rounded-full"
								aria-hidden="true"
							></span>
						{/if}
						<span class={cn('font-medium', row.done ? 'text-ink' : 'text-ink-strong')}>
							{formatScannerName(row.id)}
						</span>
					</td>
					<td class="text-right">
						<span
							class={cn(
								'font-mono text-[11px] uppercase',
								row.done ? 'text-ink-faint' : 'text-accent-ink'
							)}
						>
							{row.done ? 'done' : 'running'}
						</span>
					</td>
				</tr>
			{/each}
		</tbody>
	</table>
</div>
