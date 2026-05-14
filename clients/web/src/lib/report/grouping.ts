import type {
	IssueDetail,
	IssueGroup,
	IssueSeverity,
	PageSummary,
	ScannerSummary
} from '$lib/types/unified-report';

import { compareSeverity, getWorstSeverity } from './severity';

export type IssueGroupKey = 'none' | 'rule' | 'page' | 'scanner' | 'category';

export interface IssueBucket {
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

export function groupIssues(
	issues: IssueDetail[],
	key: IssueGroupKey,
	{ pagesById = {}, scannersById = {} }: GroupOptions = {}
): IssueBucket[] {
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

	const result: IssueBucket[] = Array.from(groups.entries()).map(([id, grouped]) => {
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



const SEVERITY_RANK: Record<string, number> = {
critical: 5,
serious: 4,
moderate: 3,
minor: 2,
info: 1
};

function maxSeverity(values: IssueSeverity[]): IssueSeverity {
let best: IssueSeverity = 'info';
let bestRank = 0;
for (const v of values) {
const r = SEVERITY_RANK[v] ?? 0;
if (r > bestRank) {
bestRank = r;
best = v;
}
}
return best;
}

/**
 * Aggregate flat issues into stable per-rule groups (Sentry-style).
 *
 * Group key (fingerprint): `${scanner}:${ruleId}`. Deterministic ordering:
 *   1. Max severity rank desc
 *   2. Occurrence count desc
 *   3. Fingerprint asc
 *
 * The fingerprint is stable across reorderings of the input so it is safe
 * to use as a baseline key for a future diff-scan feature.
 */
export function groupIssuesByRule(issues: IssueDetail[]): IssueGroup[] {
const buckets = new Map<string, IssueDetail[]>();
for (const issue of issues) {
const fingerprint = `${issue.scanner}:${issue.ruleId}`;
const list = buckets.get(fingerprint) ?? [];
list.push(issue);
buckets.set(fingerprint, list);
}

const groups: IssueGroup[] = Array.from(buckets.entries()).map(([fingerprint, occurrences]) => {
const head = occurrences[0];
const pageIds = Array.from(new Set(occurrences.map((i) => i.pageId).filter(Boolean)));
const wcagTags = Array.from(
new Set(occurrences.flatMap((i) => i.wcagTags ?? []).filter(Boolean))
);
const group: IssueGroup = {
fingerprint,
ruleId: head.ruleId,
scanner: head.scanner,
title: head.title,
description: head.description,
severity: maxSeverity(occurrences.map((i) => i.severity)),
occurrences,
pageIds
};
if (head.helpUrl) group.helpUrl = head.helpUrl;
if (wcagTags.length) group.wcagTags = wcagTags;
if (head.category) group.category = head.category;
return group;
});

return groups.sort((a, b) => {
const sevDiff = (SEVERITY_RANK[b.severity] ?? 0) - (SEVERITY_RANK[a.severity] ?? 0);
if (sevDiff !== 0) return sevDiff;
const countDiff = b.occurrences.length - a.occurrences.length;
if (countDiff !== 0) return countDiff;
return a.fingerprint.localeCompare(b.fingerprint);
});
}
