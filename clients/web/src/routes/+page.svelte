<script lang="ts">
	import GithubIcon from '$lib/components/icons/GithubIcon.svelte';
	import NexusDashboardHero from '$lib/components/landing/NexusDashboardHero.svelte';
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
			bg: 'bg-blue-50/50',
			border: 'border-t-blue-500',
			glow: 'rgba(59, 130, 246, 0.15)',
			brand: '#3b82f6'
		},
		{
			icon: Zap,
			name: 'Lighthouse',
			desc: 'Performance, SEO, and best-practices scoring from the same run pipeline.',
			color: 'text-amber-600',
			bg: 'bg-amber-50/50',
			border: 'border-t-amber-500',
			glow: 'rgba(245, 158, 11, 0.15)',
			brand: '#f59e0b'
		},
		{
			icon: Search,
			name: 'SEO',
			desc: 'Meta, headings, structured data, and crawlability checks surfaced in one report.',
			color: 'text-rose-600',
			bg: 'bg-rose-50/50',
			border: 'border-t-rose-500',
			glow: 'rgba(244, 63, 94, 0.15)',
			brand: '#f43f5e'
		},
		{
			icon: Shield,
			name: 'Security Headers',
			desc: 'CSP, HSTS, X-Frame-Options, and critical header posture scoring.',
			color: 'text-violet-600',
			bg: 'bg-violet-50/50',
			border: 'border-t-violet-500',
			glow: 'rgba(139, 92, 246, 0.15)',
			brand: '#8b5cf6'
		},
		{
			icon: Link2,
			name: 'Link Checker',
			desc: 'Broken links, redirect chains, and dead-end paths caught before release.',
			color: 'text-emerald-600',
			bg: 'bg-emerald-50/50',
			border: 'border-t-emerald-500',
			glow: 'rgba(16, 185, 129, 0.15)',
			brand: '#10b981'
		},
		{
			icon: Bot,
			name: 'AI Navigator',
			desc: 'Goal-based journey simulation that catches experience breakpoints early.',
			color: 'text-fuchsia-600',
			bg: 'bg-fuchsia-50/50',
			border: 'border-t-fuchsia-500',
			glow: 'rgba(217, 70, 239, 0.15)',
			brand: '#d946ef'
		},
		{
			icon: Search,
			name: 'Open Graph',
			desc: 'Social preview metadata validation for Open Graph tags, Twitter cards, and share readiness.',
			color: 'text-cyan-600',
			bg: 'bg-cyan-50/50',
			border: 'border-t-cyan-500',
			glow: 'rgba(6, 182, 212, 0.15)',
			brand: '#06b6d4'
		},
		{
			icon: CheckCircle,
			name: 'Spelling & Grammar',
			desc: 'AI-powered spelling, grammar, and content quality checks for polished publication-ready copy.',
			color: 'text-lime-700',
			bg: 'bg-lime-50/50',
			border: 'border-t-lime-500',
			glow: 'rgba(132, 204, 22, 0.15)',
			brand: '#84cc16'
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
	<!-- Hero Section -->
	<section class="landing-hero relative overflow-hidden pt-24 pb-8 sm:pt-28 lg:pt-32">
		<!-- Cybernetic Backdrop Auroras -->
		<div
			class="from-accent/15 pointer-events-none absolute -top-40 -left-40 z-0 h-96 w-96 rounded-full bg-radial to-transparent blur-3xl"
		></div>
		<div
			class="pointer-events-none absolute top-20 right-[-10%] z-0 h-[500px] w-[500px] rounded-full bg-radial from-blue-500/10 to-transparent blur-3xl"
		></div>
		<div
			class="pointer-events-none absolute bottom-[-10%] left-[30%] z-0 h-[450px] w-[450px] rounded-full bg-radial from-fuchsia-500/8 to-transparent blur-3xl"
		></div>

		<div class="container-width relative z-10">
			<!-- Centered headline + CTAs -->
			<div class="mx-auto max-w-3xl text-center">
				<div class="mb-5 flex flex-wrap items-center justify-center gap-3">
					<p class="section-kicker">Open-Source Scanning Platform</p>
					<Chip
						tone="muted"
						size="sm"
						class="landing-chip hover:bg-surface-muted/85 cursor-pointer gap-1.5 transition-colors"
					>
						<GithubIcon class="h-3.5 w-3.5" />
						Open Source
					</Chip>
				</div>

				<h1 class="h1-display landing-headline">
					Catch accessibility, performance, and security issues
					<span class="text-accent">before release day.</span>
				</h1>

				<p class="landing-subhead mx-auto mt-5">
					{SITE.tagline}. StageFlow merges eight scanners into one operational flow, so you can move
					from URL to remediation with less friction.
				</p>

				<div class="mt-7 flex flex-wrap justify-center gap-3">
					<a
						href="/playground"
						class={cn(
							buttonVariants({ variant: 'default', size: 'lg' }),
							'gap-2 px-7 text-base transition-transform hover:scale-102 hover:shadow-[0_0_15px_rgba(13,92,99,0.15)]'
						)}
					>
						Start scanning
						<ArrowRight class="h-4 w-4" />
					</a>
					<a
						href={SITE.githubUrl}
						target="_blank"
						rel="noopener noreferrer"
						class={cn(
							buttonVariants({ variant: 'outline', size: 'lg' }),
							'gap-2 px-7 text-base transition-transform hover:scale-102'
						)}
					>
						<GithubIcon class="h-4 w-4" />
						View source
					</a>
				</div>

				<div class="mt-6 flex flex-wrap items-center justify-center gap-x-6 gap-y-2">
					<span class="hero-preview-legend font-semibold">Powered by</span>
					{#each poweredBy as item (item.label)}
						{@const PoweredIcon = item.icon}
						<span class="hero-preview-legend flex items-center gap-1.5">
							<PoweredIcon class="h-3.5 w-3.5" />
							{item.label}
						</span>
					{/each}
				</div>
			</div>

			<!-- Full-width Nexus bento dashboard card -->
			<div class="mt-12">
				<NexusDashboardHero />
			</div>
		</div>
	</section>

	<div class="section-divider"></div>

	<!-- Capabilities Section -->
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

			<div class="mt-12 grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
				{#each scanners as scanner, index (scanner.name)}
					{@const ScannerIcon = scanner.icon}
					<article
						class="editorial-card scanner-card group border-line bg-surface relative overflow-hidden rounded-2xl border p-6 transition-all duration-300 hover:-translate-y-1 hover:scale-102"
						style="--glow-color: {scanner.glow}; --brand-color: {scanner.brand}; border-top-width: 4px;"
					>
						<div class="mb-4 flex items-start justify-between gap-4">
							<div
								class={cn(
									'flex h-12 w-12 items-center justify-center rounded-xl shadow-sm transition-all duration-300 group-hover:scale-110',
									scanner.bg
								)}
							>
								<ScannerIcon
									class={cn('h-5.5 w-5.5 transition-colors duration-300', scanner.color)}
								/>
							</div>
							<span
								class={cn(
									'rounded-full px-2 py-0.5 text-[10px] font-semibold tabular-nums',
									scanner.bg,
									scanner.color
								)}
							>
								{String(index + 1).padStart(2, '0')}
							</span>
						</div>
						<h3 class="text-ink-strong text-base font-semibold tracking-[-0.01em]">
							{scanner.name}
						</h3>
						<p class="text-ink-muted mt-2 text-sm leading-relaxed">{scanner.desc}</p>
					</article>
				{/each}
			</div>
		</div>
	</section>

	<div class="section-divider"></div>

	<!-- Workflow Section -->
	<section class="workflow-band section-padding">
		<div class="container-width">
			<div class="mx-auto max-w-2xl text-center lg:text-left">
				<p class="section-kicker mb-2">Workflow</p>
				<h2 class="h2-display">From target input to a release-ready report</h2>
			</div>

			<div class="relative mt-12 grid gap-5 lg:grid-cols-3">
				<div class="workflow-line"></div>
				{#each workflowSteps as item (item.step)}
					<article class="workflow-step group transition-all duration-300 hover:-translate-y-1">
						<div class="mb-5 flex items-center gap-2">
							<span
								class="bg-accent/10 text-accent group-hover:bg-accent group-hover:text-surface inline-flex h-7 w-7 items-center justify-center rounded-lg font-mono text-xs font-semibold transition-all duration-300"
								>{item.step}</span
							>
							<div class="bg-line h-px flex-1 opacity-60"></div>
						</div>
						<h3 class="text-ink mt-3 text-lg font-semibold">{item.title}</h3>
						<p class="text-ink-muted mt-2 text-sm leading-relaxed">{item.desc}</p>
						<p class="text-accent-deep mt-4 text-xs leading-relaxed font-medium">{item.signal}</p>
					</article>
				{/each}
			</div>
		</div>
	</section>

	<div class="section-divider"></div>

	<!-- Why StageFlow Section -->
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
						<article class="editorial-card p-6 transition-all duration-300 hover:-translate-y-0.5">
							<div class="flex gap-4">
								<div
									class="bg-accent/12 text-accent mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-xl transition-transform duration-300 group-hover:scale-105"
								>
									<CheckCircle class="h-4.5 w-4.5" />
								</div>
								<div>
									<h3 class="text-ink-strong text-sm font-semibold tracking-[-0.01em]">
										{item.title}
									</h3>
									<p class="text-ink-muted mt-1.5 text-sm leading-relaxed">{item.desc}</p>
								</div>
							</div>
						</article>
					{/each}
				</div>
			</div>
		</div>
	</section>

	<div class="section-divider"></div>

	<!-- CTA Section -->
	<section class="pt-16 pb-24">
		<div class="container-width">
			<div
				class="cta-panel border-accent/20 relative mx-auto max-w-4xl overflow-hidden rounded-3xl border px-8 py-14 text-center shadow-2xl sm:px-12"
			>
				<!-- Custom CTA Backdrop Auroras -->
				<div
					class="from-accent/35 pointer-events-none absolute -top-32 -right-32 h-80 w-80 rounded-full bg-radial to-transparent blur-3xl"
				></div>
				<div
					class="pointer-events-none absolute -bottom-32 -left-32 h-80 w-80 rounded-full bg-radial from-fuchsia-500/20 to-transparent blur-3xl"
				></div>

				<div class="relative z-10">
					<p class="section-kicker text-accent-subtle mb-3">Ready to scan</p>
					<h2 class="h2-display text-surface text-2xl sm:text-3xl lg:text-4xl">
						Audit your site with one pipeline
					</h2>
					<p class="text-surface/70 mx-auto mt-3 max-w-2xl text-sm leading-relaxed sm:text-base">
						No account required. Open the playground, choose your scanners, and start a complete run
						in under a minute.
					</p>
					<div class="mt-7">
						<a
							href="/playground"
							class={cn(
								buttonVariants({ variant: 'outline', size: 'lg' }),
								'border-surface/40 text-surface hover:bg-surface/15 gap-2.5 rounded-xl px-8 text-base font-semibold transition-all duration-300 hover:scale-102 hover:shadow-[0_0_20px_rgba(255,255,255,0.15)]'
							)}
						>
							Open Playground
							<ArrowRight class="h-4 w-4" />
						</a>
					</div>
				</div>
			</div>
		</div>
	</section>
</div>
