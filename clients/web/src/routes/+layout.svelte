<script lang="ts">
	// Self-hosted fonts via fontsource
	import '@fontsource-variable/inter';
	import '@fontsource-variable/jetbrains-mono';
	import '@fontsource-variable/source-serif-4';

	import '../app.css';
	import { page } from '$app/state';
	import Footer from '$lib/components/Footer.svelte';
	import { Navigation } from '$lib/components/nav';
	import { SITE, buildSiteUrl } from '$lib/config/site';

	let { children } = $props();
	const pathname = $derived(page.url.pathname);
	const canonicalUrl = $derived(buildSiteUrl(pathname));
	const shareImageUrl = buildSiteUrl('/og-image.png');
</script>

<svelte:head>
	<title>{SITE.siteTitle}</title>
	<meta name="description" content={SITE.tagline} />
	<link rel="canonical" href={canonicalUrl} />

	<meta property="og:title" content={SITE.siteTitle} />
	<meta property="og:description" content={SITE.tagline} />
	<meta property="og:url" content={canonicalUrl} />
	<meta property="og:image" content={shareImageUrl} />
	<meta property="og:image:alt" content="StageFlow scan dashboard preview card" />
	<meta property="og:type" content="website" />
	<meta property="og:site_name" content={SITE.name} />

	<meta name="twitter:card" content="summary_large_image" />
	<meta name="twitter:image" content={shareImageUrl} />
	<meta name="twitter:image:alt" content="StageFlow scan dashboard preview card" />
	<meta name="twitter:title" content={SITE.siteTitle} />
	<meta name="twitter:description" content={SITE.tagline} />
</svelte:head>

<div class="min-h-screen font-sans antialiased">
	<a href="#main-content" class="skip-link">Skip to main content</a>
	<Navigation />
	<main id="main-content" class="flex-1">
		{@render children()}
	</main>
	<Footer />
</div>
