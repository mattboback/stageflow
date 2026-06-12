<script lang="ts">
	import type { ScanResult, ScanStatus } from '$lib/types/scan';
	import type { StatusPillTone } from '$lib/components/ui/StatusPill.svelte';

	import { StatusPill } from '$lib/components/ui';

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
	const pillTone = $derived<StatusPillTone>(
		status === 'complete' ? 'strong' : isProcessing ? 'watch' : 'failing'
	);
	const pillLabel = $derived(
		status === 'complete' ? 'Complete' : isProcessing ? 'Running' : 'Failed'
	);
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

<div>
	<div class="flex flex-wrap items-center gap-x-3 gap-y-2">
		<h1 class="text-ink-strong text-sm font-semibold">Scan status</h1>
		<span class="text-ink-faint font-mono text-[11px]" title={`Job ${id}`}>
			job {id.length > 12 ? `${id.slice(0, 8)}…` : id}
		</span>
		<span class="text-ink-faint hidden sm:inline" aria-hidden="true">·</span>
		<span class="text-ink-muted font-mono text-[11px] tabular-nums"
			>{formatTime(elapsed)} elapsed</span
		>
		<StatusPill tone={pillTone} label={pillLabel} size="sm" class="ml-auto" />
	</div>
	<p class="text-ink-muted mt-1.5 text-sm leading-relaxed">{summaryText}</p>
</div>
