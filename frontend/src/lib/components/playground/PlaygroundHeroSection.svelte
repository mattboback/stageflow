<script lang="ts">
	import { Chip } from '$lib/components/ui';
	import { CheckCircle2, Clock, Radar } from 'lucide-svelte';
</script>

<section
	class="border-line from-surface to-paper relative overflow-hidden border-b bg-gradient-to-b"
>
	<!-- Animated grid background -->
	<div class="bg-grid-pattern absolute inset-0 opacity-50"></div>

	<!-- Accent glow -->
	<div
		class="bg-accent/10 pointer-events-none absolute -top-40 -right-40 h-80 w-80 rounded-full blur-3xl"
	></div>
	<div
		class="bg-accent/5 pointer-events-none absolute -bottom-20 -left-20 h-60 w-60 rounded-full blur-3xl"
	></div>

	<div class="container-width relative pt-24 pb-12">
		<!-- Terminal-style label -->
		<div class="mb-4 flex items-center gap-2">
			<span class="section-label-tech">Scan Engine</span>
			<Chip tone="success" size="xs" class="gap-1 font-medium">
				<span class="h-1.5 w-1.5 animate-pulse rounded-full bg-emerald-500"></span>
				Online
			</Chip>
		</div>

		<h1 class="h1-display mb-4 max-w-3xl">
			Test your website's <span class="text-gradient">accessibility & performance</span>
		</h1>

		<p class="text-ink-muted max-w-2xl text-lg leading-relaxed">
			Run comprehensive scans to identify accessibility issues, performance bottlenecks, broken
			links, and security vulnerabilities. Get actionable insights in seconds.
		</p>

		<!-- Quick stats -->
		<div class="mt-8 flex flex-wrap items-center gap-6">
			<div class="text-ink-muted flex items-center gap-2 text-sm">
				<Radar class="text-accent h-4 w-4" />
				<span class="stat-mono font-medium">5 Scanners</span>
			</div>
			<div class="text-ink-muted flex items-center gap-2 text-sm">
				<Clock class="text-accent h-4 w-4" />
				<span class="stat-mono font-medium">&lt;30s Results</span>
			</div>
			<div class="text-ink-muted flex items-center gap-2 text-sm">
				<CheckCircle2 class="text-accent h-4 w-4" />
				<span class="stat-mono font-medium">100% Free</span>
			</div>
		</div>
	</div>
</section>
