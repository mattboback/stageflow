<script lang="ts">
	import { GithubIcon } from '$lib/components/icons';
	import { Modal } from '$lib/components/ui';
	import { navLinks } from '$lib/config/nav';
	import { SITE } from '$lib/config/site';
	import { cn } from '$lib/utils';

	interface Props {
		isOpen: boolean;
		onClose: () => void;
		isActive: (href: string) => boolean;
	}

	let { isOpen, onClose, isActive }: Props = $props();
</script>

<Modal
	open={isOpen}
	{onClose}
	id="mobile-menu"
	ariaLabel="Navigation menu"
	backdrop="none"
	layout="fullscreen"
	padding="none"
	z={40}
	size="full"
	overlayClass="bg-paper/95 pt-24 pb-6 backdrop-blur-xl"
	contentClass="container-width flex flex-1 flex-col justify-between"
	closeOnBackdrop={false}
>
	<nav class="flex flex-col gap-2" aria-label="Mobile navigation">
		{#each navLinks as link (link.href)}
			<div>
				<a
					href={link.href}
					aria-current={isActive(link.href) ? 'page' : undefined}
					onclick={onClose}
					class={cn(
						'block rounded-2xl px-4 py-4 text-lg font-semibold tracking-[-0.01em]',
						isActive(link.href)
							? 'border-line/70 bg-surface text-ink border shadow-[0_8px_20px_-14px_rgba(15,15,15,0.45)]'
							: 'text-ink-muted hover:bg-surface/80 hover:text-ink'
					)}
				>
					{link.label}
				</a>
			</div>
		{/each}
	</nav>

	<div class="border-line mt-8 space-y-6 border-t pt-8">
		<a
			href={SITE.githubUrl}
			target="_blank"
			class="border-line bg-surface text-ink-muted hover:bg-surface-muted hover:text-ink flex items-center justify-center gap-2 rounded-2xl border py-3 font-semibold"
			rel="noreferrer"
		>
			<GithubIcon class="h-5 w-5" /> GitHub
		</a>
	</div>
</Modal>
