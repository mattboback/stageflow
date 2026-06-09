<script lang="ts">
	import { fade } from 'svelte/transition';

	import { mockPages } from './NexusDashboardHeroData';
</script>

<div class="nf-tab-container" in:fade={{ duration: 120 }}>
	<div class="nf-section-header">
		<div class="nf-panel-toprow">
			<span class="nf-panel-title">Site Hierarchy & Crawled Index</span>
			<span class="nf-pill">4 entries audited</span>
		</div>
	</div>
	<div class="nf-pages-grid">
		{#each mockPages as p (p.path)}
			<div class="nf-page-card">
				<div class="nf-page-header">
					<span class="nf-page-path font-mono">{p.path}</span>
					<span class="nf-page-badge" class:nf-page-badge-success={p.issues === 0}>
						{p.issues === 0 ? 'CLEAN' : `${p.issues} FINDINGS`}
					</span>
				</div>
				<p class="nf-page-label">{p.label}</p>
				<div class="nf-page-metrics">
					<div class="nf-page-metric">
						<span class="nf-metric-label">Score</span>
						<span class="nf-metric-value" style="color: {p.score > 90 ? '#10b981' : '#f59e0b'}"
							>{p.score}</span
						>
					</div>
					<div class="nf-page-metric">
						<span class="nf-metric-label">Audited</span>
						<span class="nf-metric-value text-ink-muted">{p.loadTime}</span>
					</div>
				</div>
			</div>
		{/each}
	</div>
</div>
