import { Link, useSearchParams, type MetaFunction } from 'react-router';
import {
	CircleCheck,
	Clock,
	FileArchive,
	FolderKanban,
	Info,
	KeyRound,
	Layers,
	Save,
	Target,
	X
} from 'lucide-react';

import { SiteHeader } from '../components/SiteHeader';
import { SiteFooter } from '../components/SiteFooter';
import { Pill } from '../components/Pill';
import { PlaygroundAuthConfig } from '../components/playground/PlaygroundAuthConfig';
import { usePlaygroundSession } from '../lib/hooks/usePlaygroundSession';
import { SCANNER_META, scannerLabel } from '../lib/report/scanner-identity';
import { buildSiteMeta, PLAYGROUND_DESCRIPTION, PLAYGROUND_TITLE } from '../lib/site-metadata';
import playgroundStyles from './playground.css?url';

export const links = () => [{ rel: 'stylesheet', href: playgroundStyles }];

export const meta: MetaFunction = () =>
	buildSiteMeta({
		title: PLAYGROUND_TITLE,
		description: PLAYGROUND_DESCRIPTION,
		path: '/playground'
	});

const CATEGORY_LABELS: Record<string, string> = {
	seo: 'SEO',
	custom: 'Custom'
};

function categoryLabel(s: string): string {
	if (CATEGORY_LABELS[s]) return CATEGORY_LABELS[s];
	return s ? s.charAt(0).toUpperCase() + s.slice(1) : s;
}

interface PlaygroundSessionProps {
	projectId: string | null;
	seedUrl: string | null;
}

/**
 * Keep execution-only form state scoped to the project query that created it.
 *
 * React Router reuses this route component when only the query string changes.
 * Keying the stateful session makes a project switch (including removing the
 * query) synchronously discard credentials, files, and draft form
 * values before the next project's asynchronous load starts.
 */
export default function Playground() {
	const [searchParams] = useSearchParams();
	const projectId = searchParams.get('project');
	const seedUrl = searchParams.get('url');
	const sessionKey = projectId ? `project:${projectId}` : `standalone:${seedUrl ?? ''}`;

	return <PlaygroundSession key={sessionKey} projectId={projectId} seedUrl={seedUrl} />;
}

