<script lang="ts">
	import type { ScanResult } from '$lib/types/scan';

	import { Progress } from '$lib/components/ui';
	import { Clock, FileSearch, Loader2, Server, ShieldCheck, TerminalSquare } from 'lucide-svelte';

	import ScanTerminal from './ScanTerminal.svelte';

	interface Props {
		result: ScanResult | null;
		logs: string[];
	}

	let { result, logs }: Props = $props();

	function getStageInfo(
		state?: string,
		progress?: { current_page: number; total_pages: number }
	): { stage: string; icon: typeof Clock; description: string } {
		const normalizedState = (state || '').toUpperCase();

		switch (normalizedState) {
			case 'PENDING':
				return {
					stage: 'Queued',
					icon: Clock,
					description: 'Waiting for available worker…'
				};
			case 'EXTRACTING':
				return {
					stage: 'Extracting',
					icon: Server,
					description: 'Processing uploaded content…'
				};
			case 'READY_TO_SCAN':
				return {
					stage: 'Preparing',
					icon: FileSearch,
					description: 'Analyzing site structure…'
				};
			case 'SCANNING':
				if (progress) {
					return {
						stage: 'Scanning',
						icon: Loader2,
						description: `Auditing page ${progress.current_page} of ${progress.total_pages}`
					};
				}
				return {
					stage: 'Scanning',
					icon: Loader2,
					description: 'Running accessibility checks…'
				};
			case 'COMPLETING':
				return {
					stage: 'Finalizing',
					icon: FileSearch,
					description: 'Generating reports…'
				};
			default:
				return {
					stage: 'Processing',
					icon: Loader2,
					description: 'Initializing scan environment…'
				};
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
	const expectedScannerCount = $derived(
		result?.expected_scanners?.length ??
			Math.max(completedScanners.length + remainingScanners.length, completedScanners.length)
	);
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

<div class="space-y-8">
	<div class="grid gap-4 lg:grid-cols-[1.2fr,0.8fr]">
		<div class="border-line bg-surface-muted rounded-2xl border p-5">
			<div class="flex items-start gap-4">
				<div
					class="bg-accent-soft text-accent-ink flex h-12 w-12 items-center justify-center rounded-xl"
				>
					<stageInfo.icon class="h-6 w-6" />
				</div>
				<div class="min-w-0 flex-1">
					<p class="text-ink-faint text-[11px] font-semibold tracking-[0.14em] uppercase">
						What&apos;s happening now
					</p>
					<h3 class="text-ink mt-1 text-lg font-semibold">
						{stageInfo.stage}
					</h3>
					<p class="text-ink-muted mt-1 text-sm">{stageInfo.description}</p>
					<p class="text-ink mt-3 text-sm font-medium">{currentAction}</p>
				</div>
			</div>

			<div class="mt-5 space-y-3">
				<div class="flex items-center justify-between text-sm">
					<span class="text-ink-muted font-medium">{progressLabel}</span>
					<span class="text-ink font-mono">{percentage}%</span>
				</div>
				<Progress value={percentage} class="h-3" />
				<div class="flex flex-wrap items-center justify-between gap-2 text-sm">
					<div class="text-ink-muted flex items-center gap-2">
						<span class="bg-accent inline-flex h-2 w-2 rounded-full"></span>
						<span>{estimatedTime}</span>
					</div>
					{#if result?.progress && result.progress.total_pages > 0}
						<span class="text-ink-muted font-medium">
							{Math.max(result.progress.total_pages - result.progress.current_page, 0)} pages remaining
						</span>
					{/if}
				</div>
			</div>
		</div>

		<div class="grid gap-3 sm:grid-cols-3 lg:grid-cols-1">
			<div class="bg-surface rounded-2xl border border-[var(--color-line)] px-4 py-3">
				<div
					class="text-ink-muted flex items-center gap-2 text-[11px] font-semibold tracking-[0.14em] uppercase"
				>
					<FileSearch class="h-3.5 w-3.5" />
					Pages
				</div>
				<p class="mt-2 text-sm font-semibold">
					{#if result?.progress?.total_pages}
						{result.progress.current_page} of {result.progress.total_pages}
					{:else}
						Awaiting totals
					{/if}
				</p>
			</div>
			<div class="bg-surface rounded-2xl border border-[var(--color-line)] px-4 py-3">
				<div
					class="text-ink-muted flex items-center gap-2 text-[11px] font-semibold tracking-[0.14em] uppercase"
				>
					<ShieldCheck class="h-3.5 w-3.5" />
					Scanners
				</div>
				<p class="mt-2 text-sm font-semibold">
					{completedScanners.length} complete{#if expectedScannerCount > 0}<span
							class="text-ink-muted"
						>
							of {expectedScannerCount}</span
						>{/if}
				</p>
			</div>
			<div class="bg-surface rounded-2xl border border-[var(--color-line)] px-4 py-3">
				<div
					class="text-ink-muted flex items-center gap-2 text-[11px] font-semibold tracking-[0.14em] uppercase"
				>
					<Clock class="h-3.5 w-3.5" />
					Handoff
				</div>
				<p class="mt-2 text-sm font-semibold">Report unlocks when all scanners finish</p>
			</div>
		</div>
	</div>

	{#if completedScanners.length > 0 || remainingScanners.length > 0}
		<div class="space-y-4">
			<div class="flex items-center justify-between">
				<h4 class="text-ink text-sm font-semibold">Scanner Activity</h4>
				<div class="text-ink-muted text-xs font-medium">
					{completedScanners.length} of {completedScanners.length + remainingScanners.length} finished
				</div>
			</div>
			{#if remainingScanners.length > 0}
				<div class="space-y-2">
					<div class="text-ink-muted text-xs font-semibold tracking-[0.18em] uppercase">
						Still running
					</div>
					<div class="flex flex-wrap gap-2">
						{#each remainingScanners as scannerType (scannerType)}
							<span
								class="border-line bg-surface text-ink inline-flex rounded-full border px-3 py-1 text-xs font-medium"
							>
								{formatScannerName(scannerType)}
							</span>
						{/each}
					</div>
				</div>
			{/if}
			{#if completedScanners.length > 0}
				<div class="space-y-2">
					<div class="text-ink-muted text-xs font-semibold tracking-[0.18em] uppercase">
						Completed
					</div>
					<div class="flex flex-wrap gap-2">
						{#each completedScanners as scannerType (scannerType)}
							<span
								class="bg-accent-soft text-accent-ink inline-flex rounded-full px-3 py-1 text-xs font-medium"
							>
								{formatScannerName(scannerType)}
							</span>
						{/each}
					</div>
				</div>
			{/if}
		</div>
	{:else}
		<div class="border-line bg-surface-muted/55 rounded-2xl border p-4">
			<p class="text-ink text-sm font-semibold">Waiting for scanner activity</p>
			<p class="text-ink-muted mt-1 text-sm">
				The scan has started, but no scanner progress has been reported yet. Logs below will update
				first if the pipeline is still warming up.
			</p>
		</div>
	{/if}

	<div class="space-y-3">
		<div class="flex items-center justify-between gap-3">
			<div>
				<h4 class="text-ink flex items-center gap-2 text-sm font-semibold">
					<TerminalSquare class="h-4 w-4" />
					Live logs
				</h4>
				<p class="text-ink-muted text-xs">
					Use logs for detailed diagnostics when the summary above is not enough.
				</p>
			</div>
		</div>
		<ScanTerminal {logs} />
	</div>
</div>
