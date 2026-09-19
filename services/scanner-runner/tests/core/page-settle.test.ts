import type { Page } from 'playwright';

import { afterEach, describe, expect, it, vi } from 'vitest';

import { waitForPageSettled } from '../../src/core/page-settle';

describe('waitForPageSettled', () => {
	afterEach(() => {
		vi.useRealTimers();
	});

	it('returns once the network is idle', async () => {
		const page = { waitForLoadState: vi.fn().mockResolvedValue(undefined) } as unknown as Page;

		await waitForPageSettled(page);

		expect(page.waitForLoadState).toHaveBeenCalledWith('networkidle');
	});

	it('gives up after five seconds on a page that never goes idle', async () => {
		vi.useFakeTimers();
		const page = {
			waitForLoadState: vi.fn().mockReturnValue(new Promise(() => undefined))
		} as unknown as Page;

		let settled = false;
		const waiting = waitForPageSettled(page).then(() => {
			settled = true;
		});

		await vi.advanceTimersByTimeAsync(4_999);
		expect(settled).toBe(false);

		await vi.advanceTimersByTimeAsync(1);
		await waiting;
		expect(settled).toBe(true);
	});
});
