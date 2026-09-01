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
	fingerprintProjectConfiguration,
	isUnifiedReport,
	type LocalBaseline,
	type LocalProject,
	type LocalProjectConfiguration,
	type LocalRun
} from '../lib/projects';
import { pageTitle, SITE_NAME } from '../lib/site-metadata';
import type { UnifiedReport } from '../lib/types/unified-report';

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
	const [localProject, setLocalProject] = useState<LocalProject | null>(null);
	const [localRun, setLocalRun] = useState<LocalRun | null>(null);
	const [localBaseline, setLocalBaseline] = useState<LocalBaseline | null>(null);
	const [projectMessage, setProjectMessage] = useState<string | null>(null);
	const [promotingBaseline, setPromotingBaseline] = useState(false);

	const {
		status,
		report: liveReport,
		job,
		error,
		screenshots,
		refreshArtifacts
	} = useScanReport(id);
	const archivedReport =
		!liveReport && localRun?.report && isUnifiedReport(localRun.report) ? localRun.report : null;
	const report = liveReport ?? archivedReport;
	const viewingArchived = Boolean(archivedReport && !liveReport);

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
			{ replace: true, preventScrollReset: true }
		);
		setProjectMessage('Saved in this browser and promoted as the local baseline.');
	}

	/* Raw, not occurrence-expanded: LocalBaselineComparison and saveLocalBaseline
	   both persist and diff the report exactly as it arrived from the API. */
	const banner = report ? (
		<>
			<ReportJobActions
				jobId={id}
				report={report}
				archived={viewingArchived}
				canDelete={!viewingArchived}
				showSave={!localProject}
				onSaveProject={saveOneOffProject}
			/>
			{localProject && localRun ? (
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
			<ReportView
				jobId={id}
				report={report}
				screenshots={screenshots}
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
			<SiteFooter slim />
		</>
	);
}

function configurationFromReport(report: UnifiedReport): LocalProjectConfiguration {
	const urls = report.meta.baseUrl
		? [report.meta.baseUrl]
		: report.pages.map((page) => page.url).filter(Boolean);
	return {
		urls: urls.length > 0 ? urls : ['https://example.com'],
		scanners: report.scanners.map((scanner) => ({ id: scanner.id, enabled: true })),
		browser: 'chromium',
		highlightStyle: 'solid'
	};
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