function PlaygroundSession({ projectId, seedUrl }: PlaygroundSessionProps) {
	const {
		catalog,
		catalogError,
		catalogLoading,
		enabledById,
		toggleScanner,
		setAllScanners,
		armed,
		total,
		mode,
		setMode,
		urls,
		updateUrl,
		addUrlRow,
		removeUrlRow,
		targetCount,
		file,
		fileInputRef,
		onFilePicked,
		authConfig,
		setAuthConfig,
		isAuthValid,
		project,
		projectName,
		setProjectName,
		savingProject,
		projectNotice,
		saveProjectConfiguration,
		error,
		urlRowErrors,
		submitting,
		runScan,
		ready,
		readyDetail,
		runtimeEstimate
	} = usePlaygroundSession({ projectId, seedUrl });

	return (
		<>
			<SiteHeader current="playground" />

			<main id="main" className="console">
				<div className="wrap wrap--app">
					<div className="page-head console__head">
						<div>
							{project ? (
								<>
									{/* The editable name replaces the visible h1, but the page
									    still needs one for heading order and landmarks. */}
									<h1 className="sr-only">{projectName || 'Configure a scan'}</h1>
									<span className="console__project-label">
										<FolderKanban size={16} aria-hidden="true" /> Local project
									</span>
									<label className="sr-only" htmlFor="project-name">
										Project name
									</label>
									<input
										id="project-name"
										className="console__project-name"
										value={projectName}
										onChange={(event) => setProjectName(event.target.value)}
										maxLength={80}
									/>
								</>
							) : (
								<h1>Configure a scan</h1>
							)}
							<p className="page-head__lede">
								{project
									? 'Configure repeatable URL scans here. Credentials are prompted per run and never saved with the project.'
									: 'Point StageFlow at any public URL or a static-site archive, choose the scanners you need, and run. No account, no install.'}
							</p>
						</div>
						<div className="page-head__actions">
							{project && (
								<>
									<Link className="btn btn--ghost btn--sm" to="/projects">
										All projects
									</Link>
									<button
										className="btn btn--ghost btn--sm"
										type="button"
										onClick={() => void saveProjectConfiguration()}
										disabled={savingProject}
									>
										<Save size={16} aria-hidden="true" />
										{savingProject ? 'Saving…' : 'Save project'}
									</button>
								</>
							)}
							{submitting && <Pill variant="live">Submitting</Pill>}
						</div>
					</div>
					{projectNotice && (
						<p className="console__project-notice" role="status">
							{projectNotice}
						</p>
					)}

					<div className="console__grid">
						{/* left: configuration stack */}
						<div className="console__main">
							<section className="panel">
								<div className="panel__head">
									<div className="panel__head-copy">
										<h2>
											<span className="stepno num" aria-hidden="true">
												1
											</span>
											Target
										</h2>
										<p className="panel__sub">
											Add one or more URLs to scan, or upload a static-site archive.
										</p>
									</div>
									<div className="seg" role="group" aria-label="Target type">
										<button
											type="button"
											aria-pressed={mode === 'url'}
											onClick={() => setMode('url')}
										>
											URL
										</button>
										<button
											type="button"
											aria-pressed={mode === 'zip'}
											onClick={() => setMode('zip')}
											disabled={Boolean(project)}
											title={
												project ? 'Local projects support URL scans in this release.' : undefined
											}
										>
											ZIP upload
										</button>
									</div>
								</div>
								<div className="panel__body intake">
									{mode === 'url' ? (
										<>
											<div className="addurls">
												{urls.map((u, i) => (
													<div className="urlrow" key={i}>
														<div className="urlrow__field">
															<input
																id={`url-input-${i}`}
																className="input"
																type="url"
																value={u}
																placeholder="https://…"
																aria-label={`URL ${i + 1}`}
																aria-invalid={urlRowErrors[i] ? true : undefined}
																aria-describedby={urlRowErrors[i] ? `urlrow-err-${i}` : undefined}
																onChange={(e) => updateUrl(i, e.target.value)}
															/>
															{urlRowErrors[i] && (
																<span className="urlrow__err" id={`urlrow-err-${i}`} role="alert">
																	{urlRowErrors[i]}
																</span>
															)}
														</div>
														<button
															type="button"
															className="btn btn--icon rm"
															aria-label={`Remove URL ${i + 1}`}
															onClick={() => removeUrlRow(i)}
														>
															<X size={16} aria-hidden="true" />
														</button>
													</div>
												))}
											</div>
											<button
												type="button"
												className="btn btn--ghost btn--sm addmore"
												onClick={addUrlRow}
											>
												+ Add another URL
											</button>
											<p className="intake__note">
												<Info size={15} aria-hidden="true" />
												Each URL is scanned as its own page — add every page you want covered.
											</p>
										</>
									) : (
										<>
											<button
												id="zip-picker"
												type="button"
												className="drop"
												onClick={() => fileInputRef.current?.click()}
											>
												<FileArchive size={22} aria-hidden="true" />
												<span className="drop__file">
													{file ? file.name : 'Choose a ZIP archive'}
												</span>
												<span className="drop__hint">.zip · static-site build · up to 100MB</span>
											</button>
											<input
												ref={fileInputRef}
												type="file"
												accept=".zip"
												hidden
												onChange={(e) => onFilePicked(e.target.files?.[0] ?? null)}
											/>
										</>
									)}
								</div>
							</section>

							<section className="panel">
								<div className="panel__head">
									<div className="panel__head-copy">
										<h2>
											<span className="stepno num" aria-hidden="true">
												2
											</span>
											Scanners
										</h2>
										<p className="panel__sub">Choose the scanners you want to run.</p>
									</div>
									<div className="chanhead__meta">
										<span className="chanhead__count num">
											{catalogLoading
												? 'Loading…'
												: armed === total
													? `${total} scanners enabled`
													: `${armed} of ${total} enabled`}
										</span>
										{!catalogLoading && !catalogError && (
											<button
												type="button"
												className="btn btn--quiet btn--sm"
												onClick={() => setAllScanners(armed < total)}
											>
												{armed < total ? 'Enable all' : 'Disable all'}
											</button>
										)}
									</div>
								</div>
								<div className="panel__body panel__body--flush">
									{catalogError ? (
										<div className="note note--err note--inset" role="status">
											<span className="note__ic" aria-hidden="true">
												!
											</span>
											<span>{catalogError}</span>
										</div>
									) : (
										<div
											id="scanner-options"
											className="rack"
											role="group"
											aria-label="Scanners"
											tabIndex={-1}
										>
											{catalog.map((scanner) => {
												const on = enabledById.get(scanner.id) ?? false;
												const meta = SCANNER_META[scanner.id];
												const Icon = meta?.icon ?? Layers;
												return (
													<button
														type="button"
														className="chan"
														role="checkbox"
														aria-checked={on}
														key={scanner.id}
														onClick={() => toggleScanner(scanner.id)}
													>
														<span className="chan__icon" aria-hidden="true">
															<Icon size={17} />
														</span>
														<span className="chan__main">
															<span className="chan__name">
																{scannerLabel(scanner.id, scanner.name)}
															</span>
															<span
																className="chan__desc"
																title={scanner.description || meta?.description || undefined}
															>
																{scanner.description || meta?.description || ''}
															</span>
														</span>
														<span className="chan__cat">
															{categoryLabel(scanner.categories[0] ?? '')}
														</span>
														<span
															className={on ? 'toggle toggle--on' : 'toggle'}
															aria-hidden="true"
														/>
													</button>
												);
											})}
										</div>
									)}
								</div>
							</section>

							{mode === 'url' && (
								<PlaygroundAuthConfig
									config={authConfig}
									isValid={isAuthValid}
									onConfigChange={setAuthConfig}
								/>
							)}
						</div>

						{/* right: scan summary dock */}
						<aside className="dock" aria-label="Scan summary">
							<div className="panel">
								<div className="panel__head">
									<h2>Review & run</h2>
								</div>
								<div className="panel__body">
									<div
										className={ready ? 'readybox readybox--go' : 'readybox'}
										role="status"
										aria-live="polite"
									>
										<CircleCheck size={19} aria-hidden="true" />
										<span>
											<b>{ready ? 'Ready to run' : 'Almost there'}</b>
											{readyDetail}
										</span>
									</div>
									<ul className="sumlist">
										<li>
											<span className="sumlist__icon" aria-hidden="true">
												<Target size={16} />
											</span>
											<span className="sumlist__lab">
												Targets
												<small>
													{mode === 'url' ? 'Starting points for the scan' : 'Static-site archive'}
												</small>
											</span>
											<b className="num">
												{mode === 'url'
													? `${targetCount} ${targetCount === 1 ? 'URL' : 'URLs'}`
													: file
														? '1 archive'
														: 'None'}
											</b>
										</li>
										<li>
											<span className="sumlist__icon" aria-hidden="true">
												<Layers size={16} />
											</span>
											<span className="sumlist__lab">
												Scanners
												<small>{armed === total ? 'All enabled' : 'Partial selection'}</small>
											</span>
											<b className="num">
												{armed} of {total || 8}
											</b>
										</li>
										{mode === 'url' && (
											<li>
												<span className="sumlist__icon" aria-hidden="true">
													<KeyRound size={16} />
												</span>
												<span className="sumlist__lab">
													Authentication
													<small>
														{authConfig.enabled
															? isAuthValid
																? 'Form login before scanning'
																: 'Required fields missing'
															: 'Scanning public pages'}
													</small>
												</span>
												<b>{authConfig.enabled ? (isAuthValid ? 'On' : 'Incomplete') : 'Off'}</b>
											</li>
										)}
										<li>
											<span className="sumlist__icon" aria-hidden="true">
												<Clock size={16} />
											</span>
											<span className="sumlist__lab">
												Estimated runtime
												<small>{runtimeEstimate.detail}</small>
											</span>
											<b className="num">{runtimeEstimate.label}</b>
										</li>
									</ul>
									<button
										type="button"
										className="btn btn--primary btn--lg run"
										onClick={() => {
											void runScan();
										}}
										disabled={submitting || !ready}
									>
										{submitting ? 'Submitting…' : 'Run scan'}{' '}
										{!submitting && (
											<span className="ar" aria-hidden="true">
												→
											</span>
										)}
									</button>
									<p className="runhint">No account required · Free to use</p>
								</div>
							</div>
							{error && (
								<div className="note note--err" role="status">
									<span className="note__ic" aria-hidden="true">
										!
									</span>
									<span>{error}</span>
								</div>
							)}
						</aside>
					</div>
				</div>

				{/* Mobile: the review summary collapses into one sticky action bar —
				    the dock's Run button is hidden there so only one Run exists. */}
				<div className="stickyrun">
					<span className="stickyrun__sum num">
						{mode === 'url'
							? `${targetCount} ${targetCount === 1 ? 'URL' : 'URLs'}`
							: file
								? '1 archive'
								: 'No archive'}{' '}
						· {armed} of {total || 8} scanners · auth{' '}
						{mode === 'url' && authConfig.enabled ? (isAuthValid ? 'on' : 'incomplete') : 'off'} ·{' '}
						{runtimeEstimate.label}
					</span>
					<button
						type="button"
						className="btn btn--primary"
						onClick={() => {
							void runScan();
						}}
						disabled={submitting || !ready}
					>
						{submitting ? 'Submitting…' : 'Run scan'}{' '}
						{!submitting && (
							<span className="ar" aria-hidden="true">
								→
							</span>
						)}
					</button>
				</div>
			</main>

			<SiteFooter />
		</>
	);
}
