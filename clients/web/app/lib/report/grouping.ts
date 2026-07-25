import type { IssueDetail, IssueGroup, IssueSeverity } from '../types/unified-report';

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

export function groupIssuesByRule(issues: IssueDetail[]): IssueGroup[] {
	const buckets = new Map<string, IssueDetail[]>();
	for (const issue of issues) {
		const fingerprint = `${issue.scanner}:${issue.ruleId}`;
		const list = buckets.get(fingerprint) ?? [];
		list.push(issue);
		buckets.set(fingerprint, list);
	}

	const groups: IssueGroup[] = Array.from(buckets.entries()).flatMap(
		([fingerprint, occurrences]) => {
			const head = occurrences[0];
			if (head === undefined) {
				// Unreachable: a bucket only exists because an issue was pushed into it.
				return [];
			}
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
			return [group];
		}
	);

	return groups.sort((a, b) => {
		const sevDiff = (SEVERITY_RANK[b.severity] ?? 0) - (SEVERITY_RANK[a.severity] ?? 0);
		if (sevDiff !== 0) return sevDiff;
		const countDiff = b.occurrences.length - a.occurrences.length;
		if (countDiff !== 0) return countDiff;
		return a.fingerprint.localeCompare(b.fingerprint);
	});
}
