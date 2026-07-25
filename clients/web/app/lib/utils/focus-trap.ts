/**
 * Tab-focus cycling for modal dialogs.
 *
 * Extracted because ConfirmDialog and IssueDetailModal had byte-identical copies
 * of this wrap-around logic. Keyboard traps are easy to get subtly wrong and
 * cheap to get wrong twice, so there is one implementation and one test.
 */

/**
 * Wraps focus to the other end of `focusable` when Tab would leave the trap.
 *
 * Treats focus that has escaped the container entirely (`activeElement` is
 * outside `focusable`, or null after a node was removed) the same as focus on
 * the boundary, so the next Tab pulls the user back inside rather than into the
 * page behind the dialog.
 *
 * @returns whether focus was moved, in which case the event was consumed.
 */
export function cycleTabFocus(focusable: readonly HTMLElement[], event: KeyboardEvent): boolean {
	const first = focusable[0];
	const last = focusable[focusable.length - 1];
	if (!first || !last) {
		return false;
	}

	const active = document.activeElement;
	const activeIsInside = active instanceof HTMLElement && focusable.includes(active);

	if (event.shiftKey) {
		if (active === first || !activeIsInside) {
			event.preventDefault();
			last.focus();
			return true;
		}
		return false;
	}

	if (active === last || !activeIsInside) {
		event.preventDefault();
		first.focus();
		return true;
	}
	return false;
}

/** Elements that can hold focus inside a dialog, in document order. */
export function focusableWithin(container: HTMLElement): HTMLElement[] {
	const candidates = container.querySelectorAll<HTMLElement>(
		'button:not(:disabled), [href], input:not(:disabled), select:not(:disabled), textarea:not(:disabled), [tabindex]:not([tabindex="-1"])'
	);
	// offsetWidth/offsetHeight are zero for display:none; getClientRects covers
	// elements laid out but clipped, which are still reachable by keyboard.
	return Array.from(candidates).filter(
		(el) => el.offsetWidth > 0 || el.offsetHeight > 0 || el.getClientRects().length > 0
	);
}
