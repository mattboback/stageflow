import { useState } from 'react';

import type { IssueDetail, IssueGroup, PageSummary } from '../../lib/types/unified-report';
import { SCANNER_META, getSeverityBadgeClass, getSeverityDotClass } from '../../lib/report';

interface Props {
	group: IssueGroup;
	pageById: Map<string, PageSummary>;
	defaultOpen?: boolean;
	// Explicitly `| undefined`: exactOptionalPropertyTypes distinguishes an absent
	// prop from one passed as undefined, and callers forward an optional handler.
	onSelectOccurrence?: ((occurrence: IssueDetail) => void) | undefined;
}

export function IssueGroupRow({ group, pageById, defaultOpen = false, onSelectOccurrence }: Props) {
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
				<span className={getSeverityDotClass(group.severity)} aria-hidden="true" />
				<span className="igroup__title" title={group.title}>
					{group.title}
				</span>
				<span className="igroup__meta">
					<span className={getSeverityBadgeClass(group.severity)}>{group.severity}</span>
					<span>
						{occurrenceCount > 1 && (
							<>
								<b className="num igroup__count">{occurrenceCount}</b> ×{' · '}
							</>
						)}
						{pageCount} page{pageCount === 1 ? '' : 's'}
					</span>
					<span className="igroup__scanner">{scannerLabel}</span>
				</span>
			</button>
			{open && (
				<div className="igroup__body">
					<div className="igroup__body-label">Affected page{pageCount === 1 ? '' : 's'}</div>
					{group.occurrences.map((occ) => {
						const page = pageById.get(occ.pageId);
						const pageLabel = page?.path ?? page?.url ?? occ.pageUrl ?? occ.pageId;
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
								<span className="igroup__occ-dot" aria-hidden="true">
									·
								</span>
								<span className="igroup__occ-page">
									<strong>{pageLabel}</strong>
									{occ.pageUrl}
								</span>
								<span className="num igroup__occ-count">
									{occ.elementCount > 1 ? `${occ.elementCount}×` : ''}
								</span>
							</div>
						);
					})}
				</div>
			)}
		</div>
	);
}
