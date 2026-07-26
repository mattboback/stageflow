import { useCallback, useMemo, type ReactNode } from 'react';
import { useSearchParams } from 'react-router';

import { Pill } from '../Pill';
import { AllClearBanner } from './AllClearBanner';
import { ErrorsView } from './ErrorsView';
import { IssueDetailModal } from './IssueDetailModal';
import { IssuesView } from './IssuesView';
import { LighthouseSummary } from './LighthouseSummary';
import { ReportHeader } from './ReportHeader';
import { ReportSectionNav, type ReportSection } from './ReportSectionNav';
import { ReportStatStrip } from './ReportStatStrip';
import { VisualReviewPanel } from './VisualReviewPanel';
import { useReviewVerdicts } from '../../lib/hooks/useReviewVerdicts';
import {
	buildOccurrenceModeReport,
	isIssueSortKey,
	needsHumanReview,
	sortIssues,
	type IssueSortKey
} from '../../lib/report';
import type { ScanStatus, ScreenshotArtifact } from '../../lib/types/scan';
import type { UnifiedReport } from '../../lib/types/unified-report';

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

export interface ReportViewProps {
	/** Keys the review-verdict store, so two reports never share decisions. */
	jobId: string;
	/**
	 * The report exactly as it arrived. Occurrence-mode expansion happens in
	 * here rather than at the call site, so a caller cannot forget it and
	 * silently render the un-expanded shape.
	 */
	report: UnifiedReport | null;
	screenshots: ScreenshotArtifact[];
	status: ScanStatus;
	/**
	 * Already-resolved message for the failure state. The route decides what
	 * counts as an error, because that is a question about how the report was
	 * fetched, not about how it is displayed.
	 */
	error: string | null;
	/** Above the report header. Local-project comparison on a live scan, null on /demo. */
	banner: ReactNode;
	/**
	 * The artifacts tab's download list. A slot rather than a prop because a
	 * live scan lists presigned URLs from a job that /demo does not have, and
	 * inventing a synthetic job to satisfy the shape would emit links that 404.
	 */
	artifactsPanel: ReactNode;
}

/*
 * Everything about presenting a report, and nothing about obtaining one.
 *
 * The seam matters because the report viewer is the most substantial thing in
 * this codebase and it used to be reachable only by starting a scan and
 * waiting a minute and a half for it. Both slots are required rather than
 * optional: under exactOptionalPropertyTypes an optional ReactNode plus a
 * possibly-null value is a type-error dance for nothing, and ReactNode already
 * admits null.
 *
 * useSearchParams stays here. Which section is open, which issue is expanded
 * and how the list is filtered are presentation, and both routes want the same
 * deep links to work.
 */
export function ReportView({
	jobId,
	report,
	screenshots,
	status,
	error,
	banner,
	artifactsPanel
}: ReportViewProps) {
	const [searchParams, setSearchParams] = useSearchParams();

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

	/* The human-review queue is the report's primary workflow: highest
	   severity first, resume at the first finding without a verdict. */
	const { getVerdict } = useReviewVerdicts(jobId);
	const reviewQueue = useMemo(
		() =>
			displayReport ? sortIssues(displayReport.issues.filter(needsHumanReview), 'severity') : [],
		[displayReport]
	);
	const reviewProgress = useMemo(() => {
		const reviewed = reviewQueue.filter((issue) => getVerdict(issue.id)).length;
		return { total: reviewQueue.length, reviewed, pending: reviewQueue.length - reviewed };
	}, [reviewQueue, getVerdict]);
	const nextReviewIssue =
		reviewQueue.find((issue) => !getVerdict(issue.id)) ?? reviewQueue[0] ?? null;
	const modalIssues =
		section === 'review' && activeIssue && needsHumanReview(activeIssue)
			? reviewQueue
			: (displayReport?.issues ?? []);

	const activeIssuePage = useMemo(() => {
		if (!activeIssue || !displayReport) return null;
		return displayReport.pages.find((p) => p.id === activeIssue.pageId) ?? null;
	}, [activeIssue, displayReport]);

	return (
		<main id="main" className="report">
			<div className="wrap">
				{displayReport ? (
					<>
						{banner}
						<ReportHeader report={displayReport} reviewProgress={reviewProgress} />
						<ReportSectionNav
							report={displayReport}
							section={section}
							onSectionChange={setSection}
						/>

						{section === 'review' && (
							<section id="report-panel-review" role="tabpanel" aria-labelledby="report-tab-review">
								{displayReport.summary.totalIssues === 0 && (
									<AllClearBanner report={displayReport} />
								)}
								{reviewQueue.length > 0 && (
									<div className="rnext">
										<div className="rnext__copy">
											<p className="rnext__count">
												{reviewProgress.pending === 0
													? `All ${reviewQueue.length} findings reviewed`
													: reviewProgress.reviewed > 0
														? `${reviewProgress.pending} of ${reviewQueue.length} findings still need review`
														: `${reviewQueue.length} findings need review`}
											</p>
											<p className="rnext__sub">
												Highest severity first — evidence and decision tools open in place.
											</p>
										</div>
										{nextReviewIssue && reviewProgress.pending > 0 && (
											<button
												type="button"
												className="btn btn--primary btn--lg"
												onClick={() => updateParams({ issue: nextReviewIssue.id })}
											>
												{reviewProgress.reviewed > 0 ? 'Continue review' : 'Start review'}{' '}
												<span className="ar" aria-hidden="true">
													→
												</span>
											</button>
										)}
									</div>
								)}
								<ReportStatStrip
									report={displayReport}
									onReviewSeverity={(severity) => updateParams({ section: 'issues', severity })}
									onSelectScanner={(scannerId) =>
										updateParams({ section: 'issues', scanner: scannerId })
									}
								/>
								<VisualReviewPanel
									report={displayReport}
									screenshots={screenshots}
									activeScanner={activeScanner}
									activePage={activePage}
									getReviewVerdict={getVerdict}
									onSelectPage={(pageId) => updateParams({ page: pageId })}
									onIssueSelect={(issue) => updateParams({ issue: issue.id })}
								/>
							</section>
						)}

						{section === 'issues' && (
							<section id="report-panel-issues" role="tabpanel" aria-labelledby="report-tab-issues">
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
									onSortChange={(v) => updateParams({ sort: v === 'severity' ? null : v })}
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
								{artifactsPanel}
								<ErrorsView errors={displayReport.errors} />
							</section>
						)}

						{activeIssue && (
							<IssueDetailModal
								issue={activeIssue}
								jobId={jobId}
								page={activeIssuePage}
								issues={modalIssues}
								screenshots={screenshots}
								onClose={() => updateParams({ issue: null })}
								onNavigate={(issueId) => updateParams({ issue: issueId })}
							/>
						)}
					</>
				) : status === 'failed' || status === 'error' ? (
					<div className="rsection-placeholder" role="alert">
						<h2>Scan failed</h2>
						<p>{error || 'The scan did not produce a report.'}</p>
					</div>
				) : (
					<div
						className="rsection-placeholder rsection-placeholder--loading"
						role="status"
						aria-live="polite"
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
	);
}
