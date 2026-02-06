<script lang="ts">
	import { GithubIcon } from '$lib/components/icons';
	import { navLinks } from '$lib/config/nav';
	import { SITE } from '$lib/config/site';
	import { cn } from '$lib/utils';

	interface Props {
		isActive: (href: string) => boolean;
	}

	const { isActive }: Props = $props();
</script>

<nav class="hidden items-center gap-1 md:flex" aria-label="Main navigation">
	{#each navLinks as link (link.href)}
		<a
			href={link.href}
			aria-current={isActive(link.href) ? 'page' : undefined}
			class={cn(
				'rounded-md px-3 py-1.5 text-sm font-medium transition-colors',
				isActive(link.href)
					? 'text-accent bg-accent-soft'
					: 'text-ink-muted hover:text-ink'
			)}
		>
			{link.label}
		</a>
	{/each}
	<span class="bg-line mx-2 h-4 w-px"></span>
	<a
		href={SITE.githubUrl}
		target="_blank"
		rel="noopener noreferrer"
		class="text-ink-faint hover:text-ink rounded-md p-1.5"
		aria-label="GitHub"
	>
		<GithubIcon class="h-4 w-4" />
	</a>
</nav>
