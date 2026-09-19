import type { IssueDetail, UnifiedReport } from '../types/unified-report';

/**
 * A Lighthouse manual audit: a fixed checklist item ("The page has a logical
 * tab order") that Lighthouse emits for every page of every site because it
 * cannot test it. It is a reminder, not evidence about the scanned site.
 */
export interface ManualCheck {
	ruleId: string;
	title: string;
	description: string;
	helpUrl?: string;
	pageCount: number;
}

export function isLighthouseManualCheck(issue: Pick<IssueDetail, 'scannerData'>): boolean {
	return issue.scannerData?.lighthouseManual === true;
}

const MANUAL_DESCRIPTION_PREFIX = 'Manual verification required: ';

/**
 * Returns the manual-audit checklist separately from findings. Current reports
 * carry it in `manualChecks`; reports saved to a local project before that
 * field existed still hold one info issue per page, so those are split out
 * here. Run this before any count is derived, so the headline, severity chips,
 * scanner chips and per-page totals describe only what the scanners found.
 */
export function splitManualChecks(report: UnifiedReport): {
	report: UnifiedReport;
	manualChecks: ManualCheck[];
} {
	const findings: IssueDetail[] = [];
	const checksByRule = new Map<string, ManualCheck>();
	for (const check of report.manualChecks ?? []) {
		checksByRule.set(check.ruleId, {
			ruleId: check.ruleId,
			title: check.title,
			description: check.description ?? '',
			...(check.helpUrl ? { helpUrl: check.helpUrl } : {}),
			pageCount: check.pageCount
		});
	}

	for (const issue of report.issues) {
		if (!isLighthouseManualCheck(issue)) {
			findings.push(issue);
			continue;
		}
		const existing = checksByRule.get(issue.ruleId);
		if (existing) {
			existing.pageCount++;
			continue;
		}
		const description = issue.description ?? '';
		checksByRule.set(issue.ruleId, {
			ruleId: issue.ruleId,
			title: issue.title ?? issue.ruleId,
			description: description.startsWith(MANUAL_DESCRIPTION_PREFIX)
				? description.slice(MANUAL_DESCRIPTION_PREFIX.length)
				: description,
			...(issue.helpUrl ? { helpUrl: issue.helpUrl } : {}),
			pageCount: 1
		});
	}

	if (checksByRule.size === 0) return { report, manualChecks: [] };
	return { report: { ...report, issues: findings }, manualChecks: [...checksByRule.values()] };
}
