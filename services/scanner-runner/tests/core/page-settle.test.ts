import type { Page } from 'playwright';

import { afterEach, describe, expect, it, vi } from 'vitest';

import { waitForPageSettled } from '../../src/core/page-settle';

describe('waitForPageSettled', () => {
	afterEach(() => {
		vi.useRealTimers();
	});

	it('returns once the network is idle', async () => {
		const page = {
			waitForLoadState: vi.fn().mockResolvedValue(undefined),
			waitForFunction: vi.fn().mockResolvedValue(undefined)
		} as unknown as Page;

		await waitForPageSettled(page);

		expect(page.waitForLoadState).toHaveBeenCalledWith('networkidle');
		expect(page.waitForFunction).toHaveBeenCalledWith(expect.any(Function), 1_000, {
			polling: 250,
			timeout: 5_000
		});
	});

	it('carries on when the text never stops changing', async () => {
		const page = {
			waitForLoadState: vi.fn().mockResolvedValue(undefined),
			waitForFunction: vi.fn().mockRejectedValue(new Error('Timeout 5000ms exceeded'))
		} as unknown as Page;

		await expect(waitForPageSettled(page)).resolves.toBeUndefined();
	});

	it('gives up after five seconds on a page that never goes idle', async () => {
		vi.useFakeTimers();
		const page = {
			waitForLoadState: vi.fn().mockReturnValue(new Promise(() => undefined)),
			waitForFunction: vi.fn().mockResolvedValue(undefined)
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
