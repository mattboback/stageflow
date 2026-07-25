import { useCallback, useEffect, useMemo, useState } from 'react';
import {
	Link,
	isRouteErrorResponse,
	useLocation,
	useParams,
	useRouteError,
	useSearchParams,
	type MetaFunction
} from 'react-router';

import { SiteHeader } from '../components/SiteHeader';
import { RouteFault } from '../components/RouteFault';
import { Pill } from '../components/Pill';
import { ReportHeader } from '../components/report/ReportHeader';
import { ReportSectionNav, type ReportSection } from '../components/report/ReportSectionNav';
import { IssuesView } from '../components/report/IssuesView';
import { ReportStatStrip } from '../components/report/ReportStatStrip';
import { IssueDetailModal } from '../components/report/IssueDetailModal';
import { VisualReviewPanel } from '../components/report/VisualReviewPanel';
import { ArtifactsView } from '../components/report/ArtifactsView';
import { LighthouseSummary } from '../components/report/LighthouseSummary';
import { ErrorsView } from '../components/report/ErrorsView';
import { LocalBaselineComparison } from '../components/report/LocalBaselineComparison';
import { AllClearBanner } from '../components/report/AllClearBanner';
import { useScanReport } from '../lib/hooks/useScanMonitor';
import {
	buildOccurrenceModeReport,
	isIssueSortKey,
	needsHumanReview,
	sortIssues,
	type IssueSortKey
} from '../lib/report';
import { useReviewVerdicts } from '../lib/hooks/useReviewVerdicts';
import reportStyles from './scan-report.css?url';
import severityStyles from '../styles/report.css?url';
import {
	getLocalBaseline,
	getLocalProject,
	getLocalRun,
	saveLocalBaseline,
	saveLocalRun
} from '../lib/local-project-store';
import {
	fingerprintProjectConfiguration,
	isUnifiedReport,
	type LocalBaseline,
	type LocalProject,
	type LocalRun
} from '../lib/projects';
import { pageTitle, SITE_NAME } from '../lib/site-metadata';

export const links = () => [
	{ rel: 'stylesheet', href: severityStyles },
	{ rel: 'stylesheet', href: reportStyles }
];

export const meta: MetaFunction = () => [
	{ title: pageTitle('Scan report') },
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
	const [searchParams] = useSearchParams();
	const requestedProjectId = searchParams.get('project');
	const sessionKey = `${id}:${requestedProjectId ?? ''}`;

	return <ScanReportSession key={sessionKey} id={id} requestedProjectId={requestedProjectId} />;
}

interface ScanReportSessionProps {
	id: string;
	requestedProjectId: string | null;
}

