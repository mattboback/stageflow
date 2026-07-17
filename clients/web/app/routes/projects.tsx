import { useEffect, useState, type FormEvent } from 'react';
import { Link, useNavigate, type MetaFunction } from 'react-router';
import { ArrowRight, FolderPlus, Gauge, LockKeyhole, Trash2 } from 'lucide-react';

import { ConfirmDialog } from '../components/ConfirmDialog';
import { SiteFooter } from '../components/SiteFooter';
import { SiteHeader } from '../components/SiteHeader';
import {
	createLocalProject,
	deleteLocalProject,
	listLocalProjects,
	listLocalProjectRuns
} from '../lib/local-project-store';
import { normalizeUrlInput, validateHttpUrls } from '../lib/components/playground/playground-utils';
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
}

async function loadProjectList(): Promise<ProjectListItem[]> {
	const projects = await listLocalProjects();
	const runs = await Promise.all(projects.map((project) => listLocalProjectRuns(project.id)));
	return projects.map((project, index) => ({ project, runs: runs[index] }));
}

export default function Projects() {
	const navigate = useNavigate();
	const [items, setItems] = useState<ProjectListItem[]>([]);
	const [loading, setLoading] = useState(true);
	const [storageError, setStorageError] = useState<string | null>(null);
	const [name, setName] = useState('My website');
	const [url, setUrl] = useState('https://example.com');
	const [formError, setFormError] = useState<string | null>(null);
	const [creating, setCreating] = useState(false);
	const [pendingDelete, setPendingDelete] = useState<LocalProject | null>(null);
	const [deleting, setDeleting] = useState(false);

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

	async function onCreate(event: FormEvent) {
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
		const { valid, invalid } = validateHttpUrls([normalizedUrl]);
		if (invalid.length > 0) {
			setFormError(invalid[0].reason);
			return;
		}

		setCreating(true);
		try {
			const project = await createLocalProject(projectName, {
				urls: valid,
				scanners: [],
				browser: 'chromium',
				highlightStyle: 'solid'
			});
			navigate(`/playground?project=${encodeURIComponent(project.id)}`);
		} catch (error) {
			setFormError(error instanceof Error ? error.message : 'Could not create the project.');
			setCreating(false);
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
					<header className="projects-page__head">
						<div>
							<p className="label">Regression workspace</p>
							<h1>Local projects</h1>
							<p>
								Group repeat scans, promote a known-good report, and see what changed on the next
								run.
							</p>
						</div>
						<div className="projects-local-note">
							<LockKeyhole size={18} aria-hidden="true" />
							<span>Stored only in this browser. Projects are not uploaded or synchronized.</span>
						</div>
					</header>

					{storageError && (
						<div className="projects-alert" role="alert">
							<strong>Local storage is unavailable.</strong> {storageError}
						</div>
					)}

					<div className="projects-grid">
						<section className="panel project-create" aria-labelledby="create-project-heading">
							<div className="panel__head">
								<h2 id="create-project-heading">Create a project</h2>
								<FolderPlus size={19} aria-hidden="true" />
							</div>
							<form className="panel__body project-create__form" onSubmit={onCreate} noValidate>
								<label className="field">
									<span className="label">Project name</span>
									<input
										className="input"
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
										value={url}
										onChange={(event) => setUrl(event.target.value)}
										autoComplete="url"
									/>
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

						<section className="project-list" aria-labelledby="saved-projects-heading">
							<div className="project-list__head">
								<h2 id="saved-projects-heading">Your projects</h2>
								{!loading && <span className="num muted">{items.length}</span>}
							</div>
							{loading ? (
								<div className="project-empty" role="status">
									Loading local projects…
								</div>
							) : items.length === 0 ? (
								<div className="project-empty">
									<Gauge size={24} aria-hidden="true" />
									<strong>No projects yet</strong>
									<span>Create one to turn your next good scan into a baseline.</span>
								</div>
							) : (
								<div className="project-cards">
									{items.map(({ project, runs }) => {
										const latest = runs[0];
										return (
											<article className="project-card" key={project.id}>
												<div className="project-card__body">
													<h3>{project.name}</h3>
													<p className="project-card__url mono">
														{project.configuration.urls[0] ?? 'No URL configured'}
													</p>
													<p className="project-card__meta">
														{runs.length === 0
															? 'No runs yet'
															: `${runs.length} recent run${runs.length === 1 ? '' : 's'}${latest?.score === undefined ? '' : ` · latest score ${latest.score}`}`}
													</p>
												</div>
												<div className="project-card__actions">
													<Link
														className="btn btn--primary btn--sm"
														to={`/playground?project=${encodeURIComponent(project.id)}`}
													>
														Configure & run
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
														className="project-card__delete"
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
