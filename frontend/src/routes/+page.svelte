<script lang="ts">
	import { buttonVariants } from '$lib/components/ui';
	import { SITE } from '$lib/config/site';
	import { cn } from '$lib/utils';
	import { ArrowRight, CheckCircle, Zap, Shield, Layers } from 'lucide-svelte';
</script>

<svelte:head>
	<title>{SITE.siteTitle}</title>
	<meta name="description" content={SITE.tagline} />
</svelte:head>

<div class="bg-paper min-h-screen">
	<!-- Hero Section -->
	<section class="container-width pt-32 pb-20">
		<div class="mx-auto max-w-3xl text-center">
			<h1 class="text-ink text-4xl font-bold tracking-tight sm:text-5xl lg:text-6xl">
				Web Quality Scanning
				<span class="text-accent block">Made Simple</span>
			</h1>
			<p class="text-ink-muted mt-6 text-lg leading-relaxed sm:text-xl">
				{SITE.tagline}. Run accessibility, performance, SEO, and security scans with unified reporting and real-time progress.
			</p>
			<div class="mt-10 flex flex-col items-center justify-center gap-4 sm:flex-row">
				<a
					href="/playground"
					class={cn(buttonVariants({ variant: 'default', size: 'lg' }), 'gap-2')}
				>
					Try the Playground
					<ArrowRight class="h-4 w-4" />
				</a>
				<a
					href={SITE.githubUrl}
					target="_blank"
					rel="noopener noreferrer"
					class={cn(buttonVariants({ variant: 'outline', size: 'lg' }))}
				>
					View on GitHub
				</a>
			</div>
		</div>
	</section>

	<!-- Features Section -->
	<section class="border-line border-t py-20">
		<div class="container-width">
			<h2 class="text-ink text-center text-2xl font-bold sm:text-3xl">Why StageFlow?</h2>
			<div class="mt-12 grid gap-8 sm:grid-cols-2 lg:grid-cols-3">
				<div class="bg-surface border-line rounded-xl border p-6">
					<div class="bg-accent-soft text-accent-ink mb-4 inline-flex rounded-lg p-3">
						<Layers class="h-6 w-6" />
					</div>
					<h3 class="text-ink text-lg font-semibold">Multi-Scanner</h3>
					<p class="text-ink-muted mt-2">
						Run Axe, Lighthouse, SEO checks, security headers, and link validation in one scan with unified reporting.
					</p>
				</div>
				<div class="bg-surface border-line rounded-xl border p-6">
					<div class="bg-accent-soft text-accent-ink mb-4 inline-flex rounded-lg p-3">
						<Zap class="h-6 w-6" />
					</div>
					<h3 class="text-ink text-lg font-semibold">Real-Time Progress</h3>
					<p class="text-ink-muted mt-2">
						Watch scans progress live with SSE streaming. No more waiting in the dark.
					</p>
				</div>
				<div class="bg-surface border-line rounded-xl border p-6">
					<div class="bg-accent-soft text-accent-ink mb-4 inline-flex rounded-lg p-3">
						<Shield class="h-6 w-6" />
					</div>
					<h3 class="text-ink text-lg font-semibold">Self-Hosted</h3>
					<p class="text-ink-muted mt-2">
						Podman-native, rootless containers. Your data stays on your infrastructure.
					</p>
				</div>
			</div>
		</div>
	</section>

	<!-- Scanners Section -->
	<section class="border-line border-t py-20">
		<div class="container-width">
			<h2 class="text-ink text-center text-2xl font-bold sm:text-3xl">Available Scanners</h2>
			<div class="mt-12 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
				{#each [
					{ name: 'Axe', desc: 'WCAG accessibility compliance' },
					{ name: 'Lighthouse', desc: 'Performance & best practices' },
					{ name: 'SEO', desc: 'Meta tags, headings, structured data' },
					{ name: 'Security Headers', desc: 'CSP, HSTS, X-Frame-Options' },
					{ name: 'Link Checker', desc: 'Broken link detection' },
					{ name: 'AI Navigator', desc: 'Goal-based flow testing' }
				] as scanner (scanner.name)}
					<div class="border-line flex items-start gap-3 rounded-lg border p-4">
						<CheckCircle class="text-accent mt-0.5 h-5 w-5 flex-shrink-0" />
						<div>
							<p class="text-ink font-medium">{scanner.name}</p>
							<p class="text-ink-muted text-sm">{scanner.desc}</p>
						</div>
					</div>
				{/each}
			</div>
		</div>
	</section>

	<!-- CTA Section -->
	<section class="border-line border-t py-20">
		<div class="container-width text-center">
			<h2 class="text-ink text-2xl font-bold sm:text-3xl">Ready to scan?</h2>
			<p class="text-ink-muted mt-4">
				Try the playground or deploy your own instance.
			</p>
			<div class="mt-8">
				<a
					href="/playground"
					class={cn(buttonVariants({ variant: 'default', size: 'lg' }), 'gap-2')}
				>
					Open Playground
					<ArrowRight class="h-4 w-4" />
				</a>
			</div>
		</div>
	</section>
</div>
