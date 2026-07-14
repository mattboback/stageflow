import type { IssueDetail } from '../types/unified-report';

import { isAxeIncompleteIssue, isColorContrastIssue } from './contrast-verify';
import { isManualReviewIssue } from './issue-kind';

/** Findings that cannot be resolved without a human decision. */
export function needsHumanReview(issue: IssueDetail): boolean {
	return isManualReviewIssue(issue) || (isColorContrastIssue(issue) && isAxeIncompleteIssue(issue));
}
