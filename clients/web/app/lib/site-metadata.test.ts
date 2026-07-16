import { describe, expect, it } from 'vitest';

import { buildHomeDescription } from './site-metadata';

describe('site metadata', () => {
	it('builds a grammatical description from the shipped tagline contract', () => {
		const description = buildHomeDescription(
			'StageFlow',
			'Self-hostable frontend quality regression scanning'
		);

		expect(description).toBe(
			'StageFlow — Self-hostable frontend quality regression scanning. Run accessibility, SEO, link, header, and visual checks in one report.'
		);
		expect(description.toLowerCase()).not.toContain('stageflow is a stageflow');
		expect(description.toLowerCase()).not.toContain('platform platform');
	});

	it('normalizes trailing tagline punctuation', () => {
		expect(buildHomeDescription('StageFlow', 'Quality scanning.')).toContain(
			'StageFlow — Quality scanning. Run accessibility'
		);
	});
});
