import { MAX_AUDIT_BYTES } from '../lib/portable-audit';
import { useEffect, useRef, useState, type SyntheticEvent } from 'react';
import { Link, useNavigate, type MetaFunction } from 'react-router';
import { ArrowRight, FolderPlus, Gauge, LockKeyhole, Trash2, Upload } from 'lucide-react';

import { ConfirmDialog } from '../components/ConfirmDialog';
import { Delta } from '../components/Delta';
import { ScoreHistory } from '../components/ScoreHistory';
import { SiteFooter } from '../components/SiteFooter';
import { SiteHeader } from '../components/SiteHeader';
import {
	createLocalProject,
	deleteLocalProject,
	exportLocalProject,
	getLocalBaseline,
	importLocalProject,
	listLocalProjects,
	listLocalProjectRuns
} from '../lib/local-project-store';
import { normalizeUrlInput, validateHttpUrl } from '../lib/url';
import { scoreBandFor, severityCounts, shownSiteScore } from '../lib/report';
import type { LocalProject, LocalRun } from '../lib/projects';
import { buildSiteMeta, pageTitle, SITE_NAME } from '../lib/site-metadata';
import projectsStyles from './projects.css?url';

export const links = () => [{ rel: 'stylesheet', href: projectsStyles }];

export const meta: MetaFunction = () =>
	buildSiteMeta({
		title: pageTitle('Local projects'),
		description: `Track ${SITE_NAME} scans and compare releases against a browser-local baseline.`,
		path: '/projects'
	});

interface ProjectListItem {
	project: LocalProject;
	runs: LocalRun[];
	/** Only what the card shows: a stored baseline holds a whole report. */
	baseline: { jobId: string; score: number | undefined } | null;
}

async function loadProjectList(): Promise<ProjectListItem[]> {
	const projects = await listLocalProjects();
	const [runs, baselines] = await Promise.all([
		Promise.all(projects.map((project) => listLocalProjectRuns(project.id))),
		Promise.all(projects.map((project) => getLocalBaseline(project.id)))
	]);
	// Promise.all preserves order, so runs[index] always pairs with projects[index].
	return projects.map((project, index) => {
		const baseline = baselines[index];
		return {
			project,
			runs: runs[index] ?? [],
			baseline: baseline
				? { jobId: baseline.jobId, score: shownSiteScore(baseline.report) ?? undefined }
				: null
		};
	});
}

