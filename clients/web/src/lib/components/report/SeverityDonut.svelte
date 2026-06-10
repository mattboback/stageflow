<script lang="ts">
	import type { SeverityCounts } from '$lib/types/unified-report';

	import { SEVERITY_LEVELS, getSeverityStrokeColor } from '$lib/report';
	import { cn } from '$lib/utils';

	interface Props {
		counts: Partial<SeverityCounts> | null | undefined;
		/** Number shown in the donut center; defaults to the sum of all counts. */
		centerValue?: number | null;
		centerLabel?: string;
		/** Rendered size in px (the SVG scales from an 80×80 viewBox). */
		size?: number;
		class?: string;
	}

	let {
		counts,
		centerValue = null,
		centerLabel = 'issues',
		size = 112,
		class: className
	}: Props = $props();

	const RADIUS = 30;
	const STROKE = 9;
	const CIRCUMFERENCE = 2 * Math.PI * RADIUS;
	// Visual gap between adjacent segments, in circumference units.
	const SEGMENT_GAP = 1.5;

	const total = $derived(SEVERITY_LEVELS.reduce((sum, level) => sum + (counts?.[level] ?? 0), 0));
	const displayValue = $derived(centerValue ?? total);

	const segments = $derived.by(() => {
		if (total === 0) return [];
		const nonZero = SEVERITY_LEVELS.filter((level) => (counts?.[level] ?? 0) > 0);
		const gap = nonZero.length > 1 ? SEGMENT_GAP : 0;
		let offset = 0;
		return nonZero.map((level) => {
			const count = counts?.[level] ?? 0;
			const arc = (count / total) * CIRCUMFERENCE;
			const segment = {
				level,
				count,
				length: Math.max(arc - gap, 0.5),
				offset
			};
			offset += arc;
			return segment;
		});
	});
</script>

<div
	class={cn('relative inline-flex shrink-0 items-center justify-center', className)}
	data-testid="severity-donut"
	role="img"
	aria-label="Severity distribution: {SEVERITY_LEVELS.map(
		(level) => `${counts?.[level] ?? 0} ${level}`
	).join(', ')}"
>
	<svg viewBox="0 0 80 80" width={size} height={size} class="-rotate-90" aria-hidden="true">
		<circle cx="40" cy="40" r={RADIUS} fill="none" class="stroke-slate-100" stroke-width={STROKE} />
		{#each segments as segment, idx (segment.level)}
			<circle
				cx="40"
				cy="40"
				r={RADIUS}
				fill="none"
				stroke={getSeverityStrokeColor(segment.level)}
				stroke-width={STROKE}
				stroke-linecap={segments.length > 1 ? 'butt' : 'round'}
				class="animate-donut-segment"
				stroke-dasharray={`${segment.length} ${CIRCUMFERENCE - segment.length}`}
				style={`stroke-dashoffset: ${-segment.offset}; --segment-length: ${segment.length}; --segment-gap: ${CIRCUMFERENCE - segment.length}; animation-delay: ${idx * 120}ms;`}
			/>
		{/each}
	</svg>
	<div class="pointer-events-none absolute inset-0 flex flex-col items-center justify-center">
		<span class="text-ink-strong font-mono text-2xl leading-none font-bold tabular-nums">
			{displayValue.toLocaleString()}
		</span>
		{#if centerLabel}
			<span class="text-ink-faint mt-0.5 text-[9px] font-semibold tracking-wider uppercase">
				{centerLabel}
			</span>
		{/if}
	</div>
</div>
