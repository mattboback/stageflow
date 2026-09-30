import { VerifyFindingDialog } from '../components/report/VerifyFindingDialog';
import { VerificationResultBanner } from '../components/report/VerificationResultBanner';
import { PortableAuditExportButton } from '../components/report/PortableAuditExportButton';
import {
	ReviewVerdictProvider,
	createReviewContext,
	getBrowserReviewStore
} from '../lib/hooks/useReviewVerdicts';
import type { ReviewContext } from '../lib/report/review-context';
import type { ReviewVerdictStore } from '../lib/report/review-verdict-store';
import { loadProjectReviewStore } from '../lib/local-project-store';
import { fetchWithTimeout } from '../lib/api/client';
import { buildApiUrl } from '../lib/api/utils';
import { preparePortableAudit, portableScreenshotDigests } from '../lib/portable-audit';
import type { ScanResult } from '../lib/types/scan';
import { useCallback, useEffect, useState } from 'react';
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
import { SiteFooter } from '../components/SiteFooter';
import { RouteFault } from '../components/RouteFault';
import { ArtifactsView } from '../components/report/ArtifactsView';
import { LocalBaselineComparison } from '../components/report/LocalBaselineComparison';
import { ReportJobActions } from '../components/report/ReportJobActions';
import { ReportView } from '../components/report/ReportView';
import { useScanReport } from '../lib/hooks/useScanMonitor';
import reportStyles from './scan-report.css?url';
import severityStyles from '../styles/report.css?url';
import {
	createLocalProject,
	getLocalBaseline,
	getLocalProject,
	getLocalRun,
	saveLocalBaseline,
	saveLocalRun
} from '../lib/local-project-store';
import {
	buildArchivedLocalRun,
	computeProjectDiff,
	fingerprintProjectConfiguration,
	isUnifiedReport,
	type LocalBaseline,
	type LocalProject,
	type LocalProjectConfiguration,
	type LocalRun
} from '../lib/projects';
import { pageTitle, SITE_NAME } from '../lib/site-metadata';
import type { IssueDetail, UnifiedReport } from '../lib/types/unified-report';

export const links = () => [
	{ rel: 'stylesheet', href: severityStyles },
	{ rel: 'stylesheet', href: reportStyles }
];

export const meta: MetaFunction = () => [
	{ title: pageTitle('Scan report') },
	{ name: 'robots', content: 'noindex' }
];

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

/*
 * Everything about *obtaining* a report: the job poll, the local-project
 * context, and promoting a run to a baseline. Presentation lives in
 * ReportView, which /demo renders from a static fixture with no API at all.
 */