export default function Projects() {
	const navigate = useNavigate();
	const [items, setItems] = useState<ProjectListItem[]>([]);
	const [loading, setLoading] = useState(true);
	const [storageError, setStorageError] = useState<string | null>(null);
	const [name, setName] = useState('');
	const [url, setUrl] = useState('');
	const [profile, setProfile] = useState<'staging' | 'production'>('staging');
	const [formError, setFormError] = useState<string | null>(null);
	const [creating, setCreating] = useState(false);
	const [pendingDelete, setPendingDelete] = useState<LocalProject | null>(null);
	const [deleting, setDeleting] = useState(false);
	const [importing, setImporting] = useState(false);
	const importInputRef = useRef<HTMLInputElement>(null);

	useEffect(() => {
		let cancelled = false;
		loadProjectList()
			.then((projects) => {
				if (cancelled) return;
				setItems(projects);
				setStorageError(null);
			})
			.catch((error: unknown) => {
				if (!cancelled) {
					setStorageError(error instanceof Error ? error.message : 'Local project storage failed.');
				}
			})
			.finally(() => {
				if (!cancelled) setLoading(false);
			});
		return () => {
			cancelled = true;
		};
	}, []);

	async function onCreate(event: SyntheticEvent) {
		event.preventDefault();
		setFormError(null);
		const projectName = name.trim();
		const normalizedUrl = normalizeUrlInput(url);
		if (!projectName) {
			setFormError('Give this project a name.');
			return;
		}
		if (!normalizedUrl) {
			setFormError('Enter the website URL for this project.');
			return;
		}
		const checkedUrl = validateHttpUrl(normalizedUrl);
		if (!checkedUrl.ok) {
			setFormError(checkedUrl.reason);
			return;
		}

		setCreating(true);
		try {
			const project = await createLocalProject(projectName, {
				urls: [checkedUrl.url],
				scanners: [],
				browser: 'chromium',
				highlightStyle: 'solid',
				profile
			});
			void navigate(`/playground?project=${encodeURIComponent(project.id)}`);
		} catch (error) {
			setFormError(error instanceof Error ? error.message : 'Could not create the project.');
			setCreating(false);
		}
	}

	async function onExport(project: LocalProject) {
		try {
			const payload = await exportLocalProject(project.id);
			const blob = new Blob([JSON.stringify(payload, null, 2)], { type: 'application/json' });
			const url = URL.createObjectURL(blob);
			const link = document.createElement('a');
			link.href = url;
			link.download = `${project.name.replaceAll(/[^\w.-]+/g, '-') || 'project'}.stageflow.json`;
			link.click();
			window.setTimeout(() => URL.revokeObjectURL(url), 1000);
		} catch (error) {
			setStorageError(error instanceof Error ? error.message : 'Could not export the project.');
		}
	}

	async function onImport(file: File | undefined) {
		if (!file) return;
		setImporting(true);
		try {
			if (file.size > MAX_AUDIT_BYTES)
				throw new Error('Project imports must be no larger than 96 MiB.');
			const parsed: unknown = JSON.parse(await file.text());
			await importLocalProject(parsed);
			setItems(await loadProjectList());
			setStorageError(null);
		} catch (error) {
			setStorageError(
				error instanceof Error ? error.message : 'Could not import that project file.'
			);
		} finally {
			setImporting(false);
		}
	}

	async function onDelete(project: LocalProject) {
		setDeleting(true);
		try {
			await deleteLocalProject(project.id);
			setItems(await loadProjectList());
			setStorageError(null);
		} catch (error) {
			setStorageError(error instanceof Error ? error.message : 'Could not delete the project.');
		} finally {
			setDeleting(false);
			setPendingDelete(null);
		}
	}

	return (
		<>
			<SiteHeader current="projects" />
			<main id="main" className="projects-page">
				<div className="wrap wrap--app projects-page__wrap">
					<header className="page-head">
						<div>
							<p className="label">Regression workspace</p>
							<h1>Local projects</h1>
							<p className="page-head__lede">
								Group repeat scans, promote a known-good report, and see what changed on the next
								run.
							</p>
						</div>
						<div className="projects-local-note">
							<LockKeyhole size={18} aria-hidden="true" />
							<span>
								Stored only in this browser. Export a project JSON if you want to move machines or
								keep a copy.
							</span>
						</div>
					</header>

					{storageError && (
						<div className="note note--err" role="alert">
							<strong>Local storage is unavailable.</strong> {storageError}
						</div>
					)}

					{/* The list leads, in the source as well as on screen: a returning
					    visitor's projects are the reason they came back, and stacking the
					    form first pushed every card below the fold on a short viewport.
					    Ordering it here rather than with CSS `order` keeps the tab order
					    matching what the eye sees at both widths. */}
					<div className="projects-grid">
						<section className="project-list" aria-labelledby="saved-projects-heading">
							<div className="project-list__head">
								<h2 id="saved-projects-heading">Your projects</h2>
								{!loading && <span className="num muted">{items.length}</span>}
								<div className="project-import">
									<button
										type="button"
										className="btn btn--ghost btn--sm"
										disabled={importing}
										onClick={() => importInputRef.current?.click()}
									>
										<Upload size={15} aria-hidden="true" />
										{importing ? 'Importing…' : 'Import'}
									</button>
									<input
										ref={importInputRef}
										type="file"
										accept="application/json,.json"
										hidden
										disabled={importing}
										onChange={(event) => {
											const file = event.target.files?.[0];
											event.target.value = '';
											void onImport(file);
										}}
									/>
								</div>
							</div>
							{loading ? (
								<div className="blankslate blankslate--framed" role="status">
									Loading local projects…
								</div>
							) : items.length === 0 ? (
								<div className="blankslate blankslate--framed">
									<Gauge size={24} aria-hidden="true" />
									<strong>No projects yet</strong>
									<span>Create one to turn your next good scan into a baseline.</span>
									<Link className="btn btn--ghost btn--sm" to="/demo">
										See an example comparison
									</Link>
								</div>
							) : (
								<div className="project-cards">
									{items.map(({ project, runs, baseline }) => {
										const latest = runs[0];
										// Oldest first. A run that failed or is still going has no score.
										const history = runs
											.flatMap((run) => {
												const score = run.report ? shownSiteScore(run.report) : run.score;
												return score == null
													? []
													: [{ id: run.jobId, score, at: run.completedAt ?? run.createdAt }];
											})
											.reverse();
										const latestScore = history.at(-1)?.score;
										const baselineScore = baseline?.score;
										const band = scoreBandFor(latestScore);
										const latestSeverities = latest?.report
											? severityCounts(latest.report.summary.bySeverity)
											: [];
										return (
											<article className="project-card" key={project.id}>
												<div className="project-card__body">
													<h3>
														<Link to={`/projects/${encodeURIComponent(project.id)}`}>
															{project.name}
														</Link>
													</h3>
													<p className="project-card__url mono">
														{project.configuration.urls[0] ?? 'No URL configured'}
													</p>
													<div className="project-card__stats">
														<span>
															{project.configuration.urls.length}{' '}
															{project.configuration.urls.length === 1 ? 'URL' : 'URLs'}
														</span>
														<span>
															{project.configuration.scanners.filter((s) => s.enabled).length ||
																project.configuration.scanners.length}{' '}
															scanners
														</span>
														{latest && (
															<span>
																Last run{' '}
																{new Date(
																	latest.completedAt ?? latest.createdAt
																).toLocaleDateString()}
															</span>
														)}
													</div>
													{runs.length === 0 ? (
														<p className="project-card__meta">No runs yet</p>
													) : (
														<div className="project-card__trend">
															<ScoreHistory
																points={history}
																baselineId={baseline?.jobId}
																baselineScore={baselineScore}
															/>
															<div>
																{latestScore !== undefined && (
																	<p className="project-card__score">
																		Latest score <strong className="num">{latestScore}</strong>
																		{band && (
																			<span
																				className={`project-card__band project-card__band--${band.tone}`}
																			>
																				{band.label}
																			</span>
																		)}
																		{baselineScore !== undefined && (
																			<>
																				{' '}
																				<span className="project-card__vs">
																					<Delta value={latestScore - baselineScore} /> vs baseline{' '}
																					{baselineScore}
																				</span>
																			</>
																		)}
																	</p>
																)}
																{latestSeverities.length > 0 && (
																	<p className="project-card__severities">
																		{latestSeverities.map(({ severity, count }) => (
																			<span key={severity} className="project-card__sev">
																				{count} {severity}
																			</span>
																		))}
																	</p>
																)}
																<p className="project-card__meta">
																	{runs.length} recent run{runs.length === 1 ? '' : 's'}
																	{latestScore !== undefined && !baseline && ' · no baseline yet'}
																</p>
															</div>
														</div>
													)}
												</div>
												<div className="project-card__actions">
													<Link
														className="btn btn--primary btn--sm"
														to={`/projects/${encodeURIComponent(project.id)}`}
													>
														{/* Honest per state: nothing to run "again" before a first run. */}
														Open project
													</Link>
													{latest && (
														<Link
															className="btn btn--ghost btn--sm"
															to={
																latest.status === 'complete'
																	? `/scan/${encodeURIComponent(latest.jobId)}/report?project=${encodeURIComponent(project.id)}`
																	: `/scan/${encodeURIComponent(latest.jobId)}?project=${encodeURIComponent(project.id)}`
															}
														>
															{latest.status === 'complete' ? 'Latest report' : 'View run'}
														</Link>
													)}
													<button
														className="btn btn--ghost btn--sm"
														type="button"
														onClick={() => void onExport(project)}
													>
														Export
													</button>
													<button
														className="btn btn--icon project-card__delete"
														type="button"
														onClick={() => setPendingDelete(project)}
														aria-label={`Delete ${project.name}`}
													>
														<Trash2 size={17} aria-hidden="true" />
													</button>
												</div>
											</article>
										);
									})}
								</div>
							)}
						</section>

						<section className="panel project-create" aria-labelledby="create-project-heading">
							<div className="panel__head">
								<h2 id="create-project-heading">Create a project</h2>
								<FolderPlus size={19} aria-hidden="true" />
							</div>
							<form
								className="panel__body project-create__form"
								onSubmit={(event) => {
									void onCreate(event);
								}}
								noValidate
							>
								<label className="field">
									<span className="label">Project name</span>
									<input
										className="input"
										placeholder="My website"
										value={name}
										onChange={(event) => setName(event.target.value)}
										maxLength={80}
										autoComplete="off"
									/>
								</label>
								<label className="field">
									<span className="label">Website URL</span>
									<input
										className="input"
										type="url"
										placeholder="https://example.com"
										value={url}
										onChange={(event) => setUrl(event.target.value)}
										autoComplete="url"
									/>
								</label>
								<label className="field">
									<span className="label">Assessment profile</span>
									<select
										className="input"
										value={profile}
										onChange={(event) => setProfile(event.target.value as 'staging' | 'production')}
									>
										<option value="staging">Staging / preview</option>
										<option value="production">Production</option>
									</select>
									<span className="muted">
										Staging treats confirmed indexing restrictions as expected. Other checks still
										apply.
									</span>
								</label>
								{formError && (
									<p className="project-create__error" role="alert">
										{formError}
									</p>
								)}
								<button className="btn btn--primary" type="submit" disabled={creating}>
									{creating ? 'Creating…' : 'Create and configure'}
									<ArrowRight size={17} aria-hidden="true" />
								</button>
							</form>
						</section>
					</div>
				</div>
			</main>
			{pendingDelete && (
				<ConfirmDialog
					title={`Delete “${pendingDelete.name}”?`}
					detail="The project, its run history, and its local baseline are removed from this browser. Completed scan reports on the server are unaffected."
					confirmLabel={deleting ? 'Deleting…' : 'Delete project'}
					destructive
					busy={deleting}
					onConfirm={() => void onDelete(pendingDelete)}
					onCancel={() => setPendingDelete(null)}
				/>
			)}
			<SiteFooter />
		</>
	);
}
