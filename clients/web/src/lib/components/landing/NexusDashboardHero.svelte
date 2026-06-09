<script lang="ts">
	import { siteDisplayPath } from '$lib/config/site';

	import { navTabs, type TabName } from './NexusDashboardHeroData';
	import NexusIssuesTab from './NexusIssuesTab.svelte';
	import NexusOverviewTab from './NexusOverviewTab.svelte';
	import NexusPagesTab from './NexusPagesTab.svelte';
	import NexusScannersTab from './NexusScannersTab.svelte';
	import './nexus-dashboard-hero.css';

	let activeTab = $state<TabName>('Overview');
</script>

<div class="nf-frame" aria-hidden="true">
	<!-- Browser chrome bar -->
	<div class="nf-chrome">
		<div class="nf-dots">
			<i class="nf-dot" style="background:#fc5f57"></i>
			<i class="nf-dot" style="background:#fdbc2d"></i>
			<i class="nf-dot" style="background:#34c749"></i>
		</div>
		<div class="nf-urlbar">{siteDisplayPath('/report/staging-deploy')}</div>
	</div>

	<!-- White dashboard card -->
	<div class="nf-card">
		<!-- Product nav strip -->
		<div class="nf-nav">
			<div class="nf-logo">
				<div class="nf-logo-icon">SF</div>
				<span class="nf-logo-text">StageFlow</span>
			</div>
			<div class="nf-tabs">
				{#each navTabs as tab (tab)}
					<button
						type="button"
						class="nf-tab"
						class:nf-tab-active={activeTab === tab}
						onclick={() => (activeTab = tab)}
					>
						{tab}
					</button>
				{/each}
			</div>
		</div>

		<!-- Conditional Svelte Views -->
		{#if activeTab === 'Overview'}
			<NexusOverviewTab onViewIssues={() => (activeTab = 'Issues')} />
		{:else if activeTab === 'Issues'}
			<NexusIssuesTab />
		{:else if activeTab === 'Pages'}
			<NexusPagesTab />
		{:else if activeTab === 'Scanners'}
			<NexusScannersTab />
		{/if}
	</div>
</div>