function ScanReportSession({ id, requestedProjectId }: ScanReportSessionProps) {
	const [, setSearchParams] = useSearchParams();
	const location = useLocation();
	const [localProject, setLocalProject] = useState<LocalProject | null>(null);
	const [localRun, setLocalRun] = useState<LocalRun | null>(null);
	const [localBaseline, setLocalBaseline] = useState<LocalBaseline | null>(null);
	// Saving adds ?project=, which remounts this session; the confirmation
	// arrives through navigation state.
	const [projectMessage, setProjectMessage] = useState<string | null>(() =>
		readProjectMessage(location.state)
	);
	const [promotingBaseline, setPromotingBaseline] = useState(false);
	const [verificationIssue, setVerificationIssue] = useState<IssueDetail | null>(null);
	const [projectContextLoaded, setProjectContextLoaded] = useState(false);
	const [reviewLoadError, setReviewLoadError] = useState<string | null>(null);
	const [reviewLoadAttempt, setReviewLoadAttempt] = useState(0);
	const [reviewSession, setReviewSession] = useState<{
		store: ReviewVerdictStore;
		context: ReviewContext;
	} | null>(null);

	const {
		status,
		report: liveReport,
		job,
		error,
		screenshots,
		refreshArtifacts
	} = useScanReport(id);
	const archivedReport =
		(localRun?.archived || !liveReport) && localRun?.report && isUnifiedReport(localRun.report)
			? localRun.report
			: null;
	const report = archivedReport ?? liveReport;
	const viewingArchived = Boolean(archivedReport);
	useEffect(() => {
		let cancelled = false;
		if (!report || !projectContextLoaded) return;
		const reportForContext = report;
		async function loadReviews() {
			try {
				const contrastPages = new Set(
					reportForContext.issues
						.filter(
							(issue) =>
								issue.ruleId === 'color-contrast' && issue.scannerData?.axeIncomplete === true
						)
						.map((issue) => issue.pageId)
				);
				const evidenceShots = (
					viewingArchived ? (localRun?.screenshots ?? []) : screenshots
				).filter((shot) => contrastPages.has(shot.page_id) && shot.scanner_id === 'axe');
				const screenshotDigests = evidenceShots.length
					? await portableScreenshotDigests(
							(
								await preparePortableAudit({
									jobId: id,
									report: reportForContext,
									screenshots: evidenceShots
								})
							).screenshots
						)
					: {};
				const [store, context] = await Promise.all([
					localProject
						? loadProjectReviewStore(localProject.id, (error) =>
								setProjectMessage(error instanceof Error ? error.message : String(error))
							)
						: Promise.resolve(getBrowserReviewStore()),
					createReviewContext(
						reportForContext,
						localProject?.id,
						localRun?.configFingerprint,
						screenshotDigests
					)
				]);
				if (!cancelled) {
					setReviewLoadError(null);
					setReviewSession({ store, context });
				}
			} catch (error) {
				if (!cancelled)
					setReviewLoadError(
						error instanceof Error ? error.message : 'Could not load saved reviews.'
					);
			}
		}
		void loadReviews();
		return () => {
			cancelled = true;
		};
	}, [
		report,
		localProject,
		localRun?.configFingerprint,
		id,
		viewingArchived,
		localRun?.screenshots,
		screenshots,
		projectContextLoaded,
		reviewLoadAttempt
	]);

	useEffect(() => {
		let cancelled = false;
		async function loadProjectContext() {
			try {
				const storedRun = await getLocalRun(id, requestedProjectId ?? undefined);
				const projectId = requestedProjectId ?? storedRun?.projectId;
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
			} finally {
				if (!cancelled) setProjectContextLoaded(true);
			}
		}
		void loadProjectContext();
		return () => {
			cancelled = true;
		};
	}, [id, requestedProjectId]);

	useEffect(() => {
		if (!report || !localRun) return;
		const completedRun = buildArchivedLocalRun(localRun, report, id);
		if (!completedRun) return;
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
		if (report.meta.scanContext?.purpose === 'verification') {
			setProjectMessage('Targeted verification cannot replace a complete baseline.');
			return;
		}
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

	/* The banner's only navigation. ReportView owns the rest of the query
	   string; this opens a specific finding from the baseline comparison. */
	const openIssue = useCallback(
		(issueId: string) => {
			setSearchParams(
				(prev) => {
					const next = new URLSearchParams(prev);
					next.set('section', 'issues');
					next.set('issue', issueId);
					return next;
				},
				{ replace: true, preventScrollReset: true }
			);
		},
		[setSearchParams]
	);

	async function saveOneOffProject(name: string) {
		if (report?.meta.scanContext?.purpose === 'verification')
			throw new Error('Targeted verification cannot become a complete baseline.');
		if (!report || !isUnifiedReport(report)) {
			throw new Error(`This report does not match the supported ${SITE_NAME} report contract.`);
		}
		const configuration = configurationFromReport(report);
		const project = await createLocalProject(name, configuration);
		const run: LocalRun = {
			jobId: id,
			projectId: project.id,
			configFingerprint: await fingerprintProjectConfiguration(configuration),
			status: 'complete',
			createdAt: report.meta.scannedAt ?? new Date().toISOString(),
			completedAt: report.meta.completedAt ?? new Date().toISOString(),
			...(report.summary.score == null ? {} : { score: report.summary.score }),
			totalIssues: report.summary.totalIssues,
			report
		};
		await saveLocalRun(run);
		await saveLocalBaseline({
			projectId: project.id,
			jobId: id,
			configFingerprint: run.configFingerprint,
			report,
			createdAt: new Date().toISOString()
		});
		setLocalProject(project);
		setLocalRun(run);
		setLocalBaseline(await getLocalBaseline(project.id));
		setSearchParams(
			(prev) => {
				const next = new URLSearchParams(prev);
				next.set('project', project.id);
				return next;
			},
			{
				replace: true,
				preventScrollReset: true,
				state: { projectMessage: 'Saved in this browser and promoted as the local baseline.' }
			}
		);
	}

	const comparison =
		report &&
		localRun &&
		localBaseline &&
		(localBaseline.configFingerprint === localRun.configFingerprint ||
			Boolean(localBaseline.report.meta.scanContext && report.meta.scanContext))
			? computeProjectDiff(localBaseline.jobId, localBaseline.report, id, report)
			: null;

	/* Raw, not occurrence-expanded: LocalBaselineComparison and saveLocalBaseline
	   both persist and diff the report exactly as it arrived from the API. */
	const banner = report ? (
		<>
			<ReportJobActions
				jobId={id}
				report={report}
				archived={viewingArchived}
				canDelete={!viewingArchived}
				showSave={!localProject && report.meta.scanContext?.purpose !== 'verification'}
				onSaveProject={saveOneOffProject}
			/>
			<PortableAuditExportButton
				jobId={id}
				report={report}
				{...(localRun?.verification ? { verification: localRun.verification } : {})}
				{...(comparison ? { comparison } : {})}
				screenshots={viewingArchived ? (localRun?.screenshots ?? []) : screenshots}
				{...(localProject ? { configuration: localProject.configuration } : {})}
				{...(localBaseline ? { baseline: localBaseline.report } : {})}
				{...(!viewingArchived
					? {
							refreshScreenshots: async () => {
								const response = await fetchWithTimeout(
									buildApiUrl(`/api/v1/jobs/${encodeURIComponent(id)}`)
								);
								if (!response.ok) throw new Error('Could not refresh screenshot links');
								const current = (await response.json()) as ScanResult;
								return current.artifacts?.screenshots ?? [];
							}
						}
					: {})}
			/>
			{localProject && localRun && report.meta.scanContext?.purpose !== 'verification' ? (
				<LocalBaselineComparison
					project={localProject}
					run={localRun}
					baseline={localBaseline}
					report={report}
					promoting={promotingBaseline}
					message={projectMessage}
					onPromote={() => void promoteAsBaseline()}
					onOpenIssue={openIssue}
				/>
			) : null}
		</>
	) : null;

	return (
		<>
			<SiteHeader
				app={{
					backTo: `/scan/${id}${localProject ? `?project=${encodeURIComponent(localProject.id)}` : ''}`,
					backLabel: 'Scan status',
					section: 'Report'
				}}
			/>
			<VerificationResultBanner
				report={report}
				jobId={id}
				status={status}
				{...(localRun?.verification ? { context: localRun.verification } : {})}
			/>
			<ReviewVerdictProvider
				store={reviewSession?.store ?? getBrowserReviewStore()}
				{...(reviewSession ? { context: reviewSession.context } : {})}
			>
				{!report ||
				(projectContextLoaded &&
					reviewSession?.context.jobId === id &&
					reviewSession.context.projectId === localProject?.id) ? (
					<ReportView
						jobId={id}
						onVerify={setVerificationIssue}
						comparison={comparison}
						report={report}
						screenshots={viewingArchived ? (localRun?.screenshots ?? []) : screenshots}
						status={status}
						error={error || job?.error || null}
						banner={banner}
						artifactsPanel={
							<ArtifactsView
								jobId={viewingArchived ? '' : id}
								job={viewingArchived ? null : job}
								{...(viewingArchived
									? {}
									: {
											onRefreshArtifacts: () => {
												void refreshArtifacts();
											}
										})}
							/>
						}
					/>
				) : (
					<div>
						{reviewLoadError ? (
							<>
								<p role="alert">{reviewLoadError}</p>
								<button
									className="btn btn--secondary"
									type="button"
									onClick={() => {
										setReviewLoadError(null);
										setReviewLoadAttempt((value) => value + 1);
									}}
								>
									Retry loading reviews
								</button>
							</>
						) : (
							<p role="status">Loading saved reviews…</p>
						)}
					</div>
				)}
			</ReviewVerdictProvider>
			{report && verificationIssue && (
				<VerifyFindingDialog
					report={report}
					issue={verificationIssue}
					onClose={() => setVerificationIssue(null)}
				/>
			)}
			<SiteFooter slim />
		</>
	);
}

function configurationFromReport(report: UnifiedReport): LocalProjectConfiguration {
	const urls = report.pages.map((page) => page.url).filter(Boolean);
	return {
		urls: urls.length > 0 ? urls : ['https://example.com'],
		...(report.meta.profile === 'staging' ? { profile: 'staging' as const } : {}),
		scanners: report.scanners.map((scanner) => ({
			id: scanner.id,
			enabled: true,
			...(report.meta.scanContext?.scannerConfigurations?.[scanner.id]
				? { config: report.meta.scanContext.scannerConfigurations[scanner.id] }
				: {})
		})),
		browser: 'chromium',
		highlightStyle: 'solid'
	};
}

function readProjectMessage(state: unknown): string | null {
	if (typeof state !== 'object' || state === null || !('projectMessage' in state)) return null;
	return typeof state.projectMessage === 'string' ? state.projectMessage : null;
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
							Configure a scan{' '}
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
			<SiteFooter slim />
		</>
	);
}
