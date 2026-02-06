<script lang="ts">
	import { buttonVariants } from '$lib/components/ui';
	import { SITE } from '$lib/config/site';
	import { cn } from '$lib/utils';
	import {
		ArrowRight,
		Bot,
		CheckCircle,
		Link2,
		Search,
		Shield,
		Zap
	} from 'lucide-svelte';
</script>

<svelte:head>
	<title>{SITE.siteTitle}</title>
	<meta name="description" content={SITE.tagline} />
</svelte:head>

<div class="bg-paper min-h-screen">
	<!-- Hero -->
	<section class="container-width pb-20 pt-28">
		<div class="mx-auto max-w-2xl">
			<p class="text-accent mb-4 text-sm font-semibold tracking-wide uppercase">
				Open-Source Scanning Platform
			</p>
			<h1 class="text-ink text-4xl font-bold tracking-tight sm:text-5xl lg:text-[3.5rem] lg:leading-[1.1]">
				Find accessibility, performance, and security issues
				<span class="text-ink-muted">before your users do.</span>
			</h1>
			<p class="text-ink-muted mt-6 max-w-xl text-lg leading-relaxed">
				{SITE.tagline}. Six scanners, real-time progress, unified reports — self-hosted on your infrastructure.
			</p>
			<div class="mt-8 flex flex-wrap gap-3">
				<a
					href="/playground"
					class={cn(buttonVariants({ variant: 'default', size: 'lg' }), 'gap-2')}
				>
					Start scanning
					<ArrowRight class="h-4 w-4" />
				</a>
				<a
					href={SITE.githubUrl}
					target="_blank"
					rel="noopener noreferrer"
					class={cn(buttonVariants({ variant: 'outline', size: 'lg' }))}
				>
					View source
				</a>
			</div>
		</div>
	</section>

	<!-- Scanners -->
	<section class="border-line border-t py-20">
		<div class="container-width">
			<div class="mx-auto max-w-2xl">
				<p class="text-accent mb-2 text-sm font-semibold tracking-wide uppercase">Capabilities</p>
				<h2 class="text-ink text-2xl font-bold tracking-tight sm:text-3xl">
					Six scanners, one report
				</h2>
				<p class="text-ink-muted mt-3 text-base">
					Each scan runs the checks you select and merges results into a single, actionable report.
				</p>
			</div>

			<div class="mt-12 grid gap-px overflow-hidden rounded-xl border border-line bg-line sm:grid-cols-2 lg:grid-cols-3">
				{#each [
					{ icon: Shield, name: 'Axe', desc: 'WCAG 2.1 AA/AAA accessibility testing with axe-core', color: 'text-blue-600' },
					{ icon: Zap, name: 'Lighthouse', desc: 'Performance, SEO, and best practices audits by Google', color: 'text-amber-500' },
					{ icon: Search, name: 'SEO', desc: 'Meta tags, headings, structured data, and crawlability', color: 'text-rose-500' },
					{ icon: Shield, name: 'Security Headers', desc: 'CSP, HSTS, X-Frame-Options, and header scoring', color: 'text-violet-600' },
					{ icon: Link2, name: 'Link Checker', desc: 'Broken links, redirect chains, and orphaned pages', color: 'text-emerald-600' },
					{ icon: Bot, name: 'AI Navigator', desc: 'LLM-powered agent that tests goal-based user flows', color: 'text-purple-600' }
				] as scanner (scanner.name)}
					<div class="bg-surface flex gap-4 p-6">
						<div class={cn('mt-0.5 flex-shrink-0', scanner.color)}>
							<svelte:component this={scanner.icon} class="h-5 w-5" />
						</div>
						<div>
							<p class="text-ink text-sm font-semibold">{scanner.name}</p>
							<p class="text-ink-muted mt-1 text-sm leading-relaxed">{scanner.desc}</p>
						</div>
					</div>
				{/each}
			</div>
		</div>
	</section>

	<!-- How it works -->
	<section class="border-line border-t py-20">
		<div class="container-width">
			<div class="mx-auto max-w-2xl">
				<p class="text-accent mb-2 text-sm font-semibold tracking-wide uppercase">Workflow</p>
				<h2 class="text-ink text-2xl font-bold tracking-tight sm:text-3xl">
					From URL to report in seconds
				</h2>
			</div>
			<div class="mx-auto mt-12 grid max-w-3xl gap-8 sm:grid-cols-3">
				{#each [
					{ step: '1', title: 'Configure', desc: 'Enter URLs or upload a ZIP archive. Select the scanners you need.' },
					{ step: '2', title: 'Scan', desc: 'Watch real-time progress via SSE. Each scanner runs in an isolated container.' },
					{ step: '3', title: 'Review', desc: 'Get a unified report with issues, screenshots, remediation tips, and WCAG references.' }
				] as item (item.step)}
					<div class="text-center sm:text-left">
						<div class="bg-accent text-white mx-auto mb-4 flex h-8 w-8 items-center justify-center rounded-full text-sm font-bold sm:mx-0">
							{item.step}
						</div>
						<h3 class="text-ink text-base font-semibold">{item.title}</h3>
						<p class="text-ink-muted mt-2 text-sm leading-relaxed">{item.desc}</p>
					</div>
				{/each}
			</div>
		</div>
	</section>

	<!-- Why -->
	<section class="border-line border-t py-20">
		<div class="container-width">
			<div class="mx-auto max-w-2xl">
				<p class="text-accent mb-2 text-sm font-semibold tracking-wide uppercase">Why StageFlow</p>
				<h2 class="text-ink text-2xl font-bold tracking-tight sm:text-3xl">
					Built for teams that ship
				</h2>
			</div>
			<div class="mx-auto mt-12 grid max-w-3xl gap-6 sm:grid-cols-2">
				{#each [
					{ title: 'Self-hosted', desc: 'Runs on your infra with Podman. No data leaves your network.' },
					{ title: 'Container-isolated', desc: 'Every scanner runs in its own rootless container. Clean, reproducible, secure.' },
					{ title: 'Real-time streaming', desc: 'SSE-powered live progress. Know exactly what\u2019s happening and when.' },
					{ title: 'Unified reports', desc: 'All scanner results merged into one report with severity, WCAG mapping, and fix guidance.' }
				] as feature (feature.title)}
					<div class="flex gap-3">
						<CheckCircle class="text-accent mt-0.5 h-5 w-5 flex-shrink-0" />
						<div>
							<p class="text-ink text-sm font-semibold">{feature.title}</p>
							<p class="text-ink-muted mt-1 text-sm leading-relaxed">{feature.desc}</p>
						</div>
					</div>
				{/each}
			</div>
		</div>
	</section>

	<!-- CTA -->
	<section class="border-line border-t py-16">
		<div class="container-width flex flex-col items-center justify-between gap-6 sm:flex-row">
			<div>
				<h2 class="text-ink text-xl font-bold">Ready to scan your site?</h2>
				<p class="text-ink-muted mt-1 text-sm">No account needed. Run your first scan in under a minute.</p>
			</div>
			<a
				href="/playground"
				class={cn(buttonVariants({ variant: 'default', size: 'lg' }), 'gap-2 shrink-0')}
			>
				Open Playground
				<ArrowRight class="h-4 w-4" />
			</a>
		</div>
	</section>
</div>
