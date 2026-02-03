<script lang="ts">
	import { page } from '$app/state';
	import { cn } from '$lib/utils';
	import { Command, Menu, X } from 'lucide-svelte';

	import DesktopNav from './DesktopNav.svelte';
	import MobileMenu from './MobileMenu.svelte';

	let mobileMenuOpen = $state(false);
	let scrolled = $state(false);

	const pathname = $derived(page.url.pathname);

	$effect(() => {
		const handleScroll = () => {
			scrolled = window.scrollY > 20;
		};
		window.addEventListener('scroll', handleScroll);
		return () => window.removeEventListener('scroll', handleScroll);
	});

	// Lock body scroll when mobile menu is open
	$effect(() => {
		if (mobileMenuOpen) {
			document.body.style.overflow = 'hidden';
		} else {
			document.body.style.overflow = 'unset';
		}
	});

	function isActive(href: string): boolean {
		if (href === '/') {
			return pathname === '/';
		}
		return pathname?.startsWith(href) || false;
	}

	function closeMobileMenu() {
		mobileMenuOpen = false;
	}

	function toggleMobileMenu() {
		mobileMenuOpen = !mobileMenuOpen;
	}
</script>

<header
	class={cn(
		'fixed top-0 right-0 left-0 z-50',
		scrolled || mobileMenuOpen
			? 'border-line/70 bg-surface/90 border-b backdrop-blur-md'
			: 'border-transparent bg-transparent py-2'
	)}
>
	<div class="container-width flex h-16 items-center justify-between">
		<!-- Logo -->
		<a
			href="/"
			class="group text-ink relative z-50 flex items-center gap-2.5 font-semibold tracking-tight"
			onclick={closeMobileMenu}
		>
			<div
				class="border-line bg-surface text-ink flex h-8 w-8 items-center justify-center rounded-lg border"
			>
				<Command class="h-4 w-4" />
			</div>
			<span class="text-lg">StageFlow</span>
		</a>

		<DesktopNav {isActive} />

		<!-- Mobile Toggle -->
		<div class="flex items-center gap-4 md:hidden">
			<button
				onclick={toggleMobileMenu}
				class="text-ink-muted hover:bg-surface-muted hover:text-ink focus-visible:ring-accent focus-visible:ring-offset-paper relative z-50 rounded-md p-2 focus-visible:ring-2 focus-visible:ring-offset-2"
				aria-label={mobileMenuOpen ? 'Close menu' : 'Open menu'}
				aria-expanded={mobileMenuOpen}
				aria-controls="mobile-menu"
			>
				{#if mobileMenuOpen}
					<X class="h-6 w-6" />
				{:else}
					<Menu class="h-6 w-6" />
				{/if}
			</button>
		</div>
	</div>
</header>

<MobileMenu isOpen={mobileMenuOpen} onClose={closeMobileMenu} {isActive} />
