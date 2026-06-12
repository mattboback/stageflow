<script lang="ts">
	import type { ScanResult } from '$lib/types/scan';

	import { Progress } from '$lib/components/ui';
	import { cn } from '$lib/utils';
	import { ChevronDown, TerminalSquare } from 'lucide-svelte';

	import ScannerActivityTable from './ScannerActivityTable.svelte';
	import ScanTerminal from './ScanTerminal.svelte';

	interface Props {
		result: ScanResult | null;
		logs: string[];
	}

	let { result, logs }: Props = $props();

	let logsOpen = $state(false);

	function getStageInfo(
		state?: string,
		progress?: { current_page: number; total_pages: number }
	): { stage: string; description: string } {
		const normalizedState = (state || '').toUpperCase();

		switch (normalizedState) {
			case 'PENDING':
				return { stage: 'Queued', description: 'Waiting for available worker…' };
			case 'EXTRACTING':
				return { stage: 'Extracting', description: 'Processing uploaded content…' };
			case 'READY_TO_SCAN':
				return { stage: 'Preparing', description: 'Analyzing site structure…' };
			case 'SCANNING':
				if (progress) {
					return {
						stage: 'Scanning',
						description: `Auditing page ${progress.current_page} of ${progress.total_pages}`
					};
				}
				return { stage: 'Scanning', description: 'Running accessibility checks…' };
			case 'COMPLETING':
				return { stage: 'Finalizing', description: 'Generating reports…' };
			default:
				return { stage: 'Processing', description: 'Initializing scan environment…' };
		}
	}

	function estimateTimeRemaining(progress?: { current_page: number; total_pages: number }): string {
		if (!progress || progress.current_page === 0) {
			return 'Calculating…';
		}

		const avgSecondsPerPage = 8;
		const remainingPages = progress.total_pages - progress.current_page;
		const remainingSeconds = remainingPages * avgSecondsPerPage;

		if (remainingSeconds < 60) {
			return `~${remainingSeconds}s remaining`;
		}
		const minutes = Math.ceil(remainingSeconds / 60);
		return `~${minutes} min remaining`;
	}

	function formatScannerName(scannerType: string): string {
		return scannerType
			.split('-')
			.map((part) => part.charAt(0).toUpperCase() + part.slice(1))
			.join(' ');
	}

	const stageInfo = $derived(getStageInfo(result?.state, result?.progress));
	const estimatedTime = $derived(
		result?.remaining_scanners?.length === 1 && result.remaining_scanners[0] === 'lighthouse'
			? 'Waiting on Lighthouse'
			: estimateTimeRemaining(result?.progress)
	);
	const percentage = $derived(result?.progress?.percentage ?? 0);
	const completedScanners = $derived(result?.completed_scanners ?? []);
	const remainingScanners = $derived(result?.remaining_scanners ?? []);
	const currentAction = $derived.by(() => {
		if (remainingScanners.length > 0) {
			return `Running ${remainingScanners
				.slice(0, 2)
				.map((scannerType) => formatScannerName(scannerType))
				.join(
					', '
				)}${remainingScanners.length > 2 ? ` and ${remainingScanners.length - 2} more` : ''}.`;
		}
		if (completedScanners.length > 0) {
			return 'Waiting for the next scanner update.';
		}
		return 'Initializing the first scanner and preparing live progress.';
	});
	const progressLabel = $derived.by(() => {
		if (result?.progress?.total_pages && result.progress.total_pages > 0) {
			return `Page ${result.progress.current_page} of ${result.progress.total_pages}`;
		}
		if (statusLooksQueued(result?.state)) {
			return 'Queued for execution';
		}
		return 'Preparing scan progress';
	});

	function statusLooksQueued(state?: string): boolean {
		return (state || '').toUpperCase() === 'PENDING';
	}
</script>

<div class="space-y-7">
	<!-- Current stage -->
	<div>
		<p class="section-tag">What&apos;s happening now</p>
		<h3 class="font-display text-ink-strong mt-1.5 text-2xl font-semibold tracking-[-0.01em]">
			{stageInfo.stage}
		</h3>
		<p class="text-ink-muted mt-1 text-sm">{stageInfo.description}</p>
		<p class="text-ink mt-2 text-sm font-medium">{currentAction}</p>
	</div>

	<!-- Progress -->
	<div>
		<div class="mb-2 flex items-baseline justify-between text-xs">
			<span class="text-ink-muted font-medium">{progressLabel}</span>
			<span class="text-accent font-mono font-bold tabular-nums">{percentage}%</span>
		</div>
		<Progress value={percentage} />
		<div
			class="text-ink-faint mt-2 flex flex-wrap items-center justify-between gap-2 font-mono text-[11px]"
		>
			<span>{estimatedTime}</span>
			{#if result?.progress && result.progress.total_pages > 0}
				<span>
					{Math.max(result.progress.total_pages - result.progress.current_page, 0)} pages remaining
				</span>
			{/if}
		</div>
	</div>

	<!-- Per-scanner state -->
	{#if completedScanners.length > 0 || remainingScanners.length > 0}
		<ScannerActivityTable
			expected={result?.expected_scanners ?? []}
			completed={completedScanners}
			remaining={remainingScanners}
		/>
	{:else}
		<div class="border-line border-l-2 py-1 pl-4">
			<p class="text-ink text-sm font-semibold">Waiting for scanner activity</p>
			<p class="text-ink-muted mt-1 text-sm">
				The scan has started, but no scanner progress has been reported yet. Logs below will update
				first if the pipeline is still warming up.
			</p>
		</div>
	{/if}

	<!-- Live logs, collapsed by default -->
	<div>
		<button
			type="button"
			class="text-ink-muted hover:text-ink flex w-full items-center gap-2 text-left text-sm font-medium transition-colors"
			aria-expanded={logsOpen}
			aria-controls="live-logs"
			onclick={() => (logsOpen = !logsOpen)}
		>
			<ChevronDown
				class={cn('h-4 w-4 transition-transform', logsOpen && 'rotate-180')}
				aria-hidden="true"
			/>
			<TerminalSquare class="h-4 w-4" aria-hidden="true" />
			Live logs
			<span class="text-ink-faint ml-auto font-mono text-[11px]">
				{logs.length} line{logs.length === 1 ? '' : 's'}
			</span>
		</button>
		{#if logsOpen}
			<div id="live-logs" class="mt-3">
				<ScanTerminal {logs} />
			</div>
		{/if}
	</div>
</div>
