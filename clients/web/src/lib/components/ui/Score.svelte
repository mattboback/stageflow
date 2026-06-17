<script lang="ts">
	import { cn } from '$lib/utils';

	import StatusPill, { type StatusPillTone } from './StatusPill.svelte';

	interface Props {
		score: number | null;
		/** Letter grade (e.g. "B", "A+") shown as the hero value when provided. */
		grade?: string | null;
		size?: 'sm' | 'md' | 'lg';
		showPill?: boolean;
		class?: string;
	}

	let { score, grade = null, size = 'md', showPill = true, class: className }: Props = $props();

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
		size === 'lg' ? 'text-4xl sm:text-5xl font-bold' : size === 'sm' ? 'text-xl' : 'text-3xl'
	);

	const denomClass = $derived(
		size === 'lg' ? 'text-lg' : size === 'sm' ? 'text-[11px]' : 'text-sm'
	);

	// SVG Circular progress configurations
	const radius = $derived(size === 'sm' ? 12 : size === 'md' ? 18 : 24);
	const strokeWidth = $derived(size === 'sm' ? 2 : 3);
	const viewBoxSize = $derived(radius * 2 + strokeWidth * 2 + 2);
	const center = $derived(viewBoxSize / 2);
	const circumference = $derived(2 * Math.PI * radius);
	const strokeDashoffset = $derived(
		rounded === null
			? circumference
			: circumference - (Math.min(100, Math.max(0, rounded)) / 100) * circumference
	);

	const strokeColorClass = $derived.by(() => {
		if (!band) return 'stroke-line';
		switch (band.tone) {
			case 'strong':
				return 'stroke-emerald-500';
			case 'watch':
				return 'stroke-blue-500';
			case 'needs-work':
				return 'stroke-amber-500';
			case 'high-risk':
				return 'stroke-orange-500';
			case 'failing':
				return 'stroke-red-500';
			default:
				return 'stroke-line';
		}
	});

	const gradeColorClass = $derived.by(() => {
		if (!band) return 'text-ink-strong';
		switch (band.tone) {
			case 'strong':
				return 'text-emerald-600';
			case 'watch':
				return 'text-blue-600';
			case 'needs-work':
				return 'text-amber-600';
			case 'high-risk':
				return 'text-orange-600';
			case 'failing':
				return 'text-red-600';
			default:
				return 'text-ink-strong';
		}
	});
</script>

<div class={cn('flex items-center gap-3.5', className)} data-testid="score">
	<div class="relative flex shrink-0 items-center justify-center">
		<svg
			width={viewBoxSize}
			height={viewBoxSize}
			viewBox="0 0 {viewBoxSize} {viewBoxSize}"
			class="rotate-[-90deg] transition-all duration-700 ease-out"
			aria-hidden="true"
		>
			<!-- Track circle -->
			<circle
				cx={center}
				cy={center}
				r={radius}
				fill="none"
				stroke="currentColor"
				class="stroke-surface-muted"
				stroke-width={strokeWidth}
			/>
			<!-- Active circle -->
			{#if rounded !== null}
				<circle
					cx={center}
					cy={center}
					r={radius}
					fill="none"
					class={cn('animate-score-ring', strokeColorClass)}
					stroke-width={strokeWidth}
					stroke-dasharray={circumference}
					stroke-dashoffset={strokeDashoffset}
					stroke-linecap="round"
					style={`--circumference: ${circumference}; --target-offset: ${strokeDashoffset};`}
				/>
			{/if}
		</svg>
	</div>

	<div class="flex flex-col gap-1">
		<div class="flex items-baseline gap-0.5">
			{#if grade && rounded !== null}
				<span
					class={cn('leading-none font-black tracking-tight', gradeColorClass, numClass)}
					data-testid="score-grade"
				>
					{grade}
				</span>
				<span class={cn('text-ink-muted ml-2 font-mono leading-none tabular-nums', denomClass)}>
					<span data-testid="score-number">{rounded}</span><span class="text-ink-faint">/100</span>
				</span>
			{:else}
				<span
					class={cn('text-ink-strong font-mono leading-none tracking-tight tabular-nums', numClass)}
					data-testid="score-number"
				>
					{#if rounded === null}
						—
					{:else}
						{rounded}
					{/if}
				</span>
				<span class={cn('text-ink-faint font-mono leading-none tabular-nums', denomClass)}>
					/100
				</span>
			{/if}
		</div>
		{#if showPill && band}
			<StatusPill
				tone={band.tone}
				label={band.label}
				size={size === 'sm' ? 'sm' : 'md'}
				class="origin-left scale-[0.95]"
			/>
		{/if}
	</div>
</div>
