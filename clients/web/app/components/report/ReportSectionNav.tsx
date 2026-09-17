import type { UnifiedReport } from '../../lib/types/unified-report';
import { useRovingTabList } from '../../lib/hooks/useRovingTabList';

export type ReportSection = 'review' | 'pages' | 'issues' | 'artifacts';

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

function countDistinctArtifacts(report: UnifiedReport): number {
	const distinctPaths = new Set<string>();
	if (report.artifacts) {
		for (const artifact of report.artifacts) {
			const key = artifact.path || artifact.dataUri || artifact.id;
			if (key) distinctPaths.add(key);
		}
	}
	const errorCount = report.errors?.length ?? 0;
	return distinctPaths.size + errorCount;
}

export function ReportSectionNav({ report, section, onSectionChange }: Props) {
	const artifactCount = countDistinctArtifacts(report);
	const pages = report.summary.pagesScanned ?? 0;
	const issues = report.summary.totalIssues ?? 0;

	const tabs: TabSpec[] = [
		{
			id: 'review',
			label: 'Review',
			count: pages > 0 ? `${pages} page${pages === 1 ? '' : 's'}` : null
		},
		/* Hidden below two pages: a single-page scan has nothing to compare and a
		   tab that always reads "1" is furniture. */
		...(report.pages.length > 1
			? [{ id: 'pages' as const, label: 'Pages', count: String(report.pages.length) }]
			: []),
		{ id: 'issues', label: 'Findings', count: issues > 0 ? String(issues) : null },
		{
			id: 'artifacts',
			label: 'Artifacts',
			count: artifactCount > 0 ? `${artifactCount} file${artifactCount === 1 ? '' : 's'}` : null
		}
	];
	const moveFocus = useRovingTabList(tabs.length, (index) => {
		const next = tabs[index];
		if (next) onSectionChange(next.id);
	});

	return (
		<nav className="tabs rnav" role="tablist" aria-label="Report sections">
			{tabs.map((tab, index) => {
				const selected = tab.id === section;
				return (
					<button
						key={tab.id}
						type="button"
						role="tab"
						id={`report-tab-${tab.id}`}
						aria-selected={selected}
						aria-controls={`report-panel-${tab.id}`}
						tabIndex={selected ? 0 : -1}
						className="tab"
						onClick={() => onSectionChange(tab.id)}
						onKeyDown={(event) => moveFocus(event, index)}
					>
						<span>{tab.label}</span>
						{tab.count !== null && <span className="tab__count">{tab.count}</span>}
					</button>
				);
			})}
		</nav>
	);
}
