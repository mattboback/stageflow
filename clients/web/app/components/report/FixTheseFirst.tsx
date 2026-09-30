import {
	compareSeverity,
	generateContextualFix,
	groupIssuesByRule,
	needsHumanReview,
	sortIssues
} from '../../lib/report';
import type { IssueDetail } from '../../lib/types/unified-report';
import { ScannerText } from './ScannerText';

interface Props {
	issues: IssueDetail[];
	onIssueSelect: (issue: IssueDetail) => void;
	newIssueIds?: ReadonlySet<string> | undefined;
}

export function FixTheseFirst({ issues, onIssueSelect, newIssueIds }: Props) {
	const groups = groupIssuesByRule(issues.filter((issue) => !needsHumanReview(issue)))
		.sort(
			(a, b) =>
				compareSeverity(a.severity, b.severity) ||
				b.pageIds.length - a.pageIds.length ||
				a.fingerprint.localeCompare(b.fingerprint)
		)
		.slice(0, 5);
	if (groups.length === 0) return null;
	return (
		<section className="fix-first" aria-labelledby="fix-first-heading">
			<h2 id="fix-first-heading">Fix these first</h2>
			<p>Confirmed findings, highest severity and widest page coverage first.</p>
			<ol>
				{groups.map((group) => {
					const issue = sortIssues(group.occurrences, 'severity')[0];
					if (!issue) return null;
					const fix =
						issue.howToFix || generateContextualFix(issue, issue.occurrences?.[0] ?? null);
					return (
						<li key={group.fingerprint}>
							<button type="button" onClick={() => onIssueSelect(issue)}>
								<span className="fix-first__title">
									<span className={`sev-dot sev-${group.severity}`} aria-hidden="true" />{' '}
									{group.title}{' '}
									{group.occurrences.some((item) => newIssueIds?.has(item.id)) && (
										<span className="issue-new">NEW</span>
									)}
								</span>
								<span className="fix-first__meta">
									{group.severity} · {group.pageIds.length} page
									{group.pageIds.length === 1 ? '' : 's'}
								</span>
								<span className="fix-first__guidance">
									<ScannerText
										text={fix || 'Open this finding to inspect the evidence and rule guidance.'}
										links={false}
									/>
								</span>
							</button>
						</li>
					);
				})}
			</ol>
		</section>
	);
}
