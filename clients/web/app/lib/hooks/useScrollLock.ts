import { useEffect } from 'react';

/*
 * Depth rather than a boolean: IssueDetailModal can open over a page that
 * already has a dialog mounted, and two independent unmounts each restoring
 * the body would unlock the page while a dialog was still up. Only the
 * outermost lock touches the body, and only the last release puts it back.
 */
let depth = 0;
let release: (() => void) | null = null;

/**
 * Freezes the page behind a modal for as long as the calling component is
 * mounted.
 *
 * A `position: fixed` backdrop stops clicks but not the wheel, so without this
 * the report scrolls underneath an open dialog and the user loses their place.
 * Locking costs the scrollbar its width, which would shift the sticky header
 * and every centred block left by ~15px on the desktop scrollbars that take up
 * space; the gap is paid back as padding so nothing moves.
 */
export function useScrollLock(): void {
	useEffect(() => {
		depth += 1;

		if (depth === 1) {
			const { body } = document;
			const previousOverflow = body.style.overflow;
			const previousPaddingRight = body.style.paddingRight;
			// Overlay scrollbars (touch, and macOS by default) report 0 here, so the
			// compensation is skipped exactly where there is nothing to compensate.
			const scrollbarWidth = window.innerWidth - document.documentElement.clientWidth;

			body.style.overflow = 'hidden';
			if (scrollbarWidth > 0) {
				const existing = Number.parseFloat(getComputedStyle(body).paddingRight) || 0;
				body.style.paddingRight = `${existing + scrollbarWidth}px`;
			}

			release = () => {
				// Restore the inline values rather than clearing them: the property may
				// have been set by something else before the first lock.
				body.style.overflow = previousOverflow;
				body.style.paddingRight = previousPaddingRight;
			};
		}

		return () => {
			depth -= 1;
			if (depth === 0) {
				release?.();
				release = null;
			}
		};
	}, []);
}