function ScanReportSession({ id, requestedProjectId }: ScanReportSessionProps) {
	const [searchParams, setSearchParams] = useSearchParams();
	const [localProject, setLocalProject] = useState<LocalProject | null>(null);
	const [localRun, setLocalRun] = useState<LocalRun | null>(null);
	const [localBaseline, setLocalBaseline] = useState<LocalBaseline | null>(null);
	const [projectMessage, setProjectMessage] = useState<string | null>(null);
	const [promotingBaseline, setPromotingBaseline] = useState(false);

	const { status, report, job, error, screenshots, refreshArtifacts } = useScanReport(id);

	const displayReport = useMemo(
		() => (report ? buildOccurrenceModeReport(report) : null),
		[report]
	);

	useEffect(() => {
		let cancelled = false;
		async function loadProjectContext() {
			try {
				const storedRun = await getLocalRun(id);
				const projectId = storedRun?.projectId ?? requestedProjectId;
				if (!projectId) return;
				const project = await getLocalProject(projectId);
				if (!project || cancelled) return;
				const run =
					storedRun ??
					({
						jobId: id,
						projectId,
						configFingerprint: await fingerprintProjectConfiguration(project.configuration),
						status: 'submitted',
						createdAt: new Date().toISOString()
					} satisfies LocalRun);
				const baseline = await getLocalBaseline(projectId);
				if (cancelled) return;
				setLocalProject(project);
				setLocalRun(run);
				setLocalBaseline(baseline);
			} catch (contextError) {
				if (!cancelled) {
					setProjectMessage(
						contextError instanceof Error
							? contextError.message
							: 'Could not load the local project context.'
					);
				}
			}
		}
		void loadProjectContext();
		return () => {
			cancelled = true;
		};
	}, [id, requestedProjectId]);

	useEffect(() => {
		if (
			!report ||
			!localRun ||
			localRun.status === 'complete' ||
			report.meta.jobId !== id ||
			localRun.jobId !== id
		) {
			return;
		}
		const completedRun: LocalRun = {
			...localRun,
			status: 'complete',
			completedAt: new Date().toISOString(),
			...(report.summary.score == null ? {} : { score: report.summary.score }),
			totalIssues: report.summary.totalIssues
		};
		saveLocalRun(completedRun)
			.then(() => setLocalRun(completedRun))
			.catch((saveError: unknown) => {
				setProjectMessage(
					saveError instanceof Error ? saveError.message : 'Could not update the local run history.'
				);
			});
	}, [id, localRun, report]);

	async function promoteAsBaseline() {
		if (!report || !localProject || !localRun) return;
		if (
			report.meta.jobId !== id ||
			localRun.jobId !== id ||
			localRun.projectId !== localProject.id
		) {
			setProjectMessage(
				'The local project context changed. Reload this report before promoting it.'
			);
			return;
		}
		if (!isUnifiedReport(report)) {
			setProjectMessage(`This report does not match the supported ${SITE_NAME} report contract.`);
			return;
		}
		setPromotingBaseline(true);
		setProjectMessage(null);
		try {
			const baseline: LocalBaseline = {
				projectId: localProject.id,
				jobId: id,
				configFingerprint: localRun.configFingerprint,
				report,
				createdAt: new Date().toISOString()
			};
			await saveLocalBaseline(baseline);
			setLocalBaseline(baseline);
			setProjectMessage('This report is now the local baseline.');
		} catch (saveError) {
			setProjectMessage(
				saveError instanceof Error ? saveError.message : 'Could not save this report as a baseline.'
			);
		} finally {
			setPromotingBaseline(false);
		}
	}

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
	const { getVerdict } = useReviewVerdicts(id);
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
		<>
			<SiteHeader
				app={{
					backTo: `/scan/${id}${localProject ? `?project=${encodeURIComponent(localProject.id)}` : ''}`,
					backLabel: 'Scan status',
					section: 'Report'
				}}
			/>

			<main id="main" className="report">
				<div className="wrap">
					{displayReport ? (
						<>
							{localProject && localRun && report && (
								<LocalBaselineComparison
									project={localProject}
									run={localRun}
									baseline={localBaseline}
									report={report}
									promoting={promotingBaseline}
									message={projectMessage}
									onPromote={() => void promoteAsBaseline()}
									onOpenIssue={(issueId) => updateParams({ section: 'issues', issue: issueId })}
								/>
							)}
							<ReportHeader report={displayReport} reviewProgress={reviewProgress} />
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
									<ArtifactsView
										jobId={id}
										job={job}
										onRefreshArtifacts={() => {
											void refreshArtifacts();
										}}
									/>
									<ErrorsView errors={displayReport.errors} />
								</section>
							)}

							{activeIssue && (
								<IssueDetailModal
									issue={activeIssue}
									jobId={id}
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

export function ErrorBoundary() {
	const error = useRouteError();
	const { pathname } = useLocation();
	const status = isRouteErrorResponse(error) ? error.status : 500;

	return (
		<>
			<SiteHeader app={{ backTo: '/playground', backLabel: 'New scan', section: 'Report' }} />
			<RouteFault
				status={status}
				title="This report can't be shown."
				detail="Scan jobs are kept for a limited window after they complete. The job id may have expired, or the report failed to load."
				traceLine={`report view ${pathname} failed to load`}
				traceHint="check the job id · try again or run a new scan"
				actions={
					<>
						<Link className="btn btn--primary" to="/playground">
							Run a new scan{' '}
							<span className="ar" aria-hidden="true">
								→
							</span>
						</Link>
						<a className="btn btn--ghost" href={pathname}>
							Try again
						</a>
					</>
				}
			/>
		</>
	);
}
