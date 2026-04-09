<script lang="ts">
	import { cn } from '$lib/utils';
	import { CheckCircle2, Loader2 } from 'lucide-svelte';

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

<div class="relative mx-auto w-full max-w-3xl px-4">
	<div class="bg-surface-muted absolute top-5 left-0 h-0.5 w-full -translate-y-1/2">
		<div
			class="bg-accent h-full transition-[width] duration-500 ease-in-out"
			style="width: {progressWidth}%"
		></div>
	</div>

	<div class="relative flex w-full justify-between">
		{#each SCAN_STEPS as step, index (step.id)}
			{@const isCompleted = index < effectiveIndex}
			{@const isCurrent = index === effectiveIndex}
			{@const isFuture = index > effectiveIndex}
			<div class="flex flex-col items-center gap-3">
				<div
					class={cn(
						'relative z-10 flex h-10 w-10 items-center justify-center rounded-full border-2 transition-[background-color,border-color,color,transform,box-shadow] duration-300',
						isCompleted
							? 'border-accent bg-accent text-white shadow-md'
							: isCurrent
								? 'border-accent bg-surface text-accent-ink scale-110'
								: 'border-line bg-surface text-ink-faint'
					)}
				>
					{#if isCompleted}
						<CheckCircle2 class="h-6 w-6" />
					{:else if isCurrent}
						<Loader2 class="h-5 w-5 animate-spin" />
					{:else}
						<div class="h-2.5 w-2.5 rounded-full bg-current"></div>
					{/if}
				</div>
				<span
					class={cn(
						'text-xs font-semibold tracking-wider uppercase',
						isFuture ? 'text-ink-faint' : 'text-ink'
					)}
				>
					{step.label}
				</span>
			</div>
		{/each}
	</div>
</div>
