import { useEffect, useId, useRef, useState, type ReactNode } from 'react';
import { Menu, X } from 'lucide-react';

import { cycleTabFocus, focusableWithin } from '../lib/utils/focus-trap';

/*
 * The narrow-viewport primary nav.
 *
 * Below 640px the marketing links were simply `display: none` with nothing in
 * their place, so a phone had no route to Projects, Configure scan, Scanners or
 * Docs at all -- the brand mark and a GitHub button were the entire nav. This
 * is that navigation, not an extra affordance.
 *
 * A disclosure rather than a dialog: it is a menu of links, not a modal task,
 * and aria-expanded on a button that controls a panel is the honest pairing.
 * Focus is still trapped while it is open, because a panel covering the page
 * with tabbable content behind it is the same trap problem a modal has, and
 * the helper is already written and used by ConfirmDialog and IssueDetailModal.
 */
export function MobileNav({ children }: { children: ReactNode }) {
	const [open, setOpen] = useState(false);
	const panelId = useId();
	const panelRef = useRef<HTMLDivElement | null>(null);
	const triggerRef = useRef<HTMLButtonElement | null>(null);

	useEffect(() => {
		if (!open) return;

		const handleKeydown = (event: KeyboardEvent) => {
			if (event.key === 'Escape') {
				event.preventDefault();
				setOpen(false);
				// Focus has to come home, or Escape leaves it on document.body and
				// the next Tab restarts from the top of the page.
				triggerRef.current?.focus();
				return;
			}
			if (event.key === 'Tab' && panelRef.current) {
				cycleTabFocus(focusableWithin(panelRef.current), event);
			}
		};

		window.addEventListener('keydown', handleKeydown);
		focusableWithin(panelRef.current ?? document.body)[0]?.focus();
		return () => {
			window.removeEventListener('keydown', handleKeydown);
		};
	}, [open]);

	return (
		<>
			<button
				ref={triggerRef}
				type="button"
				className="navmenu__trigger"
				aria-expanded={open}
				aria-controls={panelId}
				onClick={() => setOpen((prev) => !prev)}
			>
				{open ? <X size={20} aria-hidden="true" /> : <Menu size={20} aria-hidden="true" />}
				<span className="sr-only">{open ? 'Close menu' : 'Menu'}</span>
			</button>

			{open && (
				<div
					id={panelId}
					ref={panelRef}
					className="navmenu__panel"
					/* Any activation inside closes the menu. Every child is a link, so
					   listening once here beats threading a callback into each one --
					   and it keeps this component agnostic about what it contains. */
					onClick={() => setOpen(false)}
				>
					{children}
				</div>
			)}
		</>
	);
}
