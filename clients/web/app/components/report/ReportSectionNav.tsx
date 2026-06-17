import type { UnifiedReport } from '../../lib/types/unified-report';

export type ReportSection = 'overview' | 'issues' | 'pages' | 'artifacts';

interface Props {
	report: UnifiedReport;
	section: ReportSection;
	onSectionChange: (next: ReportSection) => void;
}

interface TabSpec {
	id: ReportSection;
	label: string;
	count: number | null;
}

export function ReportSectionNav({ report, section, onSectionChange }: Props) {
	const artifactCount =
		(report.artifacts?.length ?? 0) + (report.errors?.length ?? 0);

	const tabs: TabSpec[] = [
		{ id: 'overview', label: 'Overview', count: null },
		{ id: 'issues', label: 'Issues', count: report.summary.totalIssues ?? 0 },
		{ id: 'pages', label: 'Pages', count: report.summary.pagesScanned ?? 0 },
		{ id: 'artifacts', label: 'Artifacts', count: artifactCount || null }
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
						{tab.count !== null && tab.count > 0 && (
							<span className="rnav__tab-count">{tab.count}</span>
						)}
					</button>
				);
			})}
		</nav>
	);
}
