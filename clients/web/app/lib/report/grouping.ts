import type { IssueDetail, IssueGroup, IssueSeverity } from '../types/unified-report';

import { compareSeverity, getWorstSeverity } from './severity';

/**
 * The most severe issue in a group, defaulting to 'info' for an empty or entirely
 * unrecognized set.
 *
 * This file used to carry its own severity ranking table — a third ordering in
 * this workspace alone, running opposite to severity.ts. It now delegates.
 */
function maxSeverity(values: IssueSeverity[]): IssueSeverity {
	return getWorstSeverity(values) ?? 'info';
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
		const sevDiff = compareSeverity(a.severity, b.severity);
		if (sevDiff !== 0) return sevDiff;
		const countDiff = b.occurrences.length - a.occurrences.length;
		if (countDiff !== 0) return countDiff;
		return a.fingerprint.localeCompare(b.fingerprint);
	});
}
