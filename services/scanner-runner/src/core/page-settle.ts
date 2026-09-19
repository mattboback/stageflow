import type { Page } from 'playwright';

const NETWORKIDLE_TIMEOUT_MS = 5_000;
const SETTLE_TIMEOUT_MS = 10_000;
const NETWORK_QUIET_MS = 500;
const TEXT_STABLE_MS = 1_000;
const POLL_MS = 100;

/**
 * Gives a client-rendered page time to draw itself. Pages are loaded to
 * `domcontentloaded`, where a single-page app is still an empty shell: reading the
 * DOM then reports zero words, no `<h1>` and no links.
 *
 * Playwright's `networkidle` fires once and cannot be awaited again. When script
 * start-up is slow it fires before the app requests its data, and the page reads
 * "Loading…" for as long as that request takes. So after it, requests are counted
 * here: the page is settled once none has been in flight for half a second and the
 * visible text has then held still for a second with no new request. Everything is
 * bounded because sites with sockets or long polling never go quiet.
 */
export async function waitForPageSettled(page: Page): Promise<void> {
	let inFlight = 0;
	let lastActivity = Date.now();
	const onRequest = (): void => {
		inFlight++;
		lastActivity = Date.now();
	};
	// Requests that began before the listeners were attached finish uncounted.
	const onRequestDone = (): void => {
		inFlight = Math.max(0, inFlight - 1);
		lastActivity = Date.now();
	};

	page.on('request', onRequest);
	page.on('requestfinished', onRequestDone);
	page.on('requestfailed', onRequestDone);

	const deadline = Date.now() + SETTLE_TIMEOUT_MS;

	try {
		await withTimeout(page.waitForLoadState('networkidle'), NETWORKIDLE_TIMEOUT_MS);

		while (Date.now() < deadline) {
			if (inFlight > 0 || Date.now() - lastActivity < NETWORK_QUIET_MS) {
				await delay(POLL_MS);
				continue;
			}

			const quietSince = lastActivity;
			await waitForStableText(page, deadline - Date.now());
			if (lastActivity === quietSince) {
				return;
			}
		}
	} finally {
		page.off('request', onRequest);
		page.off('requestfinished', onRequestDone);
		page.off('requestfailed', onRequestDone);
	}
}

async function waitForStableText(page: Page, timeoutMs: number): Promise<void> {
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
			{ polling: 250, timeout: Math.max(timeoutMs, 1) }
		)
		.catch(() => undefined);
}

async function withTimeout(operation: Promise<unknown>, timeoutMs: number): Promise<void> {
	let timer: ReturnType<typeof setTimeout> | undefined;

	try {
		await Promise.race([
			operation.catch(() => undefined),
			new Promise<void>((resolve) => {
				timer = setTimeout(resolve, timeoutMs);
			})
		]);
	} finally {
		clearTimeout(timer);
	}
}

function delay(ms: number): Promise<void> {
	return new Promise((resolve) => setTimeout(resolve, ms));
}
