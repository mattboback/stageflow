<script lang="ts">
	import type { ScanResult, ScanStatus } from '$lib/types/scan';

	import { Badge, Progress } from '$lib/components/ui';
	import { cn } from '$lib/utils';
	import { Clock } from 'lucide-svelte';

	interface Props {
		id: string;
		status: ScanStatus;
		elapsed: number;
		result?: ScanResult | null;
	}

	const { id, status, elapsed, result }: Props = $props();

	function formatTime(s: number) {
		return `${Math.floor(s / 60)}m ${s % 60}s`;
	}

	const isProcessing = $derived(!['complete', 'failed', 'error'].includes(status));
	const progress = $derived(result?.progress);
	const progressPercent = $derived(progress?.percentage ?? 0);
</script>

<div class="mb-8 space-y-4">
	<div
		class="border-line bg-surface flex flex-col items-start justify-between gap-6 rounded-2xl border p-6 shadow-sm md:flex-row md:items-center"
	>
		<div>
			<div class="mb-2 flex items-center gap-3">
				<h1 class="text-2xl font-bold text-slate-900">Scan Status</h1>
				<Badge
					class={cn(
						'px-3 py-1',
						status === 'complete'
							? 'bg-emerald-100 text-emerald-700'
							: status === 'failed' || status === 'error'
								? 'bg-red-100 text-red-700'
								: 'bg-blue-100 text-blue-600'
					)}
				>
					{status.toUpperCase()}
				</Badge>
			</div>
			<p class="font-mono text-xs text-slate-500">ID: {id}</p>
		</div>
		<div
			class="border-line bg-surface-muted flex items-center gap-4 rounded-lg border px-4 py-2 text-sm text-slate-600"
		>
			<Clock class="text-accent h-4 w-4" />
			<span class="font-medium">Duration:</span>
			<span class="font-mono">{formatTime(elapsed)}</span>
		</div>
	</div>

	<!-- Progress Bar -->
	{#if isProcessing}
		<div class="border-line bg-surface rounded-xl border p-4">
			<div class="mb-2 flex items-center justify-between text-sm">
				<span class="font-medium text-slate-700">
					{#if progress}
						{#if progress.current_page > 0}
							Scanning page {progress.current_page} of {progress.total_pages}
						{:else}
							Starting scan (1 of {progress.total_pages})
						{/if}
					{:else}
						Initializing scan...
					{/if}
				</span>
				<span class="font-mono text-slate-500">{progressPercent}%</span>
			</div>
			<Progress value={progress ? progressPercent : status === 'pending' ? 0 : 10} class="h-3" />
		</div>
	{/if}
</div>
