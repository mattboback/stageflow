import { useCallback, useMemo } from 'react';
import { useParams, useSearchParams, type MetaFunction } from 'react-router';

import { SiteHeader } from '../components/SiteHeader';
import { Pill } from '../components/Pill';
import { ReportHeader } from '../components/report/ReportHeader';
import {
	ReportSectionNav,
	type ReportSection
} from '../components/report/ReportSectionNav';
import { IssuesView } from '../components/report/IssuesView';
import { ReportStatStrip } from '../components/report/ReportStatStrip';
import { IssueDetailModal } from '../components/report/IssueDetailModal';
import { VisualReviewPanel } from '../components/report/VisualReviewPanel';
import { ArtifactsView } from '../components/report/ArtifactsView';
import { LighthouseSummary } from '../components/report/LighthouseSummary';
import { ErrorsView } from '../components/report/ErrorsView';
import { useScanReport } from '../lib/hooks/useScanMonitor';
import { buildOccurrenceModeReport, isIssueSortKey, type IssueSortKey } from '../lib/report';
import reportStyles from './scan-report.css?url';
import severityStyles from '../styles/report.css?url';

export const links = () => [
	{ rel: 'stylesheet', href: severityStyles },
	{ rel: 'stylesheet', href: reportStyles }
];

export const meta: MetaFunction = () => [
	{ title: 'Scan report — StageFlow' },
	{ name: 'robots', content: 'noindex' }
];

const SECTIONS: readonly ReportSection[] = ['review', 'issues', 'artifacts'];

function isSection(value: string | null): value is ReportSection {
	return value !== null && (SECTIONS as readonly string[]).includes(value);
}

/* Old links used ?section=overview / ?section=pages — both fold into review. */
function resolveSection(raw: string | null): ReportSection {
	if (isSection(raw)) return raw;
	if (raw === 'overview' || raw === 'pages') return 'review';
	return 'review';
}

