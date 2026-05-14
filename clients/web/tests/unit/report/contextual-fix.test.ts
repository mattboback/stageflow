// @vitest-environment jsdom
import { describe, expect, it } from 'vitest';

import { generateContextualFix } from '$lib/report/contextual-fix';
import type { IssueDetail, IssueOccurrence } from '$lib/types/unified-report';

function issue(over: Partial<IssueDetail>): IssueDetail {
	return {
		id: 'i',
		scanner: 'axe',
		ruleId: 'image-alt',
		title: 't',
		description: 'd',
		severity: 'serious',
		pageId: 'p1',
		impact: 'serious',
		occurrences: [],
		...over
	} as IssueDetail;
}

function occ(html: string, extra: Partial<IssueOccurrence> = {}): IssueOccurrence {
	return { html, ...extra } as IssueOccurrence;
}

describe('generateContextualFix', () => {
	it('image-alt: includes the src attribute', () => {
		const fix = generateContextualFix(issue({ ruleId: 'image-alt' }), occ('<img src="/logo.png">'));
		expect(fix).toMatch(/alt/);
		expect(fix).toMatch(/logo\.png/);
	});

	it('label: includes the id when present', () => {
		const fix = generateContextualFix(
			issue({ ruleId: 'label' }),
			occ('<input id="email" type="email">')
		);
		expect(fix).toMatch(/for="email"/);
	});

	it('color-contrast: uses contrast ratio when available', () => {
		const fix = generateContextualFix(
			issue({ ruleId: 'color-contrast' }),
			occ('<span>txt</span>', {
				scannerData: {
					fgColor: '#888',
					bgColor: '#fff',
					contrastRatio: 3.2,
					expectedContrastRatio: 4.5
				}
			} as unknown as IssueOccurrence)
		);
		expect(fix).toMatch(/3\.2/);
		expect(fix).toMatch(/4\.5/);
	});

	it('document-title: provides title guidance', () => {
		const fix = generateContextualFix(issue({ ruleId: 'document-title' }), null);
		expect(fix).toMatch(/<title>/);
	});

	it('heading-order: mentions the level', () => {
		const fix = generateContextualFix(issue({ ruleId: 'heading-order' }), occ('<h3>Section</h3>'));
		expect(fix).toMatch(/<h3>/);
	});

	it('link-name: includes href when present', () => {
		const fix = generateContextualFix(issue({ ruleId: 'link-name' }), occ('<a href="/about"></a>'));
		expect(fix).toMatch(/\/about/);
	});

	it('falls back to issue.howToFix when no rule matches', () => {
		const fix = generateContextualFix(
			issue({ ruleId: 'unknown-rule', howToFix: 'Custom guidance text.' }),
			null
		);
		expect(fix).toBe('Custom guidance text.');
	});

	it('returns null when no match and no fallback', () => {
		const fix = generateContextualFix(issue({ ruleId: 'unknown-rule', howToFix: '' }), null);
		expect(fix).toBeNull();
	});
});
