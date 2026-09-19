import type { Page } from 'playwright';

const SETTLE_TIMEOUT_MS = 5_000;

/**
 * Gives a client-rendered page time to draw itself. Pages are loaded to
 * `domcontentloaded`, where a single-page app is still an empty shell: reading the
 * DOM then reports zero words, no `<h1>` and no links. Bounded because sites with
 * sockets or analytics pings never reach network idle.
 */
export async function waitForPageSettled(page: Page): Promise<void> {
	let timer: ReturnType<typeof setTimeout> | undefined;

	try {
		await Promise.race([
			page.waitForLoadState('networkidle').catch(() => undefined),
			new Promise<void>((resolve) => {
				timer = setTimeout(resolve, SETTLE_TIMEOUT_MS);
			})
		]);
	} finally {
		clearTimeout(timer);
	}
}
