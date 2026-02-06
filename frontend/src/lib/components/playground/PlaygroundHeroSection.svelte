<script lang="ts">
	import { Radar, Clock, CheckCircle2 } from 'lucide-svelte';
</script>

<section class="border-line border-b bg-surface-muted/50">
	<div class="container-width pt-24 pb-10">
		<p class="text-accent mb-3 text-sm font-semibold tracking-wide uppercase">Playground</p>
		<h1 class="h1-display max-w-2xl">
			Test your site's quality
		</h1>
		<p class="text-ink-muted mt-3 max-w-xl text-base leading-relaxed">
			Run accessibility, performance, security, and SEO scans. Get actionable results in seconds.
		</p>

		<div class="mt-6 flex flex-wrap items-center gap-5">
			<div class="text-ink-muted flex items-center gap-1.5 text-sm">
				<Radar class="text-accent h-4 w-4" />
				<span class="font-medium">6 Scanners</span>
			</div>
			<div class="text-ink-muted flex items-center gap-1.5 text-sm">
				<Clock class="text-accent h-4 w-4" />
				<span class="font-medium">&lt;30s Results</span>
			</div>
			<div class="text-ink-muted flex items-center gap-1.5 text-sm">
				<CheckCircle2 class="text-accent h-4 w-4" />
				<span class="font-medium">No account needed</span>
			</div>
		</div>
	</div>
</section>
