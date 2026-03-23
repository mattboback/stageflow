<script lang="ts">
	import { GithubIcon } from '$lib/components/icons';
	import { buttonVariants } from '$lib/components/ui';
	import { navLinks } from '$lib/config/nav';
	import { SITE } from '$lib/config/site';
	import { cn } from '$lib/utils';

	interface Props {
		isActive: (href: string) => boolean;
	}

	let { isActive }: Props = $props();
</script>

<nav class="hidden items-center gap-1.5 md:flex" aria-label="Main navigation">
	{#each navLinks as link (link.href)}
		<a
			href={link.href}
			aria-current={isActive(link.href) ? 'page' : undefined}
			class={cn(
				'nav-link-pill',
				isActive(link.href)
					? 'border-line/80 bg-surface text-ink border shadow-[0_4px_12px_-10px_rgba(15,15,15,0.45)]'
					: 'text-ink-muted hover:text-ink hover:bg-surface/80'
			)}
		>
			{link.label}
		</a>
	{/each}
	<span class="bg-line mx-1.5 h-4 w-px"></span>
	<a
		href={SITE.githubUrl}
		target="_blank"
		rel="noopener noreferrer"
		class="text-ink-faint hover:text-ink border-line/70 bg-surface/70 hover:bg-surface rounded-full border p-2"
		aria-label="GitHub"
	>
		<GithubIcon class="h-4 w-4" />
	</a>
	<a
		href="/playground"
		class={cn(
			buttonVariants({ variant: 'default', size: 'sm' }),
			'ml-0.5 h-9 gap-1.5 rounded-full px-4 text-[12px] font-semibold tracking-[0.06em] uppercase'
		)}
	>
		Run scan
	</a>
</nav>
