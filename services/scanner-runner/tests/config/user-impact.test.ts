import { describe, expect, it } from 'vitest';

import { getUserImpact } from '../../src/config/user-impact';

describe('user-impact', () => {
	it('returns defaults when rule id is missing or unknown', () => {
		const missing = getUserImpact(undefined);
		expect(missing.severity).toBe('inconvenient');

		const unknown = getUserImpact('not-a-rule');
		expect(unknown.severity).toBe('inconvenient');
	});

	it('returns configured impact for known rules', () => {
		const impact = getUserImpact('color-contrast');
		expect(impact.severity).toBe('degraded');
		expect(impact.affectedGroups.length).toBeGreaterThan(0);
	});
});
