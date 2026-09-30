import { describe, expect, it } from 'vitest';

import { getRuleBehavior } from '../../src/config/rule-behaviors';

describe('rule-behaviors', () => {
	it('returns the defaults when no rule id is provided', () => {
		expect(getRuleBehavior()).toEqual({ displayMode: 'expanded' });
	});

	it('merges a built-in rule over the defaults', () => {
		expect(getRuleBehavior('color-contrast')).toMatchObject({
			displayMode: 'expanded',
			owner: 'Design',
			tags: ['WCAG 1.4.3', 'Level AA']
		});
	});

	it('returns the defaults for an unknown rule', () => {
		expect(getRuleBehavior('not-a-rule')).toEqual({ displayMode: 'expanded' });
	});
});
