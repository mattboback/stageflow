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
	const artifactCount = (report.artifacts?.length ?? 0) + (report.errors?.length ?? 0);
	const pages = report.summary.pagesScanned ?? 0;
	const issues = report.summary.totalIssues ?? 0;

	const tabs: TabSpec[] = [
		{
			id: 'review',
			label: 'Review',
			count: pages > 0 ? `${pages} page${pages === 1 ? '' : 's'}` : null
		},
		{ id: 'issues', label: 'Findings', count: issues > 0 ? String(issues) : null },
		{
			id: 'artifacts',
			label: 'Artifacts',
			count: artifactCount > 0 ? `${artifactCount} file${artifactCount === 1 ? '' : 's'}` : null
		}
	];
	const moveFocus = (event: React.KeyboardEvent<HTMLButtonElement>, index: number) => {
		let nextIndex: number | null = null;
		if (event.key === 'ArrowRight') nextIndex = (index + 1) % tabs.length;
		if (event.key === 'ArrowLeft') nextIndex = (index - 1 + tabs.length) % tabs.length;
		if (event.key === 'Home') nextIndex = 0;
		if (event.key === 'End') nextIndex = tabs.length - 1;
		if (nextIndex === null) return;
		event.preventDefault();
		const next = tabs[nextIndex];
		if (!next) return;
		onSectionChange(next.id);
		event.currentTarget.parentElement
			?.querySelectorAll<HTMLButtonElement>('[role="tab"]')
			.item(nextIndex)
			.focus();
	};

	return (
		<nav className="rnav" role="tablist" aria-label="Report sections">
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
						className="rnav__tab"
						onClick={() => onSectionChange(tab.id)}
						onKeyDown={(event) => moveFocus(event, index)}
					>
						<span>{tab.label}</span>
						{tab.count !== null && <span className="rnav__tab-count">{tab.count}</span>}
					</button>
				);
			})}
		</nav>
	);
}
