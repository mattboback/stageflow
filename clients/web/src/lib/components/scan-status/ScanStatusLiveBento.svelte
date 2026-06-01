<script lang="ts">
	import type { Snippet } from 'svelte';

	import type { ScanStatus } from '$lib/types/scan';

	interface Props {
		status: ScanStatus;
		elapsed?: number;
		scannerCount?: number;
		headerContent: Snippet;
		mainContent: Snippet;
		sidebarContent: Snippet;
		bottomLeft?: Snippet;
		bottomCenter?: Snippet;
		bottomRight?: Snippet;
	}

	let {
		status,
		elapsed = 0,
		scannerCount = 0,
		headerContent,
		mainContent,
		sidebarContent,
		bottomLeft,
		bottomCenter,
		bottomRight
	}: Props = $props();

	const statusLabel = $derived.by(() => {
		switch (status) {
			case 'complete':
				return 'Complete';
			case 'failed':
			case 'error':
				return 'Failed';
			default:
				return 'Running';
		}
	});

	const isRunning = $derived(status !== 'complete' && status !== 'failed' && status !== 'error');

	function formatElapsed(ms: number): string {
		const s = Math.floor(ms / 1000);
		const m = Math.floor(s / 60);
		const sec = s % 60;
		return m > 0 ? `${m}m ${sec}s` : `${sec}s`;
	}
</script>

<div class="bento-frame status-bento-frame" aria-label="Scan status dashboard">
	<div class="bento-chrome">
		<div class="bento-chrome-dots">
			<i class="bento-chrome-dot" style="background:#fc5f57"></i>
			<i class="bento-chrome-dot" style="background:#fdbc2d"></i>
			<i class="bento-chrome-dot" style="background:#34c749"></i>
		</div>
		<div class="bento-chrome-url">stageflow.org/scan/status</div>
	</div>

	<div class="bento-card">
		<!-- Nav/header strip -->
		<div class="bento-nav">
			<span class="bento-panel-title">Scan Status</span>
			<span
				class="ml-auto rounded-full px-2.5 py-0.5 text-[11px] font-semibold"
				class:bg-emerald-50={status === 'complete'}
				class:text-emerald-700={status === 'complete'}
				class:bg-red-50={status === 'failed' || status === 'error'}
				class:text-red-700={status === 'failed' || status === 'error'}
				class:bg-blue-50={isRunning}
				class:text-blue-700={isRunning}
			>
				{statusLabel}
			</span>
		</div>

		<!-- Header content (ScanStatusHeader) -->
		<div class="border-line border-b px-5 py-4">
			{@render headerContent()}
		</div>

		<!-- Main grid: content (left) + sidebar (right) -->
		<div class="bento-grid-main">
			<div class="bento-panel">
				{@render mainContent()}
			</div>
			<div class="bento-panel">
				{@render sidebarContent()}
			</div>
		</div>

		<!-- Bottom strip -->
		{#if bottomLeft || bottomCenter || bottomRight}
			<div class="bento-grid-bottom">
				<div class="bento-panel">
					{#if bottomLeft}
						{@render bottomLeft()}
					{/if}
				</div>
				<div class="bento-panel">
					{#if bottomCenter}
						{@render bottomCenter()}
					{/if}
				</div>
				<div class="bento-panel">
					{#if bottomRight}
						{@render bottomRight()}
					{/if}
				</div>
			</div>
		{:else}
			<div class="bento-grid-bottom">
				<div class="bento-panel">
					<span class="text-ink-muted text-[11px] font-medium tracking-wide uppercase">Elapsed</span
					>
					<p class="stat-mono text-ink-strong mt-1 text-lg font-bold">{formatElapsed(elapsed)}</p>
				</div>
				<div class="bento-panel">
					<span class="text-ink-muted text-[11px] font-medium tracking-wide uppercase"
						>Scanners</span
					>
					<p class="stat-mono text-ink-strong mt-1 text-lg font-bold">{scannerCount}</p>
				</div>
				<div class="bento-panel">
					<span class="text-ink-muted text-[11px] font-medium tracking-wide uppercase">Status</span>
					<p class="text-ink-strong mt-1 text-lg font-bold">{statusLabel}</p>
				</div>
			</div>
		{/if}
	</div>
</div>
