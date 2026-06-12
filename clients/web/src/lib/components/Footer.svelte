<script lang="ts">
	import { GithubIcon, LogoMark } from '$lib/components/icons';
	import { SITE } from '$lib/config/site';

	const year = new Date().getFullYear();
</script>

<footer class="border-line bg-paper border-t">
	<div
		class="container-width flex flex-col gap-4 py-8 sm:flex-row sm:items-center sm:justify-between"
	>
		<div class="flex flex-col gap-1.5">
			<a href="/" class="text-ink inline-flex items-center gap-2">
				<LogoMark class="text-accent" size={18} />
				<span class="font-display text-[16px] leading-none font-semibold">{SITE.name}</span>
			</a>
			<p class="text-ink-faint text-xs">
				&copy; {year} StageFlow. Open-source, operator-first, no vendor lock-in.
			</p>
		</div>

		<nav class="flex items-center gap-5" aria-label="Footer">
			<a href="/playground" class="text-ink-muted hover:text-ink text-sm font-medium">Playground</a>
			<a
				href={`${SITE.githubUrl}/blob/main/README.md`}
				target="_blank"
				rel="noopener noreferrer"
				class="text-ink-muted hover:text-ink text-sm font-medium"
			>
				Docs
			</a>
			<a
				href={SITE.githubUrl}
				target="_blank"
				rel="noopener noreferrer"
				class="text-ink-muted hover:text-ink inline-flex items-center gap-1.5 text-sm font-medium"
			>
				<GithubIcon class="h-4 w-4" />
				<span>GitHub</span>
			</a>
		</nav>
	</div>
</footer>
