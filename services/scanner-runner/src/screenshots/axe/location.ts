import type { Page } from 'playwright';

import type { PageLocationInfo } from './types';

/**
 * Capture the current scroll position and page location info.
 */
export async function captureLocationInfo(page: Page): Promise<PageLocationInfo | undefined> {
	try {
		return await page.evaluate(() => {
			const scrollY = window.scrollY || window.pageYOffset;
			const vh = window.innerHeight;
			const doc = document.documentElement;
			const docHeight = Math.max(
				doc.scrollHeight,
				doc.offsetHeight,
				doc.clientHeight,
				document.body.scrollHeight,
				document.body.offsetHeight
			);
			const centerY = scrollY + vh / 2;
			const position = docHeight > 0 ? centerY / docHeight : 0;
			return {
				scrollY,
				viewportHeight: vh,
				docHeight,
				position: Math.min(1, Math.max(0, position))
			};
		});
	} catch {
		return undefined;
	}
}
