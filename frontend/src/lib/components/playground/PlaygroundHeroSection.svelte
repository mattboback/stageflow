<script lang="ts">
	import { CheckCircle2, Clock, Radar } from 'lucide-svelte';
</script>

<section class="playground-hero relative overflow-hidden border-b border-line/80">
	<div
		class="pointer-events-none absolute inset-x-0 top-0 h-64 bg-gradient-to-b from-accent/8 via-orange-100/25 to-transparent"
	></div>
	<div class="pointer-events-none absolute -top-16 right-6 h-40 w-40 rounded-full bg-accent/10 blur-2xl"></div>
	<div class="pointer-events-none absolute top-16 -left-8 h-32 w-32 rounded-full bg-orange-200/35 blur-2xl"></div>

	<div class="container-width relative pt-24 pb-12 lg:pb-14">
		<div class="max-w-3xl">
			<p class="text-accent mb-4 text-sm font-semibold tracking-[0.12em] uppercase">Playground</p>
			<h1 class="h1-display max-w-2xl">Test your site&apos;s quality</h1>
			<p class="text-ink-muted mt-4 max-w-2xl text-lg leading-relaxed">
				Run accessibility, performance, security, and SEO scans from one control surface with
				actionable output.
			</p>
		</div>

		<div class="mt-7 grid gap-3 sm:grid-cols-3">
			<div class="glass-panel flex items-center gap-2.5 rounded-xl px-4 py-3">
				<Radar class="text-accent h-4 w-4 shrink-0" />
				<span class="text-sm font-medium">6 Scanners</span>
			</div>
			<div class="glass-panel flex items-center gap-2.5 rounded-xl px-4 py-3">
				<Clock class="text-accent h-4 w-4 shrink-0" />
				<span class="text-sm font-medium">&lt;30s Results</span>
			</div>
			<div class="glass-panel flex items-center gap-2.5 rounded-xl px-4 py-3">
				<CheckCircle2 class="text-accent h-4 w-4 shrink-0" />
				<span class="text-sm font-medium">No account needed</span>
			</div>
		</div>
	</div>
</section>
