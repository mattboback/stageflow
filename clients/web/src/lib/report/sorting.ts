import type { IssueDetail } from '$lib/types/unified-report';

import { compareSeverity } from './severity';

export type IssueSortKey = 'severity' | 'page' | 'scanner' | 'count' | 'title';

export const ISSUE_SORTS: IssueSortKey[] = ['severity', 'page', 'scanner', 'count', 'title'];

export const ISSUE_SORT_LABELS: Record<IssueSortKey, string> = {
	severity: 'By Severity',
	page: 'By Page',
	scanner: 'By Scanner',
	count: 'By Element Count',
	title: 'By Title'
};

export function isIssueSortKey(value?: string | null): value is IssueSortKey {
	return ISSUE_SORTS.includes((value ?? 'severity') as IssueSortKey);
}

export function sortIssues(issues: IssueDetail[], sortKey: IssueSortKey): IssueDetail[] {
	const sorted = [...issues];

	switch (sortKey) {
		case 'page':
			sorted.sort((a, b) => a.pageUrl.localeCompare(b.pageUrl));
			break;
		case 'scanner':
			sorted.sort((a, b) => a.scanner.localeCompare(b.scanner));
			break;
		case 'count':
			sorted.sort((a, b) => b.elementCount - a.elementCount);
			break;
		case 'title':
			sorted.sort((a, b) => a.title.localeCompare(b.title));
			break;
		default:
			sorted.sort((a, b) => {
				const severityOrder = compareSeverity(a.severity, b.severity);
				if (severityOrder !== 0) return severityOrder;
				return a.title.localeCompare(b.title);
			});
			break;
	}

	return sorted;
}