export default function ScanReport() {
	const { id = '' } = useParams();
	const [searchParams, setSearchParams] = useSearchParams();

	const { status, report, job, error, screenshots, refreshArtifacts } =
		useScanReport(id);

	const displayReport = useMemo(
		() => (report ? buildOccurrenceModeReport(report) : null),
		[report]
	);

	const section: ReportSection = resolveSection(searchParams.get('section'));
	const activeScanner = searchParams.get('scanner');
	const activeSeverity = searchParams.get('severity');
	const activePage = searchParams.get('page');
	const activeCategory = searchParams.get('category');
	const searchQuery = searchParams.get('q') ?? '';
	const sortKey: IssueSortKey = isIssueSortKey(searchParams.get('sort'))
		? (searchParams.get('sort') as IssueSortKey)
		: 'severity';
	const groupByRule = searchParams.get('group') !== 'flat';
	const activeIssueId = searchParams.get('issue');

	const updateParams = useCallback(
		(updates: Record<string, string | null>) => {
			setSearchParams(
				(prev) => {
					const next = new URLSearchParams(prev);
					for (const [key, value] of Object.entries(updates)) {
						if (value === null || value === '') {
							next.delete(key);
						} else {
							next.set(key, value);
						}
					}
					return next;
				},
				{ replace: true, preventScrollReset: true }
			);
		},
		[setSearchParams]
	);

	const setSection = useCallback(
		(next: ReportSection) => {
			updateParams({ section: next === 'review' ? null : next });
		},
		[updateParams]
	);

	const activeIssue = useMemo(() => {
		if (!activeIssueId || !displayReport) return null;
		return displayReport.issues.find((i) => i.id === activeIssueId) ?? null;
	}, [activeIssueId, displayReport]);

	const activeIssuePage = useMemo(() => {
		if (!activeIssue || !displayReport) return null;
		return displayReport.pages.find((p) => p.id === activeIssue.pageId) ?? null;
	}, [activeIssue, displayReport]);

	return (
		<>
			<SiteHeader
				app={{ backTo: `/scan/${id}`, backLabel: 'Scan status', section: 'Report' }}
			/>

			<main id="main" className="report">
				<div className="wrap">
					{displayReport ? (
						<>
							<ReportHeader report={displayReport} />
							<ReportSectionNav
								report={displayReport}
								section={section}
								onSectionChange={setSection}
							/>

							{section === 'review' && (
								<section
									id="report-panel-review"
									role="tabpanel"
									aria-labelledby="report-tab-review"
								>
									<ReportStatStrip
										report={displayReport}
										jobId={id}
										onReviewSeverity={(severity) =>
											updateParams({ section: 'issues', severity })
										}
										onSelectScanner={(scannerId) =>
											updateParams({ section: 'issues', scanner: scannerId })
										}
										onReviewManual={() => updateParams({ section: 'issues' })}
									/>
									<VisualReviewPanel
										report={displayReport}
										screenshots={screenshots}
										activeScanner={activeScanner}
										activePage={activePage}
										onSelectPage={(pageId) => updateParams({ page: pageId })}
										onIssueSelect={(issue) => updateParams({ issue: issue.id })}
									/>
								</section>
							)}

							{section === 'issues' && (
								<section
									id="report-panel-issues"
									role="tabpanel"
									aria-labelledby="report-tab-issues"
								>
									<IssuesView
										report={displayReport}
										activeScanner={activeScanner}
										activeSeverity={activeSeverity}
										activePage={activePage}
										activeCategory={activeCategory}
										searchQuery={searchQuery}
										sortKey={sortKey}
										groupByRule={groupByRule}
										onScannerChange={(v) => updateParams({ scanner: v })}
										onSeverityChange={(v) => updateParams({ severity: v })}
										onPageChange={(v) => updateParams({ page: v })}
										onCategoryChange={(v) => updateParams({ category: v })}
										onSearchChange={(v) => updateParams({ q: v.trim() ? v : null })}
										onSortChange={(v) =>
											updateParams({ sort: v === 'severity' ? null : v })
										}
										onGroupToggle={(v) => updateParams({ group: v ? null : 'flat' })}
										onClear={() =>
											updateParams({
												scanner: null,
												severity: null,
												page: null,
												category: null,
												q: null
											})
										}
										onIssueSelect={(issue) => updateParams({ issue: issue.id })}
									/>
								</section>
							)}

							{section === 'artifacts' && (
								<section
									id="report-panel-artifacts"
									role="tabpanel"
									aria-labelledby="report-tab-artifacts"
									className="rsection-artifacts"
								>
									{(displayReport.summary.lighthouseCategories?.length ?? 0) > 0 && (
										<LighthouseSummary
											categories={displayReport.summary.lighthouseCategories ?? []}
										/>
									)}
									<ArtifactsView
										jobId={id}
										job={job}
										onRefreshArtifacts={refreshArtifacts}
									/>
									<ErrorsView errors={displayReport.errors} />
								</section>
							)}

							{activeIssue && (
								<IssueDetailModal
									issue={activeIssue}
									jobId={id}
									page={activeIssuePage}
									issues={displayReport.issues}
									screenshots={screenshots}
									onClose={() => updateParams({ issue: null })}
									onNavigate={(issueId) => updateParams({ issue: issueId })}
								/>
							)}
						</>
					) : status === 'failed' || status === 'error' ? (
						<div className="rsection-placeholder" role="alert">
							<h2>Scan failed</h2>
							<p>{error || job?.error || 'The scan did not produce a report.'}</p>
						</div>
					) : (
						<div
							className="rsection-placeholder"
							role="status"
							aria-live="polite"
							style={{
								display: 'flex',
								flexDirection: 'column',
								gap: '0.6rem',
								alignItems: 'center'
							}}
						>
							<Pill variant="queued">Loading</Pill>
							<h2>Preparing report…</h2>
							<p>
								{status === 'complete'
									? 'Scan complete — aggregating report.'
									: 'Waiting for the scan to finish.'}
							</p>
						</div>
					)}
				</div>
			</main>
		</>
	);
}
