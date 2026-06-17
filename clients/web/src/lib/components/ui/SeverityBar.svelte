<script lang="ts">
	import { cn } from '$lib/utils';

	export interface SeveritySegment {
		severity: 'critical' | 'serious' | 'moderate' | 'minor' | 'info';
		count: number;
	}

	interface Props {
		counts: Partial<Record<SeveritySegment['severity'], number>>;
		showLabels?: boolean;
		height?: 'sm' | 'md';
		class?: string;
	}

	let { counts, showLabels = false, height = 'md', class: className }: Props = $props();

	const SEGMENTS: SeveritySegment['severity'][] = [
		'critical',
		'serious',
		'moderate',
		'minor',
		'info'
	];

	const textClass: Record<SeveritySegment['severity'], string> = {
		critical: 'text-red-700',
		serious: 'text-orange-700',
		moderate: 'text-amber-700',
		minor: 'text-blue-700',
		info: 'text-purple-700'
	};

	const total = $derived(SEGMENTS.reduce((sum, s) => sum + (counts[s] ?? 0), 0));

	function pct(severity: SeveritySegment['severity']): number {
		if (total === 0) return 0;
		return ((counts[severity] ?? 0) / total) * 100;
	}
</script>

<div class={cn('w-full', className)} data-testid="severity-bar">
	<div
		class={cn(
			'flex w-full overflow-hidden rounded-full bg-surface-muted',
			height === 'sm' ? 'h-1.5' : 'h-2.5'
		)}
		role="img"
		aria-label="Severity distribution: {SEGMENTS.map((s) => `${counts[s] ?? 0} ${s}`).join(', ')}"
	>
		{#each SEGMENTS as severity (severity)}
			{@const value = pct(severity)}
			{#if value > 0}
				<span
						class="h-full"
						style="width: {value}%; background-color: var(--color-severity-{severity})"
					></span>
			{/if}
		{/each}
	</div>
	{#if showLabels}
		<div class="mt-1.5 flex flex-wrap gap-x-3 gap-y-1 font-mono text-[11px] tabular-nums">
			{#each SEGMENTS as severity (severity)}
				{@const value = counts[severity] ?? 0}
				{#if value > 0}
					<span class={cn('inline-flex items-center gap-1.5', textClass[severity])}>
						<span
								class="h-1.5 w-1.5 rounded-full"
								style="background-color: var(--color-severity-{severity})"
							></span>
						<span class="font-semibold">{value}</span>
						<span class="text-ink-faint capitalize">{severity}</span>
					</span>
				{/if}
			{/each}
		</div>
	{/if}
</div>
