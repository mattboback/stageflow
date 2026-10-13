import { useEffect, useRef, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router';
import { SiteHeader } from '../components/SiteHeader';
import { SiteFooter } from '../components/SiteFooter';
import { BaselineDiffSummary } from '../components/report/BaselineDiffSummary';
import {
	getLocalProject,
	getLocalBaseline,
	listLocalProjectRuns,
	exportLocalProject,
	importCliReport
} from '../lib/local-project-store';
import {
	computeProjectDiff,
	hasStableReportIdentity,
	type LocalProject,
	type LocalBaseline,
	type LocalRun
} from '../lib/projects';
import { MAX_AUDIT_BYTES } from '../lib/portable-audit';
import { shownSiteScore } from '../lib/report';
import projectsStyles from './projects.css?url';
import reportStyles from './scan-report.css?url';

export const links = () => [
	{ rel: 'stylesheet', href: projectsStyles },
	{ rel: 'stylesheet', href: reportStyles }
];

interface Workspace {
	project: LocalProject;
	baseline: LocalBaseline | null;
	runs: LocalRun[];
}
async function loadWorkspace(id: string): Promise<Workspace | null> {
	const [project, baseline, runs] = await Promise.all([
		getLocalProject(id),
		getLocalBaseline(id),
		listLocalProjectRuns(id)
	]);
	return project ? { project, baseline, runs } : null;
}
function runPath(projectId: string, run: LocalRun): string {
	return `/scan/${encodeURIComponent(run.jobId)}${run.report || run.status === 'complete' ? '/report' : ''}?project=${encodeURIComponent(projectId)}`;
}

export default function Project() {
	const { projectId = '' } = useParams();
	// Key the session so a late import or export cannot update another project.
	return <ProjectSession key={projectId} projectId={projectId} />;
}
function ProjectSession({ projectId }: { projectId: string }) {
	const navigate = useNavigate();
	const [workspace, setWorkspace] = useState<Workspace | null>(null);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);
	const [message, setMessage] = useState<string | null>(null);
	const [importing, setImporting] = useState(false);
	const input = useRef<HTMLInputElement>(null);
	useEffect(() => {
		let cancelled = false;
		const refresh = () => {
			void loadWorkspace(projectId)
				.then((value) => {
					if (!cancelled) {
						setWorkspace(value);
						setError(null);
					}
				})
				.catch((reason: unknown) => {
					if (!cancelled)
						setError(reason instanceof Error ? reason.message : 'Could not load this project.');
				})
				.finally(() => {
					if (!cancelled) setLoading(false);
				});
		};
		refresh();
		window.addEventListener('focus', refresh);
		return () => {
			cancelled = true;
			window.removeEventListener('focus', refresh);
		};
	}, [projectId]);
	async function exportProject() {
		try {
			const payload = await exportLocalProject(projectId);
			const url = URL.createObjectURL(
				new Blob([JSON.stringify(payload, null, 2)], { type: 'application/json' })
			);
			const anchor = document.createElement('a');
			anchor.href = url;
			anchor.download = `${workspace?.project.name.replaceAll(/[^\w.-]+/g, '-') || 'project'}.stageflow.json`;
			anchor.click();
			window.setTimeout(() => URL.revokeObjectURL(url), 1000);
			setMessage('Project exported with its saved baseline and recent reports.');
		} catch (reason) {
			setError(reason instanceof Error ? reason.message : 'Could not export this project.');
		}
	}
	async function importReport(file: File | undefined) {
		if (!file) return;
		setImporting(true);
		setError(null);
		setMessage(null);
		try {
			if (file.size > MAX_AUDIT_BYTES)
				throw new Error('Report imports must be no larger than 96 MiB.');
			const value: unknown = JSON.parse(await file.text());
			await importCliReport(projectId, value);
			setWorkspace(await loadWorkspace(projectId));
			setMessage('CLI report saved. Your configuration and accepted baseline have not changed.');
		} catch (reason) {
			setError(reason instanceof Error ? reason.message : 'Could not import this report.');
		} finally {
			setImporting(false);
		}
	}
	const project = workspace?.project;
	const baseline = workspace?.baseline;
	const latest = workspace?.runs.find(
		(run) =>
			run.report && !run.verification && run.report.meta.scanContext?.purpose !== 'verification'
	);
	const report = latest?.report;
	const compatible =
		baseline &&
		latest &&
		report &&
		(baseline.configFingerprint === latest.configFingerprint ||
			Boolean(baseline.report.meta.scanContext && report.meta.scanContext)) &&
		hasStableReportIdentity(baseline.report) === hasStableReportIdentity(report);
	const diff =
		compatible && baseline && latest && report
			? computeProjectDiff(baseline.jobId, baseline.report, latest.jobId, report)
			: null;
	return (
		<>
			<SiteHeader current="projects" />
			<main id="main" className="projects-page">
				<div className="wrap wrap--app projects-page__wrap">
					<Link to="/projects">All projects</Link>
					{error && (
						<p className="note note--err" role="alert">
							{error}
						</p>
					)}
					{message && (
						<p className="note" role="status">
							{message}
						</p>
					)}
					{loading ? (
						<p role="status">Loading project…</p>
					) : !project ? (
						<section className="blankslate blankslate--framed">
							<h1>Project unavailable</h1>
							<p>
								This project is not stored in this browser. Import its project export from the
								projects page to restore it.
							</p>
						</section>
					) : (
						<>
							<header className="page-head">
								<div>
									<p className="label">Project workspace</p>
									<h1>{project.name}</h1>
									<p className="page-head__lede">
										Scan your saved pages, review the evidence, and compare the next deployment.
									</p>
								</div>
							</header>
							<div className="workspace-actions">
								<Link
									className="btn btn--primary"
									to={`/playground?project=${encodeURIComponent(projectId)}`}
								>
									Run scan
								</Link>
								{latest && (
									<Link className="btn btn--ghost" to={runPath(projectId, latest)}>
										Review findings
									</Link>
								)}
								{baseline && latest && (
									<a className="btn btn--ghost" href="#project-comparison">
										Compare with baseline
									</a>
								)}
								<button
									className="btn btn--ghost"
									type="button"
									onClick={() => void exportProject()}
								>
									Export project
								</button>
								<button
									className="btn btn--ghost"
									type="button"
									disabled={importing}
									onClick={() => input.current?.click()}
								>
									{importing ? 'Importing…' : 'Import CLI report'}
								</button>
								<input
									ref={input}
									type="file"
									hidden
									accept="application/json,.json"
									aria-label="CLI report file"
									disabled={importing}
									onChange={(event) => {
										const file = event.target.files?.[0];
										event.target.value = '';
										void importReport(file);
									}}
								/>
							</div>
							<p className="projects-local-note">
								Stored only in this browser. Browser data can be cleared or evicted. Export a backup
								to keep your configuration, baseline, and reports.
							</p>
							<div className="workspace-grid">
								<section className="panel" aria-labelledby="saved-settings">
									<div className="panel__head">
										<h2 id="saved-settings">Saved scan settings</h2>
										<Link to={`/playground?project=${encodeURIComponent(projectId)}`}>
											Edit settings
										</Link>
									</div>
									<div className="panel__body workspace-content">
										<p>
											<strong>
												{project.configuration.profile === 'staging' ? 'Staging' : 'Production'}
											</strong>{' '}
											assessment · Chromium
										</p>
										<p>
											Scanners:{' '}
											{project.configuration.scanners.length
												? project.configuration.scanners
														.filter((scanner) => scanner.enabled)
														.map((scanner) => scanner.id)
														.join(', ') || 'No scanners enabled'
												: 'All available scanners'}
										</p>
										<h3>
											{project.configuration.urls.length} saved page
											{project.configuration.urls.length === 1 ? '' : 's'}
										</h3>
										<ul className="workspace-pages">
											{project.configuration.urls.map((url) => (
												<li key={url}>
													<a href={url} target="_blank" rel="noreferrer">
														{url}
													</a>
												</li>
											))}
										</ul>
										<p className="muted">
											Repeat scans use these pages. Review and save any discovered pages in scan
											settings before changing coverage.
										</p>
									</div>
								</section>
								<section className="panel" aria-labelledby="accepted-baseline">
									<div className="panel__head">
										<h2 id="accepted-baseline">Accepted baseline</h2>
									</div>
									<div className="panel__body workspace-content">
										{baseline ? (
											<>
												<p>Accepted {new Date(baseline.createdAt).toLocaleString()}</p>
												<Link
													to={`/scan/${encodeURIComponent(baseline.jobId)}/report?project=${encodeURIComponent(projectId)}`}
												>
													Review baseline report
												</Link>
												<p>The baseline is kept separately from recent run history.</p>
											</>
										) : (
											<>
												<p>No baseline yet</p>
												<p>
													Review a complete report and choose “Promote as baseline” to start
													comparing deployments.
												</p>
											</>
										)}
										<p className="muted">
											Targeted fix checks provide feedback; use a complete repeat scan before
											replacing the baseline.
										</p>
									</div>
								</section>
							</div>
							<section
								id="project-comparison"
								className="panel"
								aria-labelledby="comparison-heading"
							>
								<div className="panel__head">
									<h2 id="comparison-heading">Latest full scan vs baseline</h2>
								</div>
								<div className="panel__body">
									{!latest ? (
										<p>
											No full scan report saved yet. Run a scan or import a CLI report to begin.
										</p>
									) : !baseline ? (
										<p>Review your findings and accept a baseline to see what changes next.</p>
									) : baseline.jobId === latest.jobId ? (
										<p>
											The latest scan is your accepted baseline. Deploy your changes and run the
											same saved pages again.
										</p>
									) : diff && report ? (
										<BaselineDiffSummary
											diff={diff}
											shownScore={shownSiteScore(report)}
											onOpenIssue={(issueId) => {
												void navigate(
													`${runPath(projectId, latest)}&section=issues&issue=${encodeURIComponent(issueId)}`
												);
											}}
										/>
									) : (
										<p role="status">
											Comparison unavailable: scan configuration or finding identities changed.
											Review both reports before accepting a new baseline. Missing findings are not
											verified fixes.
										</p>
									)}
								</div>
							</section>
							<section className="panel" aria-labelledby="recent-runs">
								<div className="panel__head">
									<h2 id="recent-runs">Recent runs</h2>
									<span className="muted">Up to 10 runs</span>
								</div>
								<div className="panel__body">
									{workspace.runs.length === 0 ? (
										<p>No runs yet. Your first scan starts the history.</p>
									) : (
										<ul className="workspace-runs">
											{workspace.runs.map((run) => (
												<li key={run.jobId}>
													<div>
														<strong>
															{run.verification ? 'Targeted fix verification' : 'Full scan'}
														</strong>
														<p>
															{new Date(run.createdAt).toLocaleString()} ·{' '}
															{run.status === 'submitted'
																? 'Submitted — open to follow progress'
																: run.status === 'failed'
																	? 'Failed — review scan details'
																	: 'Report saved'}
															{baseline?.jobId === run.jobId ? ' · Accepted baseline' : ''}
														</p>
													</div>
													<Link className="btn btn--ghost btn--sm" to={runPath(projectId, run)}>
														{run.report || run.status === 'complete'
															? 'Open report'
															: 'View progress'}
													</Link>
												</li>
											))}
										</ul>
									)}
								</div>
							</section>
						</>
					)}
				</div>
			</main>
			<SiteFooter />
		</>
	);
}
