<script lang="ts">
	import GithubIcon from '$lib/components/icons/GithubIcon.svelte';
	import NexusDashboardHero from '$lib/components/landing/NexusDashboardHero.svelte';
	import { Chip, buttonVariants } from '$lib/components/ui';
	import { SITE } from '$lib/config/site';
	import { SCANNER_META } from '$lib/report';
	import { cn } from '$lib/utils';
	import { ArrowRight } from 'lucide-svelte';

	const poweredBy = ['Podman', 'axe-core', 'Lighthouse', 'WCAG 2.1'] as const;

	// Scanner list driven from a single source of truth (SCANNER_META).
	// Grouped by category so we can show a calm category tag instead of a
	// rainbow palette of decorative brand colours.
	const scannerCards = [
		{ id: 'axe', category: 'Accessibility' },
		{ id: 'lighthouse', category: 'Performance' },
		{ id: 'security-headers', category: 'Security' },
		{ id: 'seo', category: 'SEO' },
		{ id: 'open-graph', category: 'SEO' },
		{ id: 'link-checker', category: 'Quality' },
		{ id: 'spelling-grammar', category: 'Quality' },
		{ id: 'ai-navigator', category: 'AI' }
	] as const;

	const workflowSteps = [
		{
			step: '01',
			title: 'Configure the scope',
			desc: 'Provide URLs or upload a ZIP archive, then pick the scanners that matter for this release.'
		},
		{
			step: '02',
			title: 'Run isolated scanners',
			desc: 'Every scanner runs in a rootless container while progress streams live over SSE.'
		},
		{
			step: '03',
			title: 'Ship with one report',
			desc: 'Review merged findings with severity, evidence, and remediation in a single view.'
		}
	] as const;

	const structuredData = JSON.stringify({
		'@context': 'https://schema.org',
		'@graph': [
			{
				'@type': 'WebSite',
				name: SITE.name,
				url: SITE.siteUrl,
				description: SITE.tagline
			},
			{
				'@type': 'SoftwareApplication',
				name: SITE.name,
				url: SITE.siteUrl,
				description: SITE.tagline,
				applicationCategory: 'DeveloperApplication',
				isAccessibleForFree: true,
				operatingSystem: 'Linux, macOS, Windows',
				sameAs: [SITE.githubUrl]
			}
		]
	});
	const safeStructuredData = structuredData
		.replace(/</g, '\\u003c')
		.replace(/>/g, '\\u003e')
		.replace(/&/g, '\\u0026')
		.replace(/\u2028/g, '\\u2028')
		.replace(/\u2029/g, '\\u2029');
	const structuredDataTag =
		`<script type="application/ld+json">${safeStructuredData}<` + '/script>';
</script>

<svelte:head>
	<title>{SITE.siteTitle}</title>
	<meta name="description" content={SITE.tagline} />
	{@html structuredDataTag}
</svelte:head>

