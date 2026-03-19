import type { IssueDetail, PageSummary, ScannerSummary } from '$lib/types/unified-report';

import { compareSeverity, getWorstSeverity } from './severity';

export type IssueGroupKey = 'none' | 'rule' | 'page' | 'scanner' | 'category';

export const ISSUE_GROUPS: IssueGroupKey[] = ['none', 'rule', 'page', 'scanner', 'category'];

export const ISSUE_GROUP_LABELS: Record<IssueGroupKey, string> = {
	none: 'No grouping',
	rule: 'By Issue Type',
	page: 'By Page',
	scanner: 'By Scanner',
	category: 'By Category'
};

export interface IssueGroup {
	id: string;
	label: string;
	issues: IssueDetail[];
	count: number;
	severity: string | null;
}

interface GroupOptions {
	pagesById?: Record<string, PageSummary>;
	scannersById?: Record<string, ScannerSummary>;
}

export function isIssueGroupKey(value?: string | null): value is IssueGroupKey {
	return ISSUE_GROUPS.includes((value ?? 'none') as IssueGroupKey);
}

export function groupIssues(
	issues: IssueDetail[],
	key: IssueGroupKey,
	{ pagesById = {}, scannersById = {} }: GroupOptions = {}
): IssueGroup[] {
	if (key === 'none') {
		return [
			{
				id: 'all',
				label: 'All issues',
				issues,
				count: issues.length,
				severity: getWorstSeverity(issues.map((issue) => issue.severity))
			}
		];
	}

	const groups = new Map<string, IssueDetail[]>();

	for (const issue of issues) {
		let groupId = '';
		if (key === 'rule') groupId = issue.ruleId;
		if (key === 'page') groupId = issue.pageId;
		if (key === 'scanner') groupId = issue.scanner;
		if (key === 'category') groupId = issue.category ?? 'uncategorized';
		if (!groupId) continue;
		const existing = groups.get(groupId) ?? [];
		existing.push(issue);
		groups.set(groupId, existing);
	}

	const result: IssueGroup[] = Array.from(groups.entries()).map(([id, grouped]) => {
		let label = id;
		if (key === 'rule') {
			label = grouped[0]?.title ?? id;
		} else if (key === 'page') {
			const page = pagesById[id];
			label = page.path ?? page.url;
		} else if (key === 'scanner') {
			const scanner = scannersById[id];
			label = scanner.name ?? id;
		} else {
			label = id === 'uncategorized' ? 'Uncategorized' : id;
		}

		return {
			id,
			label,
			issues: grouped,
			count: grouped.length,
			severity: getWorstSeverity(grouped.map((issue) => issue.severity))
		};
	});

	return result.sort((a, b) => {
		const severityOrder = compareSeverity(a.severity, b.severity);
		if (severityOrder !== 0) return severityOrder;
		return b.count - a.count;
	});
}
