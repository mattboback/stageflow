<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		urlBarText?: string;
		navLabel?: string;
		readyLabel?: string;
		isReady?: boolean;
		scannerCount?: number;
		leftPanel: Snippet;
		rightPanel: Snippet;
		bottomLeft?: Snippet;
		bottomCenter?: Snippet;
		bottomRight?: Snippet;
	}

	let {
		urlBarText = 'stageflow.org/playground',
		navLabel = 'Configure Scan',
		readyLabel = '',
		isReady = false,
		scannerCount = 0,
		leftPanel,
		rightPanel,
		bottomLeft,
		bottomCenter,
		bottomRight
	}: Props = $props();
</script>

<div class="bento-frame playground-bento-frame" aria-label="Scan configuration dashboard">
	<!-- Browser chrome -->
	<div class="bento-chrome">
		<div class="bento-chrome-dots">
			<i class="bento-chrome-dot" style="background:#fc5f57"></i>
			<i class="bento-chrome-dot" style="background:#fdbc2d"></i>
			<i class="bento-chrome-dot" style="background:#34c749"></i>
		</div>
		<div class="bento-chrome-url">{urlBarText}</div>
	</div>

	<!-- White card -->
	<div class="bento-card">
		<!-- Nav strip -->
		<div class="bento-nav">
			<span class="bento-panel-title">{navLabel}</span>
			{#if readyLabel}
				<span
					class="ml-auto rounded-full px-2.5 py-0.5 text-[11px] font-semibold"
					class:bg-emerald-50={isReady}
					class:text-emerald-700={isReady}
					class:bg-amber-50={!isReady}
					class:text-amber-700={!isReady}
				>
					{readyLabel}
				</span>
			{/if}
			{#if scannerCount > 0}
				<span class="text-ink-muted text-[11px] font-medium">
					{scannerCount} scanner{scannerCount === 1 ? '' : 's'}
				</span>
			{/if}
		</div>

		<!-- Main bento grid: left (input/config) + right (scanners) -->
		<div class="bento-grid-main">
			<div class="bento-panel">
				{@render leftPanel()}
			</div>
			<div class="bento-panel">
				{@render rightPanel()}
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
		{/if}
	</div>
</div>