<div class="landing-shell min-h-screen">
	<!-- Hero -->
	<section class="landing-hero pt-24 pb-16 sm:pt-28 lg:pt-32">
		<div class="container-width">
			<div class="mx-auto max-w-3xl text-center">
				<div class="mb-5 flex flex-wrap items-center justify-center gap-2">
					<span class="section-tag">Self-hosted scanning platform</span>
					<span class="text-ink-faint hidden text-xs sm:inline" aria-hidden="true">·</span>
					<Chip tone="muted" size="sm" class="landing-chip gap-1.5">
						<GithubIcon class="h-3.5 w-3.5" />
						Open source
					</Chip>
				</div>

				<h1 class="h1-display landing-headline">
					Audit accessibility, performance, and security
					<span class="text-accent">in one operational flow.</span>
				</h1>

				<p class="landing-subhead mx-auto mt-5">
					{SITE.tagline}. Run eight scanners in rootless containers, watch progress live, and ship
					with one merged report.
				</p>

				<div class="mt-8 flex flex-wrap justify-center gap-3">
					<a
						href="/playground"
						class={cn(buttonVariants({ variant: 'default', size: 'lg' }), 'gap-2 px-7 text-base')}
					>
						Start a scan
						<ArrowRight class="h-4 w-4" aria-hidden="true" />
					</a>
					<a
						href={SITE.githubUrl}
						target="_blank"
						rel="noopener noreferrer"
						class={cn(buttonVariants({ variant: 'outline', size: 'lg' }), 'gap-2 px-7 text-base')}
					>
						<GithubIcon class="h-4 w-4" />
						View source
					</a>
				</div>

				<p class="text-ink-faint mt-7 text-xs">
					<span class="font-semibold">Powered by</span>
					{#each poweredBy as label, i (label)}
						<span class="ml-2">{label}</span>
						{#if i < poweredBy.length - 1}
							<span class="text-ink-faint/60 ml-2" aria-hidden="true">·</span>
						{/if}
					{/each}
				</p>
			</div>

			<!-- Dashboard preview -->
			<div class="mt-14">
				<NexusDashboardHero />
			</div>
		</div>
	</section>

	<!-- Workflow -->
	<section class="section-padding">
		<div class="container-width">
			<div class="mx-auto max-w-2xl">
				<p class="section-tag">How it works</p>
				<h2 class="h2-display mt-2">From a target to a release-ready report</h2>
			</div>

			<ol class="relative mt-12 grid gap-5 lg:grid-cols-3">
				<div class="workflow-line" aria-hidden="true"></div>
				{#each workflowSteps as item (item.step)}
					<li class="workflow-step">
						<div class="mb-5 flex items-center gap-2.5">
							<span
								class="border-line bg-surface text-ink-faint inline-flex h-7 w-7 items-center justify-center rounded-md border font-mono text-xs font-semibold"
								>{item.step}</span
							>
							<div class="bg-line h-px flex-1"></div>
						</div>
						<h3 class="text-ink text-lg font-semibold tracking-[-0.005em]">{item.title}</h3>
						<p class="text-ink-muted mt-2 text-sm leading-relaxed">{item.desc}</p>
					</li>
				{/each}
			</ol>
		</div>
	</section>

	<div class="section-divider"></div>

	<!-- Capabilities -->
	<section class="section-padding">
		<div class="container-width">
			<div class="grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,22rem)] lg:items-end">
				<div class="max-w-3xl">
					<p class="section-tag">Capabilities</p>
					<h2 class="h2-display mt-2">Eight scanners. One operational report.</h2>
				</div>
				<p class="text-ink-muted text-sm leading-relaxed lg:text-right">
					Pick what runs per release. Findings are normalised into one surface for engineers, PMs,
					and designers.
				</p>
			</div>

			<ul class="mt-10 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
				{#each scannerCards as scanner (scanner.id)}
					{@const meta = SCANNER_META[scanner.id]}
					{@const ScannerIcon = meta.icon}
					<li
						class="border-line bg-surface hover:border-ink-faint/50 flex flex-col rounded-2xl border p-5 transition-colors"
					>
						<div class="mb-4 flex items-center justify-between gap-3">
							<div
								class="bg-accent-mist text-accent flex h-9 w-9 items-center justify-center rounded-lg"
							>
								<ScannerIcon class="h-4.5 w-4.5" aria-hidden="true" />
							</div>
							<span class="section-tag">{scanner.category}</span>
						</div>
						<h3 class="text-ink-strong text-sm font-semibold tracking-[-0.005em]">
							{meta.label}
						</h3>
						<p class="text-ink-muted mt-1.5 text-xs leading-relaxed">{meta.description}.</p>
					</li>
				{/each}
			</ul>
		</div>
	</section>

	<!-- CTA -->
	<section class="pt-8 pb-24">
		<div class="container-width">
			<div class="cta-panel mx-auto max-w-4xl">
				<div class="relative z-10">
					<p class="section-tag text-accent-subtle">Ready to scan</p>
					<h2 class="h2-display text-surface mt-2">Audit your site in one run</h2>
					<p class="text-surface/70 mx-auto mt-3 max-w-2xl text-sm leading-relaxed sm:text-base">
						No account required. Open the playground, choose your scanners, and start a complete run
						in under a minute.
					</p>
					<div class="mt-7">
						<a
							href="/playground"
							class={cn(
								buttonVariants({ variant: 'outline', size: 'lg' }),
								'border-surface/40 text-surface hover:bg-surface/10 gap-2 rounded-xl px-7 text-base font-semibold'
							)}
						>
							Open playground
							<ArrowRight class="h-4 w-4" aria-hidden="true" />
						</a>
					</div>
				</div>
			</div>
		</div>
	</section>
</div>
