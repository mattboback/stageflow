import { useState } from 'react';
import { ChevronRight } from 'lucide-react';

import type { IssueDetail, IssueGroup, PageSummary } from '../../lib/types/unified-report';
import {
	SCANNER_META,
	getSeverityBadgeClass,
	getIssueKind,
	needsHumanReview
} from '../../lib/report';
import type { ReviewVerdict } from '../../lib/report/review-verdict';

interface Props {
	group: IssueGroup;
	pageById: Map<string, PageSummary>;
	defaultOpen?: boolean;
	newIssueIds?: ReadonlySet<string> | undefined;
	// Explicitly `| undefined`: exactOptionalPropertyTypes distinguishes an absent
	// prop from one passed as undefined, and callers forward an optional handler.
	onSelectOccurrence?: ((occurrence: IssueDetail) => void) | undefined;
	totalPages?: number | undefined;
	getVerdict?: ((issueId: string) => ReviewVerdict | null) | undefined;
}

export function IssueGroupRow({
	group,
	pageById,
	defaultOpen = false,
	onSelectOccurrence,
	newIssueIds,
	totalPages,
	getVerdict
}: Props) {
	const [open, setOpen] = useState(defaultOpen);
	const scannerLabel = SCANNER_META[group.scanner]?.label ?? group.scanner;
	const occurrenceCount = group.occurrences.length;
	const pageCount = group.pageIds.length;

	return (
		<div className={`igroup${open ? ' igroup--open' : ''}`}>
			<button
				type="button"
				className="igroup__head"
				aria-expanded={open}
				onClick={() => setOpen((v) => !v)}
			>
				<span className="igroup__caret" aria-hidden="true">
					▸
				</span>
				<span className="igroup__title" title={group.title}>
					{group.title}{' '}
					{group.occurrences.some((issue) => newIssueIds?.has(issue.id)) && (
						<span className="issue-new">NEW</span>
					)}
				</span>
				<span className="igroup__meta">
					<span className={getSeverityBadgeClass(group.severity)}>{group.severity}</span>
					<span>
						{occurrenceCount > 1 && (
							<>
								<b className="num igroup__count">{occurrenceCount}</b> occurrences ·{' '}
							</>
						)}
						{pageCount} page{pageCount === 1 ? '' : 's'}
					</span>
					{totalPages !== undefined && totalPages > 1 && pageCount === totalPages && (
						<span className="igroup__sitewide">Site-wide (all {totalPages} pages)</span>
					)}
					<span className="igroup__scanner">{scannerLabel}</span>
				</span>
			</button>
			{open && (
				<div className="igroup__body">
					<div className="igroup__body-label">Affected page{pageCount === 1 ? '' : 's'}</div>
					{group.occurrences.map((occ, occIndex) => {
						const page = pageById.get(occ.pageId);
						const pageLabel = page?.path ?? page?.url ?? occ.pageUrl ?? occ.pageId;
						const primary = occ.occurrences?.[0];
						const selector = primary?.selector ?? null;
						const snippet = primary?.html ? primary.html.replace(/\s+/g, ' ').slice(0, 90) : null;
						return (
							<div
								key={occ.id}
								className="igroup__occ"
								role={onSelectOccurrence ? 'button' : undefined}
								tabIndex={onSelectOccurrence ? 0 : undefined}
								onClick={onSelectOccurrence ? () => onSelectOccurrence(occ) : undefined}
								onKeyDown={
									onSelectOccurrence
										? (e) => {
												if (e.key === 'Enter' || e.key === ' ') {
													e.preventDefault();
													onSelectOccurrence(occ);
												}
											}
										: undefined
								}
							>
								<span className="igroup__occ-page">
									<strong>{pageLabel}</strong>
									<span className="igroup__occ-detail">
										<span className="igroup__occ-badge">Occurrence #{occIndex + 1}</span>
										{selector && (
											<span className="igroup__occ-selector" title={selector}>
												{selector}
											</span>
										)}
										{snippet && <span className="igroup__occ-snippet">{snippet}</span>}
										{!selector && !snippet && occ.pageUrl ? (
											<span className="igroup__occ-snippet">{occ.pageUrl}</span>
										) : null}
									</span>
								</span>
								{(() => {
									const verdict = getVerdict?.(occ.id)?.verdict;
									if (verdict === 'pass')
										return <span className="igroup__review igroup__review--pass">Passed</span>;
									if (verdict === 'fail')
										return (
											<span className="igroup__review igroup__review--fail">Verified fail</span>
										);
									if (needsHumanReview(occ))
										return (
											<span className="igroup__review igroup__review--pending">Needs review</span>
										);
									return null;
								})()}
								<span className="num igroup__occ-count">
									{getIssueKind(occ) === 'element' && occ.elementCount > 1
										? `${occ.elementCount}×`
										: ''}
								</span>
								{onSelectOccurrence && (
									<ChevronRight className="igroup__occ-open" size={16} aria-hidden="true" />
								)}
							</div>
						);
					})}
				</div>
			)}
		</div>
	);
}
