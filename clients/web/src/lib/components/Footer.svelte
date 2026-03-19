<script lang="ts">
import { GithubIcon, LogoMark } from "$lib/components/icons";
import { SITE } from "$lib/config/site";

const year = new Date().getFullYear();
</script>

<footer class="footer-shell border-line/70 border-t">
	<div class="container-width py-10">
		<div class="grid gap-8 md:grid-cols-[minmax(0,1fr)_auto] md:items-end">
			<div class="max-w-xl">
				<a href="/" class="text-ink inline-flex items-center gap-2.5">
					<LogoMark class="text-accent" size={18} />
					<span class="footer-title">{SITE.name}</span>
				</a>
				<p class="text-ink-muted mt-3 text-sm leading-relaxed">
					Self-hosted web scanning for accessibility, performance, SEO, and security workflows.
				</p>
			</div>

			<div class="flex flex-wrap items-center gap-4 text-sm">
				<a href="/playground" class="text-ink-muted hover:text-ink font-medium transition-colors">
					Playground
				</a>
				<a
					href={SITE.githubUrl}
					target="_blank"
					rel="noopener noreferrer"
					class="text-ink-faint hover:text-ink border-line/70 bg-surface/70 hover:bg-surface rounded-full border p-2 transition-colors"
					aria-label="GitHub"
				>
					<GithubIcon class="h-4 w-4" />
				</a>
			</div>
		</div>

		<div class="text-ink-faint border-line/70 mt-8 border-t pt-4 text-xs">
			&copy; {year} StageFlow. Open-source and operator-first.
		</div>
	</div>
</footer>
