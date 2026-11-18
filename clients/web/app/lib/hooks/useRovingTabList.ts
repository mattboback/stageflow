import { useCallback, type KeyboardEvent } from 'react';

/**
 * Roving focus for a horizontal tablist (WAI-ARIA tabs pattern): arrow keys
 * move selection and focus with wraparound, Home/End jump to the ends. The
 * returned handler goes on each tab button's onKeyDown; `onMove` receives the
 * index to select before focus lands on it.
 *
 * Deliberately not lib/utils/focus-trap: that cycles the Tab key inside a
 * dialog. A tablist is one Tab stop whose interior moves on arrows — a
 * different contract.
 */
export function useRovingTabList(count: number, onMove: (index: number) => void) {
	return useCallback(
		(event: KeyboardEvent<HTMLButtonElement>, index: number) => {
			let nextIndex: number | null = null;
			if (event.key === 'ArrowRight') nextIndex = (index + 1) % count;
			if (event.key === 'ArrowLeft') nextIndex = (index - 1 + count) % count;
			if (event.key === 'Home') nextIndex = 0;
			if (event.key === 'End') nextIndex = count - 1;
			if (nextIndex === null) return;
			event.preventDefault();
			onMove(nextIndex);
			event.currentTarget.parentElement
				?.querySelectorAll<HTMLButtonElement>('[role="tab"]')
				.item(nextIndex)
				.focus();
		},
		[count, onMove]
	);
}
