import { useEffect, useMemo, useRef, useState } from 'react';
import { Link, useNavigate, useSearchParams, type MetaFunction } from 'react-router';
import {
	CircleCheck,
	Clock,
	FileArchive,
	FolderKanban,
	Info,
	KeyRound,
	Layers,
	Save,
	Target
} from 'lucide-react';
import { SiteHeader } from '../components/SiteHeader';
import { SiteFooter } from '../components/SiteFooter';
import { Pill } from '../components/Pill';
import { fetchScanners, getDefaultScannerSelections, submitScanJob } from '../lib/api/client';
import {
	buildAiNavigatorConfig,
	buildFormAuthConfig,
	DEFAULT_AI_CONFIG,
	estimateScanRuntime,
	isAuthConfigComplete,
	normalizeUrlInput,
	validateHttpUrls,
	validatePlaygroundConfiguration,
	validateZipUploadFile,
	type AiConfigState,
	type AuthFormConfig
} from '../lib/components/playground/playground-utils';
import type { ScannerDefinition, ScannerSelection } from '../lib/types/scan';
import { SCANNER_META, scannerLabel } from '../lib/report/scanner-identity';
import { PlaygroundAuthConfig } from '../components/playground/PlaygroundAuthConfig';
import { PlaygroundAiConfig } from '../components/playground/PlaygroundAiConfig';
import { buildSiteMeta, PLAYGROUND_DESCRIPTION, PLAYGROUND_TITLE } from '../lib/site-metadata';
import { getLocalProject, saveLocalProject, saveLocalRun } from '../lib/local-project-store';
import {
	fingerprintProjectConfiguration,
	type LocalProject,
	type LocalProjectConfiguration
} from '../lib/projects';
import playgroundStyles from './playground.css?url';

export const links = () => [{ rel: 'stylesheet', href: playgroundStyles }];

export const meta: MetaFunction = () =>
	buildSiteMeta({
		title: PLAYGROUND_TITLE,
		description: PLAYGROUND_DESCRIPTION,
		path: '/playground'
	});

type Mode = 'url' | 'zip';

const CATEGORY_LABELS: Record<string, string> = {
	seo: 'SEO',
	ai: 'AI',
	custom: 'AI, opt-in'
};

function categoryLabel(s: string): string {
	if (CATEGORY_LABELS[s]) return CATEGORY_LABELS[s];
	return s ? s.charAt(0).toUpperCase() + s.slice(1) : s;
}

function mergeProjectSelections(
	catalog: ScannerDefinition[],
	stored: ScannerSelection[]
): ScannerSelection[] {
	if (stored.length === 0) return getDefaultScannerSelections(catalog);
	const storedById = new Map(stored.map((selection) => [selection.id, selection]));
	return catalog.map((scanner) => storedById.get(scanner.id) ?? { id: scanner.id, enabled: false });
}

function preserveUnavailableProjectSelections(
	catalog: ScannerDefinition[],
	current: ScannerSelection[],
	stored: ScannerSelection[]
): ScannerSelection[] {
	const availableIds = new Set(catalog.map((scanner) => scanner.id));
	return [...current, ...stored.filter((selection) => !availableIds.has(selection.id))];
}

