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

<nav class="hidden md:block" aria-label="Main navigation">
	<ul class="flex items-center gap-1">
		{#each navLinks as link (link.href)}
			<li>
				<a
					href={link.href}
					aria-current={isActive(link.href) ? 'page' : undefined}
					class={cn(
						'relative rounded-full px-4 py-2 text-sm font-medium',
						isActive(link.href) ? 'text-accent-ink' : 'text-ink-muted hover:text-ink'
					)}
				>
					{#if isActive(link.href)}
						<div class="bg-accent-soft absolute inset-0 -z-10 rounded-full"></div>
					{/if}
					{link.label}
				</a>
			</li>
		{/each}
	</ul>
</nav>

<div class="hidden items-center gap-3 md:flex">
	<a
		href={SITE.githubUrl}
		target="_blank"
		rel="noopener noreferrer"
		class="text-ink-faint hover:bg-surface-muted hover:text-ink rounded-md p-2"
		aria-label="GitHub"
	>
		<GithubIcon class="h-5 w-5" />
	</a>
</div>
