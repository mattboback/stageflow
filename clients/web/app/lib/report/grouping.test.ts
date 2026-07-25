import { describe, expect, it } from 'vitest';

import { groupIssuesByRule } from './grouping';
import { makeIssue } from './test-issues';

describe('groupIssuesByRule', () => {
	it('buckets by scanner and rule together, not by rule alone', () => {
		// Two scanners can report the same ruleId; collapsing them would attribute one
		// scanner's findings to the other.
		const groups = groupIssuesByRule([
			makeIssue({ id: 'a', scanner: 'axe', ruleId: 'image-alt' }),
			makeIssue({ id: 'b', scanner: 'lighthouse', ruleId: 'image-alt' })
		]);

		expect(groups.map((group) => group.fingerprint)).toEqual([
			'axe:image-alt',
			'lighthouse:image-alt'
		]);
		expect(groups.every((group) => group.occurrences.length === 1)).toBe(true);
	});

	it('collects every occurrence of one rule into a single group', () => {
		const groups = groupIssuesByRule([
			makeIssue({ id: 'a', ruleId: 'label', pageId: 'home' }),
			makeIssue({ id: 'b', ruleId: 'label', pageId: 'pricing' }),
			makeIssue({ id: 'c', ruleId: 'label', pageId: 'home' })
		]);

		expect(groups).toHaveLength(1);
		expect(groups[0]?.occurrences.map((issue) => issue.id)).toEqual(['a', 'b', 'c']);
	});

	it('reports the distinct pages a rule was seen on, deduplicated', () => {
		const groups = groupIssuesByRule([
			makeIssue({ id: 'a', ruleId: 'label', pageId: 'home' }),
			makeIssue({ id: 'b', ruleId: 'label', pageId: 'pricing' }),
			makeIssue({ id: 'c', ruleId: 'label', pageId: 'home' })
		]);

		expect(groups[0]?.pageIds).toEqual(['home', 'pricing']);
	});

	it('takes the worst severity in the group, not the first one seen', () => {
		const groups = groupIssuesByRule([
			makeIssue({ id: 'a', ruleId: 'label', severity: 'minor' }),
			makeIssue({ id: 'b', ruleId: 'label', severity: 'critical' }),
			makeIssue({ id: 'c', ruleId: 'label', severity: 'moderate' })
		]);

		expect(groups[0]?.severity).toBe('critical');
	});

	it('unions wcagTags across occurrences without duplicates', () => {
		const groups = groupIssuesByRule([
			makeIssue({ id: 'a', ruleId: 'label', wcagTags: ['wcag2a', 'wcag412'] }),
			makeIssue({ id: 'b', ruleId: 'label', wcagTags: ['wcag2a', 'wcag131'] })
		]);

		expect(groups[0]?.wcagTags).toEqual(['wcag2a', 'wcag412', 'wcag131']);
	});

	it('omits optional fields entirely rather than setting them undefined', () => {
		const groups = groupIssuesByRule([makeIssue({ ruleId: 'label' })]);
		const group = groups[0];

		expect(group).toBeDefined();
		expect(group && 'helpUrl' in group).toBe(false);
		expect(group && 'wcagTags' in group).toBe(false);
		expect(group && 'category' in group).toBe(false);
	});

	it('carries the optional fields through when the first occurrence has them', () => {
		const groups = groupIssuesByRule([
			makeIssue({
				ruleId: 'label',
				helpUrl: 'https://dequeuniversity.com/rules/axe/4.10/label',
				category: 'forms'
			})
		]);

		expect(groups[0]?.helpUrl).toBe('https://dequeuniversity.com/rules/axe/4.10/label');
		expect(groups[0]?.category).toBe('forms');
	});

	it('sorts by severity, then by occurrence count, then by fingerprint', () => {
		const groups = groupIssuesByRule([
			makeIssue({ id: '1', scanner: 'axe', ruleId: 'zebra', severity: 'moderate' }),
			makeIssue({ id: '2', scanner: 'axe', ruleId: 'alpha', severity: 'moderate' }),
			makeIssue({ id: '3', scanner: 'axe', ruleId: 'busy', severity: 'moderate' }),
			makeIssue({ id: '4', scanner: 'axe', ruleId: 'busy', severity: 'moderate' }),
			makeIssue({ id: '5', scanner: 'axe', ruleId: 'worst', severity: 'critical' })
		]);

		expect(groups.map((group) => group.fingerprint)).toEqual([
			'axe:worst', // critical outranks every moderate
			'axe:busy', // two occurrences outrank one
			'axe:alpha', // tie on count, so fingerprint decides
			'axe:zebra'
		]);
	});

	it('returns an empty list for no issues', () => {
		expect(groupIssuesByRule([])).toEqual([]);
	});

	it('defaults to info when no occurrence has a recognized severity', () => {
		const groups = groupIssuesByRule([
			// The contract types severity as an enum, but reports are parsed from JSON
			// at runtime, so an unrecognized value is reachable in production.
			makeIssue({ ruleId: 'label', severity: 'catastrophic' as never })
		]);

		expect(groups[0]?.severity).toBe('info');
	});
});
