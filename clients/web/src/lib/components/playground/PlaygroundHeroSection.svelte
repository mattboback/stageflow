<script lang="ts">
	import { CheckCircle2, Clock, Radar } from 'lucide-svelte';
</script>

<section class="playground-hero border-line relative overflow-hidden border-b">
	<div class="container-width relative pt-24 pb-12 lg:pb-14">
		<div class="max-w-3xl">
			<p class="section-kicker mb-4">Playground</p>
			<h1 class="h1-display max-w-2xl">Test your site&apos;s quality</h1>
			<p class="text-ink-muted mt-4 max-w-2xl text-lg leading-relaxed">
				Run accessibility, performance, security, and SEO scans from one control surface with
				actionable output.
			</p>
		</div>

		<div class="mt-7 grid gap-3 sm:grid-cols-3">
			<div class="glass-panel flex items-center gap-2.5 rounded-xl px-4 py-3.5">
				<Radar class="text-accent h-4 w-4 shrink-0" />
				<span class="text-sm font-medium"
					><span class="text-accent font-extrabold">8</span> Scanners</span
				>
			</div>
			<div class="glass-panel flex items-center gap-2.5 rounded-xl px-4 py-3.5">
				<Clock class="text-accent h-4 w-4 shrink-0" />
				<span class="text-sm font-medium"
					><span class="text-accent font-extrabold">&lt;30s</span> Results</span
				>
			</div>
			<div class="glass-panel flex items-center gap-2.5 rounded-xl px-4 py-3.5">
				<CheckCircle2 class="text-accent h-4 w-4 shrink-0" />
				<span class="text-sm font-medium">No account needed</span>
			</div>
		</div>
	</div>
</section>
