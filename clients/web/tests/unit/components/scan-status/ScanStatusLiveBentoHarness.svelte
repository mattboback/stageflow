<script lang="ts">
	import ScanStatusLiveBento from '$lib/components/scan-status/ScanStatusLiveBento.svelte';
</script>

<ScanStatusLiveBento status="processing" elapsed={45000} scannerCount={6}>
	{#snippet headerContent()}
		<p>Header content</p>
	{/snippet}
	{#snippet mainContent()}
		<p>Main content</p>
	{/snippet}
	{#snippet sidebarContent()}
		<p>Sidebar content</p>
	{/snippet}
</ScanStatusLiveBento>
