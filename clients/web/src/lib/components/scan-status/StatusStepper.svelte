<script lang="ts">
	import { cn } from '$lib/utils';

	interface Props {
		currentStatus: string;
	}

	const SCAN_STEPS = [
		{ id: 'pending', label: 'Queued' },
		{ id: 'extracting', label: 'Preparing' },
		{ id: 'scanning', label: 'Scanning' },
		{ id: 'completing', label: 'Finalizing' },
		{ id: 'complete', label: 'Ready' }
	];

	let { currentStatus }: Props = $props();

	const currentIndex = $derived(SCAN_STEPS.findIndex((s) => s.id === currentStatus));
	const effectiveIndex = $derived(
		currentIndex === -1 ? (currentStatus === 'failed' ? -1 : 0) : currentIndex
	);
	const progressWidth = $derived(
		effectiveIndex === -1 ? 0 : (effectiveIndex / (SCAN_STEPS.length - 1)) * 100
	);
</script>

<div aria-label="Scan pipeline progress">
	<div class="bg-line h-px w-full">
		<div
			class="bg-accent h-px transition-[width] duration-500 ease-in-out"
			style="width: {progressWidth}%"
		></div>
	</div>
	<div class="mt-2.5 flex justify-between">
		{#each SCAN_STEPS as step, index (step.id)}
			{@const isCurrent = index === effectiveIndex}
			{@const isFuture = index > effectiveIndex}
			<span
				class={cn(
					'font-mono text-[10px] tracking-[0.08em] uppercase',
					isCurrent ? 'text-accent-ink font-bold' : isFuture ? 'text-ink-faint' : 'text-ink-muted'
				)}
				aria-current={isCurrent ? 'step' : undefined}
			>
				{step.label}
			</span>
		{/each}
	</div>
</div>
