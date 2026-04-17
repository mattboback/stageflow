<script lang="ts">
	import { page } from '$app/state';
	import ReportShell from '$lib/components/report/ReportShell.svelte';
	import { SITE, buildSiteUrl } from '$lib/config/site';
	import { createScanReportStore } from '$lib/stores/scan-report.svelte';

	const jobId = $derived(page.params.id ?? '');
	const canonicalUrl = $derived(buildSiteUrl(page.url.pathname));
	const pageTitle = $derived(`Scan Report | ${SITE.name}`);
	const pageDescription = 'Multi-scanner report view with issue evidence and remediation guidance.';
	const shareImageUrl = buildSiteUrl('/og-image.png');

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
	<title>{pageTitle}</title>
	<meta name="description" content={pageDescription} />
	<link rel="canonical" href={canonicalUrl} />

	<meta property="og:title" content={pageTitle} />
	<meta property="og:description" content={pageDescription} />
	<meta property="og:url" content={canonicalUrl} />
	<meta property="og:image" content={shareImageUrl} />
	<meta property="og:image:alt" content="StageFlow scan report preview card" />
	<meta property="og:type" content="article" />
	<meta property="og:site_name" content={SITE.name} />
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
