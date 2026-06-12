import { createContrastVerdictsStore } from '$lib/stores/contrast-verdicts.svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('$app/environment', () => ({ browser: true }));

const STORAGE_KEY = 'contrast-verdicts';

describe('Contrast Verdicts Store', () => {
	beforeEach(() => {
		localStorage.clear();
		vi.useFakeTimers();
		vi.setSystemTime(new Date('2026-06-12T10:00:00.000Z'));
	});

	afterEach(() => {
		vi.useRealTimers();
		localStorage.clear();
	});

	it('starts empty and returns null for unknown verdicts', () => {
		const store = createContrastVerdictsStore();
		expect(store.getVerdict('job-1', 'issue-a')).toBeNull();
	});

	it('records and reads back a verdict', () => {
		const store = createContrastVerdictsStore();
		store.setVerdict('job-1', 'issue-a', {
			verdict: 'pass',
			fg: '#1a1714',
			bg: '#faf9f7',
			ratio: 15.2
		});

		const verdict = store.getVerdict('job-1', 'issue-a');
		expect(verdict).not.toBeNull();
		expect(verdict!.verdict).toBe('pass');
		expect(verdict!.fg).toBe('#1a1714');
		expect(verdict!.at).toBe('2026-06-12T10:00:00.000Z');
	});

	it('persists to localStorage and loads in a fresh instance', () => {
		const store = createContrastVerdictsStore();
		store.setVerdict('job-1', 'issue-a', { verdict: 'fail', fg: '#888', bg: '#999', ratio: 1.2 });

		const fresh = createContrastVerdictsStore();
		expect(fresh.getVerdict('job-1', 'issue-a')?.verdict).toBe('fail');
	});

	it('clears a verdict and removes empty jobs', () => {
		const store = createContrastVerdictsStore();
		store.setVerdict('job-1', 'issue-a', { verdict: 'pass', fg: '#000', bg: '#fff', ratio: 21 });
		store.clearVerdict('job-1', 'issue-a');

		expect(store.getVerdict('job-1', 'issue-a')).toBeNull();
		expect(JSON.parse(localStorage.getItem(STORAGE_KEY)!)).toEqual({});
	});

	it('summarizes reviewed and failed counts for a set of issues', () => {
		const store = createContrastVerdictsStore();
		store.setVerdict('job-1', 'issue-a', { verdict: 'pass', fg: '#000', bg: '#fff', ratio: 21 });
		store.setVerdict('job-1', 'issue-b', { verdict: 'fail', fg: '#888', bg: '#999', ratio: 1.2 });

		const summary = store.summarize('job-1', ['issue-a', 'issue-b', 'issue-c']);
		expect(summary).toEqual({ total: 3, reviewed: 2, failed: 1 });
	});

	it('keeps verdicts scoped per job', () => {
		const store = createContrastVerdictsStore();
		store.setVerdict('job-1', 'issue-a', { verdict: 'pass', fg: '#000', bg: '#fff', ratio: 21 });

		expect(store.getVerdict('job-2', 'issue-a')).toBeNull();
	});

	it('trims the oldest jobs beyond the cap', () => {
		const store = createContrastVerdictsStore();
		for (let i = 0; i < 21; i++) {
			vi.setSystemTime(new Date(Date.UTC(2026, 5, 12, 10, i)));
			store.setVerdict(`job-${i}`, 'issue-a', {
				verdict: 'pass',
				fg: '#000',
				bg: '#fff',
				ratio: 21
			});
		}

		expect(Object.keys(store.verdicts)).toHaveLength(20);
		expect(store.getVerdict('job-0', 'issue-a')).toBeNull();
		expect(store.getVerdict('job-20', 'issue-a')).not.toBeNull();
	});

	it('survives corrupted localStorage', () => {
		localStorage.setItem(STORAGE_KEY, 'not-json{');
		const store = createContrastVerdictsStore();
		expect(store.getVerdict('job-1', 'issue-a')).toBeNull();
	});
});
