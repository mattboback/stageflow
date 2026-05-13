<script lang="ts">
	import { page } from '$app/state';
	import {
		ScanArtifactsSidebar,
		ScanStatusContent,
		ScanStatusHeader
	} from '$lib/components/scan-status';
	import ScanStatusLiveBento from '$lib/components/scan-status/ScanStatusLiveBento.svelte';
	import { PageSection } from '$lib/components/ui';
	import { SITE } from '$lib/config/site';
	import { createScanStatusStore } from '$lib/stores/scan-status.svelte';

	const scanId = $derived(page.params.id ?? '');
	let scanStore = $state<ReturnType<typeof createScanStatusStore> | null>(null);

	$effect(() => {
		if (!scanId) return;

		const store = createScanStatusStore(scanId);
		scanStore = store;
		store.start();

		return () => store.cleanup();
	});

	const status = $derived(scanStore?.status ?? 'loading');
	const result = $derived(scanStore?.result ?? null);
	const elapsed = $derived(scanStore?.elapsed ?? 0);
	const logs = $derived(scanStore?.logs ?? []);
	const scannerCount = $derived(
		result?.expected_scanners?.length ??
			Math.max(
				(result?.completed_scanners?.length ?? 0) + (result?.remaining_scanners?.length ?? 0),
				result?.completed_scanners?.length ?? 0
			)
	);
</script>

<svelte:head>
	<title>Scan Results | {SITE.name}</title>
	<meta name="description" content="View scan status and generated artifacts." />
	<meta name="robots" content="noindex" />
</svelte:head>

<PageSection containerClass="max-w-7xl xl:max-w-[1400px]">
	<ScanStatusLiveBento {status} {elapsed} {scannerCount}>
		{#snippet headerContent()}
			<ScanStatusHeader id={scanId} {status} {elapsed} {result} />
		{/snippet}
		{#snippet mainContent()}
			<ScanStatusContent {status} {result} {logs} />
		{/snippet}
		{#snippet sidebarContent()}
			<ScanArtifactsSidebar {status} {result} />
		{/snippet}
	</ScanStatusLiveBento>
</PageSection>
