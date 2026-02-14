<script lang="ts">
	import { GithubIcon, LogoMark } from '$lib/components/icons';
	import { SITE } from '$lib/config/site';

	const year = new Date().getFullYear();
</script>

<footer class="border-line border-t">
	<div class="container-width flex items-center justify-between py-6">
		<div class="flex items-center gap-4">
			<a href="/" class="text-ink-muted hover:text-ink flex items-center gap-1.5 text-sm font-medium">
				<LogoMark class="text-ink-faint" size={16} />
				{SITE.name}
			</a>
			<span class="text-ink-faint text-xs">&copy; {year}</span>
		</div>
		<div class="flex items-center gap-4">
			<a href="/playground" class="text-ink-faint hover:text-ink text-sm transition-colors">
				Playground
			</a>
			<a
				href={SITE.githubUrl}
				target="_blank"
				rel="noopener noreferrer"
				class="text-ink-faint hover:text-ink -m-2 rounded-md p-2"
				aria-label="GitHub"
			>
				<GithubIcon class="h-4 w-4" />
			</a>
		</div>
	</div>
</footer>