function aiStateFromSelections(selections: ScannerSelection[]): AiConfigState {
	const config = selections.find((selection) => selection.id === 'ai-navigator')?.config;
	const goal = config?.goal as Record<string, unknown> | undefined;
	const vision = config?.vision as Record<string, unknown> | undefined;
	const criteria = Array.isArray(goal?.successCriteria)
		? goal.successCriteria.flatMap((criterion) => {
				if (!criterion || typeof criterion !== 'object') return [];
				const candidate = criterion as Record<string, unknown>;
				return typeof candidate.type === 'string' && typeof candidate.value === 'string'
					? [{ type: candidate.type, value: candidate.value }]
					: [];
			})
		: [];

	return {
		objective: typeof goal?.objective === 'string' ? goal.objective : DEFAULT_AI_CONFIG.objective,
		maxSteps: typeof goal?.maxSteps === 'number' ? goal.maxSteps : DEFAULT_AI_CONFIG.maxSteps,
		maxWallTimeMs:
			typeof goal?.maxWallTimeMs === 'number'
				? goal.maxWallTimeMs
				: DEFAULT_AI_CONFIG.maxWallTimeMs,
		model: typeof vision?.model === 'string' ? vision.model : DEFAULT_AI_CONFIG.model,
		inputValues: [],
		successCriteria: criteria
	};
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
 * query) synchronously discard credentials, AI inputs, files, and draft form
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
	const navigate = useNavigate();

	const [catalog, setCatalog] = useState<ScannerDefinition[]>([]);
	const [selections, setSelections] = useState<ScannerSelection[]>([]);
	const [catalogError, setCatalogError] = useState<string | null>(null);
	const [catalogLoading, setCatalogLoading] = useState(true);
	const [project, setProject] = useState<LocalProject | null>(null);
	const [projectName, setProjectName] = useState('');
	const [savingProject, setSavingProject] = useState(false);
	const [projectNotice, setProjectNotice] = useState<string | null>(null);

	const [mode, setMode] = useState<Mode>('url');
	const [urls, setUrls] = useState<string[]>(() => {
		return seedUrl ? [seedUrl, ''] : ['https://example.com', ''];
	});
	const [file, setFile] = useState<File | null>(null);
	const fileInputRef = useRef<HTMLInputElement>(null);

	const [error, setError] = useState<string | null>(null);
	const [urlRowErrors, setUrlRowErrors] = useState<Record<number, string>>({});
	const [submitting, setSubmitting] = useState(false);

	const [authConfig, setAuthConfig] = useState<AuthFormConfig>({
		enabled: false,
		loginUrl: '',
		username: '',
		password: '',
		usernameSelector: '',
		passwordSelector: '',
		submitSelector: '',
		successStrategy: 'auto',
		successSelector: ''
	});
	const isAuthValid = isAuthConfigComplete(authConfig);

	const [aiConfig, setAiConfig] = useState<AiConfigState>(DEFAULT_AI_CONFIG);
	const isAiNavigatorEnabled = selections.some((s) => s.id === 'ai-navigator' && s.enabled);

	useEffect(() => {
		const controller = new AbortController();
		Promise.all([
			fetchScanners(controller.signal),
			projectId ? getLocalProject(projectId) : Promise.resolve(null)
		])
			.then(([res, storedProject]) => {
				if (controller.signal.aborted) return;
				setCatalog(res.scanners);
				if (projectId && !storedProject) {
					setProject(null);
					setSelections(getDefaultScannerSelections(res.scanners));
					setError('That local project was not found. It may have been deleted in this browser.');
				} else if (storedProject) {
					setProject(storedProject);
					setProjectName(storedProject.name);
					setMode('url');
					setUrls([...storedProject.configuration.urls, '']);
					setSelections(mergeProjectSelections(res.scanners, storedProject.configuration.scanners));
					setAiConfig(aiStateFromSelections(storedProject.configuration.scanners));
				} else {
					setSelections(getDefaultScannerSelections(res.scanners));
				}
				setCatalogError(null);
			})
			.catch((err: unknown) => {
				if (controller.signal.aborted) return;
				setCatalogError(err instanceof Error ? err.message : 'Failed to load scanners.');
			})
			.finally(() => {
				if (!controller.signal.aborted) setCatalogLoading(false);
			});
		return () => controller.abort();
	}, [projectId]);

	const enabledById = useMemo(() => {
		const map = new Map<string, boolean>();
		for (const s of selections) map.set(s.id, s.enabled);
		return map;
	}, [selections]);

	const armed = selections.filter((s) => s.enabled).length;
	const total = selections.length;

	function toggleScanner(id: string) {
		setSelections((prev) => prev.map((s) => (s.id === id ? { ...s, enabled: !s.enabled } : s)));
	}

	function setAllScanners(enabled: boolean) {
		setSelections((prev) => prev.map((s) => ({ ...s, enabled })));
	}

	function updateUrl(index: number, value: string) {
		setUrls((prev) => prev.map((u, i) => (i === index ? value : u)));
		setUrlRowErrors((prev) => {
			if (!(index in prev)) return prev;
			const next = { ...prev };
			delete next[index];
			return next;
		});
	}

	function addUrlRow() {
		setUrls((prev) => [...prev, '']);
	}

	function removeUrlRow(index: number) {
		setUrls((prev) => (prev.length <= 1 ? prev : prev.filter((_, i) => i !== index)));
		setUrlRowErrors({});
	}

	function onFilePicked(f: File | null) {
		if (!f) {
			setFile(null);
			return;
		}
		const problem = validateZipUploadFile(f);
		if (problem) {
			setError(problem);
			setFile(null);
			return;
		}
		setError(null);
		setFile(f);
	}

	function selectionsForCurrentConfiguration(): ScannerSelection[] {
		return isAiNavigatorEnabled
			? selections.map((selection) =>
					selection.id === 'ai-navigator'
						? { ...selection, config: buildAiNavigatorConfig(aiConfig) }
						: selection
				)
			: selections;
	}

	function selectionsForProjectPersistence(current: ScannerSelection[]): ScannerSelection[] {
		return project
			? preserveUnavailableProjectSelections(catalog, current, project.configuration.scanners)
			: current;
	}

	function collectValidUrls(): { valid: string[]; rowErrors: Record<number, string> } {
		const rowErrors: Record<number, string> = {};
		const valid: string[] = [];
		urls.forEach((raw, index) => {
			const normalized = normalizeUrlInput(raw);
			if (!normalized) return;
			const result = validateHttpUrls([normalized]);
			if (result.invalid.length > 0) {
				rowErrors[index] = result.invalid[0].reason;
			} else {
				valid.push(result.valid[0]);
			}
		});
		return { valid, rowErrors };
	}

	function currentProjectConfiguration(
		validUrls: string[],
		scanners: ScannerSelection[]
	): LocalProjectConfiguration {
		return {
			urls: validUrls,
			scanners,
			browser: project?.configuration.browser ?? 'chromium',
			highlightStyle: project?.configuration.highlightStyle ?? 'solid'
		};
	}

	async function saveProjectConfiguration() {
		if (!project) return;
		setProjectNotice(null);
		setError(null);
		const { valid, rowErrors } = collectValidUrls();
		setUrlRowErrors(rowErrors);
		if (Object.keys(rowErrors).length > 0) return;
		if (valid.length === 0) {
			setError('Enter at least one URL before saving this project.');
			return;
		}
		if (!projectName.trim()) {
			setError('Give this project a name before saving.');
			return;
		}

		setSavingProject(true);
		try {
			const currentSelections = selectionsForCurrentConfiguration();
			const saved = await saveLocalProject({
				...project,
				name: projectName,
				configuration: currentProjectConfiguration(
					valid,
					selectionsForProjectPersistence(currentSelections)
				)
			});
			setProject(saved);
			setProjectName(saved.name);
			setProjectNotice('Project configuration saved locally.');
		} catch (saveError) {
			setError(saveError instanceof Error ? saveError.message : 'Could not save this project.');
		} finally {
			setSavingProject(false);
		}
	}

	async function runScan() {
		setError(null);
		setUrlRowErrors(validation.urlRowErrors);
		if (!validation.ready) {
			setError(validation.error);
			if (validation.focusId) {
				requestAnimationFrame(() => document.getElementById(validation.focusId ?? '')?.focus());
			}
			return;
		}

		const validUrls = validation.validUrls;
		const auth = mode === 'url' ? buildFormAuthConfig(authConfig) : null;
		const scannersForSubmit = selectionsForCurrentConfiguration();

		setSubmitting(true);
		try {
			let configFingerprint: string | null = null;
			if (project) {
				const runConfiguration = currentProjectConfiguration(validUrls, scannersForSubmit);
				const savedConfiguration = currentProjectConfiguration(
					validUrls,
					selectionsForProjectPersistence(scannersForSubmit)
				);
				const saved = await saveLocalProject({
					...project,
					name: projectName,
					configuration: savedConfiguration
				});
				setProject(saved);
				configFingerprint = await fingerprintProjectConfiguration(runConfiguration);
			}
			const { job_id } = await submitScanJob({
				mode,
				file,
				urls: validUrls,
				scanners: scannersForSubmit,
				highlightStyle: project?.configuration.highlightStyle ?? 'solid',
				engine: project?.configuration.browser ?? 'chromium',
				auth
			});
			if (project && configFingerprint) {
				try {
					await saveLocalRun({
						jobId: job_id,
						projectId: project.id,
						configFingerprint,
						status: 'submitted',
						createdAt: new Date().toISOString()
					});
				} catch {
					// The scan is already running. Preserve project context in the URL so the
					// report can still offer a recovery path if browser storage was exhausted.
				}
			}
			const projectQuery = project ? `?project=${encodeURIComponent(project.id)}` : '';
			navigate(`/scan/${job_id}${projectQuery}`);
		} catch (err: unknown) {
			setError(err instanceof Error ? err.message : 'Scan failed. Please try again.');
			setSubmitting(false);
		}
	}

	const targetCount = mode === 'url' ? urls.filter((u) => u.trim()).length : file ? 1 : 0;
	const validation = validatePlaygroundConfiguration({
		mode,
		urls,
		file,
		selections,
		auth: authConfig,
		ai: aiConfig,
		aiEnabled: isAiNavigatorEnabled,
		catalogLoading,
		catalogError,
		projectName: project ? projectName : undefined
	});
	const ready = validation.ready;
	const readyDetail = validation.message;
	const runtimeEstimate = estimateScanRuntime(catalog, selections, targetCount, mode);

	return (
		<>
			<SiteHeader current="playground" />

			<main id="main" className="console">
				<div className="wrap wrap--app">
					<div className="console__head">
						<div>
							{project ? (
								<>
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
							<p>
								{project
									? 'Configure repeatable URL scans here. Credentials and AI input values are prompted per run and never saved with the project.'
									: 'Point StageFlow at any public URL or a static-site archive, choose the scanners you need, and run. No account, no install.'}
							</p>
						</div>
						<div className="console__head-actions">
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
															className="rm"
															aria-label={`Remove URL ${i + 1}`}
															onClick={() => removeUrlRow(i)}
														>
															✕
														</button>
													</div>
												))}
											</div>
											<button type="button" className="addmore" onClick={addUrlRow}>
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
												className="chanhead__bulk-btn"
												onClick={() => setAllScanners(armed < total)}
											>
												{armed < total ? 'Enable all' : 'Disable all'}
											</button>
										)}
									</div>
								</div>
								<div className="panel__body" style={{ padding: 0 }}>
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
														<span className="tog" aria-hidden="true" />
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

							{isAiNavigatorEnabled && (
								<PlaygroundAiConfig config={aiConfig} onConfigChange={setAiConfig} />
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
										onClick={runScan}
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
						onClick={runScan}
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
