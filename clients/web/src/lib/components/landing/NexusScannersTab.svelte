<script lang="ts">
	import { fade } from 'svelte/transition';

	import { scanners } from './NexusDashboardHeroData';
</script>

<div class="nf-tab-container" in:fade={{ duration: 120 }}>
	<div class="nf-section-header">
		<div class="nf-panel-toprow">
			<span class="nf-panel-title">Auditing Engines Status</span>
			<span class="nf-pill">All tests completed</span>
		</div>
	</div>
	<div class="nf-scanners-grid">
		{#each scanners as s (s.name)}
			<div class="nf-scanner-card">
				<div class="nf-scanner-card-header">
					<div class="nf-scanner-circle-icon" style="background:{s.bg}">
						<div class="nf-scanner-circle-dot" style="background:{s.color}"></div>
					</div>
					<span class="nf-scanner-card-name">{s.name}</span>
					<span class="nf-scanner-card-score">{s.score}</span>
				</div>
				<div class="nf-scanner-card-status">
					<span class="nf-status-wave"></span>
					Active diagnostics verified
				</div>
			</div>
		{/each}
	</div>
</div>
