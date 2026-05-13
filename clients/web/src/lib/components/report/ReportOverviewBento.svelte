<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		urlBarText?: string;
		navTabs?: string[];
		activeTab?: string;
		mainPanel: Snippet;
		sidePanel: Snippet;
		bottomLeft?: Snippet;
		bottomCenter?: Snippet;
		bottomRight?: Snippet;
	}

	let {
		urlBarText = 'stageflow.org/scan/report',
		navTabs = [],
		activeTab = '',
		mainPanel,
		sidePanel,
		bottomLeft,
		bottomCenter,
		bottomRight
	}: Props = $props();
</script>

<div class="bento-frame" aria-label="Report overview dashboard">
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
		<!-- Nav strip with tabs -->
		{#if navTabs.length > 0}
			<div class="bento-nav">
				{#each navTabs as tab (tab)}
					<span
						class="rounded-full px-3 py-1.5 text-xs font-medium"
						class:bg-accent={tab === activeTab}
						class:text-white={tab === activeTab}
						class:text-ink-muted={tab !== activeTab}
					>
						{tab}
					</span>
				{/each}
			</div>
		{/if}

		<!-- Main bento grid -->
		<div class="bento-grid-main">
			<div class="bento-panel">
				{@render mainPanel()}
			</div>
			<div class="bento-panel">
				{@render sidePanel()}
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
