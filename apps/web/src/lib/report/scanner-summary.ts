import type { IssueDetail, PageSummary } from '$lib/types/unified-report';

export interface RuleSummary {
	ruleId: string;
	title: string;
	count: number;
}

export interface PageSummaryCount {
	pageId: string;
	label: string;
	count: number;
}

export function summarizeIssuesByRule(issues: IssueDetail[]): RuleSummary[] {
	const counts: Record<string, RuleSummary> = {};
	for (const issue of issues) {
		if (issue.ruleId in counts) {
			counts[issue.ruleId].count += 1;
		} else {
			counts[issue.ruleId] = {
				ruleId: issue.ruleId,
				title: issue.title,
				count: 1
			};
		}
	}

	return Object.values(counts).sort((a, b) => b.count - a.count);
}

export function summarizeIssuesByPage(
	issues: IssueDetail[],
	pagesById: Record<string, PageSummary>
): PageSummaryCount[] {
	const counts: Record<string, PageSummaryCount> = {};
	for (const issue of issues) {
		if (issue.pageId in counts) {
			counts[issue.pageId].count += 1;
		} else {
			const page = pagesById[issue.pageId];
			counts[issue.pageId] = {
				pageId: issue.pageId,
				label: page.path ?? page.url,
				count: 1
			};
		}
	}

	return Object.values(counts).sort((a, b) => b.count - a.count);
}
