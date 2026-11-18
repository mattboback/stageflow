import type { IssueDetail } from '../../lib/types/unified-report';
import {
	SCANNER_META,
	getSeverityBadgeClass,
	getSeverityDotClass,
	getIssueKind,
	rewriteIssueTitle
} from '../../lib/report';

import { ScannerText } from './ScannerText';
import { TruncatedEvidence } from './TruncatedEvidence';

interface Props {
	issue: IssueDetail;
	// See IssueGroupRow: exactOptionalPropertyTypes requires the explicit
	// `| undefined` for props that callers forward from their own optional prop.
	onSelect?: ((issue: IssueDetail) => void) | undefined;
	showPageUrl?: boolean;
	isNew?: boolean;
}

export function IssueRowCard({ issue, onSelect, showPageUrl = true, isNew = false }: Props) {
	const scannerLabel = SCANNER_META[issue.scanner]?.label ?? issue.scanner;
	const title = rewriteIssueTitle(issue);
	const occCount = issue.elementCount ?? issue.occurrences?.length ?? 0;

	const content = (
		<>
			<div className="irow__sev" aria-hidden="true">
				<span className={getSeverityDotClass(issue.severity)} />
			</div>
			<div className="irow__main">
				<h3 className="irow__title">
					{title} {isNew && <span className="issue-new">NEW</span>}
				</h3>
				<div className="irow__meta">
					<span className={getSeverityBadgeClass(issue.severity)}>{issue.severity}</span>
					<span className="irow__meta-item">
						<span className="irow__meta-lab">scanner</span>
						<b className="irow__meta-val">{scannerLabel}</b>
					</span>
					{issue.ruleId && (
						<span className="irow__meta-item">
							<span className="irow__meta-lab">rule</span>
							<b className="irow__meta-val">{issue.ruleId}</b>
						</span>
					)}
					{showPageUrl && issue.pageUrl && (
						<span className="irow__meta-item irow__meta-item--url" title={issue.pageUrl}>
							{issue.pageUrl}
						</span>
					)}
				</div>
				{issue.description && (
					<p className="irow__desc">
						<ScannerText text={issue.description} links={false} />
					</p>
				)}
				<TruncatedEvidence issue={issue} />
			</div>
			{getIssueKind(issue) === 'element' && occCount > 0 && (
				<div className="irow__right">
					<span className="irow__count">{occCount}</span>
					<span className="irow__count-lab">{occCount === 1 ? 'occurrence' : 'occurrences'}</span>
				</div>
			)}
		</>
	);

	if (onSelect) {
		return (
			<button type="button" className="irow" onClick={() => onSelect(issue)}>
				{content}
			</button>
		);
	}

	return <div className="irow">{content}</div>;
}
