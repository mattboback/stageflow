import { useState } from 'react';

import type { IssueDetail, IssueGroup, PageSummary } from '../../lib/types/unified-report';
import {
	SCANNER_META,
	getSeverityBadgeClass,
	getSeverityDotClass
} from '../../lib/report';

interface Props {
	group: IssueGroup;
	pageById: Map<string, PageSummary>;
	defaultOpen?: boolean;
	onSelectOccurrence?: (occurrence: IssueDetail) => void;
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
								<b
									style={{
										color: 'var(--ink-strong)',
										fontVariantNumeric: 'tabular-nums'
									}}
								>
									{occurrenceCount}
								</b>{' '}
								×{' · '}
							</>
						)}
						{pageCount} page{pageCount === 1 ? '' : 's'}
					</span>
					<span style={{ color: 'var(--ink-faint)' }}>{scannerLabel}</span>
				</span>
			</button>
			{open && (
				<div className="igroup__body">
					<div className="igroup__body-label">
						Affected page{pageCount === 1 ? '' : 's'}
					</div>
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
								style={onSelectOccurrence ? { cursor: 'pointer' } : undefined}
							>
								<span aria-hidden="true" style={{ color: 'var(--ink-faint)' }}>
									·
								</span>
								<span className="igroup__occ-page">
									<strong>{pageLabel}</strong>
									{occ.pageUrl}
								</span>
								<span
									style={{
										fontSize: '0.8125rem',
										fontVariantNumeric: 'tabular-nums',
										color: 'var(--ink-muted)'
									}}
								>
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
