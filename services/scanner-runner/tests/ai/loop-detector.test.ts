import { describe, expect, it } from 'vitest';

import { detectLoop } from '../../src/ai/loop-detector';

describe('detectLoop', () => {
	it('returns false when there are fewer than 6 steps', () => {
		expect(
			detectLoop([
				{
					stepNumber: 1,
					url: 'https://a',
					action: { type: 'click', selector: '#a' },
					reasoning: '',
					success: true,
					durationMs: 1
				}
			])
		).toBe(false);
	});

	it('returns true when a URL repeats 3+ times in the last 6 steps', () => {
		const base = {
			action: { type: 'click', selector: '#a' } as const,
			reasoning: '',
			success: true,
			durationMs: 1
		};

		expect(
			detectLoop([
				{ stepNumber: 1, url: 'https://a', ...base },
				{ stepNumber: 2, url: 'https://b', ...base },
				{ stepNumber: 3, url: 'https://a', ...base },
				{ stepNumber: 4, url: 'https://c', ...base },
				{ stepNumber: 5, url: 'https://a', ...base },
				{ stepNumber: 6, url: 'https://d', ...base }
			])
		).toBe(true);
	});
});
