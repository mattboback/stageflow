import { describe, expect, it, vi } from 'vitest';

import type { IssueDetail, IssueOccurrence } from '../types/unified-report';
import {
	failureGroupLabel,
	generateContextualFix,
	humanizeFailureSummary,
	parseFailureSummary
} from './contextual-fix';

// Shapes copied from the demo report's axe output.
const CHOICE =
	'Fix any of the following:\n  Element has no title attribute\n  aria-label attribute does not exist or is empty';
const LINK_NAME =
	'Fix all of the following:\n  Element is in tab order and does not have accessible text\n\n' +
	'Fix any of the following:\n  Element does not have text that is visible to screen readers\n' +
	'  aria-label attribute does not exist or is empty';

function issue(ruleId: string, howToFix?: string): IssueDetail {
	return {
		id: ruleId,
		scanner: 'axe',
		ruleId,
		severity: 'serious',
		title: ruleId,
		description: '',
		pageId: 'page',
		pageUrl: 'https://example.com/',
		elementCount: 1,
		...(howToFix ? { howToFix } : {})
	};
}

describe('parseFailureSummary', () => {
	it('splits axe output into its blocks, in order', () => {
		expect(parseFailureSummary(LINK_NAME)).toEqual([
			{ mode: 'all', checks: ['Element is in tab order and does not have accessible text'] },
			{
				mode: 'any',
				checks: [
					'Element does not have text that is visible to screen readers',
					'aria-label attribute does not exist or is empty'
				]
			}
		]);
	});

	it('reads a check written on the header line', () => {
		expect(
			parseFailureSummary('Fix all of the following: Document does not have a main landmark')
		).toEqual([{ mode: 'all', checks: ['Document does not have a main landmark'] }]);
	});

	it('returns null for prose, headers without checks, and empty input', () => {
		expect(
			parseFailureSummary('Send Referrer-Policy: strict-origin-when-cross-origin.')
		).toBeNull();
		expect(parseFailureSummary('Fix any of the following:')).toBeNull();
		expect(parseFailureSummary('')).toBeNull();
		expect(parseFailureSummary(undefined)).toBeNull();
	});
});

describe('failureGroupLabel', () => {
	it('promises a single fix only when the choice stands alone', () => {
		const choice = { mode: 'any' as const, checks: ['a', 'b'] };
		expect(failureGroupLabel(choice, 0, 1)).toBe('Fixing any one of these resolves it:');
		expect(failureGroupLabel(choice, 0, 2)).toBe('One of these needs fixing:');
		expect(failureGroupLabel(choice, 1, 2)).toBe('And one of these:');
	});

	it('treats a one-check choice as required', () => {
		expect(failureGroupLabel({ mode: 'any', checks: ['only'] }, 1, 2)).toBe('And this:');
		expect(failureGroupLabel({ mode: 'all', checks: ['only'] }, 0, 2)).toBe('This needs fixing:');
		expect(failureGroupLabel({ mode: 'all', checks: ['a', 'b'] }, 0, 1)).toBe(
			'All of these need fixing:'
		);
	});
});

describe('humanizeFailureSummary', () => {
	it('reads a lone choice as resolvable by any one check', () => {
		expect(humanizeFailureSummary(CHOICE)).toBe(
			'Fixing any one of these resolves it: Element has no title attribute. aria-label attribute does not exist or is empty.'
		);
	});

	it('keeps the required check required when a choice follows it', () => {
		expect(humanizeFailureSummary(LINK_NAME)).toBe(
			'This needs fixing: Element is in tab order and does not have accessible text. And one of these: Element does not have text that is visible to screen readers. aria-label attribute does not exist or is empty.'
		);
	});

	it('drops the wrapper around a single check', () => {
		expect(
			humanizeFailureSummary('Fix any of the following:\n  Element has insufficient color contrast')
		).toBe('Element has insufficient color contrast.');
	});

	it('passes prose through as sentences', () => {
		expect(humanizeFailureSummary('Update the href\nor remove the link')).toBe(
			'Update the href. or remove the link.'
		);
		expect(humanizeFailureSummary(null)).toBeNull();
	});
});

describe('generateContextualFix', () => {
	it('prefers rule guidance to the raw axe summary in howToFix', () => {
		expect(generateContextualFix(issue('button-name', CHOICE), null)).toMatch(
			/^Provide an accessible name for this button/
		);
	});

	it('matches rule ids exactly, so near-miss rules keep their own summary', () => {
		// Both contain a curated rule's id but need different advice.
		expect(generateContextualFix(issue('form-field-multiple-labels', CHOICE), null)).toBe(CHOICE);
		expect(generateContextualFix(issue('input-button-name', CHOICE), null)).toBe(CHOICE);
	});

	it('names the field from its captured markup', () => {
		const occurrence: IssueOccurrence = { selector: '#country', html: '<select id="country">' };
		expect(generateContextualFix(issue('select-name', CHOICE), occurrence)).toMatch(
			/^Associate this select with a <label for="country">/
		);
	});

	it('reads the markup the same way without a DOM, as the /demo prerender does', () => {
		vi.stubGlobal('DOMParser', undefined);
		try {
			const link: IssueOccurrence = { html: `<a class='x' href="/careers?a=1&amp;b=2" id=x>` };
			expect(generateContextualFix(issue('link-name', CHOICE), link)).toMatch(
				/^The <a href="\/careers\?a=1&b=2"> has no accessible name/
			);
			const image: IssueOccurrence = { html: '<img alt="a > b" src="/hero.png">' };
			expect(generateContextualFix(issue('image-alt'), image)).toMatch(
				/^Add a descriptive alt attribute to the <img src="\/hero.png">/
			);
		} finally {
			vi.unstubAllGlobals();
		}
	});

	it('reads invalid character references as U+FFFD, as a browser does, instead of throwing', () => {
		const link: IssueOccurrence = {
			html: '<a title="&#99999999;" data-x="&#xD800;" href="/a&#x110000;b&#0;">'
		};
		expect(generateContextualFix(issue('link-name', CHOICE), link)).toMatch(
			/^The <a href="\/a�b�"> has no accessible name/
		);
		// A rule that never reads the element does not parse it at all.
		expect(generateContextualFix(issue('button-name', CHOICE), link)).toMatch(
			/^Provide an accessible name/
		);
	});

	it('returns null when there is neither rule guidance nor scanner text', () => {
		expect(generateContextualFix(issue('region'), null)).toBeNull();
	});
});
