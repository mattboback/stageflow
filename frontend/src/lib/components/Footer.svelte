<script lang="ts">
	import { GithubIcon } from '$lib/components/icons';
	import { SITE } from '$lib/config/site';
	import { Command } from 'lucide-svelte';

	const year = new Date().getFullYear();
</script>

<footer class="border-line bg-paper border-t">
	<div class="mx-auto max-w-7xl px-4 py-12 sm:px-6 lg:px-8">
		<div class="flex flex-col items-center gap-6 sm:flex-row sm:justify-between">
			<!-- Brand -->
			<a href="/" class="text-ink flex items-center gap-2 text-xl font-bold">
				<div class="bg-accent flex h-8 w-8 items-center justify-center rounded-lg text-white">
					<Command class="h-4 w-4" />
				</div>
				{SITE.name}
			</a>

			<!-- Links -->
			<nav class="flex items-center gap-6">
				<a href="/playground" class="text-ink-muted hover:text-ink text-sm transition-colors">
					Playground
				</a>
				<a
					href={SITE.githubUrl}
					target="_blank"
					rel="noopener noreferrer"
					class="text-ink-muted hover:text-ink flex items-center gap-2 text-sm transition-colors"
				>
					<GithubIcon class="h-5 w-5" />
					<span>GitHub</span>
				</a>
			</nav>
		</div>

		<div class="border-line mt-8 border-t pt-6">
			<p class="text-ink-muted text-center text-xs">
				&copy; {year} {SITE.name}. MIT License.
			</p>
		</div>
	</div>
</footer>
