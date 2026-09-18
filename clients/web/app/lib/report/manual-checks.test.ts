import { describe, expect, it } from 'vitest';

import type { IssueDetail, UnifiedReport } from '../types/unified-report';
import { splitManualChecks } from './manual-checks';

function issue(overrides: Partial<IssueDetail> = {}): IssueDetail {
	return {
		id: 'issue-1',
		scanner: 'lighthouse',
		ruleId: 'focus-traps',
		severity: 'info',
		title: 'User focus is not accidentally trapped in a region',
		description:
			'Manual verification required: Check focus. [Learn more](https://example.com/focus).',
		pageId: 'page-1',
		pageUrl: 'https://example.com',
		elementCount: 0,
		occurrences: [],
		...overrides
	};
}

function reportWith(issues: IssueDetail[]): UnifiedReport {
	return { issues } as UnifiedReport;
}

describe('splitManualChecks', () => {
	it('lists a manual audit once however many pages reported it', () => {
		const manual = { scannerData: { lighthouseManual: true } };
		const finding = issue({ id: 'seo-1', scanner: 'seo', ruleId: 'seo-thin-content' });
		const { report, manualChecks } = splitManualChecks(
			reportWith([
				issue({ id: 'a', pageId: 'page-1', ...manual }),
				issue({ id: 'b', pageId: 'page-2', ...manual }),
				finding
			])
		);

		expect(report.issues).toEqual([finding]);
		expect(manualChecks).toEqual([
			{
				ruleId: 'focus-traps',
				title: 'User focus is not accidentally trapped in a region',
				description: 'Check focus. [Learn more](https://example.com/focus).',
				pageCount: 2
			}
		]);
	});

	it('keeps Lighthouse findings that are not manual audits', () => {
		const failed = issue({ id: 'c', ruleId: 'robots-txt', severity: 'serious' });
		const { report, manualChecks } = splitManualChecks(reportWith([failed]));

		expect(report.issues).toEqual([failed]);
		expect(manualChecks).toEqual([]);
	});
});
