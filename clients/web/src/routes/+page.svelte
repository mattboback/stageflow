<script lang="ts">
	import GithubIcon from '$lib/components/icons/GithubIcon.svelte';
	import HeroScanPreview from '$lib/components/landing/HeroScanPreview.svelte';
	import { Chip, buttonVariants } from '$lib/components/ui';
	import { SITE } from '$lib/config/site';
	import { cn } from '$lib/utils';
	import {
		ArrowRight,
		Bot,
		Box,
		CheckCircle,
		Gauge,
		Link2,
		Search,
		Shield,
		ShieldCheck,
		Zap
	} from 'lucide-svelte';

	const metrics = [
		{ value: '8', label: 'Scanners in one run' },
		{ value: 'SSE', label: 'Live progress stream' },
		{ value: '100%', label: 'Self-hosted control' }
	] as const;

	const poweredBy = [
		{ icon: Box, label: 'Podman' },
		{ icon: ShieldCheck, label: 'axe-core' },
		{ icon: Gauge, label: 'Lighthouse' },
		{ icon: Shield, label: 'WCAG 2.1' }
	] as const;

	const scanners = [
		{
			icon: Shield,
			name: 'Axe',
			desc: 'WCAG 2.1 AA/AAA checks with full node-level evidence and violation context.',
			color: 'text-blue-600',
			bg: 'bg-blue-50',
			border: 'border-t-blue-500'
		},
		{
			icon: Zap,
			name: 'Lighthouse',
			desc: 'Performance, SEO, and best-practices scoring from the same run pipeline.',
			color: 'text-amber-600',
			bg: 'bg-amber-50',
			border: 'border-t-amber-500'
		},
		{
			icon: Search,
			name: 'SEO',
			desc: 'Meta, headings, structured data, and crawlability checks surfaced in one report.',
			color: 'text-rose-600',
			bg: 'bg-rose-50',
			border: 'border-t-rose-500'
		},
		{
			icon: Shield,
			name: 'Security Headers',
			desc: 'CSP, HSTS, X-Frame-Options, and critical header posture scoring.',
			color: 'text-violet-600',
			bg: 'bg-violet-50',
			border: 'border-t-violet-500'
		},
		{
			icon: Link2,
			name: 'Link Checker',
			desc: 'Broken links, redirect chains, and dead-end paths caught before release.',
			color: 'text-emerald-600',
			bg: 'bg-emerald-50',
			border: 'border-t-emerald-500'
		},
		{
			icon: Bot,
			name: 'AI Navigator',
			desc: 'Goal-based journey simulation that catches experience breakpoints early.',
			color: 'text-fuchsia-600',
			bg: 'bg-fuchsia-50',
			border: 'border-t-fuchsia-500'
		},
		{
			icon: Search,
			name: 'Open Graph',
			desc: 'Social preview metadata validation for Open Graph tags, Twitter cards, and share readiness.',
			color: 'text-cyan-600',
			bg: 'bg-cyan-50',
			border: 'border-t-cyan-500'
		},
		{
			icon: CheckCircle,
			name: 'Spelling & Grammar',
			desc: 'AI-powered spelling, grammar, and content quality checks for polished publication-ready copy.',
			color: 'text-lime-700',
			bg: 'bg-lime-50',
			border: 'border-t-lime-500'
		}
	] as const;

	const workflowSteps = [
		{
			step: '01',
			title: 'Configure the scope',
			desc: 'Provide URLs or upload a ZIP archive, then choose the scanners needed for that release.',
			signal: 'Boundary validation prevents invalid scan states before execution.'
		},
		{
			step: '02',
			title: 'Run isolated scanners',
			desc: 'Every scanner runs in rootless containers while progress streams live over SSE.',
			signal: 'Container isolation keeps scanner behavior reproducible and debuggable.'
		},
		{
			step: '03',
			title: 'Ship with one report',
			desc: 'Review merged findings with severity, evidence, and WCAG mapping in a single view.',
			signal: 'Unified output removes context switching across scanner tools.'
		}
	] as const;

	const differentiators = [
		{
			title: 'Built for self-hosted teams',
			desc: 'Run everything on your infrastructure. No third-party data relay required.'
		},
		{
			title: 'Operationally transparent',
			desc: 'Realtime logs and status events show exactly what scanner is running and why.'
		},
		{
			title: 'Actionable remediation',
			desc: 'Findings include fix guidance and references designed for fast implementation.'
		},
		{
			title: 'One scanning workflow',
			desc: 'Accessibility, SEO, security, and performance auditing in one execution path.'
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
</script>

<svelte:head>
	<title>{SITE.siteTitle}</title>
	<meta name="description" content={SITE.tagline} />
	{@html `<script type="application/ld+json">${structuredData}</script>`}
</svelte:head>

<div class="landing-shell min-h-screen">
	<section class="landing-hero pt-32 pb-20 sm:pt-36">
		<div class="container-width">
			<div class="landing-grid lg:grid-cols-[minmax(0,1fr)_420px]">
				<div class="max-w-3xl">
					<div class="mb-6 flex flex-wrap items-center gap-3">
						<p class="section-kicker">Open-Source Scanning Platform</p>
						<Chip tone="muted" size="sm" class="landing-chip gap-1.5">
							<GithubIcon class="h-3.5 w-3.5" />
							Open Source
						</Chip>
					</div>

					<h1 class="h1-display landing-headline">
						Catch accessibility, performance, and security issues
						<span class="text-accent">before release day.</span>
					</h1>

					<p class="landing-subhead mt-6">
						{SITE.tagline}. StageFlow merges eight scanners into one operational flow, so you can
						move from URL to remediation with less friction.
					</p>

					<div class="mt-8 flex flex-wrap gap-3">
						<a
							href="/playground"
							class={cn(buttonVariants({ variant: 'default', size: 'lg' }), 'gap-2 px-7 text-base')}
						>
							Start scanning
							<ArrowRight class="h-4 w-4" />
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

					<div class="mt-8 flex flex-wrap items-center gap-x-6 gap-y-2">
						<span class="hero-preview-legend font-semibold">Powered by</span>
						{#each poweredBy as item (item.label)}
							{@const PoweredIcon = item.icon}
							<span class="hero-preview-legend flex items-center gap-1.5">
								<PoweredIcon class="h-3.5 w-3.5" />
								{item.label}
							</span>
						{/each}
					</div>

					<div class="metric-strip">
						{#each metrics as item (item.label)}
							<div class="metric-card">
								<p class="metric-value">{item.value}</p>
								<p class="metric-label">{item.label}</p>
							</div>
						{/each}
					</div>
				</div>

				<div class="space-y-4">
					<HeroScanPreview />
					<div class="hero-summary-card hidden xl:block">
						<p class="hero-summary-label">Latest run signal</p>
						<p class="hero-summary-value">94 / 100</p>
						<p class="text-ink-muted mt-2 text-sm leading-relaxed">
							Accessibility and security passed baseline. SEO flags reduced by 37% from the previous
							sweep.
						</p>
					</div>
				</div>
			</div>
		</div>
	</section>

	<div class="section-divider"></div>

	<section class="section-padding">
		<div class="container-width">
			<div class="grid gap-6 lg:grid-cols-[minmax(0,1fr)_320px] lg:items-end">
				<div class="max-w-3xl">
					<p class="section-kicker mb-2">Capabilities</p>
					<h2 class="h2-display">Eight scanners. One operational report.</h2>
				</div>
				<p class="text-ink-muted text-sm leading-relaxed lg:text-right">
					Choose scanners per run and keep output normalized in one report surface for engineers,
					PMs, and designers.
				</p>
			</div>

			<div class="mt-12 grid gap-5 md:grid-cols-2 xl:grid-cols-3">
				{#each scanners as scanner, index (scanner.name)}
					{@const ScannerIcon = scanner.icon}
					<article class={cn('editorial-card scanner-card border-t-2 p-6', scanner.border)}>
						<div class="mb-5 flex items-start justify-between gap-4">
							<div class={cn('flex h-11 w-11 items-center justify-center rounded-xl', scanner.bg)}>
								<ScannerIcon class={cn('h-5 w-5', scanner.color)} />
							</div>
							<p class="hero-preview-legend">Scanner {index + 1}</p>
						</div>
						<h3 class="text-ink text-base font-semibold">{scanner.name}</h3>
						<p class="text-ink-muted mt-2 text-sm leading-relaxed">{scanner.desc}</p>
					</article>
				{/each}
			</div>
		</div>
	</section>

	<div class="section-divider"></div>

	<section class="workflow-band section-padding">
		<div class="container-width">
			<div class="mx-auto max-w-2xl text-center lg:text-left">
				<p class="section-kicker mb-2">Workflow</p>
				<h2 class="h2-display">From target input to a release-ready report</h2>
			</div>

			<div class="relative mt-12 grid gap-5 lg:grid-cols-3">
				<div class="workflow-line"></div>
				{#each workflowSteps as item (item.step)}
					<article class="workflow-step">
						<p class="workflow-index">{item.step}</p>
						<h3 class="text-ink mt-3 text-lg font-semibold">{item.title}</h3>
						<p class="text-ink-muted mt-2 text-sm leading-relaxed">{item.desc}</p>
						<p class="text-accent-deep mt-4 text-xs leading-relaxed font-medium">{item.signal}</p>
					</article>
				{/each}
			</div>
		</div>
	</section>

	<div class="section-divider"></div>

	<section class="section-padding">
		<div class="container-width">
			<div class="grid gap-8 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)] lg:items-start">
				<div class="max-w-2xl">
					<p class="section-kicker mb-2">Why StageFlow</p>
					<h2 class="h2-display">Built for teams that ship under real constraints</h2>
					<p class="text-ink-muted mt-4 text-base leading-relaxed">
						StageFlow was designed for operators who need fast confidence without handing
						infrastructure control to third parties.
					</p>
				</div>
				<div class="grid gap-4">
					{#each differentiators as item (item.title)}
						<article class="editorial-card p-5">
							<div class="flex gap-3">
								<div
									class="bg-accent/10 text-accent mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg"
								>
									<CheckCircle class="h-4 w-4" />
								</div>
								<div>
									<h3 class="text-ink text-sm font-semibold">{item.title}</h3>
									<p class="text-ink-muted mt-1 text-sm leading-relaxed">{item.desc}</p>
								</div>
							</div>
						</article>
					{/each}
				</div>
			</div>
		</div>
	</section>

	<div class="section-divider"></div>

	<section class="pt-16 pb-24">
		<div class="container-width">
			<div class="cta-panel mx-auto max-w-4xl">
				<p class="section-kicker mb-3">Ready to Run</p>
				<h2 class="h2-display text-3xl sm:text-4xl">Audit your site with one scan pipeline</h2>
				<p class="text-ink-muted mx-auto mt-3 max-w-2xl text-sm leading-relaxed sm:text-base">
					No account required. Open the playground, choose your scanners, and start a complete run
					in under a minute.
				</p>
				<div class="mt-7">
					<a
						href="/playground"
						class={cn(buttonVariants({ variant: 'glow', size: 'lg' }), 'gap-2 rounded-full px-7')}
					>
						Open Playground
						<ArrowRight class="h-4 w-4" />
					</a>
				</div>
			</div>
		</div>
	</section>
</div>
