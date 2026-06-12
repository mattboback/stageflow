<script lang="ts">
	import GithubIcon from '$lib/components/icons/GithubIcon.svelte';
	import LandingDataPreview from '$lib/components/landing/LandingDataPreview.svelte';
	import { buttonVariants } from '$lib/components/ui';
	import { SITE } from '$lib/config/site';
	import { SCANNER_META } from '$lib/report';
	import { cn } from '$lib/utils';
	import { ArrowRight } from 'lucide-svelte';

	const facts = [
		{ value: '8', label: 'scanners per run' },
		{ value: 'WCAG 2.1 AA', label: 'accessibility baseline' },
		{ value: '0', label: 'accounts required' }
	] as const;

	// Scanner list driven from a single source of truth (SCANNER_META).
	const scannerIndex = [
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

<div class="bg-paper min-h-screen">
	<!-- Masthead -->
	<section class="pt-28 pb-14 sm:pt-32 lg:pt-36">
		<div class="container-width">
			<div class="max-w-3xl">
				<p class="section-kicker">Self-hosted · Open source</p>
				<h1 class="h1-display mt-4">Eight scanners. One release-ready report.</h1>
				<p class="text-ink-muted mt-5 max-w-2xl text-lg leading-relaxed">
					StageFlow runs accessibility, performance, SEO, and security scanners in rootless
					containers, streams progress live, and merges every finding into a single report you can
					ship against.
				</p>

				<div class="mt-8 flex flex-wrap gap-3">
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
						View on GitHub
					</a>
				</div>
			</div>

			<!-- Fact strip -->
			<dl
				class="border-line mt-14 grid grid-cols-1 gap-y-4 border-y py-5 sm:grid-cols-3 sm:gap-y-0"
			>
				{#each facts as fact (fact.label)}
					<div
						class="border-line flex items-baseline gap-3 sm:flex-col sm:gap-1 sm:not-first:border-l sm:not-first:pl-8"
					>
						<dt class="text-ink-faint order-2 text-xs font-medium">{fact.label}</dt>
						<dd class="stat-mono text-ink-strong order-1 text-2xl font-bold">{fact.value}</dd>
					</div>
				{/each}
			</dl>
		</div>
	</section>

	<!-- Report preview -->
	<section class="pb-16 lg:pb-24">
		<div class="container-width">
			<div class="max-w-2xl">
				<p class="section-kicker">The deliverable</p>
				<h2 class="h2-display mt-2">What a report looks like</h2>
				<p class="text-ink-muted mt-3 text-sm leading-relaxed sm:text-base">
					Every scanner's findings are normalised into one surface — severity, evidence, and
					remediation side by side, for engineers, PMs, and designers.
				</p>
			</div>
			<div class="mt-8 max-w-3xl">
				<LandingDataPreview />
			</div>
		</div>
	</section>

	<div class="section-divider"></div>

	<!-- How it works -->
	<section class="section-padding">
		<div class="container-width">
			<div class="max-w-2xl">
				<p class="section-kicker">How it works</p>
				<h2 class="h2-display mt-2">From a target to a release-ready report</h2>
			</div>

			<ol class="border-line mt-10 max-w-3xl border-t">
				{#each workflowSteps as item (item.step)}
					<li
						class="border-line grid grid-cols-[3.5rem_1fr] gap-x-6 border-b py-7 sm:grid-cols-[5rem_1fr]"
					>
						<span
							class="stat-mono text-paper-deep text-4xl leading-none font-bold select-none sm:text-5xl"
							aria-hidden="true">{item.step}</span
						>
						<div>
							<h3 class="font-display text-ink-strong text-xl font-semibold tracking-[-0.005em]">
								{item.title}
							</h3>
							<p class="text-ink-muted mt-2 max-w-xl text-sm leading-relaxed">{item.desc}</p>
						</div>
					</li>
				{/each}
			</ol>
		</div>
	</section>

	<div class="section-divider"></div>

	<!-- Scanner index -->
	<section class="section-padding">
		<div class="container-width">
			<div class="max-w-2xl">
				<p class="section-kicker">Capabilities</p>
				<h2 class="h2-display mt-2">The scanner index</h2>
				<p class="text-ink-muted mt-3 text-sm leading-relaxed sm:text-base">
					Pick what runs per release. Each scanner runs isolated and reports into the same unified
					format.
				</p>
			</div>

			<ul class="border-line mt-10 border-t">
				{#each scannerIndex as scanner (scanner.id)}
					{@const meta = SCANNER_META[scanner.id]}
					<li
						class="border-line grid grid-cols-1 gap-x-6 gap-y-1 border-b py-4 sm:grid-cols-[11rem_7rem_1fr] sm:items-baseline"
					>
						<h3 class="text-ink-strong text-sm font-semibold tracking-[-0.005em]">
							{meta.label}
						</h3>
						<span class="section-tag">{scanner.category}</span>
						<p class="text-ink-muted text-sm leading-relaxed">{meta.description}.</p>
					</li>
				{/each}
			</ul>
		</div>
	</section>

	<!-- CTA band -->
	<section class="bg-ink-strong mt-8">
		<div class="container-width py-16 lg:py-20">
			<div class="max-w-2xl">
				<p class="section-kicker text-accent-subtle">Ready to scan</p>
				<h2
					class="font-display text-paper mt-2 text-[1.75rem] leading-[1.18] font-semibold tracking-[-0.01em] sm:text-[2rem] lg:text-[2.5rem] lg:leading-[1.15]"
				>
					Audit your site in one run
				</h2>
				<p class="text-paper/60 mt-3 max-w-xl text-sm leading-relaxed sm:text-base">
					No account required. Open the playground, choose your scanners, and start a complete run
					in under a minute.
				</p>
				<div class="mt-7">
					<a
						href="/playground"
						class={cn(
							buttonVariants({ variant: 'outline', size: 'lg' }),
							'border-paper/40 text-paper hover:bg-paper/10 gap-2 px-7 text-base font-semibold'
						)}
					>
						Open playground
						<ArrowRight class="h-4 w-4" aria-hidden="true" />
					</a>
				</div>
			</div>
		</div>
	</section>
</div>
