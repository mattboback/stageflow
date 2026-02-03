<script lang="ts">
	import type { Snippet } from 'svelte';

	import { cn, getNextFocusIndex } from '$lib/utils';
	import { onDestroy, tick } from 'svelte';

	import {
		modalContentVariants,
		modalOverlayVariants,
		type ModalBackdrop,
		type ModalLayout,
		type ModalPadding,
		type ModalSize,
		type ModalZ
	} from './modal';

	interface Props {
		open: boolean;
		onClose: () => void;
		id?: string;
		ariaLabel?: string;
		ariaLabelledBy?: string;
		backdrop?: ModalBackdrop;
		layout?: ModalLayout;
		padding?: ModalPadding;
		z?: ModalZ;
		size?: ModalSize;
		closeOnBackdrop?: boolean;
		closeOnEscape?: boolean;
		lockScroll?: boolean;
		trapFocus?: boolean;
		overlayClass?: string;
		contentClass?: string;
		children: Snippet;
	}

	const {
		open,
		onClose,
		id,
		ariaLabel,
		ariaLabelledBy,
		backdrop,
		layout,
		padding,
		z,
		size,
		closeOnBackdrop = true,
		closeOnEscape = true,
		lockScroll = true,
		trapFocus = true,
		overlayClass,
		contentClass,
		children
	}: Props = $props();

	let contentRef = $state<HTMLDivElement | null>(null);
	let lastFocused: Element | null = null;
	let previousBodyOverflow: string | null = null;
	let wasOpen = false;

	function getFocusable(): HTMLElement[] {
		if (!contentRef) return [];
		return Array.from(
			contentRef.querySelectorAll<HTMLElement>(
				'a[href], button, textarea, input, select, [tabindex]:not([tabindex="-1"])'
			)
		).filter((el) => !el.hasAttribute('disabled') && el.getAttribute('aria-hidden') !== 'true');
	}

	function focusInitial() {
		if (!contentRef) return;

		const preferred = contentRef.querySelector<HTMLElement>('[data-modal-initial-focus]');
		if (preferred) {
			preferred.focus();
			return;
		}

		const focusable = getFocusable();
		if (focusable.length) {
			focusable[0]?.focus();
			return;
		}

		contentRef.focus();
	}

	function handleTrapFocus(event: KeyboardEvent) {
		if (!trapFocus || event.key !== 'Tab') return;

		const focusable = getFocusable();
		if (focusable.length === 0) return;

		const active = document.activeElement as HTMLElement | null;
		const activeIndex = active ? focusable.indexOf(active) : -1;
		const nextIndex = getNextFocusIndex(focusable.length, activeIndex, event.shiftKey);

		if (nextIndex < 0) return;
		event.preventDefault();
		focusable[nextIndex]?.focus();
	}

	function handleKeydown(event: KeyboardEvent) {
		if (closeOnEscape && event.key === 'Escape') {
			event.preventDefault();
			onClose();
			return;
		}
		handleTrapFocus(event);
	}

	$effect(() => {
		if (open && !wasOpen) {
			wasOpen = true;
			lastFocused = document.activeElement;

			if (lockScroll) {
				previousBodyOverflow = document.body.style.overflow;
				document.body.style.overflow = 'hidden';
			}

			if (contentRef) {
				focusInitial();
			} else {
				void (async () => {
					await tick();
					if (!open) return;
					focusInitial();
				})();
			}
			return;
		}

		if (!open && wasOpen) {
			wasOpen = false;

			if (lockScroll) {
				document.body.style.overflow = previousBodyOverflow ?? '';
				previousBodyOverflow = null;
			}

			void (async () => {
				await tick();
				if (lastFocused instanceof HTMLElement) {
					lastFocused.focus();
				}
			})();
		}
	});

	onDestroy(() => {
		if (wasOpen && lockScroll) {
			document.body.style.overflow = previousBodyOverflow ?? '';
		}
		if (lastFocused instanceof HTMLElement) {
			lastFocused.focus();
		}
	});
</script>

{#if open}
	<div
		{id}
		class={cn(modalOverlayVariants({ backdrop, layout, padding, z }), overlayClass)}
		role="dialog"
		aria-modal="true"
		aria-label={ariaLabelledBy ? undefined : ariaLabel}
		aria-labelledby={ariaLabelledBy}
		tabindex="-1"
		onclick={(event) => {
			if (!closeOnBackdrop) return;
			if (event.target === event.currentTarget) onClose();
		}}
		onkeydown={handleKeydown}
	>
		<div
			bind:this={contentRef}
			class={cn(modalContentVariants({ size }), contentClass)}
			role="document"
			tabindex="-1"
		>
			{@render children()}
		</div>
	</div>
{/if}
