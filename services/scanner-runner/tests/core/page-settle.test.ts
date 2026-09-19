import type { Page } from 'playwright';

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { waitForPageSettled } from '../../src/core/page-settle';

type Listener = () => void;

const createPage = (overrides: Record<string, unknown> = {}) => {
	const listeners = new Map<string, Listener>();
	const page = {
		on: vi.fn((event: string, listener: Listener) => listeners.set(event, listener)),
		off: vi.fn((event: string) => listeners.delete(event)),
		waitForLoadState: vi.fn().mockResolvedValue(undefined),
		waitForFunction: vi.fn().mockResolvedValue(undefined),
		...overrides
	};

	return {
		page: page as unknown as Page,
		mocks: page,
		emit: (event: string) => listeners.get(event)?.()
	};
};

describe('waitForPageSettled', () => {
	beforeEach(() => {
		vi.useFakeTimers();
	});

	afterEach(() => {
		vi.useRealTimers();
	});

	it('settles once the network is quiet and the text holds still', async () => {
		const { page, mocks } = createPage();

		const waiting = waitForPageSettled(page);
		await vi.advanceTimersByTimeAsync(600);
		await waiting;

		expect(mocks.waitForLoadState).toHaveBeenCalledWith('networkidle');
		expect(mocks.waitForFunction).toHaveBeenCalledTimes(1);
		expect(mocks.off).toHaveBeenCalledTimes(3);
	});

	it('keeps waiting while a request that started after network idle is in flight', async () => {
		const { page, mocks, emit } = createPage();

		const waiting = waitForPageSettled(page);
		emit('request');
		await vi.advanceTimersByTimeAsync(3_000);
		expect(mocks.waitForFunction).not.toHaveBeenCalled();

		emit('requestfinished');
		await vi.advanceTimersByTimeAsync(600);
		await waiting;

		expect(mocks.waitForFunction).toHaveBeenCalledTimes(1);
	});

	it('checks the text again when a request arrives while it was settling', async () => {
		const { page, mocks, emit } = createPage();
		mocks.waitForFunction.mockImplementationOnce(() => {
			emit('request');
			emit('requestfinished');
			return Promise.resolve(undefined);
		});

		const waiting = waitForPageSettled(page);
		await vi.advanceTimersByTimeAsync(2_000);
		await waiting;

		expect(mocks.waitForFunction).toHaveBeenCalledTimes(2);
	});

	it('gives up after ten seconds on a page that never goes quiet', async () => {
		const { page, mocks, emit } = createPage();
		let settled = false;

		const waiting = waitForPageSettled(page).then(() => {
			settled = true;
		});
		emit('request');

		await vi.advanceTimersByTimeAsync(9_900);
		expect(settled).toBe(false);

		await vi.advanceTimersByTimeAsync(200);
		await waiting;
		expect(mocks.waitForFunction).not.toHaveBeenCalled();
	});
});
