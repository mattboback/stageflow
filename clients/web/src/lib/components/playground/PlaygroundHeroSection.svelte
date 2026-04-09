<script lang="ts">
	import { CheckCircle2, Clock, Radar } from 'lucide-svelte';
</script>

<section class="playground-hero border-line relative overflow-hidden border-b">
	<div class="container-width relative pt-22 pb-8 lg:pt-22 lg:pb-10">
		<div class="max-w-3xl">
			<p class="section-kicker mb-3">Playground</p>
			<h1 class="h1-display max-w-2xl text-[clamp(3rem,6vw,4.5rem)] leading-[0.94]">
				Test your site&apos;s quality
			</h1>
			<p class="text-ink-muted mt-3 max-w-2xl text-base leading-relaxed sm:text-lg">
				Run accessibility, performance, security, and SEO scans from one control surface with
				actionable output.
			</p>
		</div>

		<div class="mt-6 grid gap-3 sm:grid-cols-3">
			<div class="glass-panel flex items-center gap-2.5 rounded-2xl px-4 py-3">
				<Radar class="text-accent h-4 w-4 shrink-0" />
				<span class="text-sm font-medium"
					><span class="text-accent font-extrabold">8</span> Scanners</span
				>
			</div>
			<div class="glass-panel flex items-center gap-2.5 rounded-2xl px-4 py-3">
				<Clock class="text-accent h-4 w-4 shrink-0" />
				<span class="text-sm font-medium"
					><span class="text-accent font-extrabold">&lt;30s</span> Results</span
				>
			</div>
			<div class="glass-panel flex items-center gap-2.5 rounded-2xl px-4 py-3">
				<CheckCircle2 class="text-accent h-4 w-4 shrink-0" />
				<span class="text-sm font-medium">No account needed</span>
			</div>
		</div>
	</div>
</section>
