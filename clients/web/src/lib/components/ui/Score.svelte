<script lang="ts">
	import { cn } from '$lib/utils';

	import StatusPill, { type StatusPillTone } from './StatusPill.svelte';

	interface Props {
		score: number | null;
		size?: 'sm' | 'md' | 'lg';
		showPill?: boolean;
		class?: string;
	}

	let { score, size = 'md', showPill = true, class: className }: Props = $props();

	function bandFor(value: number): { tone: StatusPillTone; label: string } {
		if (value >= 90) return { tone: 'strong', label: 'Strong' };
		if (value >= 80) return { tone: 'watch', label: 'Watch' };
		if (value >= 70) return { tone: 'needs-work', label: 'Needs work' };
		if (value >= 60) return { tone: 'high-risk', label: 'High risk' };
		return { tone: 'failing', label: 'Failing' };
	}

	const rounded = $derived(score === null ? null : Math.round(score));
	const band = $derived(rounded === null ? null : bandFor(rounded));

	const numClass = $derived(
		size === 'lg' ? 'text-5xl sm:text-6xl' : size === 'sm' ? 'text-2xl' : 'text-4xl'
	);

	const denomClass = $derived(size === 'lg' ? 'text-xl' : size === 'sm' ? 'text-sm' : 'text-base');
</script>

<div class={cn('flex flex-col gap-1.5', className)} data-testid="score">
	<div class="flex items-baseline gap-1.5">
		<span
			class={cn(
				'text-ink-strong font-mono leading-none font-semibold tracking-tight tabular-nums',
				numClass
			)}
			data-testid="score-number"
		>
			{#if rounded === null}
				—
			{:else}
				{rounded}
			{/if}
		</span>
		<span class={cn('text-ink-faint font-mono leading-none tabular-nums', denomClass)}> /100 </span>
	</div>
	{#if showPill && band}
		<StatusPill tone={band.tone} label={band.label} size={size === 'sm' ? 'sm' : 'md'} />
	{/if}
</div>
