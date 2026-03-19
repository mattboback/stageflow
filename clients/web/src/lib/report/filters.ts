import type { IssueDetail } from "$lib/types/unified-report";

export interface IssueFilterOptions {
	scannerId?: string | null;
	pageId?: string | null;
	severity?: string | null;
	category?: string | null;
	query?: string | null;
}

export function filterIssues(
	issues: IssueDetail[],
	{ scannerId, pageId, severity, category, query }: IssueFilterOptions,
): IssueDetail[] {
	let filtered = issues;

	if (scannerId) {
		filtered = filtered.filter((issue) => issue.scanner === scannerId);
	}
	if (pageId) {
		filtered = filtered.filter((issue) => issue.pageId === pageId);
	}
	if (severity) {
		filtered = filtered.filter((issue) => issue.severity === severity);
	}
	if (category) {
		filtered = filtered.filter((issue) => issue.category === category);
	}

	const trimmed = query?.trim();
	if (trimmed) {
		const needle = trimmed.toLowerCase();
		filtered = filtered.filter((issue) => {
			return (
				issue.title.toLowerCase().includes(needle) ||
				issue.description.toLowerCase().includes(needle) ||
				issue.ruleId.toLowerCase().includes(needle) ||
				issue.pageUrl.toLowerCase().includes(needle) ||
				issue.scanner.toLowerCase().includes(needle) ||
				(issue.category?.toLowerCase().includes(needle) ?? false) ||
				(issue.wcagTags?.some((tag) => tag.toLowerCase().includes(needle)) ??
					false)
			);
		});
	}

	return filtered;
}
