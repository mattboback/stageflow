<script lang="ts">
	import { page } from '$app/state';
	import {
		ScanArtifactsSidebar,
		ScanStatusContent,
		ScanStatusHeader
	} from '$lib/components/scan-status';
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
</script>

<svelte:head>
	<title>Scan Results | {SITE.ownerName}</title>
	<meta name="description" content="View scan status and generated artifacts." />
	<meta name="robots" content="noindex" />
</svelte:head>

<PageSection containerClass="max-w-7xl xl:max-w-[1400px]">
	<ScanStatusHeader id={scanId} {status} {elapsed} {result} />
	<div class="grid gap-6 lg:grid-cols-[1fr,380px] lg:items-start">
		<ScanStatusContent {status} {result} {logs} />
		<ScanArtifactsSidebar {status} {result} />
	</div>
</PageSection>
