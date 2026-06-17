import { useMemo, useState } from 'react';

import type { IssueDetail, UnifiedReport } from '../../lib/types/unified-report';
import {
	ISSUE_SORTS,
	ISSUE_SORT_LABELS,
	filterIssues,
	groupIssuesByRule,
	isIssueSortKey,
	sortIssues,
	type IssueSortKey
} from '../../lib/report';

import { IssueFilterSidebar } from './IssueFilterSidebar';
import { IssueGroupRow } from './IssueGroupRow';
import { IssueRowCard } from './IssueRowCard';

interface Props {
	report: UnifiedReport;
	activeScanner: string | null;
	activeSeverity: string | null;
	activePage: string | null;
	activeCategory: string | null;
	searchQuery: string;
	sortKey: IssueSortKey;
	groupByRule: boolean;
	onScannerChange: (next: string | null) => void;
	onSeverityChange: (next: string | null) => void;
	onPageChange: (next: string | null) => void;
	onCategoryChange: (next: string | null) => void;
	onSearchChange: (next: string) => void;
	onSortChange: (next: IssueSortKey) => void;
	onGroupToggle: (next: boolean) => void;
	onClear: () => void;
	onIssueSelect?: (issue: IssueDetail) => void;
}

export function IssuesView({
	report,
	activeScanner,
	activeSeverity,
	activePage,
	activeCategory,
	searchQuery,
	sortKey,
	groupByRule,
	onScannerChange,
	onSeverityChange,
	onPageChange,
	onCategoryChange,
	onSearchChange,
	onSortChange,
	onGroupToggle,
	onClear,
	onIssueSelect
}: Props) {
	const [, setBumpKey] = useState(0); // keeps URL→state reactive when groupToggle changes

	const categories = useMemo(() => {
		const set = new Set<string>();
		for (const issue of report.issues) {
			if (issue.category) set.add(issue.category);
		}
		return Array.from(set).sort();
	}, [report.issues]);

	const pageById = useMemo(() => {
		const map = new Map<string, (typeof report.pages)[number]>();
		for (const page of report.pages) map.set(page.id, page);
		return map;
	}, [report.pages]);

	const filtered = useMemo(
		() =>
			filterIssues(report.issues, {
				scannerId: activeScanner,
				pageId: activePage,
				severity: activeSeverity,
				category: activeCategory,
				query: searchQuery
			}),
		[
			report.issues,
			activeScanner,
			activePage,
			activeSeverity,
			activeCategory,
			searchQuery
		]
	);

	const sorted = useMemo(() => sortIssues(filtered, sortKey), [filtered, sortKey]);
	const groups = useMemo(() => (groupByRule ? groupIssuesByRule(sorted) : []), [
		sorted,
		groupByRule
	]);

	const totalCount = report.issues.length;
	const filteredCount = sorted.length;

	return (
		<div className="iview">
			<IssueFilterSidebar
				scanners={report.scanners}
				pages={report.pages}
				categories={categories}
				activeScanner={activeScanner}
				activeSeverity={activeSeverity}
				activePage={activePage}
				activeCategory={activeCategory}
				searchQuery={searchQuery}
				totalCount={totalCount}
				filteredCount={filteredCount}
				onScannerChange={onScannerChange}
				onSeverityChange={onSeverityChange}
				onPageChange={onPageChange}
				onCategoryChange={onCategoryChange}
				onSearchChange={onSearchChange}
				onClear={onClear}
			/>

			<div className="iview__panel" style={{ padding: 0 }}>
				<div className="ilist__head" style={{ padding: '0.7rem 1.1rem 0.85rem' }}>
					<span className="ilist__head-count">
						<b>{filteredCount}</b> issue{filteredCount === 1 ? '' : 's'}
						{groupByRule && groups.length > 0 && (
							<>
								{' '}
								· <b>{groups.length}</b> rule{groups.length === 1 ? '' : 's'}
							</>
						)}
					</span>
					<div className="ilist__sort">
						<label
							style={{
								display: 'inline-flex',
								alignItems: 'center',
								gap: '0.35rem',
								fontSize: '0.78rem',
								color: 'var(--ink-muted)'
							}}
						>
							<input
								type="checkbox"
								checked={groupByRule}
								onChange={(e) => {
									onGroupToggle(e.target.checked);
									setBumpKey((n) => n + 1);
								}}
							/>
							Group by rule
						</label>
						<label className="ilist__sort" style={{ gap: '0.4rem' }}>
							<span>Sort</span>
							<select
								className="isidebar__select"
								style={{ width: 'auto', padding: '0.2rem 0.4rem' }}
								value={sortKey}
								onChange={(e) => {
									const next = e.target.value;
									if (isIssueSortKey(next)) onSortChange(next);
								}}
								aria-label="Sort issues"
							>
								{ISSUE_SORTS.map((key) => (
									<option key={key} value={key}>
										{ISSUE_SORT_LABELS[key]}
									</option>
								))}
							</select>
						</label>
					</div>
				</div>

				{filteredCount === 0 ? (
					<div className="ilist__empty">
						<p>No issues match the current filters.</p>
						<small>
							{totalCount === 0
								? 'This scan completed without any issues.'
								: 'Clear filters above to see all issues.'}
						</small>
					</div>
				) : groupByRule ? (
					groups.map((group, index) => (
						<IssueGroupRow
							key={group.fingerprint}
							group={group}
							pageById={pageById}
							defaultOpen={index === 0}
							onSelectOccurrence={onIssueSelect}
						/>
					))
				) : (
					sorted.map((issue) => (
						<IssueRowCard key={issue.id} issue={issue} onSelect={onIssueSelect} />
					))
				)}
			</div>
		</div>
	);
}
