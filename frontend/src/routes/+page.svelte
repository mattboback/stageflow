<script lang="ts">
	import GithubIcon from '$lib/components/icons/GithubIcon.svelte';
	import HeroScanPreview from '$lib/components/landing/HeroScanPreview.svelte';
	import { buttonVariants, Chip } from '$lib/components/ui';
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
</script>

<svelte:head>
	<title>{SITE.siteTitle}</title>
	<meta name="description" content={SITE.tagline} />
</svelte:head>

<div class="bg-paper min-h-screen">
	<!-- Hero -->
	<section class="landing-hero pb-20 pt-28">
		<div class="container-width">
			<div class="grid items-center gap-12 lg:grid-cols-[1fr_auto]">
				<div class="mx-auto max-w-3xl lg:mx-0">
					<div class="mb-5 flex flex-wrap items-center gap-3">
						<p class="section-kicker">Open-Source Scanning Platform</p>
						<Chip tone="muted" size="sm" class="gap-1.5">
							<GithubIcon class="h-3.5 w-3.5" />
							Open Source
						</Chip>
					</div>
					<h1 class="h1-display max-w-3xl">
						Find accessibility, performance, and security issues
						<span class="text-accent">before your users do.</span>
					</h1>
					<p class="text-ink-muted mt-6 max-w-xl text-lg leading-relaxed">
						{SITE.tagline}. Six scanners, real-time progress, unified reports — self-hosted on your infrastructure.
					</p>
					<div class="mt-8 flex flex-wrap gap-3">
						<a
							href="/playground"
							class={cn(buttonVariants({ variant: 'default', size: 'lg' }), 'gap-2 text-base px-7')}
						>
							Start scanning
							<ArrowRight class="h-4 w-4" />
						</a>
						<a
							href={SITE.githubUrl}
							target="_blank"
							rel="noopener noreferrer"
							class={cn(buttonVariants({ variant: 'outline', size: 'lg' }), 'gap-2 text-base px-7')}
						>
							<GithubIcon class="h-4 w-4" />
							View source
						</a>
					</div>
					<div class="mt-8 flex flex-wrap items-center gap-x-6 gap-y-2 text-ink-faint text-xs">
						<span class="font-medium uppercase tracking-wider">Powered by</span>
						<span class="flex items-center gap-1.5"><Box class="h-3.5 w-3.5" /> Podman</span>
						<span class="flex items-center gap-1.5"><ShieldCheck class="h-3.5 w-3.5" /> axe-core</span>
						<span class="flex items-center gap-1.5"><Gauge class="h-3.5 w-3.5" /> Lighthouse</span>
						<span class="flex items-center gap-1.5"><Shield class="h-3.5 w-3.5" /> WCAG 2.1</span>
					</div>
				</div>

				<HeroScanPreview />
			</div>
		</div>
	</section>

	<div class="section-divider"></div>

	<!-- Scanners -->
	<section class="py-20">
		<div class="container-width">
			<div class="mx-auto max-w-2xl">
				<p class="section-kicker mb-2">Capabilities</p>
				<h2 class="h2-display">Six scanners, one report</h2>
				<p class="text-ink-muted mt-3 text-base">
					Each scan runs the checks you select and merges results into a single, actionable report.
				</p>
			</div>

			<div class="mt-12 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
				{#each [
					{ icon: Shield, name: 'Axe', desc: 'WCAG 2.1 AA/AAA accessibility testing with axe-core', color: 'text-blue-600', border: 'border-t-blue-500', bg: 'bg-blue-50' },
					{ icon: Zap, name: 'Lighthouse', desc: 'Performance, SEO, and best practices audits by Google', color: 'text-amber-500', border: 'border-t-amber-500', bg: 'bg-amber-50' },
					{ icon: Search, name: 'SEO', desc: 'Meta tags, headings, structured data, and crawlability', color: 'text-rose-500', border: 'border-t-rose-500', bg: 'bg-rose-50' },
					{ icon: Shield, name: 'Security Headers', desc: 'CSP, HSTS, X-Frame-Options, and header scoring', color: 'text-violet-600', border: 'border-t-violet-600', bg: 'bg-violet-50' },
					{ icon: Link2, name: 'Link Checker', desc: 'Broken links, redirect chains, and orphaned pages', color: 'text-emerald-600', border: 'border-t-emerald-600', bg: 'bg-emerald-50' },
					{ icon: Bot, name: 'AI Navigator', desc: 'LLM-powered agent that tests goal-based user flows', color: 'text-purple-600', border: 'border-t-purple-600', bg: 'bg-purple-50' }
				] as scanner (scanner.name)}
					{@const ScanIcon = scanner.icon}
					<div class={cn('scanner-card rounded-xl border border-line border-t-2 bg-surface p-6', scanner.border)}>
						<div class="flex gap-4">
							<div class={cn('flex h-10 w-10 shrink-0 items-center justify-center rounded-lg', scanner.bg)}>
								<ScanIcon class={cn('h-5 w-5', scanner.color)} />
							</div>
							<div>
								<p class="text-ink text-sm font-bold">{scanner.name}</p>
								<p class="text-ink-muted mt-1 text-sm leading-relaxed">{scanner.desc}</p>
							</div>
						</div>
					</div>
				{/each}
			</div>
		</div>
	</section>

	<div class="section-divider"></div>

	<!-- How it works -->
	<section class="bg-accent-mist py-20">
		<div class="container-width">
			<div class="mx-auto max-w-2xl">
				<p class="section-kicker mb-2">Workflow</p>
				<h2 class="h2-display">From URL to report in seconds</h2>
			</div>
			<div class="relative mx-auto mt-12 grid max-w-3xl gap-8 sm:grid-cols-3">
				<!-- Dashed connector between steps (sm+ only) -->
				<div class="pointer-events-none absolute top-5 right-0 left-0 hidden sm:block">
					<div class="mx-auto h-px w-2/3 border-t border-dashed border-accent/20"></div>
				</div>
				{#each [
					{ step: '1', title: 'Configure', desc: 'Enter URLs or upload a ZIP archive. Select the scanners you need.' },
					{ step: '2', title: 'Scan', desc: 'Watch real-time progress via SSE. Each scanner runs in an isolated container.' },
					{ step: '3', title: 'Review', desc: 'Get a unified report with issues, screenshots, remediation tips, and WCAG references.' }
				] as item (item.step)}
					<div class="relative text-center sm:text-left">
						<div class="bg-accent text-white mx-auto mb-4 flex h-10 w-10 items-center justify-center rounded-full text-base font-bold sm:mx-0">
							{item.step}
						</div>
						<h3 class="text-ink text-base font-semibold">{item.title}</h3>
						<p class="text-ink-muted mt-2 text-sm leading-relaxed">{item.desc}</p>
					</div>
				{/each}
			</div>
		</div>
	</section>

	<div class="section-divider"></div>

	<!-- Why -->
	<section class="py-20">
		<div class="container-width">
			<div class="mx-auto max-w-2xl">
				<p class="section-kicker mb-2">Why StageFlow</p>
				<h2 class="h2-display">Built for teams that ship</h2>
			</div>
			<div class="mx-auto mt-12 grid max-w-3xl gap-6 sm:grid-cols-2">
				{#each [
					{ title: 'Self-hosted', desc: 'Runs on your infra with Podman. No data leaves your network.' },
					{ title: 'Container-isolated', desc: 'Every scanner runs in its own rootless container. Clean, reproducible, secure.' },
					{ title: 'Real-time streaming', desc: 'SSE-powered live progress. Know exactly what\u2019s happening and when.' },
					{ title: 'Unified reports', desc: 'All scanner results merged into one report with severity, WCAG mapping, and fix guidance.' }
				] as feature (feature.title)}
					<div class="rounded-xl border border-line bg-surface p-5">
						<div class="flex gap-3">
							<div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-accent/10">
								<CheckCircle class="text-accent h-4 w-4" />
							</div>
							<div>
								<p class="text-ink text-sm font-bold">{feature.title}</p>
								<p class="text-ink-muted mt-1 text-sm leading-relaxed">{feature.desc}</p>
							</div>
						</div>
					</div>
				{/each}
			</div>
		</div>
	</section>

	<div class="section-divider"></div>

	<!-- CTA -->
	<section class="py-16">
		<div class="container-width">
			<div class="mx-auto max-w-3xl rounded-2xl bg-accent-soft px-8 py-12 text-center">
				<h2 class="h3-display">Ready to scan your site?</h2>
				<p class="text-ink-muted mt-2 text-sm">No account needed. Run your first scan in under a minute.</p>
				<div class="mt-6">
					<a
						href="/playground"
						class={cn(buttonVariants({ variant: 'glow', size: 'lg' }), 'gap-2')}
					>
						Open Playground
						<ArrowRight class="h-4 w-4" />
					</a>
				</div>
			</div>
		</div>
	</section>
</div>
