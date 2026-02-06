<script lang="ts">
	import type { LighthouseCategorySummary } from '$lib/types/unified-report';

	import { Panel } from '$lib/components/ui';

	interface Props {
		categories: LighthouseCategorySummary[];
	}

	const { categories }: Props = $props();

	function getScoreColor(score: number): string {
		const pct = score * 100;
		if (pct >= 90) return 'bg-emerald-500';
		if (pct >= 50) return 'bg-amber-500';
		return 'bg-red-500';
	}

	function getScoreTextColor(score: number): string {
		const pct = score * 100;
		if (pct >= 90) return 'text-emerald-700';
		if (pct >= 50) return 'text-amber-700';
		return 'text-red-700';
	}
</script>

<div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
	{#each categories as category (category.id)}
		<Panel class="shadow-xs" padding="sm" rounded="2xl">
			<p class="text-ink-muted text-xs font-semibold tracking-wide uppercase">
				{category.title}
			</p>
			<div class="mt-2 flex items-center justify-between">
				<div class={`text-2xl font-bold ${getScoreTextColor(category.avgScore)}`}>
					{Math.round(category.avgScore * 100)}
				</div>
				<div class="bg-surface-muted h-2 w-24 overflow-hidden rounded-full">
					<div
						class={`h-full ${getScoreColor(category.avgScore)}`}
						style={`width: ${Math.round(category.avgScore * 100)}%`}
					></div>
				</div>
			</div>
		</Panel>
	{/each}
</div>
