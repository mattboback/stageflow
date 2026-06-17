import { useCallback, useMemo } from 'react';
import { Link, useParams, useSearchParams, type MetaFunction } from 'react-router';

import { SiteHeader } from '../components/SiteHeader';
import { SiteFooter } from '../components/SiteFooter';
import { Pill } from '../components/Pill';
import { ReportHeader } from '../components/report/ReportHeader';
import {
	ReportSectionNav,
	type ReportSection
} from '../components/report/ReportSectionNav';
import { IssuesView } from '../components/report/IssuesView';
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

const SECTIONS: readonly ReportSection[] = ['overview', 'issues', 'pages', 'artifacts'];

function isSection(value: string | null): value is ReportSection {
	return value !== null && (SECTIONS as readonly string[]).includes(value);
}

export default function ScanReport() {
	const { id = '' } = useParams();
	const [searchParams, setSearchParams] = useSearchParams();

	const { status, report, job, error } = useScanReport(id);

	const displayReport = useMemo(
		() => (report ? buildOccurrenceModeReport(report) : null),
		[report]
	);

	const rawSection = searchParams.get('section');
	const section: ReportSection = isSection(rawSection) ? rawSection : 'issues';
	const activeScanner = searchParams.get('scanner');
	const activeSeverity = searchParams.get('severity');
	const activePage = searchParams.get('page');
	const activeCategory = searchParams.get('category');
	const searchQuery = searchParams.get('q') ?? '';
	const sortKey: IssueSortKey = isIssueSortKey(searchParams.get('sort'))
		? (searchParams.get('sort') as IssueSortKey)
		: 'severity';
	const groupByRule = searchParams.get('group') !== 'flat';

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
			updateParams({ section: next === 'issues' ? null : next });
		},
		[updateParams]
	);

	return (
		<>
			<SiteHeader />

			<main id="main" className="report">
				<div className="wrap">
					<Link to={`/scan/${id}`} className="report__back">
						Back to scan status
					</Link>

					{displayReport ? (
						<>
							<ReportHeader jobId={id} report={displayReport} />
							<ReportSectionNav
								report={displayReport}
								section={section}
								onSectionChange={setSection}
							/>

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
									/>
								</section>
							)}

							{section === 'overview' && (
								<section
									id="report-panel-overview"
									role="tabpanel"
									aria-labelledby="report-tab-overview"
									className="rsection-placeholder"
								>
									<h2>Overview dashboard</h2>
									<p>Coming in Phase 3 — severity donut, scanner status grid, top pages.</p>
								</section>
							)}

							{section === 'pages' && (
								<section
									id="report-panel-pages"
									role="tabpanel"
									aria-labelledby="report-tab-pages"
									className="rsection-placeholder"
								>
									<h2>Visual review</h2>
									<p>
										Coming in Phase 4 — screenshot viewer with issue overlay markers and
										live WCAG contrast sampler.
									</p>
								</section>
							)}

							{section === 'artifacts' && (
								<section
									id="report-panel-artifacts"
									role="tabpanel"
									aria-labelledby="report-tab-artifacts"
									className="rsection-placeholder"
								>
									<h2>Artifacts & errors</h2>
									<p>
										Coming in Phase 4 — JSON/HTML report downloads, screenshot grid,
										scanner error details.
									</p>
								</section>
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
							style={{ display: 'flex', flexDirection: 'column', gap: '0.6rem', alignItems: 'center' }}
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

			<SiteFooter />
		</>
	);
}
