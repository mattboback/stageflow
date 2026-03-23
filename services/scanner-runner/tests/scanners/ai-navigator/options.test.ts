import { describe, expect, it } from 'vitest';

import { parseAiNavigatorOptions } from '../../../src/scanners/ai-navigator/options';

describe('parseAiNavigatorOptions', () => {
	it('parses goal + vision config', () => {
		const parsed = parseAiNavigatorOptions({
			goal: {
				objective: 'Sign in',
				maxSteps: 5,
				maxWallTimeMs: 5_000,
				inputValues: { email: 'me@example.com', bad: 123 },
				successCriteria: [
					{ type: 'url-contains', value: '/dashboard' },
					{ type: 'nope', value: 'x' }
				]
			},
			vision: {
				model: 'openai/gpt-4.1-mini',
				maxTokens: 500,
				timeoutMs: 10_000,
				maxImageBytes: 123_456,
				maxConcurrency: 2,
				retry: { maxAttempts: 3, baseDelayMs: 250 }
			}
		});

		expect(parsed.goal.objective).toBe('Sign in');
		expect(parsed.goal.maxSteps).toBe(5);
		expect(parsed.goal.maxWallTimeMs).toBe(5_000);
		expect(parsed.goal.inputValues).toEqual({ email: 'me@example.com' });
		expect(parsed.goal.successCriteria).toEqual([{ type: 'url-contains', value: '/dashboard' }]);

		expect(parsed.vision.provider).toBe('openrouter');
		expect(parsed.vision.model).toBe('openai/gpt-4.1-mini');
		expect(parsed.vision.maxTokens).toBe(500);
		expect(parsed.vision.timeoutMs).toBe(10_000);
		expect(parsed.vision.maxImageBytes).toBe(123_456);
		expect(parsed.vision.maxConcurrency).toBe(2);
		expect(parsed.vision.retry).toEqual({ maxAttempts: 3, baseDelayMs: 250 });
	});

	it('rejects non-object options', () => {
		expect(() => parseAiNavigatorOptions(null)).toThrow(/SCANNER_OPTIONS/i);
		expect(() => parseAiNavigatorOptions([])).toThrow(/SCANNER_OPTIONS/i);
	});

	it('rejects vision.apiKey in options', () => {
		expect(() =>
			parseAiNavigatorOptions({
				goal: { objective: 'x' },
				vision: { model: 'm', apiKey: 'nope' }
			})
		).toThrow(/vision\.apiKey/i);
	});

	it('rejects OpenRouter header fields in options', () => {
		expect(() =>
			parseAiNavigatorOptions({
				goal: { objective: 'x' },
				vision: { model: 'm', appTitle: 'title' }
			})
		).toThrow(/vision\.appTitle/i);

		expect(() =>
			parseAiNavigatorOptions({
				goal: { objective: 'x' },
				vision: { model: 'm', appReferer: 'https://example.com' }
			})
		).toThrow(/vision\.appReferer/i);
	});
});
