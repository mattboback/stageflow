import type { IssueDetail } from '../types/unified-report';
import type { ReviewVerdict } from './review-verdict';

import { isAxeIncompleteIssue, isColorContrastIssue } from './contrast-verify';
import { isManualReviewIssue } from './issue-kind';

export interface ReviewProgress {
	total: number;
	reviewed: number;
	pending: number;
}

export interface ReviewGroupStatus {
	label: string;
	tone: 'pending' | 'pass' | 'fail' | 'mixed';
}

/** Findings that cannot be resolved without a human decision. */
export function needsHumanReview(issue: IssueDetail): boolean {
	return isManualReviewIssue(issue) || (isColorContrastIssue(issue) && isAxeIncompleteIssue(issue));
}

export function getReviewGroupStatus(
	issues: IssueDetail[],
	getVerdict: (issueId: string) => ReviewVerdict | null
): ReviewGroupStatus | null {
	const reviewIssues = issues.filter(needsHumanReview);
	if (reviewIssues.length === 0) return null;
	const pending = reviewIssues.filter((issue) => !getVerdict(issue.id)).length;
	if (pending > 0) {
		return { label: pending === 1 ? 'needs review' : `${pending} need review`, tone: 'pending' };
	}
	const decisions = new Set(reviewIssues.map((issue) => getVerdict(issue.id)?.verdict));
	if (decisions.size > 1) return { label: 'reviewed · mixed', tone: 'mixed' };
	const decision = decisions.has('fail') ? 'fail' : 'pass';
	return { label: `reviewed · ${decision}`, tone: decision };
}
