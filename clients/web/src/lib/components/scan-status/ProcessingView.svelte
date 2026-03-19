<script lang="ts">
	import type { ScanResult } from '$lib/types/scan';

	import { Progress } from '$lib/components/ui';
	import { Clock, FileSearch, Loader2, Server } from 'lucide-svelte';

	import ScanTerminal from './ScanTerminal.svelte';

	interface Props {
		result: ScanResult | null;
		logs: string[];
	}

	const { result, logs }: Props = $props();

	function getStageInfo(
		state?: string,
		progress?: { current_page: number; total_pages: number }
	): { stage: string; icon: typeof Clock; description: string } {
		const normalizedState = (state || '').toUpperCase();

		switch (normalizedState) {
			case 'PENDING':
				return { stage: 'Queued', icon: Clock, description: 'Waiting for available worker...' };
			case 'EXTRACTING':
				return { stage: 'Extracting', icon: Server, description: 'Processing uploaded content...' };
			case 'READY_TO_SCAN':
				return { stage: 'Preparing', icon: FileSearch, description: 'Analyzing site structure...' };
			case 'SCANNING':
				if (progress) {
					return {
						stage: 'Scanning',
						icon: Loader2,
						description: `Auditing page ${progress.current_page} of ${progress.total_pages}`
					};
				}
				return { stage: 'Scanning', icon: Loader2, description: 'Running accessibility checks...' };
			case 'COMPLETING':
				return { stage: 'Finalizing', icon: FileSearch, description: 'Generating reports...' };
			default:
				return {
					stage: 'Processing',
					icon: Loader2,
					description: 'Initializing scan environment...'
				};
		}
	}

	function estimateTimeRemaining(progress?: { current_page: number; total_pages: number }): string {
		if (!progress || progress.current_page === 0) {
			return 'Calculating...';
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
		switch (scannerType) {
			case 'axe':
				return 'axe-core';
			case 'seo':
				return 'SEO';
			case 'ai-navigator':
				return 'AI Navigator';
			case 'link-checker':
				return 'Link Checker';
			case 'security-headers':
				return 'Security Headers';
			case 'lighthouse':
				return 'Lighthouse';
			default:
				return scannerType
					.split('-')
					.map((part) => part.charAt(0).toUpperCase() + part.slice(1))
					.join(' ');
		}
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
</script>

<div class="space-y-8">
	<!-- Status Header -->
	<div class="border-line bg-surface-muted flex items-center justify-between rounded-xl border p-4">
		<div class="flex items-center gap-4">
			<div
				class="bg-accent-soft text-accent-ink flex h-12 w-12 items-center justify-center rounded-xl"
			>
				<stageInfo.icon class="h-6 w-6" />
			</div>
			<div>
				<h3 class="text-ink text-lg font-semibold">
					{stageInfo.stage}
				</h3>
				<p class="text-ink-muted text-sm">{stageInfo.description}</p>
			</div>
		</div>
		<div class="text-right">
			<div class="text-accent-ink text-2xl font-bold tabular-nums">
				{percentage}%
			</div>
			<div class="text-ink-muted text-xs">{estimatedTime}</div>
		</div>
	</div>

	<!-- Progress Bar -->
	<div class="space-y-3">
		<Progress value={percentage} class="h-3" />
		<div class="flex justify-between text-sm">
			<div class="text-ink-muted flex items-center gap-2">
				<span class="bg-accent inline-flex h-2 w-2 rounded-full"></span>
				{#if result?.progress}
					Page {result.progress.current_page} of {result.progress.total_pages}
				{:else}
					Initializing...
				{/if}
			</div>
			{#if result?.progress && result.progress.total_pages > 0}
				<span class="text-ink-muted font-medium">
					{result.progress.total_pages - result.progress.current_page} pages remaining
				</span>
			{/if}
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
	{/if}

	<!-- Terminal -->
	<ScanTerminal {logs} />
</div>
