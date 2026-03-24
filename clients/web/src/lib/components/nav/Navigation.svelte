<script lang="ts">
	import { page } from '$app/state';
	import { LogoMark } from '$lib/components/icons';
	import { cn } from '$lib/utils';
	import { Menu, X } from 'lucide-svelte';

	import DesktopNav from './DesktopNav.svelte';
	import MobileMenu from './MobileMenu.svelte';

	let mobileMenuOpen = $state(false);
	let scrolled = $state(false);

	const pathname = $derived(page.url.pathname);
	const isHome = $derived(pathname === '/');

	$effect(() => {
		const handleScroll = () => {
			scrolled = window.scrollY > 20;
		};
		window.addEventListener('scroll', handleScroll);
		return () => window.removeEventListener('scroll', handleScroll);
	});

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
		'nav-shell fixed top-0 right-0 left-0 z-50 border-b',
		scrolled || mobileMenuOpen
			? 'border-line bg-paper shadow-sm'
			: isHome
				? 'border-transparent bg-transparent'
				: 'border-line bg-paper'
	)}
>
	<div class="container-width flex h-16 items-center justify-between">
		<!-- Logo -->
		<a href="/" class="text-ink relative z-50 flex items-center gap-2.5" onclick={closeMobileMenu}>
			<LogoMark class="text-accent" size={24} />
			<span class="font-display text-[19px] leading-none font-semibold tracking-[-0.01em]"
				>StageFlow</span
			>
		</a>

		<DesktopNav {isActive} />

		<!-- Mobile Toggle -->
		<div class="flex items-center gap-4 md:hidden">
			<button
				onclick={toggleMobileMenu}
				class="text-ink-muted hover:text-ink hover:bg-surface/80 relative z-50 rounded-full p-3"
				aria-label={mobileMenuOpen ? 'Close menu' : 'Open menu'}
				aria-expanded={mobileMenuOpen}
				aria-controls="mobile-menu"
			>
				{#if mobileMenuOpen}
					<X class="h-5 w-5" />
				{:else}
					<Menu class="h-5 w-5" />
				{/if}
			</button>
		</div>
	</div>

	{#if scrolled || mobileMenuOpen}
		<div class="bg-line h-px"></div>
	{/if}
</header>

<MobileMenu isOpen={mobileMenuOpen} onClose={closeMobileMenu} {isActive} />
