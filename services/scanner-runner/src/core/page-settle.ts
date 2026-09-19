import type { Page } from 'playwright';

const SETTLE_TIMEOUT_MS = 5_000;
const TEXT_STABLE_MS = 1_000;

/**
 * Gives a client-rendered page time to draw itself. Pages are loaded to
 * `domcontentloaded`, where a single-page app is still an empty shell: reading the
 * DOM then reports zero words, no `<h1>` and no links. Bounded because sites with
 * sockets or analytics pings never reach network idle.
 *
 * Network idle alone is not enough: when script start-up is slow the network goes
 * quiet before the app requests its data, and the page still reads "Loading…". So
 * the visible text must also stop changing.
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

	await page
		.waitForFunction(
			(stableMs) => {
				const state = window as unknown as { __sfTextLength?: number; __sfTextSince?: number };
				const length = document.body.innerText.length;
				const now = Date.now();
				if (state.__sfTextLength !== length) {
					state.__sfTextLength = length;
					state.__sfTextSince = now;
					return false;
				}
				return now - (state.__sfTextSince ?? now) >= stableMs;
			},
			TEXT_STABLE_MS,
			{ polling: 250, timeout: SETTLE_TIMEOUT_MS }
		)
		.catch(() => undefined);
}
