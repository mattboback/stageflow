import { describe, expect, it } from 'vitest';

import {
	buildHomeDescription,
	PLAYGROUND_TITLE,
	pageTitle,
	SITE_NAME,
	SITE_TAGLINE
} from './site-metadata';

describe('site metadata', () => {
	it('builds a grammatical description from the shipped tagline', () => {
		const description = buildHomeDescription(SITE_NAME, SITE_TAGLINE);

		expect(description).toBe(
			'StageFlow — Audit a frontend and catch regressions between deployments. Run accessibility, SEO, link, header, and visual checks in one report.'
		);
		expect(description.toLowerCase()).not.toContain('stageflow is a stageflow');
		expect(description.toLowerCase()).not.toContain('platform platform');
	});

	it('normalizes trailing tagline punctuation', () => {
		expect(buildHomeDescription('StageFlow', 'Quality scanning.')).toContain(
			'StageFlow — Quality scanning. Run accessibility'
		);
	});

	it('sets the playground title to Configure a scan', () => {
		expect(PLAYGROUND_TITLE).toBe(pageTitle('Configure a scan'));
	});
});
