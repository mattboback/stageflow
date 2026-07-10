import type { UnifiedReport } from '../../lib/types/unified-report';

export type ReportSection = 'review' | 'issues' | 'artifacts';

interface Props {
	report: UnifiedReport;
	section: ReportSection;
	onSectionChange: (next: ReportSection) => void;
}

interface TabSpec {
	id: ReportSection;
	label: string;
	/* Worded, so "Review · 5 pages" can't be confused with an issue count. */
	count: string | null;
}

export function ReportSectionNav({ report, section, onSectionChange }: Props) {
	const artifactCount =
		(report.artifacts?.length ?? 0) + (report.errors?.length ?? 0);
	const pages = report.summary.pagesScanned ?? 0;
	const issues = report.summary.totalIssues ?? 0;

	const tabs: TabSpec[] = [
		{
			id: 'review',
			label: 'Review',
			count: pages > 0 ? `${pages} page${pages === 1 ? '' : 's'}` : null
		},
		{ id: 'issues', label: 'Issues', count: issues > 0 ? String(issues) : null },
		{
			id: 'artifacts',
			label: 'Artifacts',
			count:
				artifactCount > 0 ? `${artifactCount} file${artifactCount === 1 ? '' : 's'}` : null
		}
	];

	return (
		<nav className="rnav" role="tablist" aria-label="Report sections">
			{tabs.map((tab) => {
				const selected = tab.id === section;
				return (
					<button
						key={tab.id}
						type="button"
						role="tab"
						id={`report-tab-${tab.id}`}
						aria-selected={selected}
						aria-controls={`report-panel-${tab.id}`}
						className="rnav__tab"
						onClick={() => onSectionChange(tab.id)}
					>
						<span>{tab.label}</span>
						{tab.count !== null && (
							<span className="rnav__tab-count">{tab.count}</span>
						)}
					</button>
				);
			})}
		</nav>
	);
}
