import type { IssueDetail } from '../../lib/types/unified-report';
import {
	SCANNER_META,
	getSeverityBadgeClass,
	getSeverityDotClass,
	rewriteIssueTitle
} from '../../lib/report';

import { ScannerText } from './ScannerText';

interface Props {
	issue: IssueDetail;
	// See IssueGroupRow: exactOptionalPropertyTypes requires the explicit
	// `| undefined` for props that callers forward from their own optional prop.
	onSelect?: ((issue: IssueDetail) => void) | undefined;
	showPageUrl?: boolean;
}

export function IssueRowCard({ issue, onSelect, showPageUrl = true }: Props) {
	const scannerLabel = SCANNER_META[issue.scanner]?.label ?? issue.scanner;
	const title = rewriteIssueTitle(issue);
	const occCount = issue.elementCount ?? issue.occurrences?.length ?? 1;

	const content = (
		<>
			<div className="irow__sev" aria-hidden="true">
				<span className={getSeverityDotClass(issue.severity)} />
			</div>
			<div className="irow__main">
				<h3 className="irow__title">{title}</h3>
				<div className="irow__meta">
					<span className={getSeverityBadgeClass(issue.severity)}>{issue.severity}</span>
					<span className="irow__meta-item">
						<span style={{ color: 'var(--ink-faint)' }}>scanner</span>
						<b style={{ color: 'var(--ink)' }}>{scannerLabel}</b>
					</span>
					{issue.ruleId && (
						<span className="irow__meta-item">
							<span style={{ color: 'var(--ink-faint)' }}>rule</span>
							<b style={{ color: 'var(--ink)' }}>{issue.ruleId}</b>
						</span>
					)}
					{showPageUrl && issue.pageUrl && (
						<span
							className="irow__meta-item"
							title={issue.pageUrl}
							style={{
								maxWidth: '20rem',
								overflow: 'hidden',
								textOverflow: 'ellipsis',
								whiteSpace: 'nowrap'
							}}
						>
							{issue.pageUrl}
						</span>
					)}
				</div>
				{issue.description && (
					<p className="irow__desc">
						<ScannerText text={issue.description} links={false} />
					</p>
				)}
			</div>
			<div className="irow__right">
				<span className="irow__count">{occCount}</span>
				<span className="irow__count-lab">{occCount === 1 ? 'occurrence' : 'occurrences'}</span>
			</div>
		</>
	);

	if (onSelect) {
		return (
			<button
				type="button"
				className="irow"
				onClick={() => onSelect(issue)}
				style={{
					width: '100%',
					textAlign: 'left',
					background: 'transparent',
					border: 0,
					font: 'inherit',
					cursor: 'pointer'
				}}
			>
				{content}
			</button>
		);
	}

	return <div className="irow">{content}</div>;
}
