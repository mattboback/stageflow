<script lang="ts">
	import { page } from '$app/state';
	import ReportShell from '$lib/components/report/ReportShell.svelte';
	import { SITE } from '$lib/config/site';
	import { createScanReportStore } from '$lib/stores/scan-report.svelte';

	const jobId = $derived(page.params.id ?? '');

	let reportStore = $state<ReturnType<typeof createScanReportStore> | null>(null);

	$effect(() => {
		if (!jobId) return;

		const store = createScanReportStore(jobId);
		reportStore = store;
		store.start();

		return () => store.cleanup();
	});

	const status = $derived(reportStore?.status ?? 'loading');
	const report = $derived(reportStore?.report ?? null);
	const job = $derived(reportStore?.job ?? null);
	const logs = $derived(reportStore?.logs ?? []);
	const screenshots = $derived(reportStore?.screenshots ?? []);
	const error = $derived(reportStore?.error ?? null);
</script>

<svelte:head>
	<title>Scan Report | {SITE.ownerName}</title>
	<meta name="description" content="Multi-scanner report view" />
	<meta name="robots" content="noindex" />
</svelte:head>

<ReportShell
	{jobId}
	{status}
	{report}
	{job}
	{logs}
	{screenshots}
	{error}
	onRefreshArtifacts={() => reportStore?.refreshArtifacts()}
/>
