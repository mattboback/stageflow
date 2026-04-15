<script lang="ts">
	import type { ScanResult, ScanStatus } from '$lib/types/scan';

	import { Badge } from '$lib/components/ui';
	import { cn } from '$lib/utils';
	import { Clock, FileSearch, Link2, ShieldCheck } from 'lucide-svelte';

	interface Props {
		id: string;
		status: ScanStatus;
		elapsed: number;
		result?: ScanResult | null;
	}

	let { id, status, elapsed, result }: Props = $props();

	function formatTime(s: number) {
		return `${Math.floor(s / 60)}m ${s % 60}s`;
	}

	const isProcessing = $derived(!['complete', 'failed', 'error'].includes(status));
	const progress = $derived(result?.progress);
	const completedScanners = $derived(result?.completed_scanners?.length ?? 0);
	const expectedScanners = $derived(
		result?.expected_scanners?.length ??
			Math.max(
				result?.completed_scanners?.length ?? 0,
				(result?.completed_scanners?.length ?? 0) + (result?.remaining_scanners?.length ?? 0)
			)
	);
	const pagesComplete = $derived(progress?.current_page ?? 0);
	const pagesTotal = $derived(progress?.total_pages ?? 0);
	const summaryText = $derived.by(() => {
		if (status === 'complete') {
			return 'Scan complete. Open the unified report to review the highest-priority findings first.';
		}
		if (status === 'failed' || status === 'error') {
			return 'Scan stopped before completion. Check the failure details and logs to understand what blocked it.';
		}
		if (progress && progress.total_pages > 0) {
			return `Currently scanning page ${progress.current_page} of ${progress.total_pages}.`;
		}
		if (status === 'pending') {
			return 'The job is queued and waiting for a worker to begin.';
		}
		if (status === 'extracting') {
			return 'Preparing the target and scan environment before scanner execution.';
		}
		if (status === 'completing') {
			return 'Finalizing artifacts and aggregating the report.';
		}
		return 'Preparing live scan progress and scanner activity.';
	});
</script>

<div class="mb-8 space-y-4">
	<div
		class="border-line bg-surface flex flex-col items-start justify-between gap-5 rounded-3xl border p-6 shadow-sm"
	>
		<div class="flex w-full flex-col gap-5 lg:flex-row lg:items-start lg:justify-between">
			<div class="min-w-0">
				<div class="mb-2 flex items-center gap-3">
					<h1 class="text-ink text-2xl font-bold tracking-tight">Scan Status</h1>
					<Badge
						class={cn(
							'px-3 py-1',
							status === 'complete'
								? 'bg-emerald-100 text-emerald-700'
								: status === 'failed' || status === 'error'
									? 'bg-red-100 text-red-700'
									: 'bg-blue-100 text-blue-700'
						)}
					>
						{status.toUpperCase()}
					</Badge>
				</div>
				<p class="text-ink mt-2 max-w-3xl text-sm leading-relaxed">{summaryText}</p>
				<p class="text-ink-faint mt-3 font-mono text-[11px]">Job {id}</p>
			</div>

			<div
				class="border-line bg-surface-muted text-ink-muted flex items-center gap-3 rounded-2xl border px-4 py-3 text-sm"
			>
				<Clock class="text-accent h-4 w-4" />
				<div>
					<p class="text-[11px] font-semibold tracking-[0.14em] uppercase">Elapsed</p>
					<p class="text-ink font-mono">{formatTime(elapsed)}</p>
				</div>
			</div>
		</div>

		<div class="grid w-full gap-3 sm:grid-cols-3">
			<div class="bg-surface-muted/65 rounded-2xl px-4 py-3">
				<div
					class="text-ink-muted flex items-center gap-2 text-[11px] font-semibold tracking-[0.14em] uppercase"
				>
					<FileSearch class="h-3.5 w-3.5" />
					Pages
				</div>
				<p class="mt-2 text-sm font-semibold">
					{#if pagesTotal > 0}
						{pagesComplete} of {pagesTotal}
					{:else}
						Waiting for page totals
					{/if}
				</p>
			</div>
			<div class="bg-surface-muted/65 rounded-2xl px-4 py-3">
				<div
					class="text-ink-muted flex items-center gap-2 text-[11px] font-semibold tracking-[0.14em] uppercase"
				>
					<ShieldCheck class="h-3.5 w-3.5" />
					Scanners
				</div>
				<p class="mt-2 text-sm font-semibold">
					{completedScanners} complete{#if expectedScanners > 0}<span class="text-ink-muted">
							of {expectedScanners}</span
						>{/if}
				</p>
			</div>
			<div class="bg-surface-muted/65 rounded-2xl px-4 py-3">
				<div
					class="text-ink-muted flex items-center gap-2 text-[11px] font-semibold tracking-[0.14em] uppercase"
				>
					<Link2 class="h-3.5 w-3.5" />
					Next step
				</div>
				<p class="mt-2 text-sm font-semibold">
					{#if isProcessing}
						Watch live progress
					{:else if status === 'complete'}
						Open report
					{:else}
						Review failure details
					{/if}
				</p>
			</div>
		</div>
	</div>
</div>
